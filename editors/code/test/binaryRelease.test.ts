import { expect, test } from "bun:test";
import { parseInstallManifest, selectReleaseAsset } from "../src/services/binaryRelease";

const repo = "albertocavalcante/bazelle";
const sha = "7b5ffcc3a19ac69f50779b8bb9a0a01097c03008803d758eae52f3d2d4b33239";
const artifact = "bazelle-linux-amd64.tar.gz";
const url = `https://github.com/${repo}/releases/download/v0.2.0/${artifact}`;

function release(overrides: Record<string, unknown> = {}, asset: Record<string, unknown> = {}) {
  return {
    tag_name: "v0.2.0",
    draft: false,
    prerelease: false,
    assets: [
      {
        name: `${artifact}.sha256`,
        browser_download_url: `${url}.sha256`,
        digest: `sha256:${sha}`,
      },
      { name: artifact, browser_download_url: url, digest: `sha256:${sha}`, ...asset },
    ],
    ...overrides,
  };
}

test("selects the platform archive and its GitHub-computed digest", () => {
  expect(selectReleaseAsset(release(), artifact, repo)).toEqual({
    tag: "v0.2.0",
    url,
    sha256: sha,
  });
});

test("refuses prereleases and drafts such as the moving nightly", () => {
  expect(() => selectReleaseAsset(release({ prerelease: true }), artifact, repo)).toThrow(
    "not a stable release",
  );
  expect(() => selectReleaseAsset(release({ draft: true }), artifact, repo)).toThrow(
    "not a stable release",
  );
});

test("refuses an asset without a SHA-256 digest", () => {
  expect(() => selectReleaseAsset(release({}, { digest: null }), artifact, repo)).toThrow(
    "no SHA-256 digest",
  );
  expect(() => selectReleaseAsset(release({}, { digest: "md5:abc" }), artifact, repo)).toThrow(
    "no SHA-256 digest",
  );
});

test("refuses a missing asset or a download URL outside the repository", () => {
  expect(() => selectReleaseAsset(release(), "bazelle-windows-amd64.zip", repo)).toThrow(
    "has no bazelle-windows-amd64.zip",
  );
  expect(() =>
    selectReleaseAsset(
      release({}, { browser_download_url: `https://evil.example/${artifact}` }),
      artifact,
      repo,
    ),
  ).toThrow("unexpected download URL");
});

test("refuses a tag that is not a safe directory name", () => {
  expect(selectReleaseAsset(release({ tag_name: "v1.0.0+b1" }), artifact, repo).tag).toBe(
    "v1.0.0+b1",
  );
  expect(() => selectReleaseAsset(release({ tag_name: "../x" }), artifact, repo)).toThrow(
    "invalid tag",
  );
});

test("reads the install manifest and rejects malformed ones", () => {
  const manifest = { tag: "v0.2.0", binary: "bin/v0.2.0/bazelle", binarySha256: sha };
  expect(parseInstallManifest(JSON.stringify(manifest))).toEqual(manifest);
  expect(parseInstallManifest("not json")).toBeUndefined();
  expect(parseInstallManifest(JSON.stringify({ ...manifest, binarySha256: "x" }))).toBeUndefined();
  expect(parseInstallManifest(JSON.stringify({ ...manifest, binary: 1 }))).toBeUndefined();
});
