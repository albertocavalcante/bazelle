import { afterEach, expect, test } from "bun:test";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { chooseFolder, isBazelRoot } from "../src/services/workspaceFolders";

const dirs: string[] = [];
afterEach(() => {
  for (const dir of dirs.splice(0)) rmSync(dir, { recursive: true, force: true });
});

function folderWith(...files: string[]): string {
  const dir = mkdtempSync(join(tmpdir(), "bzw-"));
  dirs.push(dir);
  for (const file of files) {
    mkdirSync(join(dir, file, ".."), { recursive: true });
    writeFileSync(join(dir, file), "");
  }
  return dir;
}

test("a folder with a Bazel root marker file is a Bazel workspace", async () => {
  for (const marker of ["MODULE.bazel", "REPO.bazel", "WORKSPACE.bazel", "WORKSPACE"]) {
    expect(await isBazelRoot(folderWith(marker))).toBe(true);
  }
});

test("BUILD files or nested markers alone do not make a Bazel workspace", async () => {
  expect(await isBazelRoot(folderWith("BUILD.bazel", "sub/MODULE.bazel"))).toBe(false);
  expect(await isBazelRoot(folderWith())).toBe(false);
});

test("a directory named like a marker is not a marker", async () => {
  const dir = folderWith();
  mkdirSync(join(dir, "WORKSPACE"));
  expect(await isBazelRoot(dir)).toBe(false);
});

const isRoot = (roots: string[]) => (folder: string) => roots.includes(folder);

test("prefers the active editor's folder, then the only folder", () => {
  expect(chooseFolder(["a", "b"], "b", isRoot(["a"]))).toEqual({ kind: "folder", folder: "b" });
  expect(chooseFolder(["a"], undefined, isRoot([]))).toEqual({ kind: "folder", folder: "a" });
  expect(chooseFolder([], undefined, isRoot([]))).toEqual({ kind: "none" });
});

test("picks the single Bazel root in a multi-root workspace", () => {
  expect(chooseFolder(["docs", "repo", "web"], undefined, isRoot(["repo"]))).toEqual({
    kind: "folder",
    folder: "repo",
  });
});

test("asks when several folders, or none of them, are Bazel roots", () => {
  expect(chooseFolder(["a", "b", "c"], undefined, isRoot(["a", "c"]))).toEqual({
    kind: "ambiguous",
    candidates: ["a", "c"],
  });
  expect(chooseFolder(["a", "b"], undefined, isRoot([]))).toEqual({
    kind: "ambiguous",
    candidates: ["a", "b"],
  });
});
