#!/bin/sh
# Bazelle installer script
# Usage: curl -sSL https://raw.githubusercontent.com/albertocavalcante/bazelle/main/install.sh | sh
#
# Environment variables:
#   BAZELLE_VERSION  - Version to install (default: latest)
#   INSTALL_DIR      - Installation directory (default: ~/.local/bin)

set -e

REPO="albertocavalcante/bazelle"
BINARY_NAME="bazelle"

# Colors (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    NC=''
fi

info() {
    printf "${BLUE}==>${NC} %s\n" "$1"
}

success() {
    printf "${GREEN}==>${NC} %s\n" "$1"
}

warn() {
    printf "${YELLOW}warning:${NC} %s\n" "$1"
}

error() {
    printf "${RED}error:${NC} %s\n" "$1" >&2
    exit 1
}

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       error "Unsupported operating system: $(uname -s)" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             error "Unsupported architecture: $(uname -m)" ;;
    esac
}

# Get latest version from GitHub
get_latest_version() {
    curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | \
        grep '"tag_name":' | \
        sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and verify
download() {
    local url="$1"
    local output="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$url" -O "$output"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# Verify SHA256 checksum
verify_checksum() {
    local file="$1"
    local expected="$2"
    local actual

    if command -v sha256sum >/dev/null 2>&1; then
        actual=$(sha256sum "$file" | cut -d' ' -f1)
    elif command -v shasum >/dev/null 2>&1; then
        actual=$(shasum -a 256 "$file" | cut -d' ' -f1)
    else
        warn "Neither sha256sum nor shasum found. Skipping checksum verification."
        return 0
    fi

    if [ "$actual" != "$expected" ]; then
        error "Checksum verification failed.\nExpected: $expected\nActual:   $actual"
    fi
}

main() {
    local os arch version install_dir artifact_name download_url checksum_url

    os=$(detect_os)
    arch=$(detect_arch)

    info "Detected platform: ${os}-${arch}"

    # Determine version
    if [ -n "$BAZELLE_VERSION" ]; then
        version="$BAZELLE_VERSION"
    else
        info "Fetching latest version..."
        version=$(get_latest_version)
        if [ -z "$version" ]; then
            error "Failed to determine latest version. Set BAZELLE_VERSION manually or check https://github.com/${REPO}/releases"
        fi
    fi

    info "Installing bazelle ${version}"

    # Set install directory
    install_dir="${INSTALL_DIR:-$HOME/.local/bin}"

    # Construct artifact name
    artifact_name="bazelle-${os}-${arch}"
    download_url="https://github.com/${REPO}/releases/download/${version}/${artifact_name}.tar.gz"
    checksum_url="https://github.com/${REPO}/releases/download/${version}/${artifact_name}.tar.gz.sha256"

    # Create temp directory
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    # Download binary
    info "Downloading ${artifact_name}.tar.gz..."
    download "$download_url" "$tmp_dir/${artifact_name}.tar.gz" || \
        error "Failed to download. Check if ${os}-${arch} is supported at https://github.com/${REPO}/releases"

    # Download and verify checksum
    info "Verifying checksum..."
    download "$checksum_url" "$tmp_dir/checksum.sha256" || \
        warn "Could not download checksum file. Skipping verification."

    if [ -f "$tmp_dir/checksum.sha256" ]; then
        expected_checksum=$(cut -d' ' -f1 "$tmp_dir/checksum.sha256")
        verify_checksum "$tmp_dir/${artifact_name}.tar.gz" "$expected_checksum"
        success "Checksum verified"
    fi

    # Extract
    info "Extracting..."
    tar -xzf "$tmp_dir/${artifact_name}.tar.gz" -C "$tmp_dir"

    # Install
    info "Installing to ${install_dir}..."
    mkdir -p "$install_dir"
    mv "$tmp_dir/bazelle" "$install_dir/bazelle"
    chmod +x "$install_dir/bazelle"

    success "bazelle ${version} installed successfully!"

    # Check if install_dir is in PATH
    case ":$PATH:" in
        *":$install_dir:"*) ;;
        *)
            echo ""
            warn "$install_dir is not in your PATH."
            echo "Add it to your shell profile:"
            echo ""
            echo "  export PATH=\"\$PATH:$install_dir\""
            echo ""
            ;;
    esac

    echo ""
    echo "Run 'bazelle --help' to get started."
}

main "$@"
