import { execFileSync } from "node:child_process";
import { createHash } from "node:crypto";

export function verifyArchiveChecksum(data: Buffer, expected: string): void {
  if (!/^[a-f0-9]{64}$/iu.test(expected)) {
    throw new Error("Invalid SHA-256 checksum pin");
  }
  const actual = createHash("sha256").update(data).digest("hex");
  if (actual !== expected.toLowerCase()) {
    throw new Error("Archive checksum mismatch");
  }
}

export function extractArchive(
  archive: string,
  destination: string,
  run: typeof execFileSync = execFileSync,
): void {
  run("tar", ["-xzf", archive, "-C", destination], { stdio: "pipe" });
}
