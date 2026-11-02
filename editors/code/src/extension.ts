/**
 * Bazelle VSCode Extension
 */

import * as vscode from "vscode";
import { registerCommands } from "./commands";
import { getConfiguration } from "./configuration";
import { BazelleService } from "./services/bazelle";
import { BinaryResolver, type ResolvedBinary } from "./services/binaryResolver";
import { formatWatchEvent, updateCount } from "./services/cliOutput";
import { coalesce } from "./services/coalesce";
import { WatchService } from "./services/watch";
import { StatusBarManager } from "./ui/statusBar";
import { BazelWorkspaces } from "./workspace";

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const version = (context.extension.packageJSON.version as string | undefined) ?? "dev";
  const subscriptions = context.subscriptions;

  const output = vscode.window.createOutputChannel("Bazelle");
  subscriptions.push(output);
  output.appendLine(`Bazelle extension v${version} activating...`);

  const resolver = new BinaryResolver(
    context.extensionPath,
    context.globalStorageUri.fsPath,
    output,
  );
  const workspaces = new BazelWorkspaces();
  subscriptions.push(workspaces);
  const statusBar = new StatusBarManager(version, workspaces);
  const bazelle = new BazelleService("bazelle", output);
  const watch = new WatchService(bazelle, output);
  subscriptions.push(statusBar, watch);
  subscriptions.push(
    watch.onModeChanged((mode) => statusBar.setWatchMode(mode)),
    watch.onEvent((event) => {
      output.appendLine(formatWatchEvent(event));
      statusBar.addWatchUpdates(updateCount(event));
    }),
  );

  let hasBinary = false;
  const applyBinary = (binary: ResolvedBinary | undefined): ResolvedBinary | undefined => {
    if (binary) {
      bazelle.setBinaryPath(binary.path);
      output.appendLine(`Using binary: ${binary.path} (source: ${binary.source})`);
      statusBar.setError(undefined);
      if (!hasBinary) {
        hasBinary = true;
        startConfiguredWatch().catch((err) => output.appendLine(`Auto-watch failed: ${err}`));
      }
    } else {
      statusBar.setError(
        "bazelle CLI not found. Set bazelle.path or run Bazelle: Download Binary.",
      );
    }
    return binary;
  };

  const startConfiguredWatch = async () => {
    if (!getConfiguration().watchEnabled || watch.isRunning) return;
    // Same choice as the watch command, so multi-root workspaces do not
    // silently watch whichever folder happens to be first.
    const cwd = await workspaces.pickFolder({ quiet: true });
    if (!cwd) {
      output.appendLine("Auto-watch skipped: no single Bazel folder; run Start Watch Mode");
      return;
    }
    if (watch.isRunning) return;
    await watch.start(cwd);
  };

  // Register everything before resolving the binary so commands work (and
  // report a missing CLI) even while the download prompt is still open.
  registerCommands(context, {
    bazelle,
    watch,
    output,
    statusBar,
    workspaces,
    downloadBinary: async () => applyBinary(await resolver.download()),
  });

  subscriptions.push(
    vscode.workspace.onDidChangeConfiguration(async (e) => {
      if (!e.affectsConfiguration("bazelle.path")) return;
      const previous = bazelle.binaryPath;
      const binary = applyBinary(await resolver.resolve());
      if (binary && binary.path !== previous && watch.restartSubprocess()) {
        output.appendLine(`Watch restarted with ${binary.path}`);
      }
    }),
  );

  const autoUpdate = coalesce(async (cwd: string) => {
    const result = await statusBar.track(() => bazelle.update(cwd, { incremental: true }));
    if (!result.success) output.appendLine(`Auto-update failed: ${result.error}`);
  });
  subscriptions.push(
    vscode.workspace.onDidSaveTextDocument((document) => {
      if (!hasBinary || !getConfiguration().autoUpdate || watch.isRunning) return;
      const folder = vscode.workspace.getWorkspaceFolder(document.uri);
      if (folder && document.uri.scheme === "file") autoUpdate(folder.uri.fsPath);
    }),
  );

  if (!applyBinary(await resolver.resolve())) {
    output.appendLine("No binary found, prompting user...");
    // Not awaited: an unanswered notification must not keep activation pending.
    void resolver.promptDownload().then(applyBinary);
  }

  output.appendLine("Bazelle extension activated");
}

export function deactivate(): void {
  // Everything is disposed through context.subscriptions.
}
