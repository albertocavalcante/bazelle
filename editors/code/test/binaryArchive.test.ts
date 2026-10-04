import { expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { extractArchive, verifyArchiveChecksum } from "../src/services/binaryArchive";

test("rejects a downloaded archive when its pinned digest differs", () => {
  const data = Buffer.from("different release");
  expect(() => verifyArchiveChecksum(data, "0".repeat(64))).toThrow("checksum mismatch");
});

test("accepts only a valid matching SHA-256 digest", () => {
  const data = Buffer.from("pinned release");
  const digest = createHash("sha256").update(data).digest("hex");
  expect(() => verifyArchiveChecksum(data, digest)).not.toThrow();
  expect(() => verifyArchiveChecksum(data, "invalid")).toThrow("Invalid SHA-256");
});

test("passes archive paths as literal process arguments", () => {
  const calls: Array<{ command: string; args: string[] }> = [];
  const archive = "/tmp/extension's $(touch unsafe)/bazelle.tar.gz";
  const destination = "/tmp/output with spaces";
  extractArchive(archive, destination, (command, args) => {
    calls.push({ command, args });
  });
  expect(calls).toEqual([{ command: "tar", args: ["-xzf", archive, "-C", destination] }]);
});
