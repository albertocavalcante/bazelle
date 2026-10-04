import { expect, mock, test } from "bun:test";
import { EventEmitter as NodeEventEmitter } from "node:events";

class FakeEventEmitter<T> {
  private listeners = new Set<(value: T) => void>();
  readonly event = (listener: (value: T) => void) => {
    this.listeners.add(listener);
    return { dispose: () => this.listeners.delete(listener) };
  };
  fire(value: T): void {
    for (const listener of this.listeners) listener(value);
  }
  dispose(): void {
    this.listeners.clear();
  }
}

class FakeDaemonClient {
  static instances: FakeDaemonClient[] = [];
  static failConnect = false;
  readonly connectionState = new FakeEventEmitter<string>();
  readonly onConnectionStateChanged = this.connectionState.event;
  readonly onWatchEvent = new FakeEventEmitter<unknown>().event;
  readonly onError = new FakeEventEmitter<Error>().event;
  watchStarts = 0;
  disposed = false;
  constructor() {
    FakeDaemonClient.instances.push(this);
  }
  async connect(): Promise<void> {
    if (FakeDaemonClient.failConnect) throw new Error("connection refused");
  }
  async ping(): Promise<{ version: string; uptime: string }> {
    return { version: "test", uptime: "1s" };
  }
  async watchStart(): Promise<{ status: string; paths: string[] }> {
    this.watchStarts++;
    return { status: "watching", paths: ["/workspace"] };
  }
  dispose(): void {
    this.disposed = true;
  }
}

mock.module("vscode", () => ({ EventEmitter: FakeEventEmitter }));
mock.module("../src/services/daemon", () => ({
  DaemonClient: FakeDaemonClient,
  getDaemonConfig: () => ({ enabled: true, socketPath: "/tmp/bazelle.sock" }),
}));

const { WatchService } = await import("../src/services/watch");

function makeWatchService(): InstanceType<typeof WatchService> {
  const process = new NodeEventEmitter() as NodeEventEmitter & {
    stdout: NodeEventEmitter;
    stderr: NodeEventEmitter;
  };
  process.stdout = new NodeEventEmitter();
  process.stderr = new NodeEventEmitter();
  const bazelle = { watch: () => process };
  const output = { append: () => {}, appendLine: () => {} };
  return new WatchService(bazelle as never, output as never);
}

test("cleans up a failed daemon client before subprocess fallback", async () => {
  FakeDaemonClient.instances = [];
  FakeDaemonClient.failConnect = true;
  const watch = makeWatchService();

  await watch.start("/workspace");

  expect(watch.mode).toBe("subprocess");
  expect(FakeDaemonClient.instances[0]?.disposed).toBe(true);
});

test("restores watch registration after a daemon reconnects", async () => {
  FakeDaemonClient.instances = [];
  FakeDaemonClient.failConnect = false;
  const watch = makeWatchService();

  await watch.start("/workspace");
  const client = FakeDaemonClient.instances[0];
  expect(client?.watchStarts).toBe(1);

  client?.connectionState.fire("reconnecting");
  client?.connectionState.fire("connected");
  await new Promise((resolve) => setTimeout(resolve, 0));

  expect(client?.watchStarts).toBe(2);
});
