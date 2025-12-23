#!/bin/bash
# Vectra Guard Uninstaller
# Usage: curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/uninstall.sh | bash

set -e

echo "🗑️  Vectra Guard Uninstaller"
echo "============================"
echo ""

BINARY_NAME="vectra-guard"
INSTALL_DIR="/usr/local/bin"

# Check if installed
if ! command -v vectra-guard &> /dev/null; then
    echo "❌ Vectra Guard is not installed"
    exit 0
fi

echo "Found Vectra Guard at: $(which vectra-guard)"
CURRENT_VERSION=$(vectra-guard --help 2>&1 | head -1 || echo "unknown")
echo "Version: $CURRENT_VERSION"
echo ""

# Confirm uninstall
read -p "Remove Vectra Guard? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "❌ Uninstall cancelled"
    exit 0
fi

echo ""
echo "📋 Uninstalling components..."
echo ""

# 1. Remove binary
echo "1/4: Removing binary..."
if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    if [ -w "$INSTALL_DIR" ]; then
        rm -f "$INSTALL_DIR/$BINARY_NAME"
    else
        sudo rm -f "$INSTALL_DIR/$BINARY_NAME"
    fi
    echo "  ✅ Binary removed"
else
    echo "  ℹ️  Binary not found"
fi

# 2. Remove shell integration
echo ""
echo "2/4: Removing shell integration..."
REMOVED_INTEGRATION=false

for shell_rc in ~/.bashrc ~/.zshrc ~/.config/fish/config.fish; do
    if [ -f "$shell_rc" ]; then
        # Check if integration exists
        if grep -q "Vectra Guard Integration" "$shell_rc" 2>/dev/null; then
            # Remove Vectra Guard section
            if [[ "$OSTYPE" == "darwin"* ]]; then
                # macOS (BSD sed)
                sed -i '' '/# ====.*Vectra Guard Integration/,/# End Vectra Guard Integration/d' "$shell_rc"
            else
                # Linux (GNU sed)
                sed -i '/# ====.*Vectra Guard Integration/,/# End Vectra Guard Integration/d' "$shell_rc"
            fi
            echo "  ✅ Removed from $(basename $shell_rc)"
            REMOVED_INTEGRATION=true
        fi
    fi
done

if [ "$REMOVED_INTEGRATION" = false ]; then
    echo "  ℹ️  No shell integration found"
fi

# 3. Remove session file
echo ""
echo "3/4: Cleaning up session data..."
if [ -f ~/.vectra-guard-session ]; then
    rm -f ~/.vectra-guard-session
    echo "  ✅ Session file removed"
else
    echo "  ℹ️  No session file found"
fi

# 4. Ask about data directory
echo ""
echo "4/4: Configuration and session data..."
if [ -d ~/.vectra-guard ]; then
    echo ""
    echo "Found data directory: ~/.vectra-guard"
    echo "Contains:"
    du -sh ~/.vectra-guard 2>/dev/null || echo "  (unable to calculate size)"
    echo ""
    read -p "Remove all configuration and session data? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        rm -rf ~/.vectra-guard
        echo "  ✅ Data directory removed"
    else
        echo "  ℹ️  Keeping data directory (can be removed manually later)"
    fi
else
    echo "  ℹ️  No data directory found"
fi

# 5. Restore backups if they exist
echo ""
echo "Checking for backups..."
RESTORED_BACKUP=false

for backup in ~/.bashrc.vectra-backup ~/.zshrc.vectra-backup ~/.config/fish/config.fish.vectra-backup; do
    if [ -f "$backup" ]; then
        original="${backup%.vectra-backup}"
        echo ""
        read -p "Restore backup: $(basename $original)? [Y/n] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            cp "$backup" "$original"
            echo "  ✅ Restored $(basename $original)"
            RESTORED_BACKUP=true
        fi
    fi
done

if [ "$RESTORED_BACKUP" = false ]; then
    echo "  ℹ️  No backups found"
fi

echo ""
echo "=========================================="
echo "✅ Vectra Guard Uninstalled"
echo "=========================================="
echo ""

if [ "$REMOVED_INTEGRATION" = true ]; then
    echo "⚠️  Shell integration was removed."
    echo "   Please restart your terminal or run:"
    echo "   source ~/.bashrc  # or ~/.zshrc"
    echo ""
fi

if [ -d ~/.vectra-guard ]; then
    echo "ℹ️  Data directory preserved at: ~/.vectra-guard"
    echo "   Remove manually if desired: rm -rf ~/.vectra-guard"
    echo ""
fi

echo "To reinstall: curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash"
echo ""

