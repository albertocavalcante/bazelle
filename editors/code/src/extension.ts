/**
 * Bazelle VSCode Extension
 */

import * as vscode from "vscode";
import { registerCommands } from "./commands";
import { BazelleService } from "./services/bazelle";
import { BinaryResolver } from "./services/binaryResolver";
import { WatchService } from "./services/watch";
import { StatusBarManager } from "./ui/statusBar";

let statusBar: StatusBarManager | undefined;
let bazelle: BazelleService | undefined;
let watch: WatchService | undefined;
let output: vscode.OutputChannel | undefined;
let resolver: BinaryResolver | undefined;

export const getStatusBarManager = () => statusBar;
export const getBazelleService = () => bazelle;
export const getWatchService = () => watch;
export const getOutputChannel = () => output;
export const getBinaryResolver = () => resolver;

export async function activate(context: vscode.ExtensionContext): Promise<void> {
  const version = (context.extension.packageJSON.version as string) ?? "dev";

  output = vscode.window.createOutputChannel("Bazelle");
  context.subscriptions.push(output);

  output.appendLine(`Bazelle extension v${version} activating...`);

  // Initialize binary resolver
  resolver = new BinaryResolver(context.extensionPath, output);

  // Resolve binary
  let binary = await resolver.resolve();
  if (!binary) {
    output.appendLine("No binary found, prompting user...");
    binary = await resolver.promptDownload();
  }

  if (!binary) {
    output.appendLine("No binary available, extension running in limited mode");
    statusBar = new StatusBarManager(version);
    statusBar.setError();
    context.subscriptions.push(statusBar);

    // Register commands anyway (they'll show appropriate errors)
    bazelle = new BazelleService("bazelle", output); // Placeholder
    watch = new WatchService(bazelle, output);
    registerCommands(context, bazelle, watch, output);
    return;
  }

  output.appendLine(`Using binary: ${binary.path} (source: ${binary.source})`);

  // Initialize services
  bazelle = new BazelleService(binary.path, output);
  watch = new WatchService(bazelle, output);

  // Status bar
  statusBar = new StatusBarManager(version);
  context.subscriptions.push(statusBar);

  // Commands
  registerCommands(context, bazelle, watch, output);

  // Config watcher - update binary path if setting changes
  context.subscriptions.push(
    vscode.workspace.onDidChangeConfiguration(async (e) => {
      if (e.affectsConfiguration("bazelle.path")) {
        const newBinary = await resolver?.resolve();
        if (newBinary && bazelle) {
          bazelle.setBinaryPath(newBinary.path);
          output?.appendLine(`Binary path updated: ${newBinary.path}`);
        }
      }
    }),
  );

  // Auto-start watch if configured
  const config = vscode.workspace.getConfiguration("bazelle");
  if (config.get<boolean>("watch.enabled")) {
    const folder = vscode.workspace.workspaceFolders?.[0];
    if (folder) {
      watch.start(folder.uri.fsPath).catch((err) => {
        output?.appendLine(`Auto-watch failed: ${err}`);
      });
    }
  }

  statusBar.setReady();
  output.appendLine("Bazelle extension activated");
}

export async function deactivate(): Promise<void> {
  output?.appendLine("Deactivating...");
  watch?.dispose();
}
