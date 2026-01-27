# Bazelle development tasks

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
