# Kotlin Parser

This directory contains parser infrastructure for the Gazelle Kotlin extension.

## Current Implementation

The current parser (`../parser.go` and `../fqn_scanner.go`) uses:

1. **Regex-based import parsing** - Fast, pure Go, handles most cases
2. **FQN heuristic scanning** - Detects fully qualified names used inline

This approach is sufficient for most Kotlin projects and provides:
- Star imports (`import com.example.*`)
- Aliased imports (`import com.example.Foo as Bar`)
- File annotations (`@file:JvmName`)
- FQN detection in code body (for AI-generated code resilience)

## ANTLR Grammar (Future Enhancement)

The `grammar/KotlinImports.g4` file contains a minimal ANTLR grammar focused only on:
- Package declarations
- Import statements

This is **not** a full Kotlin grammar - it's intentionally minimal for performance.

### Why ANTLR?

- **Pure Go runtime** - No CGO required (unlike tree-sitter)
- **Better edge case handling** - Proper lexer handles comments, strings, etc.
- **Cross-compilation** - Works perfectly with `go build` for any platform

### Generating Go Parser (when needed)

1. Install ANTLR4:
   ```bash
   # macOS
   brew install antlr

   # Or download jar
   curl -O https://www.antlr.org/download/antlr-4.13.1-complete.jar
   ```

2. Generate Go code:
   ```bash
   cd grammar
   antlr4 -Dlanguage=Go -o ../antlr KotlinImports.g4
   ```

3. Add antlr4-go runtime dependency:
   ```bash
   # In MODULE.bazel, add:
   go_deps.from_file(go_mod = "//:go.mod")

   # Then in go.mod:
   require github.com/antlr4-go/antlr/v4 v4.13.0
   ```

### When to Use ANTLR

Consider switching to ANTLR parser when:
- Regex parsing hits edge cases (rare)
- Need to parse more than imports (type references in signatures)
- Want guaranteed correct comment/string handling

For now, the regex + FQN scanner approach is recommended as it's:
- Simpler
- Faster
- Zero dependencies
- Sufficient for dependency resolution
