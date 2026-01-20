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
