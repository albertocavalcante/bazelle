/**
 * Status bar state and its presentation. Pure, so it is testable without VS Code.
 */

import type { WatchMode } from "../services/watch";

export interface StatusState {
  /** Number of in-flight one-off CLI runs (update, fix, init, ...). */
  busy: number;
  watchMode: WatchMode;
  daemonConnected: boolean;
  /** BUILD files (or directories) the running watch has updated. */
  watchUpdates: number;
  /** Persistent problem (e.g. no CLI binary); cleared when resolved. */
  error: string | undefined;
}

export const initialStatus: StatusState = {
  busy: 0,
  watchMode: "none",
  daemonConnected: false,
  watchUpdates: 0,
  error: undefined,
};

export type StatusKind = "ready" | "updating" | "watching" | "error";

export interface StatusView {
  kind: StatusKind;
  text: string;
  label: string;
  daemonConnected: boolean;
}

const ICONS: Record<StatusKind, string> = {
  ready: "$(pass-filled)",
  updating: "$(loading~spin)",
  watching: "$(eye)",
  error: "$(error)",
};

const LABELS: Record<StatusKind, string> = {
  ready: "Ready",
  updating: "Updating...",
  watching: "Watching",
  error: "Error",
};

const MODE_LABELS: Record<WatchMode, string> = {
  daemon: "Daemon",
  subprocess: "Subprocess",
  none: "",
};

export function describeStatus(state: StatusState): StatusView {
  const kind: StatusKind = state.error
    ? "error"
    : state.busy > 0
      ? "updating"
      : state.watchMode !== "none"
        ? "watching"
        : "ready";
  const count = state.watchUpdates > 0 ? ` · ${state.watchUpdates}` : "";
  const suffix = kind === "watching" ? ` (${MODE_LABELS[state.watchMode]})${count}` : "";
  return {
    kind,
    text: `${ICONS[kind]} Bazelle${suffix}`,
    label: LABELS[kind],
    daemonConnected: state.daemonConnected || state.watchMode === "daemon",
  };
}
