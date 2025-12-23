#!/bin/bash
# Install Vectra Guard Shell Wrapper
# This intercepts shell invocations to enforce security policies

set -e

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BACKUP_DIR="${BACKUP_DIR:-/usr/local/bin/vectra-guard-backups}"

echo "🛡️  Vectra Guard Shell Wrapper Installer"
echo "========================================"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "❌ Error: This script must be run as root"
    echo "   Run: sudo $0"
    exit 1
fi

# Check if vectra-guard is installed
if ! command -v vectra-guard &> /dev/null; then
    echo "❌ Error: vectra-guard not found in PATH"
    echo "   Install vectra-guard first"
    exit 1
fi

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Function to wrap a shell
wrap_shell() {
    local shell_path="$1"
    local shell_name=$(basename "$shell_path")
    
    if [ ! -f "$shell_path" ]; then
        echo "⏭️  Skipping $shell_name (not found)"
        return
    fi
    
    # Check if already wrapped
    if [ -f "${shell_path}.real" ]; then
        echo "⚠️  $shell_name already wrapped, skipping"
        return
    fi
    
    echo "🔧 Wrapping $shell_name..."
    
    # Backup original
    cp "$shell_path" "$BACKUP_DIR/$shell_name.original"
    
    # Rename original
    mv "$shell_path" "${shell_path}.real"
    
    # Create wrapper
    cat > "$shell_path" << 'WRAPPER_EOF'
#!/bin/bash
# Vectra Guard Shell Wrapper
# Intercepts all shell invocations for security enforcement

REAL_SHELL="SHELL_PATH_PLACEHOLDER.real"
VECTRA_GUARD_BIN="$(which vectra-guard 2>/dev/null || echo /usr/local/bin/vectra-guard)"

# Check if we're in a guarded session
if [ -n "$VECTRAGUARD_SESSION_ID" ] && [ -x "$VECTRA_GUARD_BIN" ]; then
    # Build command string
    CMD_STRING="$*"
    
    # Validate command through vectra-guard
    if ! "$VECTRA_GUARD_BIN" validate-inline "$CMD_STRING" 2>/dev/null; then
        # Command is risky - log and potentially block
        "$VECTRA_GUARD_BIN" exec --session "$VECTRAGUARD_SESSION_ID" -- "$REAL_SHELL" "$@"
        exit $?
    fi
fi

# Execute real shell
exec "$REAL_SHELL" "$@"
WRAPPER_EOF
    
    # Replace placeholder
    sed -i "s|SHELL_PATH_PLACEHOLDER|$shell_path|g" "$shell_path"
    
    # Make executable
    chmod +x "$shell_path"
    
    echo "✅ $shell_name wrapped successfully"
}

# Wrap common shells
echo "Wrapping system shells..."
echo ""

wrap_shell "/bin/bash"
wrap_shell "/bin/sh"
wrap_shell "/bin/zsh"
wrap_shell "/bin/dash"

echo ""
echo "✅ Shell wrapper installed successfully!"
echo ""
echo "⚠️  IMPORTANT: To enable protection for a session, set:"
echo "   export VECTRAGUARD_SESSION_ID=\$(vectra-guard session start --agent YOUR_AGENT)"
echo ""
echo "📝 Original shells backed up to: $BACKUP_DIR"
echo ""
echo "🔄 To uninstall, run: $0 --uninstall"

