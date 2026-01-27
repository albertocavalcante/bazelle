/**
 * Status bar management for Bazelle extension
 */

import * as vscode from "vscode";
import { getConfiguration } from "../configuration";
import type { WatchMode } from "../services/watch";

type ServerState = "ready" | "updating" | "watching" | "error" | "stopped";

const STATUS_ICONS: Record<ServerState, string> = {
  ready: "$(pass-filled)",
  updating: "$(loading~spin)",
  watching: "$(eye)",
  error: "$(error)",
  stopped: "$(stop-circle)",
};

const STATUS_COLORS: Record<ServerState, vscode.ThemeColor | undefined> = {
  ready: undefined,
  updating: new vscode.ThemeColor("statusBarItem.warningBackground"),
  watching: new vscode.ThemeColor("statusBarItem.prominentBackground"),
  error: new vscode.ThemeColor("statusBarItem.errorBackground"),
  stopped: undefined,
};

const WATCH_MODE_LABELS: Record<WatchMode, string> = {
  daemon: "Daemon",
  subprocess: "Subprocess",
  none: "",
};

export class StatusBarManager implements vscode.Disposable {
  private statusBarItem: vscode.StatusBarItem;
  private currentState: ServerState = "stopped";
  private watchMode: WatchMode = "none";
  private daemonConnected = false;
  private extensionVersion: string;
  private disposables: vscode.Disposable[] = [];

  constructor(extensionVersion: string) {
    this.extensionVersion = extensionVersion;

    // Create status bar item (left-aligned)
    this.statusBarItem = vscode.window.createStatusBarItem(
      "bazelle.status",
      vscode.StatusBarAlignment.Left,
      99,
    );
    this.statusBarItem.name = "Bazelle";
    this.statusBarItem.command = "bazelle.showStatusMenu";

    // Smart visibility - only show in Bazel workspaces
    this.disposables.push(
      vscode.window.onDidChangeActiveTextEditor((editor) => {
        this.updateVisibility(editor);
      }),
    );

    // Listen for configuration changes
    this.disposables.push(
      vscode.workspace.onDidChangeConfiguration((e) => {
        if (e.affectsConfiguration("bazelle.statusBar")) {
          this.updateVisibility(vscode.window.activeTextEditor);
        }
      }),
    );

    // Initial visibility check
    this.updateVisibility(vscode.window.activeTextEditor);
    this.updateView();
  }

  private updateVisibility(_editor: vscode.TextEditor | undefined): void {
    const config = getConfiguration();

    switch (config.statusBarShow) {
      case "never":
        this.statusBarItem.hide();
        break;
      case "always":
        this.statusBarItem.show();
        break;
      default:
        // Check if we're in a Bazel workspace
        if (this.isInBazelWorkspace()) {
          this.statusBarItem.show();
        } else {
          this.statusBarItem.hide();
        }
        break;
    }
  }

  private isInBazelWorkspace(): boolean {
    const workspaceFolders = vscode.workspace.workspaceFolders;
    if (!workspaceFolders) return false;

    // Check if any workspace folder has Bazel files
    // This is a simple check - the actual visibility is controlled by activation events
    return true; // If extension is activated, we're likely in a Bazel workspace
  }

  private updateView(): void {
    const icon = STATUS_ICONS[this.currentState];
    const color = STATUS_COLORS[this.currentState];

    // Add mode indicator when watching
    let text = `${icon} Bazelle`;
    if (this.currentState === "watching" && this.watchMode !== "none") {
      const modeLabel = WATCH_MODE_LABELS[this.watchMode];
      text = `${icon} Bazelle (${modeLabel})`;
    }

    this.statusBarItem.text = text;
    this.statusBarItem.backgroundColor = color;
    this.statusBarItem.tooltip = this.buildTooltip();
  }

  private buildTooltip(): vscode.MarkdownString {
    const md = new vscode.MarkdownString();
    md.isTrusted = true;
    md.supportHtml = true;

    md.appendMarkdown("### Bazelle\n\n");
    md.appendMarkdown(`**Status:** ${this.getStateLabel()}\n\n`);
    md.appendMarkdown(`**Version:** ${this.extensionVersion}\n\n`);

    // Show daemon status
    if (this.currentState === "watching") {
      const modeLabel = this.watchMode === "daemon" ? "Daemon" : "Subprocess";
      md.appendMarkdown(`**Watch Mode:** ${modeLabel}\n\n`);
    }

    if (this.daemonConnected) {
      md.appendMarkdown("**Daemon:** $(check) Connected\n\n");
    }

    md.appendMarkdown("---\n\n");
    md.appendMarkdown("[Update BUILD Files](command:bazelle.update) | ");
    md.appendMarkdown("[Check Status](command:bazelle.status) | ");
    md.appendMarkdown("[Show Output](command:bazelle.showOutput)\n\n");

    // Daemon commands
    md.appendMarkdown("[Daemon Status](command:bazelle.daemon.status) | ");
    if (this.daemonConnected) {
      md.appendMarkdown("[Stop Daemon](command:bazelle.daemon.stop)");
    } else {
      md.appendMarkdown("[Start Daemon](command:bazelle.daemon.start)");
    }

    return md;
  }

  private getStateLabel(): string {
    switch (this.currentState) {
      case "ready":
        return "Ready";
      case "updating":
        return "Updating...";
      case "watching":
        return "Watching";
      case "error":
        return "Error";
      case "stopped":
        return "Stopped";
    }
  }

  setReady(): void {
    this.currentState = "ready";
    this.watchMode = "none";
    this.updateView();
  }

  setUpdating(): void {
    this.currentState = "updating";
    this.updateView();
  }

  setWatching(mode: WatchMode = "subprocess"): void {
    this.currentState = "watching";
    this.watchMode = mode;
    this.updateView();
  }

  setError(): void {
    this.currentState = "error";
    this.updateView();
  }

  setStopped(): void {
    this.currentState = "stopped";
    this.watchMode = "none";
    this.updateView();
  }

  setDaemonConnected(connected: boolean): void {
    this.daemonConnected = connected;
    this.updateView();
  }

  getWatchMode(): WatchMode {
    return this.watchMode;
  }

  dispose(): void {
    this.statusBarItem.dispose();
    for (const d of this.disposables) {
      d.dispose();
    }
  }
}
