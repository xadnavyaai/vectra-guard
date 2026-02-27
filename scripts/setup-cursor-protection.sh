#!/bin/bash
# Vectra Guard: Cursor IDE Protection Setup
# Automatically configures Cursor to use vectra-guard for all commands

set -e

WORKSPACE="${WORKSPACE:-$(pwd)}"
AGENT_NAME="${AGENT_NAME:-cursor-ai}"

echo "🛡️  Vectra Guard - Cursor IDE Protection Setup"
echo "=============================================="
echo ""
echo "For end-to-end setup docs, see README.md (IDE integration) and GETTING_STARTED.md."
echo ""
echo "Workspace: $WORKSPACE"
echo "Agent Name: $AGENT_NAME"
echo ""

# Check if vectra-guard is installed
if ! command -v vectra-guard &> /dev/null; then
    echo "❌ Error: vectra-guard not found in PATH"
    echo "   Please install vectra-guard first"
    exit 1
fi

# Check if running in correct directory
if [ ! -f "main.go" ] && [ ! -f "vectra-guard" ]; then
    echo "⚠️  Warning: Not in vectra-guard directory"
    echo "   Some features may not work correctly"
fi

echo "Step 1/6: Installing shell wrapper..."
echo "--------------------------------------"
if [ -f "scripts/install-shell-wrapper.sh" ]; then
    if [ "$EUID" -eq 0 ]; then
        ./scripts/install-shell-wrapper.sh
    else
        echo "ℹ️  Shell wrapper installation requires root privileges"
        echo "   Run: sudo ./scripts/install-shell-wrapper.sh"
        echo "   Skipping for now (you can install later)"
    fi
else
    echo "⚠️  Shell wrapper script not found, skipping"
fi
echo ""

echo "Step 2/6: Creating workspace configuration..."
echo "--------------------------------------"
mkdir -p "$WORKSPACE/.vscode" "$WORKSPACE/.cursor" "$WORKSPACE/.vectra-guard"

# VSCode/Cursor settings
cat > "$WORKSPACE/.vscode/settings.json" << 'EOF'
{
  "terminal.integrated.env.osx": {
    "VECTRAGUARD_SESSION_ID": "${env:VECTRAGUARD_SESSION_ID}"
  },
  "terminal.integrated.env.linux": {
    "VECTRAGUARD_SESSION_ID": "${env:VECTRAGUARD_SESSION_ID}"
  },
  "terminal.integrated.env.windows": {
    "VECTRAGUARD_SESSION_ID": "${env:VECTRAGUARD_SESSION_ID}"
  },
  "terminal.integrated.defaultProfile.osx": "bash",
  "terminal.integrated.profiles.osx": {
    "bash": {
      "path": "/bin/bash",
      "args": ["-l"],
      "icon": "terminal-bash"
    }
  }
}
EOF

echo "✅ Created .vscode/settings.json"

# VSCode/Cursor tasks
cat > "$WORKSPACE/.vscode/tasks.json" << 'EOF'
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "🛡️ Protected: Install Dependencies",
      "type": "shell",
      "command": "vectra-guard",
      "args": ["exec", "--session", "${env:VECTRAGUARD_SESSION_ID}", "--", "npm", "install"],
      "group": "build",
      "presentation": {
        "echo": true,
        "reveal": "always",
        "panel": "shared",
        "showReuseMessage": false
      }
    },
    {
      "label": "🛡️ Protected: Run Tests",
      "type": "shell",
      "command": "vectra-guard",
      "args": ["exec", "--session", "${env:VECTRAGUARD_SESSION_ID}", "--", "npm", "test"],
      "group": "test"
    },
    {
      "label": "🛡️ Protected: Build",
      "type": "shell",
      "command": "vectra-guard",
      "args": ["exec", "--session", "${env:VECTRAGUARD_SESSION_ID}", "--", "npm", "run", "build"],
      "group": {
        "kind": "build",
        "isDefault": true
      }
    },
    {
      "label": "🛡️ Protected: Deploy (Interactive)",
      "type": "shell",
      "command": "vectra-guard",
      "args": ["exec", "--interactive", "--session", "${env:VECTRAGUARD_SESSION_ID}", "--", "./scripts/deploy.sh"],
      "group": "none"
    },
    {
      "label": "🛡️ Start Session",
      "type": "shell",
      "command": "source .vectra-guard/init.sh",
      "group": "none"
    },
    {
      "label": "🛡️ View Session",
      "type": "shell",
      "command": "vectra-guard session show ${env:VECTRAGUARD_SESSION_ID}",
      "group": "none"
    }
  ]
}
EOF

echo "✅ Created .vscode/tasks.json"
echo ""

echo "Step 3/6: Creating session initialization..."
echo "--------------------------------------"
cat > "$WORKSPACE/.vectra-guard/init.sh" << 'INITEOF'
#!/bin/bash
# Vectra Guard Session Initializer for Cursor IDE

if [ -z "$VECTRAGUARD_SESSION_ID" ]; then
    # Check if we already have a session file
    if [ -f ~/.vectra-guard-session-id ]; then
        SESSION=$(cat ~/.vectra-guard-session-id)
        # Verify session is still valid
        if vectra-guard session show "$SESSION" &>/dev/null; then
            export VECTRAGUARD_SESSION_ID=$SESSION
            echo "🛡️  Vectra Guard session resumed: $SESSION"
        else
            # Session expired, start new one
            SESSION=$(vectra-guard session start --agent "cursor-ai" --workspace "$(pwd)")
            export VECTRAGUARD_SESSION_ID=$SESSION
            echo $SESSION > ~/.vectra-guard-session-id
            echo "🛡️  Vectra Guard session started: $SESSION"
        fi
    else
        # No existing session, start new one
        SESSION=$(vectra-guard session start --agent "cursor-ai" --workspace "$(pwd)")
        export VECTRAGUARD_SESSION_ID=$SESSION
        echo $SESSION > ~/.vectra-guard-session-id
        echo "🛡️  Vectra Guard session started: $SESSION"
    fi
else
    echo "🛡️  Vectra Guard session active: $VECTRAGUARD_SESSION_ID"
fi
INITEOF

chmod +x "$WORKSPACE/.vectra-guard/init.sh"
echo "✅ Created .vectra-guard/init.sh"
echo ""

echo "Step 4/6: Setting up git hooks (disabled by default)..."
echo "------------------------------------------------------"
if [ -d "$WORKSPACE/.git" ]; then
    if [ "${VG_ENABLE_PRECOMMIT:-0}" = "1" ]; then
        mkdir -p "$WORKSPACE/.git/hooks"
        
        cat > "$WORKSPACE/.git/hooks/pre-commit" << 'HOOKEOF'
#!/bin/bash
# Vectra Guard Pre-commit Hook
# Validates all scripts before commit

echo "🛡️  Vectra Guard: Validating scripts..."

SCRIPTS=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.(sh|bash)$')

if [ -n "$SCRIPTS" ]; then
    for script in $SCRIPTS; do
        if [ -f "$script" ]; then
            echo "   Checking: $script"
            if ! vectra-guard validate "$script"; then
                echo ""
                echo "❌ Security issues found in: $script"
                echo ""
                echo "To see details, run:"
                echo "  vectra-guard explain $script"
                echo ""
                echo "Fix issues or use 'git commit --no-verify' to bypass (not recommended)"
                exit 1
            fi
        fi
    done
    echo "✅ All scripts validated"
fi

# Check Dockerfile if present
if git diff --cached --name-only | grep -q "Dockerfile"; then
    echo "   Checking: Dockerfile"
    if ! vectra-guard validate Dockerfile 2>/dev/null; then
        echo "⚠️  Warning: Dockerfile may contain risky patterns"
    fi
fi

exit 0
HOOKEOF
        
        chmod +x "$WORKSPACE/.git/hooks/pre-commit"
        echo "✅ Created git pre-commit hook (VG_ENABLE_PRECOMMIT=1)"
    else
        echo "ℹ️  Skipping pre-commit hook (set VG_ENABLE_PRECOMMIT=1 to enable)"
    fi
else
    echo "⚠️  No .git directory found, skipping git hooks"
fi
echo ""

echo "Step 5/6: Initializing vectra-guard configuration..."
echo "--------------------------------------"
if [ ! -f ~/.config/vectra-guard/config.yaml ] && [ ! -f "$WORKSPACE/vectra-guard.yaml" ]; then
    vectra-guard init
    echo "✅ Created ~/.config/vectra-guard/config.yaml"
else
    echo "ℹ️  Config already exists, skipping"
fi
echo ""

echo "Step 6/6: Creating helper scripts..."
echo "--------------------------------------"

# Create a quick session starter
cat > "$WORKSPACE/.vectra-guard/start-session.sh" << 'STARTEOF'
#!/bin/bash
# Quick session starter

SESSION=$(vectra-guard session start --agent "cursor-ai" --workspace "$(pwd)")
export VECTRAGUARD_SESSION_ID=$SESSION
echo $SESSION > ~/.vectra-guard-session-id
echo "🛡️  Session started: $SESSION"
echo "   To use in current shell, run:"
echo "   export VECTRAGUARD_SESSION_ID=$SESSION"
STARTEOF

chmod +x "$WORKSPACE/.vectra-guard/start-session.sh"
echo "✅ Created start-session.sh"

# Create session viewer
cat > "$WORKSPACE/.vectra-guard/view-session.sh" << 'VIEWEOF'
#!/bin/bash
# View current session

if [ -z "$VECTRAGUARD_SESSION_ID" ]; then
    if [ -f ~/.vectra-guard-session-id ]; then
        SESSION=$(cat ~/.vectra-guard-session-id)
    else
        echo "❌ No active session"
        echo "   Run: .vectra-guard/start-session.sh"
        exit 1
    fi
else
    SESSION=$VECTRAGUARD_SESSION_ID
fi

vectra-guard session show "$SESSION"
VIEWEOF

chmod +x "$WORKSPACE/.vectra-guard/view-session.sh"
echo "✅ Created view-session.sh"

echo ""
echo "=========================================="
echo "✅ Setup Complete!"
echo "=========================================="
echo ""
echo "Next steps:"
echo ""
echo "1. Restart Cursor IDE"
echo "2. Open a new terminal in Cursor"
echo "3. Run: source .vectra-guard/init.sh"
echo "4. Verify: echo \$VECTRAGUARD_SESSION_ID"
echo ""
echo "Quick commands:"
echo "  .vectra-guard/start-session.sh    # Start new session"
echo "  .vectra-guard/view-session.sh     # View current session"
echo "  vectra-guard session list         # List all sessions"
echo ""
echo "Tasks available in Cursor:"
echo "  - 🛡️ Protected: Install Dependencies"
echo "  - 🛡️ Protected: Run Tests"
echo "  - 🛡️ Protected: Build"
echo "  - 🛡️ Protected: Deploy (Interactive)"
echo "  - 🛡️ View Session"
echo ""
echo "All commands executed in Cursor terminal are now protected! 🛡️"
echo ""

# Optionally start a session now
read -p "Start a vectra-guard session now? [Y/n] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
    ./.vectra-guard/start-session.sh
fi

