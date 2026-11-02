/**
 * Bazel workspace detection and folder selection. Pure, so they are testable.
 */

import { stat } from "node:fs/promises";
import { join } from "node:path";

/** Files that make a directory a Bazel repository root. */
export const BAZEL_ROOT_MARKERS = ["MODULE.bazel", "REPO.bazel", "WORKSPACE.bazel", "WORKSPACE"];

const isFile = (path: string): Promise<boolean> =>
  stat(path).then(
    (stats) => stats.isFile(),
    () => false,
  );

/** Whether `dir` itself (not a subdirectory) is a Bazel repository root. */
export async function isBazelRoot(dir: string): Promise<boolean> {
  const found = await Promise.all(BAZEL_ROOT_MARKERS.map((marker) => isFile(join(dir, marker))));
  return found.includes(true);
}

export type FolderChoice<T> =
  | { kind: "none" }
  | { kind: "folder"; folder: T }
  | { kind: "ambiguous"; candidates: T[] };

/**
 * The folder a command applies to: the active editor's folder, the only
 * folder, or the only Bazel root. Otherwise the user must pick, preferably
 * among the Bazel roots.
 */
export function chooseFolder<T>(
  folders: readonly T[],
  active: T | undefined,
  isRoot: (folder: T) => boolean,
): FolderChoice<T> {
  if (active !== undefined) return { kind: "folder", folder: active };
  const [only] = folders;
  if (folders.length === 1 && only !== undefined) return { kind: "folder", folder: only };
  if (folders.length === 0) return { kind: "none" };
  const roots = folders.filter(isRoot);
  const [root] = roots;
  if (roots.length === 1 && root !== undefined) return { kind: "folder", folder: root };
  return { kind: "ambiguous", candidates: roots.length > 0 ? roots : [...folders] };
}
