# Tree-sitter Abstraction Layer

This package provides a pluggable tree-sitter backend abstraction for bazelle, enabling accurate lossless parsing of source code for import extraction.

## Backends

| Backend | Implementation | CGO Required | Languages |
|---------|---------------|--------------|-----------|
| **CGO** | [smacker/go-tree-sitter](https://github.com/smacker/go-tree-sitter) | Yes | 29 languages |
| **Wazero** | [malivvan/tree-sitter](https://github.com/malivvan/tree-sitter) | No (WASM) | C, C++ (limited) |

## Usage

```go
import "github.com/albertocavalcante/bazelle/pkg/treesitter"

// Auto-select backend (CGO preferred, wazero fallback)
backend, err := treesitter.NewBackendFromEnv()
if err != nil {
    return err
}
defer backend.Close()

// Create parser for Java
parser, err := backend.NewParser(treesitter.Java)
if err != nil {
    return err
}
defer parser.Close()

// Parse source code
tree, err := parser.Parse(ctx, sourceCode)
if err != nil {
    return err
}
defer tree.Close()

// Extract imports using helper functions
imports := treesitter.FindByType(tree.RootNode(), "import_declaration")
```

## Backend Selection

Set the `BAZELLE_TREESITTER_BACKEND` environment variable:

```bash
export BAZELLE_TREESITTER_BACKEND=cgo     # Force CGO backend
export BAZELLE_TREESITTER_BACKEND=wazero  # Force WASM backend
export BAZELLE_TREESITTER_BACKEND=auto    # Auto-detect (default)
```

## Supported Languages (CGO Backend)

Go, Java, Kotlin, Scala, Rust, Python, JavaScript, TypeScript, TSX,
Groovy, C, C++, C#, Ruby, Swift, Bash, HTML, CSS, YAML, TOML,
Protobuf, HCL, Dockerfile, Lua, Elixir, Elm, OCaml, Svelte, Cue

### Excluded Languages

The following languages are excluded due to Bazel build compatibility issues:

| Language | Issue | Upstream Issue |
|----------|-------|----------------|
| PHP | Uses `#include "tree_sitter/parser.h"` instead of `#include "parser.h"` | [smacker/go-tree-sitter#175](https://github.com/smacker/go-tree-sitter/issues/175) |
| SQL | Same include path issue | Same root cause |
| Markdown | Complex multi-parser structure with same issue | Same root cause |

## Known Issues & Upstream References

### Bazel CGO Include Path Issue

**Problem**: Some go-tree-sitter language parsers use inconsistent include paths:
- Most languages: `#include "parser.h"` (works in Bazel) ✅
- php, sql: `#include "tree_sitter/parser.h"` (fails in Bazel) ❌

**Root Cause**: The generated `parser.c` files come from upstream tree-sitter grammars with different include conventions. When built with vanilla `go build`, CGO adds parent directories to the include path, making both styles work. Bazel's stricter include path handling exposes this inconsistency.

**Upstream Issues**:
- [smacker/go-tree-sitter#175](https://github.com/smacker/go-tree-sitter/issues/175) - Python: `../array.h` file not found
- [bazel-contrib/bazel-gazelle#2059](https://github.com/bazel-contrib/bazel-gazelle/issues/2059) - Incorrect BUILD file for go_library with C includes
- [bazel-contrib/rules_go#4298](https://github.com/bazel-contrib/rules_go/pull/4298) - Pass headers along as transitive dependencies (partial fix)

**Workarounds**:
1. Exclude problematic languages (current approach)
2. Fork go-tree-sitter and normalize include paths
3. Use `go_deps.module_override` with patches

### Wazero Backend Limitations

The [malivvan/tree-sitter](https://github.com/malivvan/tree-sitter) WASM backend currently only supports C and C++. To add more languages:
1. Fork the repo
2. Add language grammars to the Makefile
3. Rebuild the WASM bundle

## Future Considerations

### Should We Fork go-tree-sitter?

**Repo Status** (as of Jan 2025):
- Last commit: August 2024
- Stars: 533
- Open issues: 42
- Forks: 147

**Pros of forking**:
- Fix include path inconsistencies for Bazel compatibility
- Update to latest tree-sitter grammars
- Add missing languages

**Cons of forking**:
- Maintenance burden
- Divergence from upstream

**Recommendation**: The include path issue is an upstream problem that should ideally be fixed in go-tree-sitter or handled by gazelle/rules_go. For now, excluding 3 languages (php, sql, markdown) is acceptable since they're not critical for bazelle's primary use cases (Java, Kotlin, Scala, Go, Python, Rust, TypeScript).

If upstream remains inactive and more languages need Bazel support, consider:
1. Contributing a fix to go-tree-sitter
2. Forking only if upstream is unresponsive
