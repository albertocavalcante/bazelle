# Bazelle development tasks

# Default: show available recipes
default:
    @just --list

# ─────────────────────────────────────────────────────────────────────────────
# Build & Test
# ─────────────────────────────────────────────────────────────────────────────

# Install bazelle to ~/.local/bin
install:
    bazel run //:install

# Build bazelle
build:
    bazel build //cmd/bazelle

# Run all tests
test:
    bazel test //...

# Update BUILD files
update:
    bazel run //:install && bazelle update

# Check if BUILD files are up to date
check:
    bazel run //:install && bazelle update --check

# ─────────────────────────────────────────────────────────────────────────────
# Documentation
# ─────────────────────────────────────────────────────────────────────────────

# Install docs dependencies
docs-install:
    cd docs && bun install

# Build docs for production
docs-build:
    cd docs && bun run build

# Serve docs locally (dev mode with hot reload)
docs-dev:
    cd docs && bun run dev

# Preview production build locally
docs-preview:
    cd docs && bun run preview

# Build and serve docs (production build)
docs: docs-build docs-preview

# ─────────────────────────────────────────────────────────────────────────────
# VSCode Extension
# ─────────────────────────────────────────────────────────────────────────────

# Install VSCode extension dependencies
vscode-install:
    cd editors/code && bun install

# Build VSCode extension
vscode-build:
    cd editors/code && bun run build

# Package VSCode extension (.vsix)
vscode-package:
    cd editors/code && bun run package

# Run VSCode extension checks (typecheck, lint, format)
vscode-check:
    cd editors/code && bun run check

# ─────────────────────────────────────────────────────────────────────────────
# Linting & Formatting
# ─────────────────────────────────────────────────────────────────────────────

# Format all code
fmt:
    bazel run @buildifier_prebuilt//:buildifier -- -r .
    cd editors/code && bun run format
    yamlfmt .

# Lint all code
lint:
    bazel run @buildifier_prebuilt//:buildifier -- -mode=check -lint=warn -r .
    actionlint
    cd editors/code && bun run lint

# ─────────────────────────────────────────────────────────────────────────────
# Open in Browser / Editor
# ─────────────────────────────────────────────────────────────────────────────

# Open repository in browser
open:
    open https://github.com/albertocavalcante/bazelle

# Open GitHub Actions
actions:
    open https://github.com/albertocavalcante/bazelle/actions

# Open GitHub Issues
issues:
    open https://github.com/albertocavalcante/bazelle/issues

# Open Pull Requests
prs:
    open https://github.com/albertocavalcante/bazelle/pulls

# Open Releases
releases:
    open https://github.com/albertocavalcante/bazelle/releases

# Open in VSCode
code:
    code .

# Open in Cursor
cursor:
    cursor .

# Open in Zed
zed:
    zed .

# ─────────────────────────────────────────────────────────────────────────────
# CI / Workflows
# ─────────────────────────────────────────────────────────────────────────────

# Trigger nightly build
nightly:
    gh workflow run nightly.yml --field force=true

# Watch CI status
ci-watch:
    gh run watch

# List recent workflow runs
ci-list:
    gh run list --limit 10
