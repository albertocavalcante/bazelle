/**
 * Binary resolver for Bazelle CLI
 *
 * Resolution order:
 * 1. Configured path (bazelle.path setting)
 * 2. PATH lookup (bazelle command)
 * 3. Bundled binary (bin/{platform}-{arch}/bazelle)
 * 4. Download from GitHub releases (optional)
 */

import { execSync } from "node:child_process";
import { chmodSync, existsSync, rmSync } from "node:fs";
import { mkdir, writeFile } from "node:fs/promises";
import { join } from "node:path";
import * as vscode from "vscode";

const GITHUB_REPO = "albertocavalcante/bazelle";

type Platform = "linux" | "darwin" | "windows";
type Arch = "amd64" | "arm64";
type ArchiveExt = "tar.gz" | "zip";
type BinarySource = "configured" | "path" | "bundled" | "downloaded";

interface PlatformConfig {
  platform: Platform;
  arch: Arch;
  binary: string;
  archive: ArchiveExt;
}

// Bundled platforms (no darwin-amd64, no windows-arm64)
const PLATFORMS: Record<string, PlatformConfig> = {
  "linux-x64": { platform: "linux", arch: "amd64", binary: "bazelle", archive: "tar.gz" },
  "linux-arm64": { platform: "linux", arch: "arm64", binary: "bazelle", archive: "tar.gz" },
  "darwin-arm64": { platform: "darwin", arch: "arm64", binary: "bazelle", archive: "tar.gz" },
  "win32-x64": { platform: "windows", arch: "amd64", binary: "bazelle.exe", archive: "zip" },
} as const;

export interface ResolvedBinary {
  path: string;
  source: BinarySource;
}

export class BinaryResolver {
  readonly #extensionPath: string;
  readonly #output: vscode.OutputChannel;
  readonly #platformKey: string;
  readonly #config: PlatformConfig | undefined;

  constructor(extensionPath: string, output: vscode.OutputChannel) {
    this.#extensionPath = extensionPath;
    this.#output = output;
    this.#platformKey = `${process.platform}-${process.arch}`;
    this.#config = PLATFORMS[this.#platformKey];
  }

  get isSupported(): boolean {
    return this.#config !== undefined;
  }

  get platformKey(): string {
    return this.#platformKey;
  }

  /**
   * Resolve binary using priority: configured > PATH > bundled
   */
  async resolve(): Promise<ResolvedBinary | undefined> {
    return this.#fromConfig() ?? this.#fromPath() ?? this.#fromBundle();
  }

  /**
   * Download binary from GitHub releases
   */
  async download(version = "nightly"): Promise<ResolvedBinary | undefined> {
    if (!this.#config) {
      void vscode.window.showErrorMessage(
        `Platform ${this.#platformKey} is not supported for download`,
      );
      return undefined;
    }

    const { platform, arch, binary, archive } = this.#config;
    const dir = join(this.#extensionPath, "bin", `${platform}-${arch}`);
    const binaryPath = join(dir, binary);
    const artifact = `bazelle-${platform}-${arch}.${archive}`;
    const url = `https://github.com/${GITHUB_REPO}/releases/download/${version}/${artifact}`;

    return vscode.window.withProgress(
      {
        location: vscode.ProgressLocation.Notification,
        title: `Downloading bazelle (${version})...`,
        cancellable: false,
      },
      async () => {
        try {
          this.#log(`Downloading: ${url}`);

          const response = await fetch(url);
          if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
          }

          await mkdir(dir, { recursive: true });

          const archivePath = join(dir, artifact);
          await writeFile(archivePath, Buffer.from(await response.arrayBuffer()));

          this.#extract(archivePath, dir, archive);
          rmSync(archivePath);

          if (process.platform !== "win32") {
            chmodSync(binaryPath, 0o755);
          }

          this.#log(`Downloaded to: ${binaryPath}`);
          void vscode.window.showInformationMessage("Bazelle downloaded successfully");

          return { path: binaryPath, source: "downloaded" as const };
        } catch (err) {
          const msg = err instanceof Error ? err.message : String(err);
          this.#log(`Download failed: ${msg}`);
          void vscode.window.showErrorMessage(`Failed to download bazelle: ${msg}`);
          return undefined;
        }
      },
    );
  }

  /**
   * Prompt user when no binary found
   */
  async promptDownload(): Promise<ResolvedBinary | undefined> {
    if (!this.#config) {
      const action = await vscode.window.showErrorMessage(
        `Bazelle not found. Platform ${this.#platformKey} doesn't support auto-download. Please install manually.`,
        "Documentation",
      );
      if (action === "Documentation") {
        void vscode.env.openExternal(
          vscode.Uri.parse(`https://github.com/${GITHUB_REPO}#installation`),
        );
      }
      return undefined;
    }

    const action = await vscode.window.showWarningMessage(
      "Bazelle binary not found. Download it?",
      "Download",
      "Configure Path",
    );

    if (action === "Download") return this.download();
    if (action === "Configure Path") {
      void vscode.commands.executeCommand("workbench.action.openSettings", "bazelle.path");
    }
    return undefined;
  }

  #fromConfig(): ResolvedBinary | undefined {
    const configured = vscode.workspace.getConfiguration("bazelle").get<string>("path");

    if (!configured || configured === "bazelle") return undefined;

    const expanded = configured.replace(/^~/, process.env.HOME ?? "");
    if (!existsSync(expanded)) {
      this.#log(`Configured path not found: ${expanded}`);
      return undefined;
    }

    this.#log(`Using configured: ${expanded}`);
    return { path: expanded, source: "configured" };
  }

  #fromPath(): ResolvedBinary | undefined {
    try {
      const cmd = process.platform === "win32" ? "where" : "which";
      const result = execSync(`${cmd} bazelle`, { encoding: "utf-8", stdio: "pipe" }).trim();

      if (result && existsSync(result)) {
        this.#log(`Using PATH: ${result}`);
        return { path: result, source: "path" };
      }
    } catch {
      // Not found
    }
    return undefined;
  }

  #fromBundle(): ResolvedBinary | undefined {
    if (!this.#config) {
      this.#log(`Platform ${this.#platformKey} has no bundled binary`);
      return undefined;
    }

    const { platform, arch, binary } = this.#config;
    const bundled = join(this.#extensionPath, "bin", `${platform}-${arch}`, binary);

    if (!existsSync(bundled)) return undefined;

    if (process.platform !== "win32") {
      try {
        chmodSync(bundled, 0o755);
      } catch {
        // Ignore
      }
    }

    this.#log(`Using bundled: ${bundled}`);
    return { path: bundled, source: "bundled" };
  }

  #extract(archive: string, dest: string, type: ArchiveExt): void {
    if (type === "zip") {
      execSync(
        `powershell -Command "Expand-Archive -Path '${archive}' -DestinationPath '${dest}' -Force"`,
        {
          stdio: "pipe",
        },
      );
    } else {
      execSync(`tar -xzf "${archive}" -C "${dest}"`, { stdio: "pipe" });
    }
  }

  #log(msg: string): void {
    this.#output.appendLine(`[BinaryResolver] ${msg}`);
  }
}
