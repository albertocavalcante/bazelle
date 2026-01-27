# Bazelle Daemon Mode - Phase 1 Specification

## Overview

Phase 1 implements the foundational daemon infrastructure: a background server process that clients can connect to via Unix domain sockets, with basic lifecycle management commands.

## Goals

1. Single daemon process per workspace that runs in the background
2. Unix socket-based IPC for client-daemon communication
3. CLI commands for daemon lifecycle management
4. Migrate existing watch functionality to run inside daemon
5. VS Code extension connects to daemon instead of spawning subprocess

## Non-Goals (Future Phases)

- TCP/network support (Phase 2)
- Authentication/authorization (Phase 2)
- Multi-workspace daemon (Phase 2)
- systemd/launchd service files (Phase 3)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Daemon Process                          │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │
│  │   Watcher   │  │  Debouncer  │  │   Gazelle Runner    │  │
│  └──────┬──────┘  └──────┬──────┘  └──────────┬──────────┘  │
│         │                │                     │             │
│         └────────────────┴─────────────────────┘             │
│                          │                                   │
│                   ┌──────┴──────┐                           │
│                   │   Server    │                           │
│                   │ (RPC over   │                           │
│                   │ Unix Socket)│                           │
│                   └──────┬──────┘                           │
└──────────────────────────┼──────────────────────────────────┘
                           │ ~/.bazelle/daemon.sock
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   ┌────┴────┐       ┌─────┴─────┐      ┌─────┴─────┐
   │ CLI     │       │ VS Code   │      │ Other     │
   │ Client  │       │ Extension │      │ Tools     │
   └─────────┘       └───────────┘      └───────────┘
```

## File Structure

```
cmd/bazelle/internal/daemon/
├── server.go       # Unix socket server, connection handling
├── client.go       # Client library for connecting to daemon
├── protocol.go     # JSON-RPC message types
├── lifecycle.go    # PID file, lockfile, process management
└── handler.go      # RPC method handlers

cmd/bazelle/internal/cli/
├── daemon.go       # New: daemon command group
├── daemon_start.go # New: start subcommand
├── daemon_stop.go  # New: stop subcommand
└── daemon_status.go# New: status subcommand
```

## Socket & PID File Locations

```
~/.bazelle/
├── daemon.sock     # Unix domain socket
├── daemon.pid      # PID file with process ID
└── daemon.log      # Daemon log output (when not in foreground)
```

For workspace-specific daemons (future):
```
<workspace>/.bazelle/
├── daemon.sock
├── daemon.pid
└── daemon.log
```

## Protocol

JSON-RPC 2.0 over Unix socket with newline-delimited messages.

### Message Format

**Request:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "watch/start",
  "params": {
    "paths": ["/path/to/workspace"],
    "languages": ["go", "python"]
  }
}
```

**Response:**
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "status": "watching",
    "paths": ["/path/to/workspace"]
  }
}
```

**Notification (server → client):**
```json
{
  "jsonrpc": "2.0",
  "method": "watch/event",
  "params": {
    "type": "update",
    "directories": ["pkg/foo"],
    "timestamp": "2024-01-27T12:00:00Z"
  }
}
```

### RPC Methods

| Method | Direction | Description |
|--------|-----------|-------------|
| `ping` | client→server | Health check, returns `pong` |
| `shutdown` | client→server | Graceful daemon shutdown |
| `watch/start` | client→server | Start watching paths |
| `watch/stop` | client→server | Stop watching |
| `watch/status` | client→server | Get current watch status |
| `watch/event` | server→client | File change notification |
| `update/run` | client→server | Trigger manual update |
| `status/get` | client→server | Get staleness status |

## CLI Commands

### `bazelle daemon start`

Start the daemon process in the background.

```bash
bazelle daemon start [flags]

Flags:
  --foreground    Run in foreground (don't daemonize)
  --socket PATH   Custom socket path (default: ~/.bazelle/daemon.sock)
  --log PATH      Log file path (default: ~/.bazelle/daemon.log)
```

**Behavior:**
1. Check if daemon already running (via PID file + process check)
2. If running, print "Daemon already running (PID: xxx)" and exit 0
3. Fork/exec background process (or run foreground if flag set)
4. Write PID to `daemon.pid`
5. Create Unix socket and start listening
6. Print "Daemon started (PID: xxx)"

### `bazelle daemon stop`

Stop the running daemon.

```bash
bazelle daemon stop [flags]

Flags:
  --force        Force kill if graceful shutdown fails
  --socket PATH  Custom socket path (if using non-default)
```

**Behavior:**
1. Read PID from `daemon.pid`
2. Send `shutdown` RPC via socket
3. Wait up to 5 seconds for graceful shutdown
4. If `--force` and still running, send SIGKILL
5. Remove PID file and socket file
6. Print "Daemon stopped"

### `bazelle daemon status`

Check daemon status.

```bash
bazelle daemon status [flags]

Flags:
  --json         Output as JSON
  --socket PATH  Custom socket path (if using non-default)
```

**Output:**
```
Daemon: running (PID: 12345)
Socket: ~/.bazelle/daemon.sock
Uptime: 2h 15m
Watching: 3 paths
  - /home/user/project1
  - /home/user/project2
  - /home/user/project3
Connected clients: 2
```

### `bazelle daemon restart`

Convenience command: stop + start.

```bash
bazelle daemon restart [flags]

Flags:
  --force        Force kill if graceful stop fails
  --socket PATH  Custom socket path (if using non-default)
```

## Lifecycle Management

### PID File (`daemon.pid`)

```
12345
```

Simple text file containing just the PID.

### Startup Sequence

1. Acquire exclusive lock on `daemon.lock` (flock)
2. Check for stale PID file (process not running)
3. Clean up stale socket file if exists
4. Write new PID file
5. Create Unix socket
6. Initialize watcher, debouncer
7. Release lock, enter main loop

### Shutdown Sequence

1. Stop accepting new connections
2. Send shutdown notification to connected clients
3. Stop watcher
4. Close all client connections
5. Remove socket file
6. Remove PID file
7. Exit

### Crash Recovery

On startup, detect and clean up after crash:
- Stale PID file (process not running) → remove
- Stale socket file → remove
- Incomplete state → reinitialize

## Server Implementation

### Connection Handling

```go
type Server struct {
    socketPath string
    listener   net.Listener
    clients    map[*Client]bool
    clientsMu  sync.RWMutex

    watcher    *watch.Watcher
    debouncer  *watch.Debouncer

    shutdown   chan struct{}
}

type Client struct {
    conn     net.Conn
    encoder  *json.Encoder
    decoder  *json.Decoder
    server   *Server

    // Subscriptions
    watchEvents bool
}
```

### Concurrency Model

- One goroutine per client connection
- Watcher runs in dedicated goroutine
- Debouncer runs in dedicated goroutine
- Main goroutine handles accept loop
- Graceful shutdown via context cancellation

## Client Implementation

### CLI Client

```go
type Client struct {
    conn    net.Conn
    encoder *json.Encoder
    decoder *json.Decoder
    nextID  atomic.Int64
}

func Connect(socketPath string) (*Client, error)
func (c *Client) Ping() error
func (c *Client) Shutdown() error
func (c *Client) WatchStart(paths []string, languages []string) error
func (c *Client) WatchStop() error
func (c *Client) WatchStatus() (*WatchStatus, error)
func (c *Client) SubscribeEvents(handler func(Event)) error
```

### VS Code Integration

Update `editors/code/src/services/watch.ts`:

```typescript
class DaemonClient {
  private socket: net.Socket;

  async connect(): Promise<void>;
  async ping(): Promise<void>;
  async watchStart(paths: string[]): Promise<void>;
  async watchStop(): Promise<void>;
  onEvent(handler: (event: WatchEvent) => void): void;
}
```

Fallback behavior: if daemon not running, spawn `bazelle watch` as before.

## Error Handling

| Error | Response |
|-------|----------|
| Daemon not running | Exit 1: "Daemon not running. Start with: bazelle daemon start" |
| Socket permission denied | Exit 1: "Cannot connect to daemon: permission denied" |
| Daemon already running | Exit 0: "Daemon already running (PID: xxx)" |
| Invalid RPC method | JSON-RPC error: code -32601 "Method not found" |
| Invalid params | JSON-RPC error: code -32602 "Invalid params" |

## Testing Strategy

1. **Unit tests**: Protocol encoding/decoding, lifecycle helpers
2. **Integration tests**: Start/stop daemon, client connection
3. **E2E tests**: Full flow with file changes triggering updates

## Migration Path

1. Implement daemon infrastructure (this phase)
2. `bazelle watch` continues to work as-is (foreground)
3. Add `bazelle daemon` commands
4. Update VS Code extension to prefer daemon, fallback to watch
5. Future: deprecate standalone watch in favor of daemon

## Success Criteria

- [x] `bazelle daemon start` launches background process
- [x] `bazelle daemon stop` cleanly shuts down daemon
- [x] `bazelle daemon status` shows daemon info
- [x] Multiple clients can connect simultaneously
- [x] File changes trigger notifications to all connected clients
- [x] Daemon survives client disconnections
- [x] Clean shutdown on SIGTERM/SIGINT
- [x] Crash recovery works (stale files cleaned up)
- [x] VS Code extension can connect to daemon

## Implementation Status

Phase 1 implementation is complete. All success criteria have been met:

- **CLI Commands**: `daemon start`, `stop`, `status`, `restart` all implemented
- **Server**: Unix socket server with JSON-RPC 2.0 protocol
- **Client**: Go client library and TypeScript client for VS Code
- **Watch Integration**: Daemon runs file watcher, broadcasts events to clients
- **VS Code Extension**: Connects to daemon, falls back to subprocess mode
- **Testing**: Unit tests for protocol, lifecycle, handlers; integration tests for server/client
