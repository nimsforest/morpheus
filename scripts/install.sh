#!/bin/bash
# Morpheus Universal Installer
# Works on: Linux, macOS, Termux (Android)
# Auto-detects: OS, architecture, and environment

set -e

echo "🌲 Morpheus Universal Installer"
echo "================================"
echo ""

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux*)  OS="linux" ;;
    darwin*) OS="darwin" ;;
    *)
        echo "❌ Unsupported OS: $OS"
        echo "   Supported: Linux, macOS"
        exit 1
        ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    armv7* | armv8l) ARCH="arm" ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        echo "   Supported: x86_64 (amd64), aarch64 (arm64), armv7/armv8l (arm)"
        exit 1
        ;;
esac

# Detect if we're in Termux
IS_TERMUX=0
if [[ -n "$PREFIX" ]] && [[ "$PREFIX" == *"com.termux"* ]]; then
    IS_TERMUX=1
fi

echo "Detected system:"
echo "  OS: $OS"
echo "  Architecture: $ARCH"
if [[ $IS_TERMUX -eq 1 ]]; then
    echo "  Environment: Termux"
else
    echo "  Environment: Standard Linux"
fi
echo ""

# Get latest release
echo "📡 Fetching latest release..."
LATEST_URL="https://api.github.com/repos/nimsforest/morpheus/releases/latest"
RELEASE_INFO=$(curl -s "$LATEST_URL")

if [[ $? -ne 0 ]]; then
    echo "❌ Failed to fetch release information"
    echo "   Check your internet connection"
    exit 1
fi

VERSION=$(echo "$RELEASE_INFO" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [[ -z "$VERSION" ]]; then
    echo "❌ Could not parse version from GitHub API"
    exit 1
fi

echo "  Latest version: $VERSION"
echo ""

# Download binary
BINARY_NAME="morpheus-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/nimsforest/morpheus/releases/download/${VERSION}/${BINARY_NAME}"

echo "📥 Downloading $BINARY_NAME..."
TMP_DIR=$(mktemp -d)
TMP_FILE="$TMP_DIR/morpheus"

if ! curl -L -f -o "$TMP_FILE" "$DOWNLOAD_URL"; then
    echo "❌ Download failed"
    echo "   URL: $DOWNLOAD_URL"
    echo ""
    echo "Check if binary exists for your platform at:"
    echo "   https://github.com/nimsforest/morpheus/releases/latest"
    rm -rf "$TMP_DIR"
    exit 1
fi

chmod +x "$TMP_FILE"

# Verify binary
echo "🔍 Verifying binary..."
if ! "$TMP_FILE" version &>/dev/null; then
    echo "❌ Binary verification failed"
    rm -rf "$TMP_DIR"
    exit 1
fi

INSTALLED_VERSION=$("$TMP_FILE" version)
echo "  $INSTALLED_VERSION"
echo ""

# Install to appropriate location
echo "📦 Installing morpheus..."

if [[ $IS_TERMUX -eq 1 ]]; then
    # Termux installation
    INSTALL_DIR="$PREFIX/bin"
    cp "$TMP_FILE" "$INSTALL_DIR/morpheus"
    chmod +x "$INSTALL_DIR/morpheus"
    echo "  ✓ Installed to $INSTALL_DIR/morpheus"
else
    # Standard Linux/macOS installation
    if [[ -w "/usr/local/bin" ]]; then
        # Can write without sudo
        INSTALL_DIR="/usr/local/bin"
        cp "$TMP_FILE" "$INSTALL_DIR/morpheus"
        chmod +x "$INSTALL_DIR/morpheus"
        echo "  ✓ Installed to $INSTALL_DIR/morpheus"
    elif command -v sudo &>/dev/null; then
        # Need sudo
        INSTALL_DIR="/usr/local/bin"
        sudo cp "$TMP_FILE" "$INSTALL_DIR/morpheus"
        sudo chmod +x "$INSTALL_DIR/morpheus"
        echo "  ✓ Installed to $INSTALL_DIR/morpheus (with sudo)"
    else
        # No sudo, install to user bin
        INSTALL_DIR="$HOME/.local/bin"
        mkdir -p "$INSTALL_DIR"
        cp "$TMP_FILE" "$INSTALL_DIR/morpheus"
        chmod +x "$INSTALL_DIR/morpheus"
        echo "  ✓ Installed to $INSTALL_DIR/morpheus"
        echo ""
        echo "  ⚠️  Make sure $INSTALL_DIR is in your PATH:"
        echo "     export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
fi

# Cleanup
rm -rf "$TMP_DIR"

echo ""
echo "================================"
echo "✅ Installation Complete!"
echo "================================"
echo ""
echo "Verify installation:"
echo "  morpheus version"
echo ""
echo "Get started:"
echo "  morpheus help"
echo ""
echo "Next steps:"
echo "  1. Get Hetzner API token: https://console.hetzner.cloud/"
echo "  2. Set up config: ~/.morpheus/config.yaml"
echo "  3. Create your first forest: morpheus plant cloud wood"
echo ""
