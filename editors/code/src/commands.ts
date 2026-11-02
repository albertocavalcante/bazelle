/**
 * Command registration for Bazelle extension
 */

import { homedir } from "node:os";
import * as vscode from "vscode";
import { getConfiguration } from "./configuration";
import type { BazelleService, CommandResult } from "./services/bazelle";
import type { ResolvedBinary } from "./services/binaryResolver";
import { DaemonClient } from "./services/daemon";
import type { WatchService } from "./services/watch";
import type { StatusBarManager } from "./ui/statusBar";
import type { BazelWorkspaces } from "./workspace";

export interface CommandContext {
  bazelle: BazelleService;
  watch: WatchService;
  output: vscode.OutputChannel;
  statusBar: StatusBarManager;
  workspaces: BazelWorkspaces;
  downloadBinary: () => Promise<ResolvedBinary | undefined>;
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

export function registerCommands(context: vscode.ExtensionContext, ctx: CommandContext): void {
  const { bazelle, watch, output, statusBar, workspaces } = ctx;
  const pickWorkspaceFolder = () => workspaces.pickFolder();

  const register = (id: string, handler: () => unknown) =>
    context.subscriptions.push(vscode.commands.registerCommand(id, handler));

  const createDaemonClient = () =>
    new DaemonClient(output, { socketPath: getConfiguration().daemonSocketPath });

  const isDaemonAvailable = async () => {
    const client = createDaemonClient();
    try {
      return await client.isAvailable();
    } finally {
      client.dispose();
    }
  };

  /** Run a one-off CLI command with cancellable progress and report the result. */
  const runCli =
    (
      title: string,
      done: string,
      action: (cwd: string, signal: AbortSignal) => Promise<CommandResult>,
    ) =>
    async () => {
      const cwd = await pickWorkspaceFolder();
      if (!cwd) return;
      const result = await statusBar.track(() =>
        vscode.window.withProgress(
          { location: vscode.ProgressLocation.Notification, title, cancellable: true },
          (_progress, token) => {
            const abort = new AbortController();
            token.onCancellationRequested(() => abort.abort());
            return action(cwd, abort.signal);
          },
        ),
      );
      if (result.success) {
        void vscode.window.showInformationMessage(done);
      } else if (result.error !== "Cancelled") {
        void vscode.window.showErrorMessage(
          `${title.replace(/\.\.\.$/u, "")} failed: ${result.error}`,
        );
      }
    };

  register(
    "bazelle.update",
    runCli("Updating BUILD files...", "BUILD files updated", (cwd, signal) =>
      bazelle.update(cwd, {}, signal),
    ),
  );
  register(
    "bazelle.updateIncremental",
    runCli("Updating BUILD files (incremental)...", "BUILD files updated", (cwd, signal) =>
      bazelle.update(cwd, { incremental: true }, signal),
    ),
  );
  register(
    "bazelle.fix",
    runCli("Fixing BUILD files...", "BUILD files fixed", (cwd, signal) =>
      bazelle.fix(cwd, {}, signal),
    ),
  );
  register(
    "bazelle.fixDryRun",
    runCli("Fix (dry run)...", "Dry run complete - check output", (cwd, signal) => {
      output.show(true);
      return bazelle.fix(cwd, { dryRun: true }, signal);
    }),
  );
  register(
    "bazelle.init",
    runCli("Initializing Bazel project...", "Project initialized", (cwd, signal) =>
      bazelle.init(cwd, signal),
    ),
  );

  register("bazelle.status", async () => {
    const cwd = await pickWorkspaceFolder();
    if (!cwd) return;

    const result = await statusBar.track(() => bazelle.status(cwd));
    output.show(true);
    if (result.summary) {
      const { stale, staleDirs, error } = result.summary;
      const msg = error
        ? `Status unknown: ${error}. Run "Bazelle: Update BUILD Files" first.`
        : stale
          ? `${staleDirs.length} directories have stale BUILD files`
          : "All BUILD files up to date";
      void vscode.window.showInformationMessage(msg);
    } else {
      void vscode.window.showErrorMessage(`Status failed: ${result.error}`);
    }
  });

  register("bazelle.watch", async () => {
    if (watch.isRunning) {
      void vscode.window.showInformationMessage("Watch mode already running");
      return;
    }
    const cwd = await pickWorkspaceFolder();
    if (!cwd) return;

    try {
      await watch.start(cwd);
      if (watch.mode !== "none") {
        void vscode.window.showInformationMessage(`Watch mode started (${watch.mode})`);
      }
    } catch (err) {
      void vscode.window.showErrorMessage(`Watch failed: ${err}`);
    }
  });

  register("bazelle.stopWatch", async () => {
    if (!watch.isRunning) {
      void vscode.window.showInformationMessage("Watch mode not running");
      return;
    }
    await watch.stop();
    void vscode.window.showInformationMessage("Watch mode stopped");
  });

  register("bazelle.downloadBinary", () => ctx.downloadBinary());

  register("bazelle.showOutput", () => output.show());

  // ========================================
  // Daemon commands
  // ========================================

  register("bazelle.daemon.start", async () => {
    output.appendLine("Starting bazelle daemon...");
    if (await isDaemonAvailable()) {
      statusBar.setDaemonConnected(true);
      void vscode.window.showInformationMessage("Daemon is already running");
      return;
    }

    // The daemon is shared across workspaces; any folder (or home) works as cwd.
    const cwd = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? homedir();
    const result = await statusBar.track(() =>
      vscode.window.withProgress(
        { location: vscode.ProgressLocation.Notification, title: "Starting daemon..." },
        () => bazelle.startDaemon(cwd, getConfiguration().daemonSocketPath),
      ),
    );
    if (!result.success) {
      void vscode.window.showErrorMessage(`Failed to start daemon: ${result.error}`);
      return;
    }

    const running = await isDaemonAvailable();
    statusBar.setDaemonConnected(running);
    if (running) {
      void vscode.window.showInformationMessage("Daemon started successfully");
    } else {
      void vscode.window.showWarningMessage(
        "Daemon process started but not responding. Check output for details.",
      );
    }
  });

  /** Ask the daemon to shut down; resolves true once it no longer answers. */
  const shutdownDaemon = async (): Promise<boolean> => {
    const client = createDaemonClient();
    try {
      await client.connect();
      const result = await client.shutdown();
      output.appendLine(`Daemon shutdown: ${result.message}`);
    } finally {
      client.dispose();
    }
    for (let i = 0; i < 20; i++) {
      if (!(await isDaemonAvailable())) return true;
      await sleep(150);
    }
    return false;
  };

  register("bazelle.daemon.stop", async () => {
    output.appendLine("Stopping bazelle daemon...");
    try {
      const stopped = await shutdownDaemon();
      statusBar.setDaemonConnected(!stopped);
      if (stopped) void vscode.window.showInformationMessage("Daemon stopped");
      else void vscode.window.showWarningMessage("Daemon acknowledged shutdown but still answers");
    } catch (err) {
      output.appendLine(`Failed to stop daemon: ${err}`);
      void vscode.window.showErrorMessage(
        `Failed to stop daemon: ${err}. The daemon may not be running.`,
      );
    }
  });

  register("bazelle.daemon.restart", async () => {
    output.appendLine("Restarting bazelle daemon...");
    try {
      if (!(await shutdownDaemon())) {
        void vscode.window.showErrorMessage("Daemon did not shut down; not restarting");
        return;
      }
    } catch {
      output.appendLine("Daemon was not running");
    }
    await vscode.commands.executeCommand("bazelle.daemon.start");
  });

  type ActionItem = vscode.QuickPickItem & { command?: string };
  const pickAction = async (title: string, items: ActionItem[]) => {
    const pick = await vscode.window.showQuickPick(items, {
      title,
      placeHolder: "Select an action",
    });
    if (pick?.command) void vscode.commands.executeCommand(pick.command);
  };
  const separator: ActionItem = { label: "", kind: vscode.QuickPickItemKind.Separator };

  register("bazelle.daemon.status", async () => {
    output.appendLine("Checking daemon status...");
    const client = createDaemonClient();
    let items: ActionItem[];
    try {
      await client.connect();
      const ping = await client.ping();
      const watchStatus = await client.watchStatus();
      statusBar.setDaemonConnected(true);
      items = [
        {
          label: "$(check) Daemon Running",
          description: `Version: ${ping.version}`,
          detail: `Uptime: ${ping.uptime}, Started: ${ping.start_time}`,
        },
        watchStatus.watching
          ? {
              label: "$(eye) Watch Active",
              description: `Watching ${watchStatus.paths?.join(", ") ?? "no paths"}`,
            }
          : { label: "$(eye-closed) Watch Inactive", description: "Not currently watching" },
        separator,
        { label: "$(stop) Stop Daemon", command: "bazelle.daemon.stop" },
        { label: "$(refresh) Restart Daemon", command: "bazelle.daemon.restart" },
      ];
    } catch (err) {
      statusBar.setDaemonConnected(false);
      items = [
        {
          label: "$(error) Daemon Not Running",
          description: "Could not connect to daemon",
          detail: String(err),
        },
        separator,
        { label: "$(play) Start Daemon", command: "bazelle.daemon.start" },
      ];
    } finally {
      client.dispose();
    }
    await pickAction("Bazelle Daemon Status", items);
  });

  register("bazelle.showStatusMenu", async () => {
    const daemonRunning = await isDaemonAvailable();
    await pickAction("Bazelle", [
      {
        label: "$(sync) Update BUILD Files",
        description: "bazelle update",
        command: "bazelle.update",
      },
      {
        label: "$(sync) Update (Incremental)",
        description: "bazelle update --incremental",
        command: "bazelle.updateIncremental",
      },
      { label: "$(tools) Fix BUILD Files", description: "bazelle fix", command: "bazelle.fix" },
      { label: "$(eye) Check Status", description: "bazelle status", command: "bazelle.status" },
      separator,
      watch.isRunning
        ? {
            label: "$(stop) Stop Watch",
            description: `Stop watching (${watch.mode})`,
            command: "bazelle.stopWatch",
          }
        : {
            label: "$(eye) Start Watch",
            description: "Watch for changes",
            command: "bazelle.watch",
          },
      separator,
      {
        label: "$(server) Daemon Status",
        description: daemonRunning ? "Running" : "Not running",
        command: "bazelle.daemon.status",
      },
      daemonRunning
        ? { label: "$(stop-circle) Stop Daemon", command: "bazelle.daemon.stop" }
        : { label: "$(play) Start Daemon", command: "bazelle.daemon.start" },
      separator,
      {
        label: "$(cloud-download) Download Binary",
        description: "Download bazelle from GitHub",
        command: "bazelle.downloadBinary",
      },
      { label: "$(output) Show Output", command: "bazelle.showOutput" },
    ]);
  });
}
