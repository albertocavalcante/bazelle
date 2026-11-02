/**
 * Status bar management for Bazelle extension
 */

import * as vscode from "vscode";
import { getConfiguration } from "../configuration";
import type { WatchMode } from "../services/watch";
import type { BazelWorkspaces } from "../workspace";
import { describeStatus, initialStatus, type StatusKind, type StatusState } from "./statusModel";

const COLORS: Record<StatusKind, vscode.ThemeColor | undefined> = {
  ready: undefined,
  updating: new vscode.ThemeColor("statusBarItem.warningBackground"),
  watching: new vscode.ThemeColor("statusBarItem.prominentBackground"),
  error: new vscode.ThemeColor("statusBarItem.errorBackground"),
};

export class StatusBarManager implements vscode.Disposable {
  readonly #item: vscode.StatusBarItem;
  readonly #version: string;
  readonly #workspaces: Pick<BazelWorkspaces, "hasBazelRoot" | "onDidChange">;
  readonly #disposables: vscode.Disposable[] = [];
  #state: StatusState = initialStatus;

  constructor(
    extensionVersion: string,
    workspaces: Pick<BazelWorkspaces, "hasBazelRoot" | "onDidChange">,
  ) {
    this.#version = extensionVersion;
    this.#workspaces = workspaces;
    this.#item = vscode.window.createStatusBarItem(
      "bazelle.status",
      vscode.StatusBarAlignment.Left,
      99,
    );
    this.#item.name = "Bazelle";
    this.#item.command = "bazelle.showStatusMenu";

    this.#disposables.push(
      vscode.workspace.onDidChangeConfiguration((e) => {
        if (e.affectsConfiguration("bazelle.statusBar")) this.#updateVisibility();
      }),
      workspaces.onDidChange(() => this.#updateVisibility()),
    );

    this.#updateVisibility();
    this.#render();
  }

  #updateVisibility(): void {
    const show = getConfiguration().statusBarShow;
    const visible =
      show === "always" || (show === "onBazelWorkspace" && this.#workspaces.hasBazelRoot);
    if (visible) this.#item.show();
    else this.#item.hide();
  }

  #update(patch: Partial<StatusState>): void {
    this.#state = { ...this.#state, ...patch };
    this.#render();
  }

  #render(): void {
    const view = describeStatus(this.#state);
    this.#item.text = view.text;
    this.#item.backgroundColor = COLORS[view.kind];

    const md = new vscode.MarkdownString(undefined, true);
    md.isTrusted = { enabledCommands: [...TOOLTIP_COMMANDS] };
    md.appendMarkdown("### Bazelle\n\n");
    md.appendMarkdown(`**Status:** ${view.label}\n\n`);
    if (this.#state.error) md.appendText(`${this.#state.error}\n\n`);
    md.appendMarkdown(`**Version:** ${this.#version}\n\n`);
    if (this.#state.watchMode !== "none") {
      md.appendMarkdown(
        `**Watch Mode:** ${this.#state.watchMode === "daemon" ? "Daemon" : "Subprocess"}\n\n`,
      );
      md.appendMarkdown(`**Watch updates:** ${this.#state.watchUpdates} (details in output)\n\n`);
    }
    if (view.daemonConnected) md.appendMarkdown("**Daemon:** $(check) Connected\n\n");
    md.appendMarkdown("---\n\n");
    md.appendMarkdown("[Update BUILD Files](command:bazelle.update) | ");
    md.appendMarkdown("[Check Status](command:bazelle.status) | ");
    md.appendMarkdown("[Show Output](command:bazelle.showOutput)\n\n");
    md.appendMarkdown("[Daemon Status](command:bazelle.daemon.status) | ");
    md.appendMarkdown(
      view.daemonConnected
        ? "[Stop Daemon](command:bazelle.daemon.stop)"
        : "[Start Daemon](command:bazelle.daemon.start)",
    );
    this.#item.tooltip = md;
  }

  /** Show the busy state while `task` runs; overlapping tasks are counted. */
  async track<T>(task: () => PromiseLike<T>): Promise<T> {
    this.#update({ busy: this.#state.busy + 1 });
    try {
      return await task();
    } finally {
      this.#update({ busy: this.#state.busy - 1 });
    }
  }

  setWatchMode(watchMode: WatchMode): void {
    // The count survives a daemon-to-subprocess fallback, not a stop.
    this.#update(watchMode === "none" ? { watchMode, watchUpdates: 0 } : { watchMode });
  }

  addWatchUpdates(count: number): void {
    if (count > 0) this.#update({ watchUpdates: this.#state.watchUpdates + count });
  }

  setError(error: string | undefined): void {
    this.#update({ error });
  }

  setDaemonConnected(daemonConnected: boolean): void {
    this.#update({ daemonConnected });
  }

  dispose(): void {
    this.#item.dispose();
    for (const d of this.#disposables) d.dispose();
  }
}

const TOOLTIP_COMMANDS = [
  "bazelle.update",
  "bazelle.status",
  "bazelle.showOutput",
  "bazelle.daemon.status",
  "bazelle.daemon.stop",
  "bazelle.daemon.start",
] as const;
