#!/bin/sh
set -e

# Preflight installer
# Usage: curl -fsSL https://raw.githubusercontent.com/vertti/preflight/main/install.sh | sh

REPO="vertti/preflight"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *) echo "unsupported" ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64) echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *) echo "unsupported" ;;
    esac
}

# Get latest release tag
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

main() {
    OS=$(detect_os)
    ARCH=$(detect_arch)

    if [ "$OS" = "unsupported" ] || [ "$ARCH" = "unsupported" ]; then
        echo "Error: Unsupported OS or architecture: $(uname -s) $(uname -m)"
        exit 1
    fi

    VERSION=$(get_latest_version)
    if [ -z "$VERSION" ]; then
        echo "Error: Could not determine latest version"
        exit 1
    fi

    BINARY="preflight-${OS}-${ARCH}"
    if [ "$OS" = "windows" ]; then
        BINARY="${BINARY}.exe"
    fi

    BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
    URL="${BASE_URL}/${BINARY}"
    CHECKSUM_URL="${BASE_URL}/checksums.txt"

    echo "Installing preflight ${VERSION} (${OS}/${ARCH})..."

    # Create temp directory for downloads
    TMP_DIR=$(mktemp -d)
    TMP_FILE="${TMP_DIR}/${BINARY}"
    TMP_CHECKSUMS="${TMP_DIR}/checksums.txt"

    # Download checksums and binary
    echo "Downloading checksums..."
    curl -fsSL "$CHECKSUM_URL" -o "$TMP_CHECKSUMS"

    echo "Downloading binary..."
    curl -fsSL "$URL" -o "$TMP_FILE"

    # Verify checksum
    echo "Verifying checksum..."
    cd "$TMP_DIR"
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum -c checksums.txt --ignore-missing
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 -c checksums.txt --ignore-missing
    else
        echo "Warning: Could not verify checksum (sha256sum/shasum not found)"
    fi
    cd - >/dev/null

    chmod +x "$TMP_FILE"

    # Install
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_FILE" "${INSTALL_DIR}/preflight"
    else
        echo "Installing to ${INSTALL_DIR} (requires sudo)..."
        sudo mv "$TMP_FILE" "${INSTALL_DIR}/preflight"
    fi

    # Cleanup
    rm -rf "$TMP_DIR"

    echo "Successfully installed preflight to ${INSTALL_DIR}/preflight"
    echo ""
    echo "Run 'preflight --help' to get started."
}

main
