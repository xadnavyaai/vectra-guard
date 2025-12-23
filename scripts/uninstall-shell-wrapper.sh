#!/bin/bash
# Uninstall Vectra Guard Shell Wrapper

set -e

BACKUP_DIR="${BACKUP_DIR:-/usr/local/bin/vectra-guard-backups}"

echo "🔄 Vectra Guard Shell Wrapper Uninstaller"
echo "========================================="
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "❌ Error: This script must be run as root"
    echo "   Run: sudo $0"
    exit 1
fi

# Function to restore a shell
restore_shell() {
    local shell_path="$1"
    local shell_name=$(basename "$shell_path")
    
    if [ ! -f "${shell_path}.real" ]; then
        echo "⏭️  Skipping $shell_name (not wrapped)"
        return
    fi
    
    echo "🔧 Restoring $shell_name..."
    
    # Remove wrapper
    rm -f "$shell_path"
    
    # Restore original
    mv "${shell_path}.real" "$shell_path"
    
    echo "✅ $shell_name restored"
}

# Restore shells
echo "Restoring system shells..."
echo ""

restore_shell "/bin/bash"
restore_shell "/bin/sh"
restore_shell "/bin/zsh"
restore_shell "/bin/dash"

echo ""
echo "✅ Shell wrapper uninstalled successfully!"
echo ""
echo "📝 Backups still available at: $BACKUP_DIR"

