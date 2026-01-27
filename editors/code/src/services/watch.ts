/**
 * Watch service - manages bazelle watch process
 */

import type { ChildProcess } from "node:child_process";
import * as vscode from "vscode";
import type { BazelleService } from "./bazelle";

export interface WatchEvent {
  type: "started" | "updated" | "error" | "file_change";
  path?: string;
  message?: string;
}

export class WatchService implements vscode.Disposable {
  readonly #bazelle: BazelleService;
  readonly #output: vscode.OutputChannel;
  readonly #onEvent = new vscode.EventEmitter<WatchEvent>();

  #process: ChildProcess | null = null;

  readonly onEvent = this.#onEvent.event;

  constructor(bazelle: BazelleService, output: vscode.OutputChannel) {
    this.#bazelle = bazelle;
    this.#output = output;
  }

  get isRunning(): boolean {
    return this.#process !== null;
  }

  async start(cwd: string): Promise<void> {
    if (this.#process) {
      throw new Error("Watch mode already running");
    }

    this.#output.appendLine("Starting watch mode...");
    this.#process = this.#bazelle.watch(cwd);

    this.#process.stdout?.on("data", (data: Buffer) => {
      const text = data.toString();
      this.#output.append(text);

      for (const line of text.split("\n")) {
        if (!line.trim()) continue;
        try {
          this.#onEvent.fire(JSON.parse(line) as WatchEvent);
        } catch {
          // Not JSON
        }
      }
    });

    this.#process.stderr?.on("data", (data: Buffer) => {
      const text = data.toString();
      this.#output.append(text);
      this.#onEvent.fire({ type: "error", message: text });
    });

    this.#process.on("error", (err) => {
      this.#output.appendLine(`Watch error: ${err.message}`);
      this.#onEvent.fire({ type: "error", message: err.message });
      this.#process = null;
    });

    this.#process.on("close", (code) => {
      this.#output.appendLine(`Watch exited (code ${code})`);
      this.#process = null;
    });

    this.#onEvent.fire({ type: "started" });
  }

  stop(): void {
    if (this.#process) {
      this.#output.appendLine("Stopping watch mode...");
      this.#process.kill();
      this.#process = null;
    }
  }

  dispose(): void {
    this.stop();
    this.#onEvent.dispose();
  }
}
