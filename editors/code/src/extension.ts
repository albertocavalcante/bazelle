/**
 * Bazelle VSCode Extension
 */

import * as vscode from "vscode";
import { registerCommands } from "./commands";
import { getConfiguration } from "./configuration";
import { BazelleService } from "./services/bazelle";
import { BinaryResolver, type ResolvedBinary } from "./services/binaryResolver";
import { coalesce } from "./services/coalesce";
import { WatchService } from "./services/watch";
import { StatusBarManager } from "./ui/statusBar";

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
  const statusBar = new StatusBarManager(version);
  const bazelle = new BazelleService("bazelle", output);
  const watch = new WatchService(bazelle, output);
  subscriptions.push(statusBar, watch);
  subscriptions.push(watch.onModeChanged((mode) => statusBar.setWatchMode(mode)));

  let hasBinary = false;
  const applyBinary = (binary: ResolvedBinary | undefined): ResolvedBinary | undefined => {
    if (binary) {
      bazelle.setBinaryPath(binary.path);
      output.appendLine(`Using binary: ${binary.path} (source: ${binary.source})`);
      statusBar.setError(undefined);
      if (!hasBinary) {
        hasBinary = true;
        startConfiguredWatch();
      }
    } else {
      statusBar.setError(
        "bazelle CLI not found. Set bazelle.path or run Bazelle: Download Binary.",
      );
    }
    return binary;
  };

  const startConfiguredWatch = () => {
    if (!getConfiguration().watchEnabled || watch.isRunning) return;
    const folder = vscode.workspace.workspaceFolders?.[0];
    if (!folder) return;
    watch.start(folder.uri.fsPath).catch((err) => {
      output.appendLine(`Auto-watch failed: ${err}`);
    });
  };

  // Register everything before resolving the binary so commands work (and
  // report a missing CLI) even while the download prompt is still open.
  registerCommands(context, {
    bazelle,
    watch,
    output,
    statusBar,
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
