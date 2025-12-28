# Installs the latest release of witr from GitHub
# Repo: https://github.com/pranshuparmar/witr

#!/usr/bin/env bash
set -euo pipefail

REPO="pranshuparmar/witr"
INSTALL_PATH="/usr/local/bin/witr"

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        ARCH=amd64
        ;;
    aarch64|arm64)
        ARCH=arm64
        ;;
    *)
        echo "Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

# Get latest release tag from GitHub API
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d '"' -f4)
if [[ -z "$LATEST" ]]; then
    echo "Could not determine latest release tag." >&2
    exit 1
fi

URL="https://github.com/$REPO/releases/download/$LATEST/witr-linux-$ARCH"
TMP=$(mktemp)
MANURL="https://github.com/$REPO/releases/download/$LATEST/witr.1"
MAN_TMP=$(mktemp)

# Download binary
curl -fL "$URL" -o "$TMP"
curl -fL "$MANURL" -o "$MAN_TMP"

# Install
sudo install -m 755 "$TMP" "$INSTALL_PATH"
rm -f "$TMP"
sudo install -D -m 644 "$MAN_TMP" /usr/local/share/man/man1/witr.1
rm -f "$MAN_TMP"

echo "witr installed successfully to $INSTALL_PATH (version: $LATEST, arch: $ARCH)"
echo "Man page installed to /usr/local/share/man/man1/witr.1"
