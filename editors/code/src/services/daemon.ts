/**
 * Daemon client - connects to bazelle daemon via Unix socket
 *
 * Implements JSON-RPC 2.0 protocol over Unix socket connection.
 */

import * as net from "node:net";
import * as os from "node:os";
import * as path from "node:path";
import * as vscode from "vscode";

// JSON-RPC 2.0 types matching cmd/bazelle/internal/daemon/protocol.go

const JSONRPC_VERSION = "2.0";

interface JsonRpcRequest {
  jsonrpc: string;
  id: number;
  method: string;
  params?: unknown;
}

interface JsonRpcNotification {
  jsonrpc: string;
  method: string;
  params?: unknown;
}

interface JsonRpcResponse {
  jsonrpc: string;
  id: number | null;
  result?: unknown;
  error?: JsonRpcError;
}

interface JsonRpcError {
  code: number;
  message: string;
  data?: unknown;
}

// RPC method names (matching protocol.go)
const Methods = {
  Ping: "ping",
  Shutdown: "shutdown",
  WatchStart: "watch/start",
  WatchStop: "watch/stop",
  WatchStatus: "watch/status",
  WatchEvent: "watch/event", // notification from server
  UpdateRun: "update/run",
  StatusGet: "status/get",
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
  type: "change" | "update" | "error";
  directories?: string[];
  files?: string[];
  message?: string;
  timestamp: string;
}

export interface UpdateRunParams {
  paths?: string[];
  incremental?: boolean;
}

export interface UpdateRunResult {
  status: string;
  updated_dirs?: string[];
  duration?: string;
}

export interface StatusGetResult {
  stale: boolean;
  stale_dirs?: string[];
}

// Connection state
export type DaemonConnectionState = "disconnected" | "connecting" | "connected" | "reconnecting";

// Events emitted by daemon client
export interface DaemonEvents {
  connectionStateChanged: DaemonConnectionState;
  watchEvent: WatchEventParams;
  error: Error;
}

// Pending request tracker
interface PendingRequest {
  resolve: (result: unknown) => void;
  reject: (error: Error) => void;
  timeout: ReturnType<typeof setTimeout>;
}

export class DaemonClient implements vscode.Disposable {
  readonly #output: vscode.OutputChannel;
  readonly #onConnectionStateChanged = new vscode.EventEmitter<DaemonConnectionState>();
  readonly #onWatchEvent = new vscode.EventEmitter<WatchEventParams>();
  readonly #onError = new vscode.EventEmitter<Error>();

  #socket: net.Socket | null = null;
  #socketPath: string;
  #connectionState: DaemonConnectionState = "disconnected";
  #buffer = "";
  #requestId = 0;
  #pendingRequests = new Map<number, PendingRequest>();
  #reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  #reconnectAttempts = 0;
  #disposed = false;

  // Configuration
  readonly #requestTimeout: number;
  readonly #maxReconnectAttempts: number;
  readonly #reconnectDelay: number;

  // Public events
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
    this.#socketPath = options?.socketPath ?? this.#getDefaultSocketPath();
    this.#requestTimeout = options?.requestTimeout ?? 30000;
    this.#maxReconnectAttempts = options?.maxReconnectAttempts ?? 5;
    this.#reconnectDelay = options?.reconnectDelay ?? 1000;
  }

  #getDefaultSocketPath(): string {
    const homeDir = os.homedir();
    return path.join(homeDir, ".bazelle", "daemon.sock");
  }

  get socketPath(): string {
    return this.#socketPath;
  }

  set socketPath(value: string) {
    if (this.#connectionState !== "disconnected") {
      throw new Error("Cannot change socket path while connected");
    }
    this.#socketPath = value;
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
      this.#onConnectionStateChanged.fire(state);
      this.#output.appendLine(`[Daemon] Connection state: ${state}`);
    }
  }

  /**
   * Connect to the daemon socket
   */
  async connect(): Promise<void> {
    if (this.#disposed) {
      throw new Error("Client has been disposed");
    }

    if (this.#connectionState === "connected" || this.#connectionState === "connecting") {
      return;
    }

    this.#setConnectionState("connecting");
    this.#reconnectAttempts = 0;

    return this.#doConnect();
  }

  async #doConnect(): Promise<void> {
    return new Promise((resolve, reject) => {
      this.#socket = net.createConnection(this.#socketPath);

      this.#socket.on("connect", () => {
        this.#setConnectionState("connected");
        this.#reconnectAttempts = 0;
        this.#output.appendLine(`[Daemon] Connected to ${this.#socketPath}`);
        resolve();
      });

      this.#socket.on("data", (data: Buffer) => {
        this.#handleData(data);
      });

      this.#socket.on("error", (err: NodeJS.ErrnoException) => {
        this.#output.appendLine(`[Daemon] Socket error: ${err.message}`);

        if (this.#connectionState === "connecting") {
          this.#setConnectionState("disconnected");
          reject(new Error(`Failed to connect to daemon: ${err.message}`));
        } else {
          this.#onError.fire(err);
          this.#handleDisconnect();
        }
      });

      this.#socket.on("close", () => {
        this.#output.appendLine("[Daemon] Socket closed");
        this.#handleDisconnect();
      });

      this.#socket.on("end", () => {
        this.#output.appendLine("[Daemon] Socket ended");
      });
    });
  }

  #handleData(data: Buffer): void {
    this.#buffer += data.toString("utf-8");

    // Process newline-delimited JSON messages
    for (;;) {
      const newlineIndex = this.#buffer.indexOf("\n");
      if (newlineIndex === -1) {
        break;
      }
      const line = this.#buffer.slice(0, newlineIndex);
      this.#buffer = this.#buffer.slice(newlineIndex + 1);

      if (line.trim()) {
        this.#processMessage(line);
      }
    }
  }

  #processMessage(line: string): void {
    try {
      const message = JSON.parse(line) as JsonRpcResponse | JsonRpcNotification;

      // Check if it's a notification (no id field or id is null)
      if (!("id" in message) || message.id === null) {
        this.#handleNotification(message as JsonRpcNotification);
        return;
      }

      // It's a response - id is guaranteed to be a number at this point
      const response = message as JsonRpcResponse;
      const responseId = response.id as number;
      const pending = this.#pendingRequests.get(responseId);

      if (!pending) {
        this.#output.appendLine(`[Daemon] Received response for unknown request ID: ${responseId}`);
        return;
      }

      this.#pendingRequests.delete(responseId);
      clearTimeout(pending.timeout);

      if (response.error) {
        pending.reject(new Error(`RPC error ${response.error.code}: ${response.error.message}`));
      } else {
        pending.resolve(response.result);
      }
    } catch (err) {
      this.#output.appendLine(`[Daemon] Failed to parse message: ${err}`);
    }
  }

  #handleNotification(notification: JsonRpcNotification): void {
    switch (notification.method) {
      case Methods.WatchEvent:
        this.#onWatchEvent.fire(notification.params as WatchEventParams);
        break;
      default:
        this.#output.appendLine(`[Daemon] Unknown notification: ${notification.method}`);
    }
  }

  #handleDisconnect(): void {
    // Clear all pending requests
    for (const [id, pending] of this.#pendingRequests) {
      clearTimeout(pending.timeout);
      pending.reject(new Error("Connection lost"));
      this.#pendingRequests.delete(id);
    }

    this.#socket = null;
    this.#buffer = "";

    if (this.#disposed) {
      this.#setConnectionState("disconnected");
      return;
    }

    // Attempt reconnect if not at max attempts
    if (this.#reconnectAttempts < this.#maxReconnectAttempts) {
      this.#setConnectionState("reconnecting");
      this.#scheduleReconnect();
    } else {
      this.#setConnectionState("disconnected");
      this.#output.appendLine("[Daemon] Max reconnect attempts reached");
    }
  }

  #scheduleReconnect(): void {
    if (this.#reconnectTimer) {
      clearTimeout(this.#reconnectTimer);
    }

    const delay = this.#reconnectDelay * 2 ** this.#reconnectAttempts;
    this.#reconnectAttempts++;

    this.#output.appendLine(
      `[Daemon] Reconnecting in ${delay}ms (attempt ${this.#reconnectAttempts}/${this.#maxReconnectAttempts})`,
    );

    this.#reconnectTimer = setTimeout(() => {
      this.#doConnect().catch((err) => {
        this.#output.appendLine(`[Daemon] Reconnect failed: ${err.message}`);
        this.#handleDisconnect();
      });
    }, delay);
  }

  /**
   * Disconnect from the daemon socket
   */
  disconnect(): void {
    if (this.#reconnectTimer) {
      clearTimeout(this.#reconnectTimer);
      this.#reconnectTimer = null;
    }

    if (this.#socket) {
      this.#socket.destroy();
      this.#socket = null;
    }

    this.#setConnectionState("disconnected");
  }

  /**
   * Send a JSON-RPC request and wait for response
   */
  async #sendRequest<T>(method: string, params?: unknown): Promise<T> {
    if (!this.isConnected || !this.#socket) {
      throw new Error("Not connected to daemon");
    }

    const id = ++this.#requestId;
    const request: JsonRpcRequest = {
      jsonrpc: JSONRPC_VERSION,
      id,
      method,
      params,
    };

    return new Promise((resolve, reject) => {
      const timeout = setTimeout(() => {
        this.#pendingRequests.delete(id);
        reject(new Error(`Request timeout: ${method}`));
      }, this.#requestTimeout);

      this.#pendingRequests.set(id, {
        resolve: resolve as (result: unknown) => void,
        reject,
        timeout,
      });

      const message = `${JSON.stringify(request)}\n`;
      const socket = this.#socket;
      if (!socket) {
        reject(new Error("Socket is not connected"));
        return;
      }
      socket.write(message, (err) => {
        if (err) {
          this.#pendingRequests.delete(id);
          clearTimeout(timeout);
          reject(new Error(`Failed to send request: ${err.message}`));
        }
      });
    });
  }

  // RPC Methods

  /**
   * Ping the daemon to check if it's alive
   */
  async ping(): Promise<PingResult> {
    return this.#sendRequest<PingResult>(Methods.Ping);
  }

  /**
   * Shutdown the daemon
   */
  async shutdown(): Promise<ShutdownResult> {
    return this.#sendRequest<ShutdownResult>(Methods.Shutdown);
  }

  /**
   * Start watching for file changes
   */
  async watchStart(params?: WatchStartParams): Promise<WatchStartResult> {
    return this.#sendRequest<WatchStartResult>(Methods.WatchStart, params);
  }

  /**
   * Stop watching for file changes
   */
  async watchStop(): Promise<WatchStopResult> {
    return this.#sendRequest<WatchStopResult>(Methods.WatchStop);
  }

  /**
   * Get current watch status
   */
  async watchStatus(): Promise<WatchStatusResult> {
    return this.#sendRequest<WatchStatusResult>(Methods.WatchStatus);
  }

  /**
   * Run an update
   */
  async updateRun(params?: UpdateRunParams): Promise<UpdateRunResult> {
    return this.#sendRequest<UpdateRunResult>(Methods.UpdateRun, params);
  }

  /**
   * Get status of BUILD files
   */
  async statusGet(): Promise<StatusGetResult> {
    return this.#sendRequest<StatusGetResult>(Methods.StatusGet);
  }

  /**
   * Check if daemon is available (socket exists and is connectable)
   */
  async isAvailable(): Promise<boolean> {
    try {
      // Check if socket file exists
      const fs = await import("node:fs/promises");
      await fs.access(this.#socketPath);

      // Try to connect and ping
      if (!this.isConnected) {
        await this.connect();
      }
      await this.ping();
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

/**
 * Get daemon configuration from VS Code settings
 */
export function getDaemonConfig(): {
  enabled: boolean;
  socketPath: string | undefined;
  autoStart: boolean;
} {
  const config = vscode.workspace.getConfiguration("bazelle");
  return {
    enabled: config.get<boolean>("daemon.enabled", true),
    socketPath: config.get<string>("daemon.socketPath"),
    autoStart: config.get<boolean>("daemon.autoStart", false),
  };
}
