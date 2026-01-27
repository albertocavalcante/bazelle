# Bazelle

Polyglot Gazelle CLI - a unified BUILD file generator with multiple language extensions baked in.

## What is Bazelle?

Bazelle wraps [Bazel Gazelle](https://github.com/bazel-contrib/bazel-gazelle) with multiple language extensions pre-configured, so you don't have to set up complex toolchains and dependencies yourself.

**Supported languages:**
- Proto (built-in)
- Go (built-in)
- Kotlin (own extension)
- Python (via rules_python)
- C/C++ (via gazelle_cc)
- Java (coming soon - blocked on Bazel 9 compatibility)
- Groovy (planned)

## Quick Start

```bash
# Build the bazelle binary
bazel build //cmd/bazelle

# Run on your project
bazel run //cmd/bazelle -- update /path/to/your/project

# Or use the wrapper target
bazel run //:gazelle
```

## Daemon Mode

Bazelle supports running as a background daemon for improved performance and editor integration. The daemon eliminates startup overhead and allows multiple clients to share a single file watcher.

### Quick Start

```bash
# Start the daemon in the background
bazelle daemon start

# Check if daemon is running
bazelle daemon status

# Stop the daemon
bazelle daemon stop
```

### Daemon Commands

| Command | Description |
|---------|-------------|
| `bazelle daemon start` | Start the daemon (background by default) |
| `bazelle daemon stop` | Stop the running daemon |
| `bazelle daemon status` | Show daemon status and info |
| `bazelle daemon restart` | Restart the daemon |

### Command Options

**`daemon start`**
```bash
bazelle daemon start [flags]

Flags:
  --foreground    Run in foreground (don't daemonize, useful for debugging)
  --socket PATH   Custom socket path (default: ~/.bazelle/daemon.sock)
  --log PATH      Log file path (default: ~/.bazelle/daemon.log)
```

**`daemon stop`**
```bash
bazelle daemon stop [flags]

Flags:
  --force         Force kill if graceful shutdown fails (sends SIGKILL)
  --socket PATH   Custom socket path
```

**`daemon status`**
```bash
bazelle daemon status [flags]

Flags:
  --json          Output as JSON (for scripting)
  --socket PATH   Custom socket path
```

**`daemon restart`**
```bash
bazelle daemon restart [flags]

Flags:
  --force         Force kill if graceful stop fails
  --socket PATH   Custom socket path
```

### File Locations

The daemon stores its files in `~/.bazelle/`:

| File | Purpose |
|------|---------|
| `daemon.sock` | Unix domain socket for client connections |
| `daemon.pid` | PID file containing the daemon process ID |
| `daemon.log` | Log output when running in background mode |

### Features

- **Persistent file watching** - No startup overhead for each operation
- **Multi-client support** - CLI, VS Code, and other tools can connect simultaneously
- **Automatic BUILD file updates** - Watch for file changes and regenerate BUILD files
- **Graceful shutdown** - Handles SIGTERM/SIGINT signals properly
- **Crash recovery** - Automatically cleans up stale socket/PID files

### VS Code Integration

The [VS Code extension](editors/code/) automatically connects to the daemon when available:

1. Start the daemon: `bazelle daemon start`
2. In VS Code, run "Bazelle: Start Watch Mode"
3. The extension connects to the daemon (falls back to subprocess if unavailable)

Configure daemon behavior in VS Code settings:
```json
{
  "bazelle.daemon.enabled": true,     // Use daemon when available (default: true)
  "bazelle.daemon.autoStart": false,  // Auto-start daemon if not running
  "bazelle.daemon.socketPath": null   // Custom socket path (optional)
}
```

### Troubleshooting

**Daemon not starting:**
```bash
# Check for stale files
ls -la ~/.bazelle/

# Clean up stale files (if daemon crashed)
rm -f ~/.bazelle/daemon.sock ~/.bazelle/daemon.pid

# Start with verbose logging
bazelle daemon start --foreground
```

**Connection refused:**
```bash
# Verify daemon is running
bazelle daemon status

# Check socket file exists and has correct permissions
ls -la ~/.bazelle/daemon.sock
```

**View daemon logs:**
```bash
# Background daemon logs
tail -f ~/.bazelle/daemon.log

# Or run in foreground to see logs in terminal
bazelle daemon stop
bazelle daemon start --foreground
```

See [docs/specs/daemon-mode-phase1.md](docs/specs/daemon-mode-phase1.md) for technical details and protocol specification.

## Extensions

| Extension | Status | BCR | Description |
|-----------|--------|-----|-------------|
| [gazelle-kotlin](./gazelle-kotlin/) | 🚧 WIP | ❌ | Kotlin support (kt_jvm_library, kt_jvm_test) |
| gazelle-groovy | 📋 Planned | ❌ | Groovy support (groovy_library, groovy_test) |

### Third-Party Extensions (via bazel_dep)

| Extension | Status | Description |
|-----------|--------|-------------|
| [@rules_python_gazelle_plugin](https://github.com/bazelbuild/rules_python) | ✅ Enabled | Python support |
| [@gazelle_cc](https://github.com/EngFlow/gazelle_cc) | ✅ Enabled | C/C++ support |
| [@contrib_rules_jvm](https://github.com/bazel-contrib/rules_jvm) | ⏸️ Disabled | Java support (Bazel 9 incompatible) |

## Architecture

```
bazelle/
├── cmd/bazelle/           # Polyglot CLI binary
├── gazelle-kotlin/        # Kotlin extension ──Copybara──▶ standalone repo
├── gazelle-groovy/        # (future) Groovy extension
└── internal/              # Shared utilities
```

This monorepo contains Gazelle language extensions with Copybara sync to standalone repos for BCR publishing.

## Configuration

Add directives to your root `BUILD.bazel`:

```starlark
# Enable/disable extensions per directory
# gazelle:kotlin_enabled true
# gazelle:python_extension enabled
# gazelle:cc_generate true
```

## Development

```bash
# Build all extensions
bazel build //...

# Test all extensions
bazel test //...

# Update BUILD files (using bazelle)
bazel run //:gazelle

# Build standalone binary
bazel build //cmd/bazelle
./bazel-bin/cmd/bazelle/bazelle_/bazelle --help
```

## Roadmap

- [ ] Java support (waiting for contrib_rules_jvm Bazel 9 fix)
- [ ] Groovy extension (gazelle-groovy)
- [ ] Hermetic C/C++ toolchains for cross-compilation
- [ ] Pre-built binaries (standalone distribution)
- [ ] BCR publishing for extensions

## License

Apache-2.0
