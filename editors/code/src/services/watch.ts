/**
 * Watch service - manages bazelle watch via daemon or subprocess
 *
 * Attempts to connect to daemon first, falls back to spawning subprocess.
 */

import type { ChildProcess } from "node:child_process";
import * as vscode from "vscode";
import type { BazelleService } from "./bazelle";
import {
  DaemonClient,
  type DaemonConnectionState,
  type WatchEventParams,
  getDaemonConfig,
} from "./daemon";

export interface WatchEvent {
  type: "started" | "updated" | "error" | "file_change";
  path?: string;
  message?: string;
  directories?: string[];
  files?: string[];
}

export type WatchMode = "daemon" | "subprocess" | "none";

export class WatchService implements vscode.Disposable {
  readonly #bazelle: BazelleService;
  readonly #output: vscode.OutputChannel;
  readonly #onEvent = new vscode.EventEmitter<WatchEvent>();
  readonly #onModeChanged = new vscode.EventEmitter<WatchMode>();

  #process: ChildProcess | null = null;
  #daemonClient: DaemonClient | null = null;
  #mode: WatchMode = "none";
  #cwd: string | null = null;
  #disposables: vscode.Disposable[] = [];

  readonly onEvent = this.#onEvent.event;
  readonly onModeChanged = this.#onModeChanged.event;

  constructor(bazelle: BazelleService, output: vscode.OutputChannel) {
    this.#bazelle = bazelle;
    this.#output = output;
  }

  get isRunning(): boolean {
    return this.#mode !== "none";
  }

  get mode(): WatchMode {
    return this.#mode;
  }

  get daemonClient(): DaemonClient | null {
    return this.#daemonClient;
  }

  /**
   * Start watch mode - tries daemon first, falls back to subprocess
   */
  async start(cwd: string): Promise<void> {
    if (this.#mode !== "none") {
      throw new Error("Watch mode already running");
    }

    this.#cwd = cwd;
    const config = getDaemonConfig();

    // Try daemon first if enabled
    if (config.enabled) {
      this.#output.appendLine("Attempting to connect to daemon...");
      try {
        await this.#startDaemon(cwd, config.socketPath);
        return;
      } catch (err) {
        this.#output.appendLine(`Daemon connection failed: ${err}`);
        this.#output.appendLine("Falling back to subprocess mode...");
      }
    }

    // Fallback to subprocess
    await this.#startSubprocess(cwd);
  }

  /**
   * Start watch via daemon connection
   */
  async #startDaemon(cwd: string, socketPath?: string): Promise<void> {
    this.#daemonClient = new DaemonClient(this.#output, {
      socketPath,
    });

    // Subscribe to daemon events
    this.#disposables.push(
      this.#daemonClient.onWatchEvent((event) => {
        this.#handleDaemonEvent(event);
      }),
    );

    this.#disposables.push(
      this.#daemonClient.onConnectionStateChanged((state) => {
        this.#handleConnectionStateChange(state);
      }),
    );

    this.#disposables.push(
      this.#daemonClient.onError((err) => {
        this.#output.appendLine(`[Daemon] Error: ${err.message}`);
        this.#onEvent.fire({ type: "error", message: err.message });
      }),
    );

    // Connect to daemon
    await this.#daemonClient.connect();

    // Verify connection with ping
    const pingResult = await this.#daemonClient.ping();
    this.#output.appendLine(
      `[Daemon] Connected to bazelle daemon v${pingResult.version} (uptime: ${pingResult.uptime})`,
    );

    // Start watching
    const result = await this.#daemonClient.watchStart({
      paths: [cwd],
    });

    this.#output.appendLine(`[Daemon] Watch started: ${result.status}`);
    this.#output.appendLine(`[Daemon] Watching paths: ${result.paths.join(", ")}`);

    this.#setMode("daemon");
    this.#onEvent.fire({ type: "started" });
  }

  #handleDaemonEvent(event: WatchEventParams): void {
    this.#output.appendLine(`[Daemon] Watch event: ${event.type} at ${event.timestamp}`);

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
        this.#onEvent.fire({
          type: "error",
          message: event.message,
        });
        break;
    }
  }

  #handleConnectionStateChange(state: DaemonConnectionState): void {
    switch (state) {
      case "disconnected":
        if (this.#mode === "daemon") {
          this.#output.appendLine("[Daemon] Lost connection");
          // Try to fall back to subprocess if we have a cwd
          if (this.#cwd) {
            this.#output.appendLine("Attempting fallback to subprocess mode...");
            this.#cleanupDaemon();
            this.#startSubprocess(this.#cwd).catch((err) => {
              this.#output.appendLine(`Subprocess fallback failed: ${err}`);
              this.#setMode("none");
            });
          } else {
            this.#setMode("none");
          }
        }
        break;
      case "reconnecting":
        this.#output.appendLine("[Daemon] Attempting to reconnect...");
        break;
      case "connected":
        if (this.#mode === "daemon") {
          this.#output.appendLine("[Daemon] Reconnected");
        }
        break;
    }
  }

  /**
   * Start watch via subprocess (existing behavior)
   */
  async #startSubprocess(cwd: string): Promise<void> {
    this.#output.appendLine("Starting watch mode (subprocess)...");
    this.#process = this.#bazelle.watch(cwd);

    this.#process.stdout?.on("data", (data: Buffer) => {
      const text = data.toString();
      this.#output.append(text);

      for (const line of text.split("\n")) {
        if (!line.trim()) continue;
        try {
          const parsed = JSON.parse(line);
          this.#onEvent.fire(this.#normalizeSubprocessEvent(parsed));
        } catch {
          // Not JSON
        }
      }
    });

    this.#process.stderr?.on("data", (data: Buffer) => {
      const text = data.toString();
      this.#output.append(text);
      this.#onEvent.fire({ type: "error", message: text });
    });

    this.#process.on("error", (err) => {
      this.#output.appendLine(`Watch error: ${err.message}`);
      this.#onEvent.fire({ type: "error", message: err.message });
      this.#process = null;
      this.#setMode("none");
    });

    this.#process.on("close", (code) => {
      this.#output.appendLine(`Watch exited (code ${code})`);
      this.#process = null;
      this.#setMode("none");
    });

    this.#setMode("subprocess");
    this.#onEvent.fire({ type: "started" });
  }

  /**
   * Normalize subprocess event to match our WatchEvent interface
   */
  #normalizeSubprocessEvent(event: Record<string, unknown>): WatchEvent {
    return {
      type: event.type as WatchEvent["type"],
      path: event.path as string | undefined,
      message: event.message as string | undefined,
      directories: event.directories as string[] | undefined,
      files: event.files as string[] | undefined,
    };
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
    if (this.#mode === "daemon" && this.#daemonClient) {
      try {
        await this.#daemonClient.watchStop();
        this.#output.appendLine("[Daemon] Watch stopped");
      } catch (err) {
        this.#output.appendLine(`[Daemon] Error stopping watch: ${err}`);
      }
      this.#cleanupDaemon();
    } else if (this.#mode === "subprocess" && this.#process) {
      this.#output.appendLine("Stopping watch mode (subprocess)...");
      this.#process.kill();
      this.#process = null;
    }

    this.#cwd = null;
    this.#setMode("none");
  }

  #cleanupDaemon(): void {
    for (const d of this.#disposables) {
      d.dispose();
    }
    this.#disposables = [];

    if (this.#daemonClient) {
      this.#daemonClient.dispose();
      this.#daemonClient = null;
    }
  }

  /**
   * Get the daemon client (for direct access)
   */
  getDaemonClient(): DaemonClient | null {
    return this.#daemonClient;
  }

  /**
   * Check if daemon is available
   */
  async isDaemonAvailable(): Promise<boolean> {
    const client = new DaemonClient(this.#output);
    try {
      return await client.isAvailable();
    } finally {
      client.dispose();
    }
  }

  dispose(): void {
    this.stop().catch(() => {
      // Ignore errors during disposal
    });
    this.#onEvent.dispose();
    this.#onModeChanged.dispose();
  }
}
