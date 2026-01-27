/**
 * Command registration for Bazelle extension
 */

import * as vscode from "vscode";
import { getBinaryResolver, getStatusBarManager } from "./extension";
import type { BazelleService } from "./services/bazelle";
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
      getStatusBarManager()?.setWatching();
      void vscode.window.showInformationMessage("Watch mode started");
    } catch (err) {
      void vscode.window.showErrorMessage(`Watch failed: ${err}`);
    }
  });

  // Stop watch mode
  register("bazelle.stopWatch", () => {
    if (!watch.isRunning) {
      void vscode.window.showInformationMessage("Watch mode not running");
      return;
    }
    watch.stop();
    getStatusBarManager()?.setReady();
    void vscode.window.showInformationMessage("Watch mode stopped");
  });

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

  // Status menu
  register("bazelle.showStatusMenu", async () => {
    const items: vscode.QuickPickItem[] = [
      { label: "$(sync) Update BUILD Files", description: "bazelle update" },
      { label: "$(sync) Update (Incremental)", description: "bazelle update --incremental" },
      { label: "$(tools) Fix BUILD Files", description: "bazelle fix" },
      { label: "$(eye) Check Status", description: "bazelle status" },
      { label: "", kind: vscode.QuickPickItemKind.Separator },
      {
        label: watch.isRunning ? "$(stop) Stop Watch" : "$(eye) Start Watch",
        description: watch.isRunning ? "Stop watching" : "Watch for changes",
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
      "$(cloud-download) Download Binary": "bazelle.downloadBinary",
      "$(output) Show Output": "bazelle.showOutput",
    };

    const cmd = commands[pick.label];
    if (cmd) void vscode.commands.executeCommand(cmd);
  });
}
