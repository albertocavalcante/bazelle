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
  static watchPaths = ["/workspace"];
  static nextWatchStart: Promise<{ status: string; paths: string[] }> | null = null;
  static connectGate: Promise<void> | null = null;
  static watchStatus = "watching";
  readonly connectionState = new FakeEventEmitter<string>();
  readonly onConnectionStateChanged = this.connectionState.event;
  readonly watchEvents = new FakeEventEmitter<unknown>();
  readonly onWatchEvent = this.watchEvents.event;
  readonly onError = new FakeEventEmitter<Error>().event;
  watchStarts = 0;
  watchStartParams: unknown[] = [];
  disposed = false;
  constructor() {
    FakeDaemonClient.instances.push(this);
  }
  async connect(): Promise<void> {
    if (FakeDaemonClient.connectGate) await FakeDaemonClient.connectGate;
    if (FakeDaemonClient.failConnect) throw new Error("connection refused");
  }
  async ping(): Promise<{ version: string; uptime: string }> {
    return { version: "test", uptime: "1s" };
  }
  async watchStart(params?: unknown): Promise<{ status: string; paths: string[] }> {
    this.watchStarts++;
    this.watchStartParams.push(params);
    if (FakeDaemonClient.nextWatchStart) return FakeDaemonClient.nextWatchStart;
    return { status: FakeDaemonClient.watchStatus, paths: FakeDaemonClient.watchPaths };
  }
  watchStops = 0;
  async watchStop(): Promise<void> {
    this.watchStops++;
  }
  dispose(): void {
    this.disposed = true;
  }
}

const config = {
  daemonEnabled: true,
  daemonSocketPath: "/tmp/bazelle.sock" as string | undefined,
  daemonAutoStart: false,
  watchDebounceMs: 500,
};

mock.module("vscode", () => ({ EventEmitter: FakeEventEmitter }));

const { WatchService } = await import("../src/services/watch");

type FakeProcess = NodeEventEmitter & {
  stdout: NodeEventEmitter;
  stderr: NodeEventEmitter;
  killed: boolean;
  kill: () => void;
};

function makeProcess(): FakeProcess {
  const child = new NodeEventEmitter() as FakeProcess;
  child.stdout = new NodeEventEmitter();
  child.stderr = new NodeEventEmitter();
  child.killed = false;
  child.kill = () => {
    child.killed = true;
  };
  return child;
}

interface FakeBazelle {
  processes: FakeProcess[];
  watchCalls: Array<{ cwd: string; debounceMs?: number }>;
  daemonStarts: Array<string | undefined>;
  watch: (cwd: string, debounceMs?: number) => FakeProcess;
  startDaemon: (cwd: string, socketPath?: string) => Promise<{ success: boolean }>;
}

let bazelle: FakeBazelle;

function makeWatchService(): InstanceType<typeof WatchService> {
  bazelle = {
    processes: [],
    watchCalls: [],
    daemonStarts: [],
    watch(cwd, debounceMs) {
      this.watchCalls.push({ cwd, debounceMs });
      const child = makeProcess();
      this.processes.push(child);
      return child;
    },
    async startDaemon(_cwd, socketPath) {
      this.daemonStarts.push(socketPath);
      FakeDaemonClient.failConnect = false;
      return { success: true };
    },
  };
  const output = { append: () => {}, appendLine: () => {} };
  return new WatchService(bazelle as never, output as never, {
    getConfig: () => config,
    createDaemonClient: () => new FakeDaemonClient() as never,
  });
}

function reset(): void {
  FakeDaemonClient.instances = [];
  FakeDaemonClient.failConnect = false;
  FakeDaemonClient.watchPaths = ["/workspace"];
  FakeDaemonClient.nextWatchStart = null;
  FakeDaemonClient.connectGate = null;
  FakeDaemonClient.watchStatus = "watching";
  config.daemonEnabled = true;
  config.daemonAutoStart = false;
  config.watchDebounceMs = 500;
}

test("cleans up a failed daemon client before subprocess fallback", async () => {
  FakeDaemonClient.instances = [];
  FakeDaemonClient.failConnect = true;
  FakeDaemonClient.watchPaths = ["/workspace"];
  const watch = makeWatchService();

  await watch.start("/workspace");

  expect(watch.mode).toBe("subprocess");
  expect(FakeDaemonClient.instances[0]?.disposed).toBe(true);
});

test("restores watch registration after a daemon reconnects", async () => {
  FakeDaemonClient.instances = [];
  FakeDaemonClient.failConnect = false;
  FakeDaemonClient.watchPaths = ["/workspace"];
  const watch = makeWatchService();

  await watch.start("/workspace");
  const client = FakeDaemonClient.instances[0];
  expect(client?.watchStarts).toBe(1);

  client?.connectionState.fire("reconnecting");
  client?.connectionState.fire("connected");
  await new Promise((resolve) => setTimeout(resolve, 0));

  expect(client?.watchStarts).toBe(2);
});

test("falls back when the daemon is watching another workspace", async () => {
  FakeDaemonClient.instances = [];
  FakeDaemonClient.failConnect = false;
  FakeDaemonClient.watchPaths = ["/other-workspace"];
  const watch = makeWatchService();

  await watch.start("/workspace");

  expect(watch.mode).toBe("subprocess");
  expect(FakeDaemonClient.instances[0]?.disposed).toBe(true);
});

test("falls back if reconnect finds a watcher for another workspace", async () => {
  FakeDaemonClient.instances = [];
  FakeDaemonClient.failConnect = false;
  FakeDaemonClient.watchPaths = ["/workspace"];
  const watch = makeWatchService();
  await watch.start("/workspace");

  FakeDaemonClient.watchPaths = ["/other-workspace"];
  const client = FakeDaemonClient.instances[0];
  client?.connectionState.fire("connected");
  await new Promise((resolve) => setTimeout(resolve, 0));

  expect(watch.mode).toBe("subprocess");
  expect(client?.disposed).toBe(true);
});

test("does not restart watching after a reconnect fails following stop", async () => {
  FakeDaemonClient.instances = [];
  FakeDaemonClient.failConnect = false;
  FakeDaemonClient.watchPaths = ["/workspace"];
  FakeDaemonClient.nextWatchStart = null;
  const watch = makeWatchService();
  await watch.start("/workspace");

  let rejectReconnect: (error: Error) => void = () => {};
  FakeDaemonClient.nextWatchStart = new Promise((_, reject) => {
    rejectReconnect = reject;
  });
  FakeDaemonClient.instances[0]?.connectionState.fire("connected");
  await watch.stop();
  rejectReconnect(new Error("reconnect failed"));
  await new Promise((resolve) => setTimeout(resolve, 0));

  expect(watch.mode).toBe("none");
  FakeDaemonClient.nextWatchStart = null;
});

test("a stale subprocess exit does not clobber a restarted watcher", async () => {
  reset();
  config.daemonEnabled = false;
  const watch = makeWatchService();
  await watch.start("/workspace");
  const first = bazelle.processes[0];
  await watch.stop();
  await watch.start("/workspace");

  first?.emit("close", null);

  expect(watch.mode).toBe("subprocess");
  expect(first?.killed).toBe(true);
});

test("rejects a second start while the first is still connecting", async () => {
  reset();
  let release: () => void = () => {};
  FakeDaemonClient.connectGate = new Promise((resolve) => {
    release = resolve;
  });
  const watch = makeWatchService();
  const first = watch.start("/workspace");

  await expect(watch.start("/workspace")).rejects.toThrow("already");
  release();
  await first;

  expect(FakeDaemonClient.instances).toHaveLength(1);
  expect(watch.mode).toBe("daemon");
});

test("stop during an in-flight daemon start leaves watch stopped", async () => {
  reset();
  let release: () => void = () => {};
  FakeDaemonClient.connectGate = new Promise((resolve) => {
    release = resolve;
  });
  const watch = makeWatchService();
  const starting = watch.start("/workspace");
  await watch.stop();
  release();
  await starting;

  expect(watch.mode).toBe("none");
  expect(FakeDaemonClient.instances[0]?.disposed).toBe(true);
  expect(bazelle.processes).toHaveLength(0);
});

test("passes the configured debounce to the daemon and the subprocess", async () => {
  reset();
  config.watchDebounceMs = 750;
  const daemonWatch = makeWatchService();
  await daemonWatch.start("/workspace");
  expect(FakeDaemonClient.instances[0]?.watchStartParams[0]).toEqual({
    paths: ["/workspace"],
    debounce: 750,
  });

  config.daemonEnabled = false;
  const subprocessWatch = makeWatchService();
  await subprocessWatch.start("/workspace");
  expect(bazelle.watchCalls).toEqual([{ cwd: "/workspace", debounceMs: 750 }]);
});

test("auto-starts the daemon when configured and it is not running", async () => {
  reset();
  config.daemonAutoStart = true;
  FakeDaemonClient.failConnect = true;
  const watch = makeWatchService();

  await watch.start("/workspace");

  expect(bazelle.daemonStarts).toEqual(["/tmp/bazelle.sock"]);
  expect(watch.mode).toBe("daemon");
});

test("emits CLI watch events split across stdout chunks", async () => {
  reset();
  config.daemonEnabled = false;
  const watch = makeWatchService();
  const events: unknown[] = [];
  watch.onEvent((event) => events.push(event));
  await watch.start("/workspace");
  const child = bazelle.processes[0];

  child?.stdout.emit("data", Buffer.from('{"event":"upd'));
  child?.stdout.emit("data", Buffer.from('ated","build_file":"pkg/BUILD.bazel"}\n'));

  expect(events).toContainEqual({ type: "updated", path: "pkg/BUILD.bazel" });
});

test("stopping after watch/start was sent still stops the daemon watcher", async () => {
  reset();
  let reply: (value: { status: string; paths: string[] }) => void = () => {};
  FakeDaemonClient.nextWatchStart = new Promise((resolve) => {
    reply = resolve;
  });
  const watch = makeWatchService();
  const starting = watch.start("/workspace");
  await new Promise((resolve) => setTimeout(resolve, 0));
  expect(FakeDaemonClient.instances[0]?.watchStarts).toBe(1);

  await watch.stop();
  reply({ status: "watching", paths: ["/workspace"] });
  await starting;

  expect(watch.mode).toBe("none");
  expect(FakeDaemonClient.instances[0]?.watchStops).toBe(1);
  expect(FakeDaemonClient.instances[0]?.disposed).toBe(true);
});

test("does not stop a daemon watcher another client started", async () => {
  reset();
  FakeDaemonClient.watchStatus = "already_watching";
  const watch = makeWatchService();
  await watch.start("/workspace");

  await watch.stop();

  expect(FakeDaemonClient.instances[0]?.watchStops).toBe(0);
  expect(FakeDaemonClient.instances[0]?.disposed).toBe(true);
});

test("falls back to a subprocess as soon as the daemon announces shutdown", async () => {
  reset();
  const watch = makeWatchService();
  await watch.start("/workspace");

  FakeDaemonClient.instances[0]?.watchEvents.fire({ type: "shutdown", timestamp: "t" });

  expect(watch.mode).toBe("subprocess");
  expect(FakeDaemonClient.instances[0]?.disposed).toBe(true);
});
