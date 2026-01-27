/**
 * Command registration for Bazelle extension
 */

import { spawn } from "node:child_process";
import * as vscode from "vscode";
import { getBinaryResolver, getStatusBarManager } from "./extension";
import type { BazelleService } from "./services/bazelle";
import { DaemonClient, getDaemonConfig } from "./services/daemon";
import type { WatchService } from "./services/watch";

export function registerCommands(
  context: vscode.ExtensionContext,
  bazelle: BazelleService,
  watch: WatchService,
  output: vscode.OutputChannel,
): void {
  const register = (id: string, handler: () => void | Promise<void>) =>
    context.subscriptions.push(vscode.commands.registerCommand(id, handler));

  const getWorkspace = () => {
    const folder = vscode.workspace.workspaceFolders?.[0];
    if (!folder) {
      void vscode.window.showWarningMessage("No workspace folder open");
      return undefined;
    }
    return folder.uri.fsPath;
  };

  const withProgress = async (title: string, task: () => Promise<void>) => {
    await vscode.window.withProgress(
      { location: vscode.ProgressLocation.Notification, title, cancellable: false },
      task,
    );
  };

  // Update BUILD files
  register("bazelle.update", async () => {
    const cwd = getWorkspace();
    if (!cwd) return;

    getStatusBarManager()?.setUpdating();
    try {
      await withProgress("Updating BUILD files...", async () => {
        const result = await bazelle.update(cwd);
        if (result.success) {
          void vscode.window.showInformationMessage("BUILD files updated");
        } else {
          void vscode.window.showErrorMessage(`Update failed: ${result.error}`);
        }
      });
    } finally {
      getStatusBarManager()?.setReady();
    }
  });

  // Update BUILD files (incremental)
  register("bazelle.updateIncremental", async () => {
    const cwd = getWorkspace();
    if (!cwd) return;

    getStatusBarManager()?.setUpdating();
    try {
      await withProgress("Updating BUILD files (incremental)...", async () => {
        const result = await bazelle.update(cwd, { incremental: true });
        if (result.success) {
          void vscode.window.showInformationMessage("BUILD files updated");
        } else {
          void vscode.window.showErrorMessage(`Update failed: ${result.error}`);
        }
      });
    } finally {
      getStatusBarManager()?.setReady();
    }
  });

  // Fix BUILD files
  register("bazelle.fix", async () => {
    const cwd = getWorkspace();
    if (!cwd) return;

    getStatusBarManager()?.setUpdating();
    try {
      await withProgress("Fixing BUILD files...", async () => {
        const result = await bazelle.fix(cwd);
        if (result.success) {
          void vscode.window.showInformationMessage("BUILD files fixed");
        } else {
          void vscode.window.showErrorMessage(`Fix failed: ${result.error}`);
        }
      });
    } finally {
      getStatusBarManager()?.setReady();
    }
  });

  // Fix BUILD files (dry run)
  register("bazelle.fixDryRun", async () => {
    const cwd = getWorkspace();
    if (!cwd) return;

    const result = await bazelle.fix(cwd, { dryRun: true });
    output.show();
    if (result.success) {
      void vscode.window.showInformationMessage("Dry run complete - check output");
    } else {
      void vscode.window.showErrorMessage(`Dry run failed: ${result.error}`);
    }
  });

  // Show status
  register("bazelle.status", async () => {
    const cwd = getWorkspace();
    if (!cwd) return;

    const result = await bazelle.status(cwd);
    output.show();

    if (result.success) {
      const msg =
        result.staleCount === 0
          ? "All BUILD files up to date"
          : `${result.staleCount} directories have stale BUILD files`;
      void vscode.window.showInformationMessage(msg);
    } else {
      void vscode.window.showErrorMessage(`Status failed: ${result.error}`);
    }
  });

  // Start watch mode
  register("bazelle.watch", async () => {
    const cwd = getWorkspace();
    if (!cwd) return;

    if (watch.isRunning) {
      void vscode.window.showInformationMessage("Watch mode already running");
      return;
    }

    try {
      await watch.start(cwd);
      getStatusBarManager()?.setWatching(watch.mode);
      const modeLabel = watch.mode === "daemon" ? "daemon" : "subprocess";
      void vscode.window.showInformationMessage(`Watch mode started (${modeLabel})`);
    } catch (err) {
      void vscode.window.showErrorMessage(`Watch failed: ${err}`);
    }
  });

  // Stop watch mode
  register("bazelle.stopWatch", async () => {
    if (!watch.isRunning) {
      void vscode.window.showInformationMessage("Watch mode not running");
      return;
    }
    await watch.stop();
    getStatusBarManager()?.setReady();
    void vscode.window.showInformationMessage("Watch mode stopped");
  });

  // Subscribe to watch mode changes
  context.subscriptions.push(
    watch.onModeChanged((mode) => {
      if (mode === "none") {
        getStatusBarManager()?.setReady();
      } else {
        getStatusBarManager()?.setWatching(mode);
      }
    }),
  );

  // Initialize project
  register("bazelle.init", async () => {
    const cwd = getWorkspace();
    if (!cwd) return;

    await withProgress("Initializing Bazel project...", async () => {
      const result = await bazelle.init(cwd);
      if (result.success) {
        void vscode.window.showInformationMessage("Project initialized");
      } else {
        void vscode.window.showErrorMessage(`Init failed: ${result.error}`);
      }
    });
  });

  // Download binary
  register("bazelle.downloadBinary", async () => {
    const resolver = getBinaryResolver();
    if (!resolver) return;

    const binary = await resolver.download();
    if (binary) {
      bazelle.setBinaryPath(binary.path);
      getStatusBarManager()?.setReady();
    }
  });

  // Show output
  register("bazelle.showOutput", () => output.show());

  // ========================================
  // Daemon commands
  // ========================================

  // Start daemon
  register("bazelle.daemon.start", async () => {
    const cwd = getWorkspace();
    if (!cwd) return;

    output.appendLine("Starting bazelle daemon...");

    // Check if daemon is already running
    const client = new DaemonClient(output);
    try {
      const isAvailable = await client.isAvailable();
      if (isAvailable) {
        void vscode.window.showInformationMessage("Daemon is already running");
        getStatusBarManager()?.setDaemonConnected(true);
        return;
      }
    } catch {
      // Daemon not running, continue to start it
    } finally {
      client.dispose();
    }

    // Start daemon process
    try {
      await withProgress("Starting daemon...", async () => {
        const binaryPath = bazelle.binaryPath;
        const child = spawn(binaryPath, ["daemon", "start"], {
          cwd,
          detached: true,
          stdio: "ignore",
        });
        child.unref();

        // Wait a moment for daemon to start
        await new Promise((resolve) => setTimeout(resolve, 1000));

        // Verify daemon is running
        const verifyClient = new DaemonClient(output);
        try {
          const isAvailable = await verifyClient.isAvailable();
          if (isAvailable) {
            getStatusBarManager()?.setDaemonConnected(true);
            void vscode.window.showInformationMessage("Daemon started successfully");
          } else {
            void vscode.window.showWarningMessage(
              "Daemon process started but not responding. Check output for details.",
            );
          }
        } finally {
          verifyClient.dispose();
        }
      });
    } catch (err) {
      output.appendLine(`Failed to start daemon: ${err}`);
      void vscode.window.showErrorMessage(`Failed to start daemon: ${err}`);
    }
  });

  // Stop daemon
  register("bazelle.daemon.stop", async () => {
    output.appendLine("Stopping bazelle daemon...");

    const client = new DaemonClient(output);
    try {
      await client.connect();
      const result = await client.shutdown();
      output.appendLine(`Daemon shutdown: ${result.message}`);
      getStatusBarManager()?.setDaemonConnected(false);
      void vscode.window.showInformationMessage("Daemon stopped");
    } catch (err) {
      output.appendLine(`Failed to stop daemon: ${err}`);
      void vscode.window.showErrorMessage(
        `Failed to stop daemon: ${err}. The daemon may not be running.`,
      );
    } finally {
      client.dispose();
    }
  });

  // Daemon status
  register("bazelle.daemon.status", async () => {
    output.appendLine("Checking daemon status...");

    const config = getDaemonConfig();
    const client = new DaemonClient(output, {
      socketPath: config.socketPath,
    });

    try {
      await client.connect();
      const pingResult = await client.ping();
      const watchStatus = await client.watchStatus();

      getStatusBarManager()?.setDaemonConnected(true);

      const items: vscode.QuickPickItem[] = [
        {
          label: "$(check) Daemon Running",
          description: `Version: ${pingResult.version}`,
          detail: `Uptime: ${pingResult.uptime}, Started: ${pingResult.start_time}`,
        },
      ];

      if (watchStatus.watching) {
        items.push({
          label: "$(eye) Watch Active",
          description: `Watching ${watchStatus.file_count ?? 0} files`,
          detail: watchStatus.paths?.join(", ") ?? "No paths",
        });
      } else {
        items.push({
          label: "$(eye-closed) Watch Inactive",
          description: "Not currently watching",
        });
      }

      items.push(
        { label: "", kind: vscode.QuickPickItemKind.Separator },
        { label: "$(stop) Stop Daemon", description: "Shutdown the daemon" },
        { label: "$(refresh) Restart Daemon", description: "Restart the daemon" },
      );

      const pick = await vscode.window.showQuickPick(items, {
        title: "Bazelle Daemon Status",
        placeHolder: "Select an action",
      });

      if (pick?.label === "$(stop) Stop Daemon") {
        void vscode.commands.executeCommand("bazelle.daemon.stop");
      } else if (pick?.label === "$(refresh) Restart Daemon") {
        void vscode.commands.executeCommand("bazelle.daemon.restart");
      }
    } catch (err) {
      getStatusBarManager()?.setDaemonConnected(false);

      const items: vscode.QuickPickItem[] = [
        {
          label: "$(error) Daemon Not Running",
          description: "Could not connect to daemon",
          detail: String(err),
        },
        { label: "", kind: vscode.QuickPickItemKind.Separator },
        { label: "$(play) Start Daemon", description: "Start the daemon" },
      ];

      const pick = await vscode.window.showQuickPick(items, {
        title: "Bazelle Daemon Status",
        placeHolder: "Select an action",
      });

      if (pick?.label === "$(play) Start Daemon") {
        void vscode.commands.executeCommand("bazelle.daemon.start");
      }
    } finally {
      client.dispose();
    }
  });

  // Restart daemon
  register("bazelle.daemon.restart", async () => {
    output.appendLine("Restarting bazelle daemon...");

    // Stop first
    const client = new DaemonClient(output);
    try {
      await client.connect();
      await client.shutdown();
      output.appendLine("Daemon stopped for restart");
    } catch {
      output.appendLine("Daemon was not running");
    } finally {
      client.dispose();
    }

    // Wait for shutdown
    await new Promise((resolve) => setTimeout(resolve, 500));

    // Start again
    void vscode.commands.executeCommand("bazelle.daemon.start");
  });

  // Status menu (updated to include daemon options)
  register("bazelle.showStatusMenu", async () => {
    const daemonClient = new DaemonClient(output);
    let daemonRunning = false;

    try {
      daemonRunning = await daemonClient.isAvailable();
    } catch {
      // Ignore
    } finally {
      daemonClient.dispose();
    }

    const items: vscode.QuickPickItem[] = [
      { label: "$(sync) Update BUILD Files", description: "bazelle update" },
      { label: "$(sync) Update (Incremental)", description: "bazelle update --incremental" },
      { label: "$(tools) Fix BUILD Files", description: "bazelle fix" },
      { label: "$(eye) Check Status", description: "bazelle status" },
      { label: "", kind: vscode.QuickPickItemKind.Separator },
      {
        label: watch.isRunning ? "$(stop) Stop Watch" : "$(eye) Start Watch",
        description: watch.isRunning ? `Stop watching (${watch.mode})` : "Watch for changes",
      },
      { label: "", kind: vscode.QuickPickItemKind.Separator },
      {
        label: "$(server) Daemon Status",
        description: daemonRunning ? "Running" : "Not running",
      },
      {
        label: daemonRunning ? "$(stop-circle) Stop Daemon" : "$(play) Start Daemon",
        description: daemonRunning ? "Shutdown daemon" : "Start daemon",
      },
      { label: "", kind: vscode.QuickPickItemKind.Separator },
      { label: "$(cloud-download) Download Binary", description: "Download bazelle from GitHub" },
      { label: "$(output) Show Output", description: "Open output channel" },
    ];

    const pick = await vscode.window.showQuickPick(items, {
      title: "Bazelle",
      placeHolder: "Select an action",
    });

    if (!pick) return;

    const commands: Record<string, string> = {
      "$(sync) Update BUILD Files": "bazelle.update",
      "$(sync) Update (Incremental)": "bazelle.updateIncremental",
      "$(tools) Fix BUILD Files": "bazelle.fix",
      "$(eye) Check Status": "bazelle.status",
      "$(eye) Start Watch": "bazelle.watch",
      "$(stop) Stop Watch": "bazelle.stopWatch",
      "$(server) Daemon Status": "bazelle.daemon.status",
      "$(play) Start Daemon": "bazelle.daemon.start",
      "$(stop-circle) Stop Daemon": "bazelle.daemon.stop",
      "$(cloud-download) Download Binary": "bazelle.downloadBinary",
      "$(output) Show Output": "bazelle.showOutput",
    };

    const cmd = commands[pick.label];
    if (cmd) void vscode.commands.executeCommand(cmd);
  });
}
