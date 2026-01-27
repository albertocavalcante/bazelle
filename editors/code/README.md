# Bazelle VSCode Extension

VSCode integration for [Bazelle](https://github.com/albertocavalcante/bazelle), a polyglot BUILD file generator for Bazel.

## Features

- **Status Bar Integration** - Shows Bazelle status and provides quick access to commands
- **BUILD File Management** - Update and fix BUILD files directly from VSCode
- **Watch Mode** - Automatically regenerate BUILD files when source files change
- **Daemon Integration** - Connect to bazelle daemon for improved performance
- **Starlark Syntax Highlighting** - Proper highlighting for BUILD files with gazelle directive support

## Commands

| Command | Description |
|---------|-------------|
| `Bazelle: Update BUILD Files` | Run `bazelle update` to regenerate BUILD files |
| `Bazelle: Update BUILD Files (Incremental)` | Run `bazelle update --incremental` for faster updates |
| `Bazelle: Fix BUILD Files` | Run `bazelle fix` to fix BUILD file issues |
| `Bazelle: Fix BUILD Files (Dry Run)` | Preview what `bazelle fix` would change |
| `Bazelle: Show Status` | Check which directories have stale BUILD files |
| `Bazelle: Start Watch Mode` | Start watching for file changes |
| `Bazelle: Stop Watch Mode` | Stop the watch mode |
| `Bazelle: Initialize Project` | Initialize a new Bazel project |
| `Bazelle: Show Output` | Open the Bazelle output channel |

### Daemon Commands

| Command | Description |
|---------|-------------|
| `Bazelle: Start Daemon` | Start the bazelle daemon process in background |
| `Bazelle: Stop Daemon` | Stop the running daemon gracefully |
| `Bazelle: Daemon Status` | Show daemon status (PID, uptime, watch state) |
| `Bazelle: Restart Daemon` | Stop and restart the daemon |

## Daemon Mode

The daemon provides improved performance by keeping the bazelle process running in the background. This eliminates startup overhead and allows multiple clients to share a single file watcher.

### How It Works

1. **Daemon Process**: A background process listens on a Unix socket (`~/.bazelle/daemon.sock`)
2. **JSON-RPC Protocol**: The extension communicates with the daemon using JSON-RPC 2.0
3. **Automatic Fallback**: If the daemon is unavailable, the extension falls back to subprocess mode

### Daemon vs Subprocess Mode

| Feature | Daemon Mode | Subprocess Mode |
|---------|------------|-----------------|
| Startup time | Instant (already running) | ~500ms per operation |
| Memory | Shared across clients | Separate per operation |
| File watching | Single shared watcher | New watcher per session |
| Multi-client | Yes (CLI + VS Code + others) | No |
| Requires setup | Yes (`bazelle daemon start`) | No |

### Using Daemon Mode

1. **Start the daemon** (from terminal or VS Code):
   ```bash
   bazelle daemon start
   ```
   Or use the "Bazelle: Start Daemon" command in VS Code.

2. **Start watch mode** in VS Code:
   - Run "Bazelle: Start Watch Mode" from Command Palette
   - The extension automatically connects to the daemon if available
   - Check the Output panel for connection status

3. **Verify connection**:
   - Run "Bazelle: Daemon Status" to see daemon info
   - The status bar shows the current mode (daemon/subprocess)

### Automatic Fallback

When watch mode starts, the extension:
1. Checks if `bazelle.daemon.enabled` is true (default)
2. Attempts to connect to the daemon socket
3. If connection fails, falls back to spawning `bazelle watch` as a subprocess
4. Logs the active mode in the Output panel

## Configuration

### General Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `bazelle.path` | `"bazelle"` | Path to the bazelle executable |
| `bazelle.autoUpdate` | `false` | Automatically update BUILD files on save |
| `bazelle.statusBar.show` | `"onBazelWorkspace"` | When to show status bar: `always`, `onBazelWorkspace`, `never` |

### Watch Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `bazelle.watch.enabled` | `false` | Start watch mode automatically on workspace open |
| `bazelle.watch.debounceMs` | `500` | Debounce delay in milliseconds for file changes |

### Daemon Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `bazelle.daemon.enabled` | `true` | Use daemon when available (falls back to subprocess) |
| `bazelle.daemon.socketPath` | `undefined` | Custom socket path (default: `~/.bazelle/daemon.sock`) |
| `bazelle.daemon.autoStart` | `false` | Automatically start daemon if not running |

### Example Configuration

```json
{
  // Use bundled or PATH bazelle
  "bazelle.path": "bazelle",

  // Don't auto-update on save (use watch mode instead)
  "bazelle.autoUpdate": false,

  // Show status bar in Bazel workspaces
  "bazelle.statusBar.show": "onBazelWorkspace",

  // Don't start watch on open (manual start)
  "bazelle.watch.enabled": false,
  "bazelle.watch.debounceMs": 500,

  // Enable daemon with auto-start
  "bazelle.daemon.enabled": true,
  "bazelle.daemon.autoStart": true
}
```

## Troubleshooting

### Daemon Connection Issues

**"Failed to connect to daemon"**
```bash
# Check if daemon is running
bazelle daemon status

# Start daemon if not running
bazelle daemon start

# Check socket file exists
ls -la ~/.bazelle/daemon.sock
```

**"Daemon not running" but socket exists**
```bash
# Clean up stale files (daemon may have crashed)
rm -f ~/.bazelle/daemon.sock ~/.bazelle/daemon.pid
bazelle daemon start
```

**Connection timeouts**
- Increase timeout in extension settings
- Check if daemon is overloaded (many watched files)
- Try restarting the daemon: `bazelle daemon restart`

### Watch Mode Issues

**Watch mode falls back to subprocess**
1. Check Output panel for error messages
2. Verify daemon is running: `bazelle daemon status`
3. Check daemon logs: `tail -f ~/.bazelle/daemon.log`

**Changes not detected**
1. Check that files are in watched paths
2. Increase debounce time if updates are too frequent
3. Restart watch mode

### Viewing Logs

- **Extension logs**: Run "Bazelle: Show Output" command
- **Daemon logs**: `tail -f ~/.bazelle/daemon.log`
- **Daemon foreground**: `bazelle daemon start --foreground`

### Socket Permission Issues

If you see permission errors:
```bash
# Check socket permissions (should be 0600)
ls -la ~/.bazelle/daemon.sock

# Remove and restart daemon
rm -f ~/.bazelle/daemon.sock
bazelle daemon start
```

## Requirements

- [Bazelle](https://github.com/albertocavalcante/bazelle) CLI installed and accessible in PATH
- A Bazel workspace (with `MODULE.bazel`, `WORKSPACE`, or `BUILD` files)
- For daemon mode: Unix-like OS (macOS, Linux) - daemon uses Unix sockets

## Installation

1. Install the extension from VSCode Marketplace (coming soon)
2. Or build from source:
   ```bash
   cd editors/code
   bun install
   bun run build
   bun run package
   code --install-extension bazelle-*.vsix
   ```

## Development

```bash
# Install dependencies
bun install

# Build the extension
bun run build

# Watch for changes
bun run watch

# Package the extension
bun run package
```

## License

Apache-2.0
