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
echo ""

# Confirm uninstall
# Use /dev/tty to read from terminal when piped through curl | bash
if [ -t 0 ]; then
    read -p "Remove Vectra Guard? [y/N] " -n 1 -r
    echo
else
    # Non-interactive mode - require explicit confirmation
    echo "⚠️  Running in non-interactive mode"
    echo "   Uninstall cancelled (use interactive terminal for removal)"
    exit 0
fi

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
    echo "  ℹ️  Data directory found: ~/.vectra-guard"
    echo "  ℹ️  Keeping data directory (remove manually if desired: rm -rf ~/.vectra-guard)"
else
    echo "  ℹ️  No data directory found"
fi

# 5. Check for backups
echo ""
echo "Checking for backups..."
FOUND_BACKUPS=false

for backup in ~/.bashrc.vectra-backup ~/.zshrc.vectra-backup ~/.config/fish/config.fish.vectra-backup; do
    if [ -f "$backup" ]; then
        echo "  ℹ️  Backup found: $backup"
        echo "     Restore with: cp $backup ${backup%.vectra-backup}"
        FOUND_BACKUPS=true
    fi
done

if [ "$FOUND_BACKUPS" = false ]; then
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

if [ "$FOUND_BACKUPS" = true ]; then
    echo "ℹ️  Backup files preserved"
    echo "   Restore manually if desired (see paths above)"
    echo ""
fi

echo "To reinstall: curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash"
echo ""

