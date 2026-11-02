import { expect, test } from "bun:test";
import { describeStatus, initialStatus } from "../src/ui/statusModel";

test("returns to the watching view after a one-off update finishes", () => {
  const watching = { ...initialStatus, watchMode: "daemon" as const };
  expect(describeStatus({ ...watching, busy: 1 }).label).toBe("Updating...");
  expect(describeStatus({ ...watching, busy: 0 })).toMatchObject({
    text: "$(eye) Bazelle (Daemon)",
    label: "Watching",
  });
});

test("stays busy until every overlapping task finishes", () => {
  expect(describeStatus({ ...initialStatus, busy: 2 }).label).toBe("Updating...");
});

test("shows a missing binary as an error until it is resolved", () => {
  const missing = { ...initialStatus, error: "bazelle binary not found" };
  expect(describeStatus(missing)).toMatchObject({ text: "$(error) Bazelle", label: "Error" });
  expect(describeStatus({ ...missing, error: undefined }).label).toBe("Ready");
});

test("a daemon-mode watch implies a connected daemon", () => {
  expect(describeStatus({ ...initialStatus, watchMode: "daemon" }).daemonConnected).toBe(true);
  expect(describeStatus(initialStatus).daemonConnected).toBe(false);
});
