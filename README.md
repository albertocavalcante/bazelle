# Bazelle

Gazelle language extensions monorepo.

## Extensions

| Extension | Status | BCR | Description |
|-----------|--------|-----|-------------|
| [gazelle-kotlin](./gazelle-kotlin/) | 🚧 WIP | ❌ | Kotlin support for Gazelle |

## Architecture

This monorepo contains Gazelle language extensions with Copybara sync to standalone repos:

```
bazelle/                          # Monorepo (source of truth)
├── gazelle-kotlin/  ──Copybara──▶ albertocavalcante/gazelle-kotlin
├── gazelle-groovy/  ──Copybara──▶ (future)
└── ...
```

## Development

```bash
# Build all extensions
bazel build //...

# Test all extensions
bazel test //...

# Update BUILD files
bazel run //:gazelle
```

## License

Apache-2.0
