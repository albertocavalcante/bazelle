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
