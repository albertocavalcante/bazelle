/**
 * Bazelle CLI service - executes bazelle commands
 */

import { type ChildProcess, spawn } from "node:child_process";
import type * as vscode from "vscode";
import { parseStatusOutput, type StatusSummary } from "./cliOutput";
import { commandFor, treeKillCommand } from "./process";

export interface CommandResult {
  success: boolean;
  output?: string;
  error?: string;
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

  update(cwd: string, opts?: UpdateOptions, signal?: AbortSignal): Promise<CommandResult> {
    const args = ["update"];
    if (opts?.incremental) args.push("--incremental");
    return this.#run(args, cwd, signal);
  }

  fix(cwd: string, opts?: FixOptions, signal?: AbortSignal): Promise<CommandResult> {
    const args = ["fix"];
    if (opts?.dryRun) args.push("--dry-run");
    if (opts?.check) args.push("--check");
    return this.#run(args, cwd, signal);
  }

  async status(cwd: string): Promise<CommandResult & { summary?: StatusSummary }> {
    const result = await this.#run(["status", "--json"], cwd);
    if (!result.success) return result;
    try {
      return { ...result, summary: parseStatusOutput(result.output ?? "") };
    } catch (err) {
      return { success: false, output: result.output, error: `Unexpected status output: ${err}` };
    }
  }

  init(cwd: string, signal?: AbortSignal): Promise<CommandResult> {
    return this.#run(["init"], cwd, signal);
  }

  /** Run `daemon start`, which daemonizes and exits once the daemon is up. */
  startDaemon(cwd: string, socketPath?: string): Promise<CommandResult> {
    const args = ["daemon", "start"];
    if (socketPath) args.push("--socket", socketPath);
    return this.#run(args, cwd);
  }

  watch(cwd: string, debounceMs?: number): ChildProcess {
    const args = ["watch", "--json"];
    if (debounceMs !== undefined) args.push("--debounce", String(Math.round(debounceMs)));
    return this.#spawn(args, cwd);
  }

  /** Kill a spawned CLI process and, on Windows, everything it started. */
  kill(child: ChildProcess): void {
    const tree = child.pid === undefined ? undefined : treeKillCommand(child.pid);
    if (!tree) {
      child.kill();
      return;
    }
    const killer = spawn(tree.command, tree.args, { stdio: "ignore", windowsHide: true });
    killer.on("error", (err) => {
      this.#output.appendLine(`${tree.command} failed: ${err.message}`);
      child.kill();
    });
  }

  #spawn(args: string[], cwd: string): ChildProcess {
    this.#output.appendLine(`$ ${this.#binaryPath} ${args.join(" ")}  (in ${cwd})`);
    const { command, shell } = commandFor(this.#binaryPath);
    return spawn(command, args, { cwd, shell, stdio: ["ignore", "pipe", "pipe"] });
  }

  #run(args: string[], cwd: string, signal?: AbortSignal): Promise<CommandResult> {
    return new Promise((resolve) => {
      if (signal?.aborted) {
        resolve({ success: false, error: "Cancelled" });
        return;
      }
      const child = this.#spawn(args, cwd);
      // Not spawn({ signal }): that kills only cmd.exe for a Windows .cmd shim.
      const onAbort = () => this.kill(child);
      signal?.addEventListener("abort", onAbort, { once: true });
      let stdout = "";
      let stderr = "";

      child.stdout?.setEncoding("utf8").on("data", (text: string) => {
        stdout += text;
        this.#output.append(text);
      });
      child.stderr?.setEncoding("utf8").on("data", (text: string) => {
        stderr += text;
        this.#output.append(text);
      });

      child.on("error", (err) => {
        signal?.removeEventListener("abort", onAbort);
        const error = signal?.aborted ? "Cancelled" : err.message;
        this.#output.appendLine(`Error: ${error}`);
        resolve({ success: false, output: stdout, error });
      });

      child.on("close", (code) => {
        signal?.removeEventListener("abort", onAbort);
        if (signal?.aborted) {
          resolve({ success: false, output: stdout, error: "Cancelled" });
          return;
        }
        resolve(
          code === 0
            ? { success: true, output: stdout }
            : { success: false, output: stdout, error: stderr.trim() || `Exit code ${code}` },
        );
      });
    });
  }
}
