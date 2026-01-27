/**
 * Bazelle CLI service - executes bazelle commands
 */

import { type ChildProcess, spawn } from "node:child_process";
import type * as vscode from "vscode";

export interface CommandResult {
  success: boolean;
  output?: string;
  error?: string;
  staleCount?: number;
}

export interface UpdateOptions {
  incremental?: boolean;
}

export interface FixOptions {
  dryRun?: boolean;
  check?: boolean;
}

export class BazelleService {
  #binaryPath: string;
  readonly #output: vscode.OutputChannel;

  constructor(binaryPath: string, output: vscode.OutputChannel) {
    this.#binaryPath = binaryPath;
    this.#output = output;
  }

  setBinaryPath(path: string): void {
    this.#binaryPath = path;
  }

  get binaryPath(): string {
    return this.#binaryPath;
  }

  async update(cwd: string, opts?: UpdateOptions): Promise<CommandResult> {
    const args = ["update"];
    if (opts?.incremental) args.push("--incremental");
    return this.#run(args, cwd);
  }

  async fix(cwd: string, opts?: FixOptions): Promise<CommandResult> {
    const args = ["fix"];
    if (opts?.dryRun) args.push("--dry-run");
    if (opts?.check) args.push("--check");
    return this.#run(args, cwd);
  }

  async status(cwd: string): Promise<CommandResult> {
    const result = await this.#run(["status", "--json"], cwd);

    if (result.success && result.output) {
      let staleCount = 0;
      for (const line of result.output.trim().split("\n")) {
        try {
          const data = JSON.parse(line);
          if (data.stale) staleCount++;
        } catch {
          // Not JSON
        }
      }
      result.staleCount = staleCount;
    }

    return result;
  }

  async init(cwd: string): Promise<CommandResult> {
    return this.#run(["init"], cwd);
  }

  watch(cwd: string): ChildProcess {
    const args = ["watch", "--json"];
    this.#output.appendLine(`$ ${this.#binaryPath} ${args.join(" ")}`);

    return spawn(this.#binaryPath, args, {
      cwd,
      stdio: ["ignore", "pipe", "pipe"],
      shell: process.platform === "win32",
    });
  }

  #run(args: string[], cwd: string): Promise<CommandResult> {
    return new Promise((resolve) => {
      this.#output.appendLine(`$ ${this.#binaryPath} ${args.join(" ")}`);

      const child = spawn(this.#binaryPath, args, {
        cwd,
        stdio: ["ignore", "pipe", "pipe"],
        shell: process.platform === "win32",
      });

      let stdout = "";
      let stderr = "";

      child.stdout?.on("data", (data: Buffer) => {
        const text = data.toString();
        stdout += text;
        this.#output.append(text);
      });

      child.stderr?.on("data", (data: Buffer) => {
        const text = data.toString();
        stderr += text;
        this.#output.append(text);
      });

      child.on("error", (err) => {
        this.#output.appendLine(`Error: ${err.message}`);
        resolve({ success: false, error: err.message });
      });

      child.on("close", (code) => {
        resolve(
          code === 0
            ? { success: true, output: stdout }
            : { success: false, output: stdout, error: stderr || `Exit code ${code}` },
        );
      });
    });
  }
}
