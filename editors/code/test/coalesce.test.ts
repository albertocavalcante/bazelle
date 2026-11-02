import { expect, test } from "bun:test";
import { coalesce } from "../src/services/coalesce";

test("runs at most once per key at a time and once more for saves during a run", async () => {
  const runs: string[] = [];
  const releases: Array<() => void> = [];
  const trigger = coalesce((key: string) => {
    runs.push(key);
    return new Promise<void>((resolve) => releases.push(resolve));
  });

  trigger("/a");
  trigger("/a");
  trigger("/a");
  trigger("/b");
  expect(runs).toEqual(["/a", "/b"]);

  releases.shift()?.();
  await Promise.resolve();
  await Promise.resolve();
  expect(runs).toEqual(["/a", "/b", "/a"]);

  releases.shift()?.();
  releases.shift()?.();
  await new Promise((resolve) => setTimeout(resolve, 0));
  expect(runs).toEqual(["/a", "/b", "/a"]);
});

test("keeps working after a run fails", async () => {
  let calls = 0;
  const trigger = coalesce(async () => {
    calls++;
    throw new Error("boom");
  });
  trigger("/a");
  await new Promise((resolve) => setTimeout(resolve, 0));
  trigger("/a");
  await new Promise((resolve) => setTimeout(resolve, 0));
  expect(calls).toBe(2);
});
