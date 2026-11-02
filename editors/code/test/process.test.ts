import { expect, test } from "bun:test";
import { commandFor, expandHome, treeKillCommand } from "../src/services/process";

test("runs executables directly without a shell", () => {
  expect(commandFor("C:\\Program Files\\bazelle\\bazelle.exe", "win32")).toEqual({
    command: "C:\\Program Files\\bazelle\\bazelle.exe",
    shell: false,
  });
  expect(commandFor("/opt/my tools/bazelle", "darwin")).toEqual({
    command: "/opt/my tools/bazelle",
    shell: false,
  });
});

test("quotes Windows batch shims that need a shell", () => {
  expect(commandFor("C:\\Users\\me\\AppData\\bazelle.CMD", "win32")).toEqual({
    command: '"C:\\Users\\me\\AppData\\bazelle.CMD"',
    shell: true,
  });
});

test("expands only a leading ~ path segment", () => {
  expect(expandHome("~/bin/bazelle", "/home/me")).toBe("/home/me/bin/bazelle");
  expect(expandHome("~", "/home/me")).toBe("/home/me");
  expect(expandHome("~\\bin\\bazelle.exe", "C:\\Users\\me")).toBe(
    "C:\\Users\\me\\bin\\bazelle.exe",
  );
  expect(expandHome("~other/bazelle", "/home/me")).toBe("~other/bazelle");
  expect(expandHome("/usr/bin/bazelle", "/home/me")).toBe("/usr/bin/bazelle");
});

test("kills the whole process tree on Windows, where a .cmd shim wraps the watcher", () => {
  expect(treeKillCommand(4242, "win32")).toEqual({
    command: "taskkill",
    args: ["/pid", "4242", "/T", "/F"],
  });
  expect(treeKillCommand(4242, "linux")).toBeUndefined();
  expect(treeKillCommand(4242, "darwin")).toBeUndefined();
});
