/**
 * Binary resolver for Bazelle CLI
 *
 * Resolution order:
 * 1. Configured path (bazelle.path setting)
 * 2. PATH lookup (bazelle command)
 * 3. Bundled binary (bin/{platform}-{arch}/bazelle)
 * 4. Download from GitHub releases (optional): the latest stable release,
 *    verified against GitHub's SHA-256 asset digest; while no stable release
 *    exists, the nightly archive pinned below.
 */

import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";
import { chmodSync, existsSync } from "node:fs";
import { mkdir, mkdtemp, readFile, rename, rm, writeFile } from "node:fs/promises";
import { homedir } from "node:os";
import { dirname, join, relative, resolve as resolvePath, sep } from "node:path";
import * as vscode from "vscode";
import { extractArchive, verifyArchiveChecksum } from "./binaryArchive";
import { type InstallManifest, parseInstallManifest, selectReleaseAsset } from "./binaryRelease";
import { expandHome } from "./process";

const GITHUB_REPO = "albertocavalcante/bazelle";

type Platform = "linux" | "darwin";
type Arch = "amd64" | "arm64";
type BinarySource = "configured" | "path" | "bundled" | "downloaded";

// Fallback while no stable release exists. The nightly tag moves, so a moved
// archive is rejected until these pins are deliberately updated from the
// release asset digests.
const PINNED_NIGHTLY = "nightly";
const DOWNLOAD_TIMEOUT_MS = 120_000;
const MANIFEST = "installed.json";

interface DownloadTarget {
  tag: string;
  url: string;
  archiveSha256: string;
  /** Known binary digest; when absent it is recorded at install time. */
  binarySha256?: string;
}

const sha256 = (data: Buffer) => createHash("sha256").update(data).digest("hex");

interface PlatformConfig {
  platform: Platform;
  arch: Arch;
  binary: string;
  /** Pins for the nightly fallback. */
  archiveSha256: string;
  binarySha256: string;
}

// Windows currently has no published CLI archive; configured/PATH binaries still work.
const PLATFORMS: Record<string, PlatformConfig> = {
  "linux-x64": {
    platform: "linux",
    arch: "amd64",
    binary: "bazelle",
    archiveSha256: "7b5ffcc3a19ac69f50779b8bb9a0a01097c03008803d758eae52f3d2d4b33239",
    binarySha256: "a553f2960282561f7ff5ab2b238dd6659217d0d72763ecba1f7217fb9fdd0bd6",
  },
  "linux-arm64": {
    platform: "linux",
    arch: "arm64",
    binary: "bazelle",
    archiveSha256: "5a7a63017ef27dab12f266e16cbe5566fa15b1b5b635f8145827e86bbc8450f0",
    binarySha256: "aa9c8cd8c34d7b909135056326af57de02c02aacf0e8a11e72809f6e3238efb2",
  },
  "darwin-arm64": {
    platform: "darwin",
    arch: "arm64",
    binary: "bazelle",
    archiveSha256: "7765cdd6f85e482226f677c10b9f7bcfc1d0c215ec5238e1bbc1b10fb7e4c263",
    binarySha256: "8214e9177687cc34afcf9e0fdccc811b89aacd42d8d14cd47274fd0e6b3a6afc",
  },
} as const;

export interface ResolvedBinary {
  path: string;
  source: BinarySource;
}

export class BinaryResolver {
  readonly #extensionPath: string;
  readonly #storagePath: string;
  readonly #output: vscode.OutputChannel;
  readonly #platformKey: string;
  readonly #config: PlatformConfig | undefined;

  constructor(extensionPath: string, storagePath: string, output: vscode.OutputChannel) {
    this.#extensionPath = extensionPath;
    this.#storagePath = storagePath;
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
    return (
      this.#fromConfig() ?? this.#fromPath() ?? (await this.#fromStorage()) ?? this.#fromBundle()
    );
  }

  /**
   * Download binary from GitHub releases
   */
  async download(): Promise<ResolvedBinary | undefined> {
    const config = this.#config;
    if (!config) {
      void vscode.window.showErrorMessage(
        `Platform ${this.#platformKey} is not supported for download`,
      );
      return undefined;
    }

    const { platform, arch, binary } = config;
    const artifact = `bazelle-${platform}-${arch}.tar.gz`;

    return vscode.window.withProgress(
      {
        location: vscode.ProgressLocation.Notification,
        title: "Downloading bazelle...",
        cancellable: true,
      },
      async (progress, token) => {
        try {
          const abort = new AbortController();
          token.onCancellationRequested(() => abort.abort(new Error("Download cancelled")));
          const signal = AbortSignal.any([abort.signal, AbortSignal.timeout(DOWNLOAD_TIMEOUT_MS)]);

          const target = await this.#resolveRelease(config, artifact, signal);
          progress.report({ message: target.tag });
          this.#log(`Downloading: ${target.url}`);
          const response = await fetch(target.url, { signal });
          if (!response.ok) {
            throw new Error(`HTTP ${response.status}: ${response.statusText}`);
          }

          const data = Buffer.from(await response.arrayBuffer());
          verifyArchiveChecksum(data, target.archiveSha256);
          const dir = join(
            this.#storagePath,
            "bin",
            `${target.tag}-${target.archiveSha256.slice(0, 12)}`,
            `${platform}-${arch}`,
          );
          const binaryPath = join(dir, binary);
          await mkdir(this.#storagePath, { recursive: true });
          const staging = await mkdtemp(join(this.#storagePath, "download-"));
          let binarySha256: string;
          try {
            const archivePath = join(staging, artifact);
            await writeFile(archivePath, data);
            extractArchive(archivePath, staging);
            const stagedBinary = join(staging, binary);
            const binaryData = await readFile(stagedBinary);
            if (target.binarySha256) verifyArchiveChecksum(binaryData, target.binarySha256);
            binarySha256 = sha256(binaryData);
            chmodSync(stagedBinary, 0o755);
            await rm(archivePath);
            await mkdir(dirname(dir), { recursive: true });
            await rm(dir, { recursive: true, force: true });
            await rename(staging, dir);
          } finally {
            await rm(staging, { recursive: true, force: true });
          }
          await this.#writeManifest({
            tag: target.tag,
            binary: relative(this.#storagePath, binaryPath),
            binarySha256,
          });

          this.#log(`Downloaded ${target.tag} to: ${binaryPath}`);
          void vscode.window.showInformationMessage(
            `Bazelle ${target.tag} downloaded successfully`,
          );

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

  /** The latest stable release, or the pinned nightly while none exists. */
  async #resolveRelease(
    config: PlatformConfig,
    artifact: string,
    signal: AbortSignal,
  ): Promise<DownloadTarget> {
    const api = `https://api.github.com/repos/${GITHUB_REPO}/releases/latest`;
    this.#log(`Resolving latest release: ${api}`);
    const response = await fetch(api, {
      headers: { Accept: "application/vnd.github+json", "X-GitHub-Api-Version": "2022-11-28" },
      signal,
    });
    if (response.status === 404) {
      // releases/latest ignores prereleases, so 404 means no stable release yet.
      this.#log(`No stable release yet; using the pinned ${PINNED_NIGHTLY} build`);
      return {
        tag: PINNED_NIGHTLY,
        url: `https://github.com/${GITHUB_REPO}/releases/download/${PINNED_NIGHTLY}/${artifact}`,
        archiveSha256: config.archiveSha256,
        binarySha256: config.binarySha256,
      };
    }
    if (!response.ok) {
      throw new Error(`Could not resolve the latest release: HTTP ${response.status}`);
    }
    const asset = selectReleaseAsset(await response.json(), artifact, GITHUB_REPO);
    return { tag: asset.tag, url: asset.url, archiveSha256: asset.sha256 };
  }

  async #writeManifest(manifest: InstallManifest): Promise<void> {
    const path = join(this.#storagePath, "bin", MANIFEST);
    await writeFile(`${path}.tmp`, JSON.stringify(manifest, null, 2));
    await rename(`${path}.tmp`, path);
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

    const expanded = expandHome(configured, homedir());
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
      const result = execFileSync(cmd, ["bazelle"], { encoding: "utf-8", stdio: "pipe" })
        .trim()
        .split(/\r?\n/u)[0];

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

  async #fromStorage(): Promise<ResolvedBinary | undefined> {
    if (!this.#config) return undefined;
    const manifest = parseInstallManifest(
      await readFile(join(this.#storagePath, "bin", MANIFEST), "utf8").catch(() => ""),
    );
    if (manifest) {
      const binaryPath = resolvePath(this.#storagePath, manifest.binary);
      if (!binaryPath.startsWith(resolvePath(this.#storagePath) + sep)) {
        this.#log(`Ignoring install manifest pointing outside storage: ${manifest.binary}`);
        return undefined;
      }
      return this.#verifiedStored(binaryPath, manifest.binarySha256);
    }
    // Downloads made before the manifest existed.
    const { platform, arch, binary, archiveSha256, binarySha256 } = this.#config;
    const legacyDir = `${PINNED_NIGHTLY}-${archiveSha256.slice(0, 12)}`;
    const legacy = join(this.#storagePath, "bin", legacyDir, `${platform}-${arch}`, binary);
    return this.#verifiedStored(legacy, binarySha256);
  }

  async #verifiedStored(
    binaryPath: string,
    expectedSha256: string,
  ): Promise<ResolvedBinary | undefined> {
    if (!existsSync(binaryPath)) return undefined;
    try {
      verifyArchiveChecksum(await readFile(binaryPath), expectedSha256);
      return { path: binaryPath, source: "downloaded" };
    } catch (error) {
      this.#log(`Stored binary failed verification: ${error}`);
      return undefined;
    }
  }

  #log(msg: string): void {
    this.#output.appendLine(`[BinaryResolver] ${msg}`);
  }
}
