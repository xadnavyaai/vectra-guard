#!/bin/bash
# Vectra Guard Quick Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash

set -e

REPO="xadnavyaai/vectra-guard"
# Allow overriding install dir for testing (e.g., INSTALL_DIR=/tmp/vg-install)
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
BINARY_NAME="vectra-guard"

echo "🛡️  Vectra Guard Installer"
echo "=========================="
echo ""

# Check if already installed
if command -v vectra-guard &> /dev/null; then
    INSTALLED_VERSION=$(vectra-guard --help 2>&1 | head -1 || echo "unknown")
    echo "ℹ️  Vectra Guard is already installed"
    echo "   Location: $(which vectra-guard)"
    echo "   Version: $INSTALLED_VERSION"
    echo ""
    
    # Prompt for upgrade (use /dev/tty if piped through curl | bash)
    if [ ! -t 0 ]; then
        # Check if we have access to /dev/tty (real terminal)
        if [ ! -c /dev/tty ]; then
            echo "⚡ Non-interactive environment - auto-upgrading..."
            UPGRADE=true
        else
            # Read directly from /dev/tty when piped
            read -p "Upgrade to latest version? [Y/n] " -n 1 -r < /dev/tty
            echo
            if [[ $REPLY =~ ^[Nn]$ ]]; then
                echo "❌ Installation cancelled"
                exit 0
            fi
            UPGRADE=true
        fi
    else
        read -p "Upgrade to latest version? [Y/n] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Nn]$ ]]; then
            echo "❌ Installation cancelled"
            exit 0
        fi
        UPGRADE=true
    fi
    echo ""
fi

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
    Darwin)
        OS="darwin"
        ;;
    Linux)
        OS="linux"
        ;;
    *)
        echo "❌ Unsupported OS: $OS"
        echo "   Please install manually from: https://github.com/${REPO}"
        exit 1
        ;;
esac

case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    arm64|aarch64)
        ARCH="arm64"
        ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        echo "   Please install manually from: https://github.com/${REPO}"
        exit 1
        ;;
esac

echo "📋 System: $OS $ARCH"
echo "📦 Install dir: $INSTALL_DIR"
echo ""

# Download pre-built release binary
echo "📦 Downloading Vectra Guard release binary..."
echo ""

BINARY_FILENAME="${BINARY_NAME}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_FILENAME}"
CHECKSUM_URL="https://github.com/${REPO}/releases/latest/download/checksums.txt"

# Ensure a download tool exists
if ! command -v curl &> /dev/null && ! command -v wget &> /dev/null; then
    echo "❌ Neither curl nor wget is installed."
    echo "   Please install one of them and re-run (no sudo recommended)."
    exit 1
fi

# Create temp file for binary
TEMP_FILE=$(mktemp)
trap "rm -f $TEMP_FILE" EXIT

DOWNLOAD_SUCCESS=false
if command -v curl &> /dev/null; then
    if curl -fsSL "$DOWNLOAD_URL" -o "$TEMP_FILE"; then
        DOWNLOAD_SUCCESS=true
    fi
elif command -v wget &> /dev/null; then
    if wget -q "$DOWNLOAD_URL" -O "$TEMP_FILE"; then
        DOWNLOAD_SUCCESS=true
    fi
fi

if [ "$DOWNLOAD_SUCCESS" = false ]; then
    echo "❌ Release download failed."
    echo "   Please check your internet connection or install manually from source:"
    echo "   https://github.com/${REPO}"
    exit 1
fi

# Optional checksum verification (best-effort)
CHECKSUM_FILE=$(mktemp)
trap "rm -f $CHECKSUM_FILE" EXIT
if command -v curl &> /dev/null; then
    curl -fsSL "$CHECKSUM_URL" -o "$CHECKSUM_FILE" || true
elif command -v wget &> /dev/null; then
    wget -q "$CHECKSUM_URL" -O "$CHECKSUM_FILE" || true
fi

if [ -s "$CHECKSUM_FILE" ]; then
    if command -v shasum &> /dev/null; then
        EXPECTED_SHA=$(grep "  ${BINARY_FILENAME}$" "$CHECKSUM_FILE" | awk '{print $1}')
        ACTUAL_SHA=$(shasum -a 256 "$TEMP_FILE" | awk '{print $1}')
        if [ -n "$EXPECTED_SHA" ] && [ "$EXPECTED_SHA" != "$ACTUAL_SHA" ]; then
            echo "❌ Checksum verification failed."
            exit 1
        fi
    elif command -v sha256sum &> /dev/null; then
        EXPECTED_SHA=$(grep "  ${BINARY_FILENAME}$" "$CHECKSUM_FILE" | awk '{print $1}')
        ACTUAL_SHA=$(sha256sum "$TEMP_FILE" | awk '{print $1}')
        if [ -n "$EXPECTED_SHA" ] && [ "$EXPECTED_SHA" != "$ACTUAL_SHA" ]; then
            echo "❌ Checksum verification failed."
            exit 1
        fi
    else
        echo "⚠️  Checksum verification skipped (no shasum/sha256sum)"
    fi
else
    echo "⚠️  Checksums not available. Skipping verification."
fi

# Make executable
chmod +x "$TEMP_FILE"

# Install
echo "📝 Installing to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
if [ ! -w "$INSTALL_DIR" ]; then
    echo "❌ Install directory is not writable: $INSTALL_DIR"
    echo "   Set INSTALL_DIR to a writable path (e.g., $HOME/.local/bin) and re-run."
    exit 1
fi
if [[ "$INSTALL_DIR" == "/usr/local/bin" || "$INSTALL_DIR" == "/usr/bin" || "$INSTALL_DIR" == "/bin" ]]; then
    if [ -z "${VECTRAGUARD_ALLOW_SYSTEM_INSTALL:-}" ]; then
        echo "❌ System-wide installs are disabled by default."
        echo "   Use INSTALL_DIR=\"$HOME/.local/bin\" or set VECTRAGUARD_ALLOW_SYSTEM_INSTALL=1 to override."
        exit 1
    fi
fi
mv "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"

# Verify installation
if [ -x "$INSTALL_DIR/$BINARY_NAME" ]; then
    if command -v vectra-guard &> /dev/null; then
        NEW_VERSION=$(vectra-guard version 2>&1 | head -1 || vectra-guard --help 2>&1 | head -1 || echo "installed")
    else
        NEW_VERSION=$("$INSTALL_DIR/$BINARY_NAME" version 2>&1 | head -1 || "$INSTALL_DIR/$BINARY_NAME" --help 2>&1 | head -1 || echo "installed")
    fi
    echo ""

    # Add install directory to PATH and vg/vectraguard aliases to shell config.
    # Always try each rc file (do not skip based on current process PATH), so that
    # .zshrc gets the block even when the installer runs under bash with PATH already set.
    ADDED_TO=""
    for RC in "$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile"; do
        if [ -f "$RC" ]; then
            # Only skip if our block is already present (comment + vg alias). Do not skip
            # just because .local/bin or PATH appears from another tool.
            if grep -q "# Vectra Guard" "$RC" 2>/dev/null && grep -q "alias vg='vectra-guard'" "$RC" 2>/dev/null; then
                continue
            fi
            echo "" >> "$RC"
            echo "# Vectra Guard: add install dir to PATH" >> "$RC"
            if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
                echo 'export PATH="$PATH:$HOME/.local/bin"' >> "$RC"
            else
                echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$RC"
            fi
            echo "alias vg='vectra-guard'" >> "$RC"
            echo "alias vectraguard='vectra-guard'" >> "$RC"
            ADDED_TO="${ADDED_TO} ${RC}"
        elif [ "$RC" = "$HOME/.zshrc" ] || [ "$RC" = "$HOME/.bashrc" ]; then
            # Create .zshrc or .bashrc if missing so PATH is set in new shells
            echo "# Vectra Guard: add install dir to PATH" >> "$RC"
            if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
                echo 'export PATH="$PATH:$HOME/.local/bin"' >> "$RC"
            else
                echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$RC"
            fi
            echo "alias vg='vectra-guard'" >> "$RC"
            echo "alias vectraguard='vectra-guard'" >> "$RC"
            ADDED_TO="${ADDED_TO} ${RC}"
        fi
    done
    if [ -n "$ADDED_TO" ]; then
        echo "📂 Added $INSTALL_DIR to PATH and aliases (vg, vectraguard) in:$ADDED_TO"
        echo "   In this shell right now, run:"
        echo "     export PATH=\"\$PATH:$INSTALL_DIR\""
        echo "     alias vg='vectra-guard'"
        echo "     alias vectraguard='vectra-guard'"
        echo "   For future shells: if you use zsh, run \"source ~/.zshrc\" in zsh (or open a new terminal)."
        echo "   If you use bash, run \"source ~/.bashrc\" in bash. Do not source ~/.zshrc from bash (zsh-only syntax can break)."
        echo ""
    fi

    if [ "${UPGRADE:-false}" = true ]; then
        echo "✅ Vectra Guard upgraded successfully!"
        echo ""
        if [ -n "${INSTALLED_VERSION:-}" ]; then
            echo "   Old: $INSTALLED_VERSION"
        fi
        echo "   New: $NEW_VERSION"
    else
    echo "✅ Vectra Guard installed successfully!"
        echo "   Version: $NEW_VERSION"
    fi
    
    echo ""
    # Optional: add a lightweight shell hook to track all commands
    if [ -t 0 ]; then
        read -p "Enable Shell Tracker (adds a small hook)? [y/N] " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-shell-tracker.sh | bash
            echo ""
        fi
    fi
    echo "🚀 Quick Start:"
    echo ""
    echo "   1. Validate a script (safe - never executes):"
    echo "      vectra-guard validate my-script.sh"
    echo ""
    echo "   2. Execute commands safely:"
    echo "      vectra-guard exec -- npm install"
    echo ""
    echo "   3. Get security explanations:"
    echo "      vectra-guard explain risky-script.sh"
    echo ""
    echo "   4. Optional - Initialize global config:"
    echo "      vectra-guard init    # creates ~/.config/vectra-guard/config.yaml"
    echo ""
    echo "   5. Optional - Enable Shell Tracker (recommended):"
    echo "      curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install-shell-tracker.sh | bash"
    echo ""
    echo "📚 Full Documentation: https://github.com/${REPO}"
    echo "🗑️  Uninstall: curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/uninstall.sh | bash"
    echo ""
else
    echo ""
    echo "❌ Installation failed - binary not found at: $INSTALL_DIR/$BINARY_NAME"
    exit 1
fi

