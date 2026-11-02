/**
 * Tracks which workspace folders are Bazel repository roots, exposes that as
 * the `bazelle.isBazelWorkspace` context key, and picks the folder a command
 * or auto-watch applies to.
 */

import * as vscode from "vscode";
import { BAZEL_ROOT_MARKERS, chooseFolder, isBazelRoot } from "./services/workspaceFolders";

export const BAZEL_WORKSPACE_CONTEXT = "bazelle.isBazelWorkspace";

export class BazelWorkspaces implements vscode.Disposable {
  readonly #onDidChange = new vscode.EventEmitter<void>();
  readonly #disposables: vscode.Disposable[] = [];
  #roots = new Set<string>();
  #refreshing: Promise<void> = Promise.resolve();
  #refreshes = 0;

  readonly onDidChange = this.#onDidChange.event;

  constructor() {
    // Rescans are a few stats per folder, so a nested marker rescanning too is fine.
    const markers = vscode.workspace.createFileSystemWatcher(
      `**/{${BAZEL_ROOT_MARKERS.join(",")}}`,
      false,
      true,
      false,
    );
    this.#disposables.push(
      markers,
      markers.onDidCreate(() => void this.refresh()),
      markers.onDidDelete(() => void this.refresh()),
      vscode.workspace.onDidChangeWorkspaceFolders(() => void this.refresh()),
      this.#onDidChange,
    );
    void this.refresh();
  }

  /** Settles once the latest scan of the workspace folders has finished. */
  get ready(): Promise<void> {
    return this.#refreshing;
  }

  get hasBazelRoot(): boolean {
    return this.#roots.size > 0;
  }

  isBazelRoot(folder: vscode.WorkspaceFolder): boolean {
    return this.#roots.has(folder.uri.toString());
  }

  refresh(): Promise<void> {
    const refresh = ++this.#refreshes;
    this.#refreshing = (async () => {
      const folders = (vscode.workspace.workspaceFolders ?? []).filter(
        (folder) => folder.uri.scheme === "file",
      );
      const flags = await Promise.all(folders.map((folder) => isBazelRoot(folder.uri.fsPath)));
      if (refresh !== this.#refreshes) return; // a newer scan owns the result
      this.#roots = new Set(folders.filter((_, i) => flags[i]).map((f) => f.uri.toString()));
      this.#onDidChange.fire();
      await vscode.commands.executeCommand(
        "setContext",
        BAZEL_WORKSPACE_CONTEXT,
        this.hasBazelRoot,
      );
    })().catch(() => {
      // setContext only fails while the window is shutting down.
    });
    return this.#refreshing;
  }

  /**
   * The folder a command applies to: the active editor's folder, the only
   * folder, the only Bazel root, or the user's pick. `quiet` suppresses the
   * "no folder" warning for automatic (non-command) use.
   */
  async pickFolder(options: { quiet?: boolean } = {}): Promise<string | undefined> {
    await this.ready;
    const folders = vscode.workspace.workspaceFolders ?? [];
    const active = vscode.window.activeTextEditor?.document.uri;
    const choice = chooseFolder(
      folders,
      active && vscode.workspace.getWorkspaceFolder(active),
      (folder) => this.isBazelRoot(folder),
    );
    switch (choice.kind) {
      case "none":
        if (!options.quiet) void vscode.window.showWarningMessage("No workspace folder open");
        return undefined;
      case "folder":
        return choice.folder.uri.fsPath;
      case "ambiguous": {
        const picked = await vscode.window.showQuickPick(
          choice.candidates.map((folder) => ({
            label: folder.name,
            description: folder.uri.fsPath,
            folder,
          })),
          { placeHolder: "Select the Bazel workspace folder" },
        );
        return picked?.folder.uri.fsPath;
      }
    }
  }

  dispose(): void {
    for (const d of this.#disposables) d.dispose();
  }
}
