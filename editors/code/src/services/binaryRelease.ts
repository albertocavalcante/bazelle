/**
 * GitHub release parsing for the CLI download. Pure, so it is testable.
 */

const SHA256 = /^[a-f0-9]{64}$/u;
const SAFE_TAG = /^(?!\.{1,2}$)[\w.+-]+$/u;

export interface ReleaseAsset {
  tag: string;
  url: string;
  /** SHA-256 of the archive, as computed by GitHub when it was uploaded. */
  sha256: string;
}

const record = (value: unknown): Record<string, unknown> =>
  typeof value === "object" && value !== null ? (value as Record<string, unknown>) : {};

/**
 * Pick `artifact` from a GitHub release API object (GET /releases/latest).
 * Only stable releases qualify, and only with a server-side SHA-256 digest.
 */
export function selectReleaseAsset(release: unknown, artifact: string, repo: string): ReleaseAsset {
  const data = record(release);
  const tag = data.tag_name;
  if (typeof tag !== "string" || !SAFE_TAG.test(tag)) {
    throw new Error(`Release has an invalid tag: ${String(tag)}`);
  }
  if (data.draft !== false || data.prerelease !== false) {
    throw new Error(`Release ${tag} is not a stable release`);
  }
  const asset = (Array.isArray(data.assets) ? data.assets : [])
    .map(record)
    .find((a) => a.name === artifact);
  if (!asset) throw new Error(`Release ${tag} has no ${artifact}`);

  const url = asset.browser_download_url;
  const prefix = `https://github.com/${repo}/releases/download/`;
  if (typeof url !== "string" || !url.startsWith(prefix)) {
    throw new Error(`Release ${tag} has an unexpected download URL for ${artifact}`);
  }
  const digest = typeof asset.digest === "string" ? asset.digest : "";
  const sha256 = digest.startsWith("sha256:") ? digest.slice("sha256:".length).toLowerCase() : "";
  if (!SHA256.test(sha256)) throw new Error(`Release ${tag} has no SHA-256 digest for ${artifact}`);
  return { tag, url, sha256 };
}

/** What a download installed, so later activations can verify it offline. */
export interface InstallManifest {
  tag: string;
  /** Binary path relative to the extension's global storage directory. */
  binary: string;
  binarySha256: string;
}

export function parseInstallManifest(text: string): InstallManifest | undefined {
  let data: Record<string, unknown>;
  try {
    data = record(JSON.parse(text));
  } catch {
    return undefined;
  }
  const { tag, binary, binarySha256 } = data;
  if (typeof tag !== "string" || typeof binary !== "string" || typeof binarySha256 !== "string") {
    return undefined;
  }
  if (!SHA256.test(binarySha256)) return undefined;
  return { tag, binary, binarySha256 };
}
