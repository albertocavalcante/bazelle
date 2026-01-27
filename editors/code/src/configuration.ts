/**
 * Configuration management for Bazelle extension
 */

import * as vscode from "vscode";

export interface BazelleConfig {
  bazellePath: string;
  autoUpdate: boolean;
  statusBarShow: "always" | "onBazelWorkspace" | "never";
  watchEnabled: boolean;
  watchDebounceMs: number;
  daemonEnabled: boolean;
  daemonSocketPath: string | undefined;
  daemonAutoStart: boolean;
}

export function getConfiguration(): BazelleConfig {
  const config = vscode.workspace.getConfiguration("bazelle");

  return {
    bazellePath: config.get<string>("path", "bazelle"),
    autoUpdate: config.get<boolean>("autoUpdate", false),
    statusBarShow: config.get<"always" | "onBazelWorkspace" | "never">(
      "statusBar.show",
      "onBazelWorkspace",
    ),
    watchEnabled: config.get<boolean>("watch.enabled", false),
    watchDebounceMs: config.get<number>("watch.debounceMs", 500),
    daemonEnabled: config.get<boolean>("daemon.enabled", true),
    daemonSocketPath: config.get<string>("daemon.socketPath"),
    daemonAutoStart: config.get<boolean>("daemon.autoStart", false),
  };
}
