/**
 * Watch service - manages bazelle watch via daemon or subprocess
 *
 * Attempts to connect to daemon first, falls back to spawning subprocess.
 */

import type { ChildProcess } from "node:child_process";
import { StringDecoder } from "node:string_decoder";
import * as vscode from "vscode";
import { type BazelleConfig, getConfiguration } from "../configuration";
import type { BazelleService } from "./bazelle";
import { LineBuffer, parseWatchEvent, type WatchEvent } from "./cliOutput";
import { DaemonClient, type DaemonConnectionState, type WatchEventParams } from "./daemon";

export type { WatchEvent } from "./cliOutput";
export type WatchMode = "daemon" | "subprocess" | "none";

export interface WatchDependencies {
  getConfig: () => Pick<
    BazelleConfig,
    "daemonEnabled" | "daemonSocketPath" | "daemonAutoStart" | "watchDebounceMs"
  >;
  createDaemonClient: (socketPath: string | undefined) => DaemonClient;
  /** Surface a problem the user did not ask about (a lost daemon watch). */
  warn: (message: string) => void;
}

/** Thrown inside a start attempt that a later stop()/start() superseded. */
class Superseded extends Error {}

export class WatchService implements vscode.Disposable {
  readonly #bazelle: BazelleService;
  readonly #output: vscode.OutputChannel;
  readonly #onEvent = new vscode.EventEmitter<WatchEvent>();
  readonly #onModeChanged = new vscode.EventEmitter<WatchMode>();

  #process: ChildProcess | null = null;
  #daemonClient: DaemonClient | null = null;
  // The daemon has one shared watcher; only stop it if this window started it.
  #ownsDaemonWatch = false;
  #daemonSubscriptions: vscode.Disposable[] = [];
  #mode: WatchMode = "none";
  #cwd: string | null = null;
  #starting = false;
  // Bumped by every start() and stop(); async work from an older generation
  // must not touch current state.
  #generation = 0;

  readonly onEvent = this.#onEvent.event;
  readonly onModeChanged = this.#onModeChanged.event;

  readonly #deps: WatchDependencies;

  constructor(
    bazelle: BazelleService,
    output: vscode.OutputChannel,
    deps: WatchDependencies = {
      getConfig: getConfiguration,
      createDaemonClient: (socketPath) => new DaemonClient(output, { socketPath }),
      warn: (message) => void vscode.window.showWarningMessage(message),
    },
  ) {
    this.#bazelle = bazelle;
    this.#output = output;
    this.#deps = deps;
  }

  get isRunning(): boolean {
    return this.#mode !== "none" || this.#starting;
  }

  get mode(): WatchMode {
    return this.#mode;
  }

  /**
   * Start watch mode - tries daemon first, falls back to subprocess
   */
  async start(cwd: string): Promise<void> {
    if (this.isRunning) {
      throw new Error("Watch mode already running");
    }

    const generation = ++this.#generation;
    this.#starting = true;
    this.#cwd = cwd;
    const config = this.#deps.getConfig();

    try {
      if (config.daemonEnabled) {
        this.#output.appendLine("Attempting to connect to daemon...");
        try {
          await this.#startDaemon(cwd, generation);
          return;
        } catch (err) {
          if (err instanceof Superseded) return;
          this.#output.appendLine(`Daemon connection failed: ${err}`);
          if (generation !== this.#generation) return;
          this.#output.appendLine("Falling back to subprocess mode...");
        }
      }
      this.#startSubprocess(cwd);
    } finally {
      if (generation === this.#generation) this.#starting = false;
    }
  }

  /**
   * Start watch via daemon connection
   */
  async #startDaemon(cwd: string, generation: number): Promise<void> {
    const { daemonSocketPath, daemonAutoStart, watchDebounceMs } = this.#deps.getConfig();
    const client = this.#deps.createDaemonClient(daemonSocketPath);
    const subscriptions = [
      client.onWatchEvent((event) => {
        if (this.#daemonClient === client) this.#handleDaemonEvent(event);
      }),
      client.onConnectionStateChanged((state) => {
        if (this.#daemonClient === client) this.#handleConnectionStateChange(state);
      }),
      client.onError((err) => {
        if (this.#daemonClient !== client) return;
        this.#onEvent.fire({ type: "error", message: err.message });
      }),
    ];
    let owns = false;
    const checkCurrent = () => {
      if (generation !== this.#generation) throw new Superseded("watch start superseded");
    };

    try {
      try {
        await client.connect();
      } catch (err) {
        if (!daemonAutoStart) throw err;
        checkCurrent();
        this.#output.appendLine("[Daemon] Not running, starting it (bazelle.daemon.autoStart)");
        const started = await this.#bazelle.startDaemon(cwd, daemonSocketPath);
        if (!started.success) throw new Error(`daemon start failed: ${started.error}`);
        checkCurrent();
        await client.connect();
      }
      checkCurrent();

      const pingResult = await client.ping();
      checkCurrent();
      this.#output.appendLine(
        `[Daemon] Connected to bazelle daemon v${pingResult.version} (uptime: ${pingResult.uptime})`,
      );

      const result = await client.watchStart({ paths: [cwd], debounce: watchDebounceMs });
      owns = result.status === "watching";
      checkCurrent();
      requireWatchingPath(result.paths, cwd);

      this.#output.appendLine(`[Daemon] Watch started: ${result.status}`);
      this.#output.appendLine(`[Daemon] Watching paths: ${result.paths.join(", ")}`);
    } catch (err) {
      for (const d of subscriptions) d.dispose();
      // A stop() that raced our watch/start must still stop what we started.
      if (owns && err instanceof Superseded) await client.watchStop().catch(() => {});
      client.dispose();
      throw err;
    }

    this.#daemonClient = client;
    this.#ownsDaemonWatch = owns;
    this.#daemonSubscriptions = subscriptions;
    this.#setMode("daemon");
    this.#onEvent.fire({ type: "started" });
  }

  #handleDaemonEvent(event: WatchEventParams): void {
    switch (event.type) {
      case "change":
        this.#onEvent.fire({
          type: "file_change",
          directories: event.directories,
          files: event.files,
        });
        break;
      case "update":
        this.#onEvent.fire({
          type: "updated",
          directories: event.directories,
          message: event.message,
        });
        break;
      case "error":
        this.#onEvent.fire({ type: "error", message: event.message });
        break;
      case "stopped":
        // The daemon's watcher died on its own (e.g. a watch limit); restarting
        // it would likely fail the same way, so watch with a subprocess.
        this.#output.appendLine(`[Daemon] Watcher stopped: ${event.message ?? ""}`);
        if (this.#mode === "daemon" && this.#cwd) {
          this.#fallBackToSubprocess(this.#cwd, "the daemon watcher stopped");
        }
        break;
      case "shutdown":
        this.#output.appendLine(`[Daemon] Shutting down at ${event.timestamp}`);
        // Fall back now instead of after the reconnect backoff (~31s).
        if (this.#mode === "daemon" && this.#cwd) {
          this.#fallBackToSubprocess(this.#cwd, "the daemon shut down");
        }
        break;
    }
  }

  #handleConnectionStateChange(state: DaemonConnectionState): void {
    const client = this.#daemonClient;
    const cwd = this.#cwd;
    if (this.#mode !== "daemon" || !client || !cwd) return;

    switch (state) {
      case "disconnected":
        this.#output.appendLine("[Daemon] Lost connection");
        this.#fallBackToSubprocess(cwd, "lost the connection to the daemon");
        break;
      case "reconnecting":
        this.#output.appendLine("[Daemon] Attempting to reconnect...");
        break;
      case "connected":
        this.#output.appendLine("[Daemon] Reconnected");
        void client
          .watchStart({ paths: [cwd], debounce: this.#deps.getConfig().watchDebounceMs })
          .then((result) => {
            if (this.#daemonClient !== client) return;
            if (result.status === "watching") this.#ownsDaemonWatch = true;
            requireWatchingPath(result.paths, cwd);
            this.#output.appendLine("[Daemon] Watch registration restored");
          })
          .catch((err) => {
            if (this.#daemonClient !== client) return;
            this.#output.appendLine(`[Daemon] Watch registration failed: ${err}`);
            this.#fallBackToSubprocess(cwd, "the daemon did not restore the watch");
          });
        break;
    }
  }

  #fallBackToSubprocess(cwd: string, reason: string): void {
    this.#output.appendLine("Attempting fallback to subprocess mode...");
    this.#deps.warn(`Bazelle watch: ${reason}; watching with a subprocess instead.`);
    this.#disposeDaemon();
    this.#startSubprocess(cwd);
  }

  /**
   * Start watch via subprocess
   */
  #startSubprocess(cwd: string): void {
    this.#output.appendLine("Starting watch mode (subprocess)...");
    const child = this.#bazelle.watch(cwd, this.#deps.getConfig().watchDebounceMs);
    this.#process = child;

    const decoder = new StringDecoder("utf8");
    // Recognized JSON events are logged by onEvent listeners; echo the rest.
    const lines = new LineBuffer((line) => {
      let event: WatchEvent | undefined;
      try {
        event = parseWatchEvent(JSON.parse(line));
      } catch {
        // Not JSON
      }
      if (!event) this.#output.appendLine(line);
      else if (this.#process === child) this.#onEvent.fire(event);
    });

    child.stdout?.on("data", (data: Buffer) => lines.push(decoder.write(data)));

    child.stderr?.on("data", (data: Buffer) => {
      this.#output.append(data.toString());
    });

    const exited = (reason: string) => {
      if (this.#process !== child) {
        this.#output.appendLine(reason); // a replaced watcher; not an error event
        return;
      }
      this.#process = null;
      this.#onEvent.fire({ type: "error", message: reason });
      this.#setMode("none");
    };
    child.on("error", (err) => exited(`Watch error: ${err.message}`));
    child.on("close", (code) => exited(`Watch exited (code ${code})`));

    this.#setMode("subprocess");
    this.#onEvent.fire({ type: "started" });
  }

  /**
   * Respawn a subprocess watcher so it runs the current binary. A daemon
   * watch runs inside the daemon and is left alone. Returns whether it restarted.
   */
  restartSubprocess(): boolean {
    const child = this.#process;
    const cwd = this.#cwd;
    if (this.#mode !== "subprocess" || this.#starting || !child || !cwd) return false;
    this.#output.appendLine("Restarting watch mode with the new bazelle binary...");
    this.#process = null; // detach first so its exit is not reported as a failure
    this.#bazelle.kill(child);
    this.#startSubprocess(cwd);
    return true;
  }

  #setMode(mode: WatchMode): void {
    if (this.#mode !== mode) {
      this.#mode = mode;
      this.#onModeChanged.fire(mode);
    }
  }

  /**
   * Stop watch mode
   */
  async stop(): Promise<void> {
    this.#generation++;
    this.#starting = false;
    this.#cwd = null;

    const child = this.#process;
    this.#process = null;
    const client = this.#daemonClient;
    const subscriptions = this.#daemonSubscriptions;
    const ownsWatch = this.#ownsDaemonWatch;
    this.#ownsDaemonWatch = false;
    this.#daemonClient = null;
    this.#daemonSubscriptions = [];
    this.#setMode("none");

    if (child) {
      this.#output.appendLine("Stopping watch mode (subprocess)...");
      this.#bazelle.kill(child);
    }
    if (client) {
      for (const d of subscriptions) d.dispose();
      if (ownsWatch) {
        try {
          await client.watchStop();
          this.#output.appendLine("[Daemon] Watch stopped");
        } catch (err) {
          this.#output.appendLine(`[Daemon] Error stopping watch: ${err}`);
        }
      } else {
        this.#output.appendLine(
          "[Daemon] Closing the connection releases this client's share of the daemon watch",
        );
      }
      client.dispose();
    }
  }

  #disposeDaemon(): void {
    this.#ownsDaemonWatch = false;
    for (const d of this.#daemonSubscriptions) d.dispose();
    this.#daemonSubscriptions = [];
    this.#daemonClient?.dispose();
    this.#daemonClient = null;
  }

  dispose(): void {
    void this.stop();
    this.#onEvent.dispose();
    this.#onModeChanged.dispose();
  }
}

function requireWatchingPath(paths: string[], cwd: string): void {
  if (!paths.includes(cwd)) {
    throw new Error(`Daemon is watching ${paths.join(", ")} instead of ${cwd}`);
  }
}
