/**
 * Parsers for bazelle CLI output. Pure, so they can be tested without VS Code.
 */

const str = (value: unknown): string | undefined => (typeof value === "string" ? value : undefined);
const strs = (value: unknown): string[] | undefined =>
  Array.isArray(value) ? value.filter((v): v is string => typeof v === "string") : undefined;

export interface StatusSummary {
  stale: boolean;
  staleDirs: string[];
  /** Set when the CLI could not compute staleness (e.g. "no state found"). */
  error?: string;
}

/**
 * Parse `bazelle status --json`, which prints a single indented JSON object
 * (cmd/bazelle/internal/cli/status.go StatusOutput).
 */
export function parseStatusOutput(stdout: string): StatusSummary {
  const data = JSON.parse(stdout) as Record<string, unknown>;
  return {
    stale: data.stale === true,
    staleDirs: strs(data.stale_dirs) ?? [],
    error: str(data.error),
  };
}

export interface WatchEvent {
  type: "started" | "file_change" | "updating" | "updated" | "error";
  path?: string;
  message?: string;
  directories?: string[];
  files?: string[];
}

/**
 * Map a `bazelle watch --json` event (cmd/bazelle/internal/watch/logger.go)
 * onto the extension's WatchEvent. Returns undefined for events we ignore.
 */
export function parseWatchEvent(event: Record<string, unknown>): WatchEvent | undefined {
  switch (event.event) {
    case "ready":
      return { type: "started", path: str(event.path) };
    case "file_changed": {
      const path = str(event.path);
      return { type: "file_change", files: path ? [path] : [] };
    }
    case "updating":
      return { type: "updating", directories: strs(event.dirs) };
    case "updated":
      return { type: "updated", path: str(event.build_file) };
    case "error":
    case "internal_error":
      return { type: "error", message: str(event.error) };
    default:
      return undefined;
  }
}

/**
 * Splits a text stream into complete lines, holding back a trailing partial
 * line until the rest of it arrives.
 */
export class LineBuffer {
  #pending = "";
  readonly #onLine: (line: string) => void;

  constructor(onLine: (line: string) => void) {
    this.#onLine = onLine;
  }

  push(chunk: string): void {
    const lines = (this.#pending + chunk).split("\n");
    this.#pending = lines.pop() ?? "";
    for (const line of lines) {
      const trimmed = line.trim();
      if (trimmed) this.#onLine(trimmed);
    }
  }
}
