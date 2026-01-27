# Bazelle VSCode Extension

VSCode integration for [Bazelle](https://github.com/aspect-build/bazelle), a polyglot BUILD file generator for Bazel.

## Features

- **Status Bar Integration** - Shows Bazelle status and provides quick access to commands
- **BUILD File Management** - Update and fix BUILD files directly from VSCode
- **Watch Mode** - Automatically regenerate BUILD files when source files change
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

## Configuration

| Setting | Default | Description |
|---------|---------|-------------|
| `bazelle.path` | `"bazelle"` | Path to the bazelle executable |
| `bazelle.autoUpdate` | `false` | Automatically update BUILD files on save |
| `bazelle.statusBar.show` | `"onBazelWorkspace"` | When to show the status bar (`always`, `onBazelWorkspace`, `never`) |
| `bazelle.watch.enabled` | `false` | Enable watch mode on startup |
| `bazelle.watch.debounceMs` | `500` | Debounce delay for watch mode |

## Requirements

- [Bazelle](https://github.com/aspect-build/bazelle) CLI installed and accessible in PATH
- A Bazel workspace (with `MODULE.bazel`, `WORKSPACE`, or `BUILD` files)

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
