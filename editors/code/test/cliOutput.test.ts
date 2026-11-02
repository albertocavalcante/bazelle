import { expect, test } from "bun:test";
import { LineBuffer, parseStatusOutput, parseWatchEvent } from "../src/services/cliOutput";

// `bazelle status --json` prints one indented object (cmd/bazelle/internal/cli/status.go).
test("counts stale directories from the indented status JSON object", () => {
  const stdout = `${JSON.stringify(
    { stale: true, stale_dirs: ["pkg/a", "pkg/b"], modified_files: ["pkg/a/x.go"] },
    null,
    2,
  )}\n`;
  expect(parseStatusOutput(stdout)).toEqual({ stale: true, staleDirs: ["pkg/a", "pkg/b"] });
});

test("reports an up-to-date workspace", () => {
  expect(parseStatusOutput('{\n  "stale": false,\n  "stale_dirs": null\n}\n')).toEqual({
    stale: false,
    staleDirs: [],
  });
});

test("rejects non-JSON status output", () => {
  expect(() => parseStatusOutput("BUILD files are up to date\n")).toThrow();
});

// `bazelle watch --json` events (cmd/bazelle/internal/watch/logger.go).
test("maps CLI watch events onto extension watch events", () => {
  expect(parseWatchEvent({ event: "ready", files: 3, path: "/ws" })).toEqual({
    type: "started",
    path: "/ws",
  });
  expect(parseWatchEvent({ event: "file_changed", path: "a.go", change: "modified" })).toEqual({
    type: "file_change",
    files: ["a.go"],
  });
  expect(parseWatchEvent({ event: "updating", dirs: ["pkg"] })).toEqual({
    type: "updating",
    directories: ["pkg"],
  });
  expect(parseWatchEvent({ event: "updated", build_file: "pkg/BUILD.bazel" })).toEqual({
    type: "updated",
    path: "pkg/BUILD.bazel",
  });
  expect(parseWatchEvent({ event: "error", error: "boom" })).toEqual({
    type: "error",
    message: "boom",
  });
  expect(parseWatchEvent({ event: "shutdown", updates: 1 })).toBeUndefined();
});

test("reassembles JSON lines split across stdout chunks", () => {
  const lines: string[] = [];
  const buffer = new LineBuffer((line) => lines.push(line));
  buffer.push('{"event":"rea');
  buffer.push('dy"}\n{"event":"error","error":"é');
  expect(lines).toEqual(['{"event":"ready"}']);
  buffer.push('"}\n');
  expect(lines).toEqual(['{"event":"ready"}', '{"event":"error","error":"é"}']);
});
