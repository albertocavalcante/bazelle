/**
 * Daemon client - connects to bazelle daemon via Unix socket
 *
 * Implements JSON-RPC 2.0 over newline-delimited messages, matching
 * cmd/bazelle/internal/daemon/protocol.go.
 */

import * as net from "node:net";
import * as os from "node:os";
import * as path from "node:path";
import * as vscode from "vscode";
import { LineBuffer } from "./cliOutput";

const JSONRPC_VERSION = "2.0";

interface JsonRpcMessage {
  jsonrpc: string;
  id?: number | null;
  method?: string;
  params?: unknown;
  result?: unknown;
  error?: { code: number; message: string; data?: unknown };
}

// RPC method names (matching protocol.go)
const Methods = {
  Ping: "ping",
  Shutdown: "shutdown",
  WatchStart: "watch/start",
  WatchStop: "watch/stop",
  WatchStatus: "watch/status",
  WatchEvent: "watch/event", // notification from server
} as const;

// Result types matching protocol.go

export interface PingResult {
  pong: boolean;
  version: string;
  uptime: string;
  start_time: string;
}

export interface ShutdownResult {
  message: string;
}

export interface WatchStartParams {
  paths?: string[];
  languages?: string[];
  debounce?: number;
}

export interface WatchStartResult {
  status: string;
  paths: string[];
  languages?: string[];
}

export interface WatchStopResult {
  status: string;
}

export interface WatchStatusResult {
  watching: boolean;
  paths?: string[];
  languages?: string[];
  file_count?: number;
  update_time?: string;
}

export interface WatchEventParams {
  type: "change" | "update" | "error" | "shutdown";
  directories?: string[];
  files?: string[];
  message?: string;
  timestamp: string;
}

export type DaemonConnectionState = "disconnected" | "connecting" | "connected" | "reconnecting";

interface PendingRequest {
  resolve: (result: unknown) => void;
  reject: (error: Error) => void;
  timeout: ReturnType<typeof setTimeout>;
}

/** Ping timeout used by isAvailable() so probes never hang the UI. */
const PROBE_TIMEOUT_MS = 2000;

export function defaultSocketPath(): string {
  return path.join(os.homedir(), ".bazelle", "daemon.sock");
}

export class DaemonClient implements vscode.Disposable {
  readonly #output: vscode.OutputChannel;
  readonly #onConnectionStateChanged = new vscode.EventEmitter<DaemonConnectionState>();
  readonly #onWatchEvent = new vscode.EventEmitter<WatchEventParams>();
  readonly #onError = new vscode.EventEmitter<Error>();

  readonly #socketPath: string;
  readonly #requestTimeout: number;
  readonly #maxReconnectAttempts: number;
  readonly #reconnectDelay: number;

  // Only events from the current socket are acted on; a replaced or
  // deliberately closed socket's late events are ignored.
  #socket: net.Socket | null = null;
  #connecting: Promise<void> | null = null;
  #connectionState: DaemonConnectionState = "disconnected";
  #requestId = 0;
  #pendingRequests = new Map<number, PendingRequest>();
  #reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  #reconnectAttempts = 0;
  #disposed = false;

  readonly onConnectionStateChanged = this.#onConnectionStateChanged.event;
  readonly onWatchEvent = this.#onWatchEvent.event;
  readonly onError = this.#onError.event;

  constructor(
    output: vscode.OutputChannel,
    options?: {
      socketPath?: string;
      requestTimeout?: number;
      maxReconnectAttempts?: number;
      reconnectDelay?: number;
    },
  ) {
    this.#output = output;
    this.#socketPath = options?.socketPath || defaultSocketPath();
    this.#requestTimeout = options?.requestTimeout ?? 30000;
    this.#maxReconnectAttempts = options?.maxReconnectAttempts ?? 5;
    this.#reconnectDelay = options?.reconnectDelay ?? 1000;
  }

  get socketPath(): string {
    return this.#socketPath;
  }

  get connectionState(): DaemonConnectionState {
    return this.#connectionState;
  }

  get isConnected(): boolean {
    return this.#connectionState === "connected";
  }

  #setConnectionState(state: DaemonConnectionState): void {
    if (this.#connectionState !== state) {
      this.#connectionState = state;
      this.#output.appendLine(`[Daemon] Connection state: ${state}`);
      this.#onConnectionStateChanged.fire(state);
    }
  }

  /**
   * Connect to the daemon socket. Rejects (without retrying) if the daemon
   * is not reachable; retries only happen after an established connection drops.
   */
  async connect(): Promise<void> {
    if (this.#disposed) throw new Error("Client has been disposed");
    if (this.isConnected) return;
    if (this.#connecting) return this.#connecting;

    this.#clearReconnectTimer();
    this.#setConnectionState("connecting");
    this.#connecting = this.#openSocket()
      .catch((err: Error) => {
        this.#setConnectionState("disconnected");
        throw new Error(`Failed to connect to daemon: ${err.message}`);
      })
      .finally(() => {
        this.#connecting = null;
      });
    return this.#connecting;
  }

  /** Open a socket; resolves once connected, rejects if it never connects. */
  #openSocket(): Promise<void> {
    return new Promise((resolve, reject) => {
      // Attach listeners before connecting so an immediate failure (ENOENT)
      // is never an unhandled 'error' event.
      const socket = new net.Socket();
      socket.setEncoding("utf8");
      this.#socket = socket;
      let established = false;
      const lines = new LineBuffer((line) => this.#processMessage(line));

      socket.on("connect", () => {
        if (socket !== this.#socket) {
          socket.destroy(); // superseded by a newer connection attempt
          return;
        }
        established = true;
        this.#reconnectAttempts = 0;
        this.#output.appendLine(`[Daemon] Connected to ${this.#socketPath}`);
        this.#setConnectionState("connected");
        resolve();
      });

      socket.on("data", (chunk: string) => {
        if (socket === this.#socket) lines.push(chunk);
      });

      socket.on("error", (err) => {
        this.#output.appendLine(`[Daemon] Socket error: ${err.message}`);
        if (!established) {
          if (socket === this.#socket) this.#socket = null;
          reject(err);
        } else if (socket === this.#socket) {
          this.#onError.fire(err);
        }
      });

      socket.on("close", () => {
        if (socket !== this.#socket) {
          // Detached by disconnect() or a newer attempt: settle, don't reconnect.
          if (!established) reject(new Error("Connection attempt abandoned"));
          return;
        }
        this.#socket = null;
        if (!established) {
          reject(new Error("Socket closed before connecting"));
          return;
        }
        this.#output.appendLine("[Daemon] Socket closed");
        this.#rejectPending(new Error("Connection lost"));
        this.#scheduleReconnect();
      });

      socket.connect(this.#socketPath);
    });
  }

  #processMessage(line: string): void {
    let message: JsonRpcMessage;
    try {
      message = JSON.parse(line) as JsonRpcMessage;
    } catch (err) {
      this.#output.appendLine(`[Daemon] Failed to parse message: ${err}`);
      return;
    }

    if (typeof message.id !== "number") {
      if (message.method === Methods.WatchEvent) {
        this.#onWatchEvent.fire(message.params as WatchEventParams);
      } else if (message.error) {
        // Go omits the id when it cannot parse a request (protocol.go Response).
        this.#output.appendLine(
          `[Daemon] RPC error ${message.error.code}: ${message.error.message}`,
        );
      } else {
        this.#output.appendLine(`[Daemon] Unknown notification: ${message.method}`);
      }
      return;
    }

    const pending = this.#pendingRequests.get(message.id);
    if (!pending) {
      this.#output.appendLine(`[Daemon] Received response for unknown request ID: ${message.id}`);
      return;
    }
    this.#pendingRequests.delete(message.id);
    clearTimeout(pending.timeout);

    if (message.error) {
      pending.reject(new Error(`RPC error ${message.error.code}: ${message.error.message}`));
    } else {
      pending.resolve(message.result);
    }
  }

  #rejectPending(error: Error): void {
    for (const pending of this.#pendingRequests.values()) {
      clearTimeout(pending.timeout);
      pending.reject(error);
    }
    this.#pendingRequests.clear();
  }

  #scheduleReconnect(): void {
    if (this.#disposed) {
      this.#setConnectionState("disconnected");
      return;
    }
    if (this.#reconnectAttempts >= this.#maxReconnectAttempts) {
      this.#output.appendLine("[Daemon] Max reconnect attempts reached");
      this.#setConnectionState("disconnected");
      return;
    }

    const delay = this.#reconnectDelay * 2 ** this.#reconnectAttempts;
    this.#reconnectAttempts++;
    this.#output.appendLine(
      `[Daemon] Reconnecting in ${delay}ms (attempt ${this.#reconnectAttempts}/${this.#maxReconnectAttempts})`,
    );
    this.#setConnectionState("reconnecting");

    this.#reconnectTimer = setTimeout(() => {
      this.#reconnectTimer = null;
      this.#openSocket().catch((err: Error) => {
        this.#output.appendLine(`[Daemon] Reconnect failed: ${err.message}`);
        // connect() or disconnect() may have taken over meanwhile.
        if (this.#connectionState === "reconnecting") this.#scheduleReconnect();
      });
    }, delay);
  }

  #clearReconnectTimer(): void {
    if (this.#reconnectTimer) {
      clearTimeout(this.#reconnectTimer);
      this.#reconnectTimer = null;
    }
  }

  /**
   * Disconnect from the daemon socket without reconnecting.
   */
  disconnect(): void {
    this.#clearReconnectTimer();
    const socket = this.#socket;
    this.#socket = null; // detach first so its close event is ignored
    socket?.destroy();
    this.#rejectPending(new Error("Disconnected"));
    this.#setConnectionState("disconnected");
  }

  #sendRequest<T>(method: string, params?: unknown, timeoutMs = this.#requestTimeout): Promise<T> {
    const socket = this.#socket;
    if (!this.isConnected || !socket) {
      return Promise.reject(new Error("Not connected to daemon"));
    }

    const id = ++this.#requestId;
    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.#pendingRequests.delete(id);
        reject(new Error(`Request timeout: ${method}`));
      }, timeoutMs);

      this.#pendingRequests.set(id, {
        resolve: resolve as (result: unknown) => void,
        reject,
        timeout,
      });

      socket.write(
        `${JSON.stringify({ jsonrpc: JSONRPC_VERSION, id, method, params })}\n`,
        (err) => {
          if (err && this.#pendingRequests.delete(id)) {
            clearTimeout(timeout);
            reject(new Error(`Failed to send request: ${err.message}`));
          }
        },
      );
    });
  }

  ping(timeoutMs?: number): Promise<PingResult> {
    return this.#sendRequest<PingResult>(Methods.Ping, undefined, timeoutMs);
  }

  shutdown(): Promise<ShutdownResult> {
    return this.#sendRequest<ShutdownResult>(Methods.Shutdown);
  }

  watchStart(params?: WatchStartParams): Promise<WatchStartResult> {
    return this.#sendRequest<WatchStartResult>(Methods.WatchStart, params);
  }

  watchStop(): Promise<WatchStopResult> {
    return this.#sendRequest<WatchStopResult>(Methods.WatchStop);
  }

  watchStatus(): Promise<WatchStatusResult> {
    return this.#sendRequest<WatchStatusResult>(Methods.WatchStatus);
  }

  /**
   * Check whether the daemon is reachable and answers a ping promptly.
   */
  async isAvailable(): Promise<boolean> {
    try {
      await this.connect();
      await this.ping(PROBE_TIMEOUT_MS);
      return true;
    } catch {
      return false;
    }
  }

  dispose(): void {
    this.#disposed = true;
    this.disconnect();
    this.#onConnectionStateChanged.dispose();
    this.#onWatchEvent.dispose();
    this.#onError.dispose();
  }
}
