#!/bin/bash
# Install locally-built bazelle binary
# Usage: bazel run //:install
#
# Environment variables:
#   INSTALL_DIR - Installation directory (default: ~/.local/bin)

set -e

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BAZELLE_PATH="${BUILD_WORKSPACE_DIRECTORY}/bazel-bin/cmd/bazelle/bazelle_/bazelle"
GAZELLE_PATH="${BUILD_WORKSPACE_DIRECTORY}/bazel-bin/cmd/gazelle/gazelle_/gazelle"

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

# Build if needed
if [ ! -f "$BAZELLE_PATH" ] || [ ! -f "$GAZELLE_PATH" ]; then
    info "Building bazelle..."
    (cd "$BUILD_WORKSPACE_DIRECTORY" && bazel build //cmd/bazelle //cmd/gazelle)
fi

# Install
info "Installing to ${INSTALL_DIR}..."
mkdir -p "$INSTALL_DIR"

rm -f "$INSTALL_DIR/bazelle" "$INSTALL_DIR/bazelle-gazelle"
cp "$BAZELLE_PATH" "$INSTALL_DIR/bazelle"
cp "$GAZELLE_PATH" "$INSTALL_DIR/bazelle-gazelle"
chmod +x "$INSTALL_DIR/bazelle" "$INSTALL_DIR/bazelle-gazelle"

success "Installed bazelle and bazelle-gazelle to ${INSTALL_DIR}/"

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
