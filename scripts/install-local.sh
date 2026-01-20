#!/bin/bash
# Install locally-built bazelle binary
# Usage: bazel run //:install
#
# Environment variables:
#   INSTALL_DIR - Installation directory (default: ~/.local/bin)

set -e

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BINARY_PATH="${BUILD_WORKSPACE_DIRECTORY}/bazel-bin/cmd/bazelle/bazelle_/bazelle"

# Colors
if [ -t 1 ]; then
    GREEN='\033[0;32m'
    BLUE='\033[0;34m'
    YELLOW='\033[0;33m'
    NC='\033[0m'
else
    GREEN='' BLUE='' YELLOW='' NC=''
fi

info() { printf "${BLUE}==>${NC} %s\n" "$1"; }
success() { printf "${GREEN}==>${NC} %s\n" "$1"; }
warn() { printf "${YELLOW}warning:${NC} %s\n" "$1"; }

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    info "Building bazelle..."
    (cd "$BUILD_WORKSPACE_DIRECTORY" && bazel build //cmd/bazelle)
fi

# Install
info "Installing to ${INSTALL_DIR}..."
mkdir -p "$INSTALL_DIR"
cp "$BINARY_PATH" "$INSTALL_DIR/bazelle"
chmod +x "$INSTALL_DIR/bazelle"

success "bazelle installed to ${INSTALL_DIR}/bazelle"

# Check PATH
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        echo ""
        warn "$INSTALL_DIR is not in your PATH."
        echo "Add it: export PATH=\"\$PATH:$INSTALL_DIR\""
        ;;
esac

# Show version
echo ""
"$INSTALL_DIR/bazelle" --version 2>/dev/null || "$INSTALL_DIR/bazelle" --help | head -1
