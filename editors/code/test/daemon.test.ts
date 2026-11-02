import { afterEach, expect, mock, test } from "bun:test";
import { mkdtempSync, rmSync } from "node:fs";
import * as net from "node:net";
import { tmpdir } from "node:os";
import { join } from "node:path";

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

mock.module("vscode", () => ({ EventEmitter: FakeEventEmitter }));
const { DaemonClient } = await import("../src/services/daemon");

const cleanups: Array<() => void> = [];
afterEach(() => {
  for (const cleanup of cleanups.splice(0)) cleanup();
});

function socketPath(): string {
  // Unix socket paths are limited to ~104 bytes, so keep this short.
  const dir = mkdtempSync(join(tmpdir(), "bzd-"));
  cleanups.push(() => rmSync(dir, { recursive: true, force: true }));
  return join(dir, "d.sock");
}

type Handler = (socket: net.Socket, request: { id: number; method: string }) => void;

async function listen(path: string, handler: Handler): Promise<net.Server> {
  const sockets = new Set<net.Socket>();
  const server = net.createServer((socket) => {
    sockets.add(socket);
    socket.on("close", () => sockets.delete(socket));
    let buffer = "";
    socket.on("data", (chunk) => {
      buffer += chunk.toString();
      let index = buffer.indexOf("\n");
      while (index !== -1) {
        handler(socket, JSON.parse(buffer.slice(0, index)));
        buffer = buffer.slice(index + 1);
        index = buffer.indexOf("\n");
      }
    });
  });
  (server as net.Server & { dropClients: () => void }).dropClients = () => {
    for (const socket of sockets) socket.destroy();
  };
  await new Promise<void>((resolve) => server.listen(path, resolve));
  cleanups.push(() => {
    for (const socket of sockets) socket.destroy();
    server.close();
  });
  return server;
}

const pong = (socket: net.Socket, id: number) =>
  socket.write(
    `${JSON.stringify({ jsonrpc: "2.0", id, result: { pong: true, version: "v", uptime: "1s" } })}\n`,
  );

function makeClient(path: string, options: Record<string, number> = {}) {
  const log: string[] = [];
  const output = { appendLine: (line: string) => log.push(line), append: () => {} };
  const client = new DaemonClient(output as never, { socketPath: path, ...options });
  const states: string[] = [];
  client.onConnectionStateChanged((state) => states.push(state));
  cleanups.push(() => client.dispose());
  return { client, log, states };
}

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

test("a failed initial connect rejects without scheduling reconnects", async () => {
  const { client, states } = makeClient(socketPath(), { reconnectDelay: 5 });

  await expect(client.connect()).rejects.toThrow("Failed to connect");
  await sleep(50);

  expect(states).not.toContain("reconnecting");
  expect(client.connectionState).toBe("disconnected");
});

test("uses every reconnect attempt after the daemon goes away", async () => {
  const path = socketPath();
  const server = await listen(path, (socket, req) => pong(socket, req.id));
  const { client, log } = makeClient(path, { reconnectDelay: 5, maxReconnectAttempts: 3 });
  await client.connect();

  server.close();
  (server as net.Server & { dropClients: () => void }).dropClients();
  await sleep(200);

  expect(log.filter((line) => line.includes("Reconnecting in"))).toHaveLength(3);
  expect(client.connectionState).toBe("disconnected");
});

test("reconnects when the daemon comes back", async () => {
  const path = socketPath();
  const server = await listen(path, (socket, req) => pong(socket, req.id));
  const { client, states } = makeClient(path, { reconnectDelay: 20, maxReconnectAttempts: 5 });
  await client.connect();

  (server as net.Server & { dropClients: () => void }).dropClients();
  await sleep(100);

  expect(states.filter((state) => state === "connected")).toHaveLength(2);
  expect((await client.ping()).pong).toBe(true);
});

test("an explicit disconnect does not reconnect", async () => {
  const path = socketPath();
  await listen(path, (socket, req) => pong(socket, req.id));
  const { client, states } = makeClient(path, { reconnectDelay: 5 });
  await client.connect();

  client.disconnect();
  await sleep(50);

  expect(states).not.toContain("reconnecting");
  expect(client.connectionState).toBe("disconnected");
});

test("decodes UTF-8 characters split across socket chunks", async () => {
  const path = socketPath();
  await listen(path, (socket, req) => {
    const bytes = Buffer.from(
      `${JSON.stringify({ jsonrpc: "2.0", id: req.id, result: { status: "ok", paths: ["/ws/café"] } })}\n`,
    );
    const split = bytes.indexOf(Buffer.from("é")) + 1; // inside the 2-byte sequence
    socket.write(bytes.subarray(0, split));
    setTimeout(() => socket.write(bytes.subarray(split)), 10);
  });
  const { client } = makeClient(path);
  await client.connect();

  expect((await client.watchStart({ paths: ["/ws/café"] })).paths).toEqual(["/ws/café"]);
});

test("isAvailable gives up quickly on a daemon that never answers", async () => {
  const path = socketPath();
  await listen(path, () => {});
  const { client } = makeClient(path, { requestTimeout: 30_000 });

  const started = Date.now();
  expect(await client.isAvailable()).toBe(false);
  expect(Date.now() - started).toBeLessThan(4000);
});
