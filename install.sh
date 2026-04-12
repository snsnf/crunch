#!/bin/sh
set -e

REPO="snsnf/crunch"
INSTALL_DIR="/usr/local/bin"

echo "Installing Crunch CLI..."

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
    darwin) PLATFORM="macos" ;;
    linux)  PLATFORM="linux" ;;
    *)      echo "Error: Unsupported OS: $OS"; exit 1 ;;
esac

case "$ARCH" in
    arm64|aarch64) ARCH="arm64" ;;
    x86_64|amd64)  ARCH="amd64" ;;
    *)             echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Linux only has amd64
if [ "$PLATFORM" = "linux" ] && [ "$ARCH" = "arm64" ]; then
    echo "Error: Linux arm64 builds are not available yet."
    exit 1
fi

# Get latest release tag
TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d '"' -f 4)
if [ -z "$TAG" ]; then
    echo "Error: Could not fetch latest release."
    exit 1
fi
echo "Latest version: $TAG"

# Set asset name and download
if [ "$PLATFORM" = "macos" ]; then
    ASSET="crunch-cli-macos-${ARCH}.zip"
else
    ASSET="crunch-cli-linux-${ARCH}.tar.gz"
fi

URL="https://github.com/$REPO/releases/download/$TAG/$ASSET"
CHECKSUM_URL="https://github.com/$REPO/releases/download/$TAG/checksums.txt"
TMP=$(mktemp -d)

echo "Downloading..."
curl -fSL -o "$TMP/$ASSET" "$URL"

# Verify checksum if available
if curl -fsSL -o "$TMP/checksums.txt" "$CHECKSUM_URL" 2>/dev/null; then
    EXPECTED=$(grep "$ASSET" "$TMP/checksums.txt" | cut -d ' ' -f 1)
    if [ -n "$EXPECTED" ]; then
        if command -v sha256sum >/dev/null 2>&1; then
            ACTUAL=$(sha256sum "$TMP/$ASSET" | cut -d ' ' -f 1)
        else
            ACTUAL=$(shasum -a 256 "$TMP/$ASSET" | cut -d ' ' -f 1)
        fi
        if [ "$ACTUAL" != "$EXPECTED" ]; then
            echo "Error: Checksum verification failed!"
            echo "  Expected: $EXPECTED"
            echo "  Got:      $ACTUAL"
            rm -rf "$TMP"
            exit 1
        fi
        echo "Checksum verified."
    fi
fi

# Extract
if echo "$ASSET" | grep -q ".zip$"; then
    unzip -o -q "$TMP/$ASSET" -d "$TMP"
else
    tar xzf "$TMP/$ASSET" -C "$TMP"
fi

# Install
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP/crunch" "$INSTALL_DIR/crunch"
    chmod +x "$INSTALL_DIR/crunch"
else
    echo "Need sudo to install to $INSTALL_DIR"
    sudo mv "$TMP/crunch" "$INSTALL_DIR/crunch"
    sudo chmod +x "$INSTALL_DIR/crunch"
fi

# Cleanup
rm -rf "$TMP"

# macOS: remove quarantine flag
if [ "$PLATFORM" = "macos" ]; then
    xattr -d com.apple.quarantine "$INSTALL_DIR/crunch" 2>/dev/null || true
fi

echo ""
echo "Crunch $TAG installed!"
echo "Run: crunch --help"
