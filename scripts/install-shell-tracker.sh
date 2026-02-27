#!/bin/bash
# Vectra Guard: Shell Tracker Installer
# Integrates with bash, zsh, and fish to track ALL commands

set -uo pipefail

REQUESTED_SHELLS="${VG_SHELLS:-}"
while [ $# -gt 0 ]; do
    case "$1" in
        --shell|--shells)
            if [ $# -lt 2 ]; then
                echo "❌ Missing value for $1 (use space-separated shells like 'bash zsh')" >&2
                exit 1
            fi
            shift
            REQUESTED_SHELLS="${1:-}"
            ;;
        --shells=*)
            REQUESTED_SHELLS="${1#*=}"
            ;;
        -h|--help)
            echo "Usage: install-shell-tracker.sh [--shells \"bash zsh\"]" >&2
            echo "       (or set VG_SHELLS=\"bash zsh\" for non-interactive selection)" >&2
            exit 0
            ;;
        *)
            echo "❌ Unknown option: $1" >&2
            exit 1
            ;;
    esac
    shift
done

echo "🛡️  Vectra Guard - Shell Tracker Installer"
echo "========================================="
echo ""
echo "For a full walkthrough, see README.md and GETTING_STARTED.md."
echo ""

# Check if vectra-guard is installed
if ! command -v vectra-guard &> /dev/null; then
    echo "❌ Error: vectra-guard not found in PATH"
    echo "   Please install vectra-guard first"
    exit 1
fi

# Detect available shells
SHELLS=()
# Check for bash (common on Linux)
if command -v bash &> /dev/null; then
    SHELLS+=("bash")
fi
# Check for zsh (common on macOS)
if command -v zsh &> /dev/null; then
    SHELLS+=("zsh")
fi
# Check for fish
if command -v fish &> /dev/null && [ -d ~/.config/fish ]; then
    SHELLS+=("fish")
fi

if [ ${#SHELLS[@]} -eq 0 ]; then
    echo "❌ No supported shells found (bash, zsh, or fish)"
    echo "   Please install bash or zsh first"
    exit 1
fi

echo "Detected shells: ${SHELLS[*]}"
echo ""

# Backup existing configs
SELECTED_SHELLS=("${SHELLS[@]}")

# Optional interactive or flag/env-based shell selection
if [ -n "$REQUESTED_SHELLS" ]; then
    SELECTED_SHELLS=()
    for shell in $REQUESTED_SHELLS; do
        for detected in "${SHELLS[@]}"; do
            if [ "$shell" = "$detected" ]; then
                SELECTED_SHELLS+=("$shell")
            fi
        done
    done
    if [ ${#SELECTED_SHELLS[@]} -eq 0 ]; then
        echo "❌ None of the requested shells are available. Detected: ${SHELLS[*]}" >&2
        exit 1
    fi
    echo "  ℹ️  Shell selection provided (--shells/VG_SHELLS): ${SELECTED_SHELLS[*]}"
elif [ -c /dev/tty ] && [ -t 0 ]; then
    echo "Select shells to enable (space-separated). Press Enter for all detected."
    echo "Detected: ${SHELLS[*]}"
    read -p "> " -r < /dev/tty || REPLY=""
    if [ -n "${REPLY:-}" ]; then
        SELECTED_SHELLS=()
        for shell in $REPLY; do
            for detected in "${SHELLS[@]}"; do
                if [ "$shell" = "$detected" ]; then
                    SELECTED_SHELLS+=("$shell")
                fi
            done
        done
        if [ ${#SELECTED_SHELLS[@]} -eq 0 ]; then
            echo "❌ No valid shells selected, using all detected shells."
            SELECTED_SHELLS=("${SHELLS[@]}")
        fi
    else
        echo "  ℹ️  No selection entered, enabling all detected shells."
    fi
else
    echo "  ℹ️  Non-interactive mode: installing for all detected shells"
fi
echo ""

# Backup existing configs
echo "Step 1/5: Backing up existing configurations..."
for shell in "${SELECTED_SHELLS[@]}"; do
    case $shell in
        bash)
            if [ -f ~/.bashrc ]; then
                cp ~/.bashrc ~/.bashrc.vectra-backup
                echo "  ✅ Backed up ~/.bashrc"
            else
                touch ~/.bashrc
                echo "  ✅ Created ~/.bashrc"
            fi
            ;;
        zsh)
            if [ -f ~/.zshrc ]; then
                cp ~/.zshrc ~/.zshrc.vectra-backup
                echo "  ✅ Backed up ~/.zshrc"
            else
                touch ~/.zshrc
                echo "  ✅ Created ~/.zshrc"
            fi
            ;;
        fish)
            mkdir -p ~/.config/fish
            if [ -f ~/.config/fish/config.fish ]; then
                cp ~/.config/fish/config.fish ~/.config/fish/config.fish.vectra-backup
                echo "  ✅ Backed up ~/.config/fish/config.fish"
            else
                touch ~/.config/fish/config.fish
                echo "  ✅ Created ~/.config/fish/config.fish"
            fi
            ;;
    esac
done
echo ""

# Install shell integrations
echo "Step 2/5: Installing shell integrations..."

# Function to add bash integration
install_bash() {
    mkdir -p ~/.vectra-guard
    cat > ~/.vectra-guard/tracker.bash << 'BASH_TRACKER'
# Vectra Guard Shell Tracker (Auto-generated)
if [ -n "$BASH_VERSION" ] && command -v vectra-guard &> /dev/null; then
  _vectra_guard_init() {
    if [ -z "$VECTRAGUARD_SESSION_ID" ]; then
      SESSION=$(vectra-guard session start --agent "${USER}-bash" --workspace "$PWD" 2>/dev/null | tail -n 1 || echo "")
      if [ -n "$SESSION" ]; then
        export VECTRAGUARD_SESSION_ID=$SESSION
        echo $SESSION > ~/.vectra-guard-session
      fi
    fi
  }

  _vectra_guard_preexec() {
    local cmd="$BASH_COMMAND"
    if [[ "$cmd" =~ ^vectra-guard ]] || [[ "$cmd" =~ ^vg[[:space:]] ]] || [[ "$cmd" =~ _vectra_guard ]] || [[ "$cmd" =~ VECTRAGUARD ]]; then
      return 0
    fi
    if [[ -z "$cmd" ]] || [[ "$cmd" =~ ^[[:space:]]*# ]] || [[ "$cmd" =~ ^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*= ]]; then
      return 0
    fi
    VECTRA_LAST_CMD="$cmd"
  }

  _vectra_guard_precmd() {
    local exit_code=$?
    if [ -n "$VECTRA_LAST_CMD" ] && [ -n "$VECTRAGUARD_SESSION_ID" ]; then
      vectra-guard session record --session "$VECTRAGUARD_SESSION_ID" --command "$VECTRA_LAST_CMD" --exit-code "$exit_code" &>/dev/null
    fi
    unset VECTRA_LAST_CMD
  }

  shopt -s extdebug
  trap '_vectra_guard_preexec' DEBUG
  PROMPT_COMMAND="_vectra_guard_precmd${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
  _vectra_guard_init
fi
BASH_TRACKER

    if ! grep -q "vectra-guard/tracker.bash" ~/.bashrc 2>/dev/null; then
        echo "source ~/.vectra-guard/tracker.bash" >> ~/.bashrc
    fi
    echo "  ✅ Bash tracker installed"
}

# Function to add zsh integration
install_zsh() {
    mkdir -p ~/.vectra-guard
    cat > ~/.vectra-guard/tracker.zsh << 'ZSH_TRACKER'
# Vectra Guard Shell Tracker (Auto-generated)
if [ -n "$ZSH_VERSION" ] && command -v vectra-guard &> /dev/null; then
  _vectra_guard_init() {
    if [[ -z "$VECTRAGUARD_SESSION_ID" ]]; then
      SESSION=$(vectra-guard session start --agent "${USER}-zsh" --workspace "$PWD" 2>/dev/null | tail -n 1 || echo "")
      if [[ -n "$SESSION" ]]; then
        export VECTRAGUARD_SESSION_ID=$SESSION
        echo $SESSION > ~/.vectra-guard-session 2>/dev/null || true
      fi
    fi
  }

  _vectra_guard_preexec() {
    local cmd="$1"
    if [[ "$cmd" =~ ^vectra-guard ]] || [[ "$cmd" =~ ^vg[[:space:]] ]] || [[ "$cmd" =~ _vectra_guard ]] || [[ "$cmd" =~ VECTRAGUARD ]]; then
      return 0
    fi
    if [[ -z "$cmd" ]] || [[ "$cmd" =~ ^[[:space:]]*# ]] || [[ "$cmd" =~ ^[[:space:]]*[A-Za-z_][A-Za-z0-9_]*= ]]; then
      return 0
    fi
    VECTRA_LAST_CMD="$cmd"
  }

  _vectra_guard_precmd() {
    local exit_code=$?
    if [[ -n "$VECTRA_LAST_CMD" && -n "$VECTRAGUARD_SESSION_ID" ]]; then
      vectra-guard session record --session "$VECTRAGUARD_SESSION_ID" --command "$VECTRA_LAST_CMD" --exit-code "$exit_code" &>/dev/null 2>&1 || true
    fi
    unset VECTRA_LAST_CMD
  }

  autoload -Uz add-zsh-hook 2>/dev/null || true
  add-zsh-hook preexec _vectra_guard_preexec 2>/dev/null || true
  add-zsh-hook precmd _vectra_guard_precmd 2>/dev/null || true
  _vectra_guard_init
fi
ZSH_TRACKER

    if ! grep -q "vectra-guard/tracker.zsh" ~/.zshrc 2>/dev/null; then
        echo "source ~/.vectra-guard/tracker.zsh" >> ~/.zshrc
    fi
    echo "  ✅ Zsh tracker installed"
}

# Function to add fish integration
install_fish() {
    mkdir -p ~/.config/fish
    cat >> ~/.config/fish/config.fish << 'FISH_EOF'

# ============================================================================
# Vectra Guard Integration (Auto-generated)
# ============================================================================

if command -v vectra-guard > /dev/null
    # Initialize session
    function _vectra_guard_init
        if not set -q VECTRAGUARD_SESSION_ID
            if test -f ~/.vectra-guard-session
                set -gx VECTRAGUARD_SESSION_ID (tail -n 1 ~/.vectra-guard-session 2>/dev/null || echo "")
                # Verify session
                if test -n "$VECTRAGUARD_SESSION_ID"; and not vectra-guard session show $VECTRAGUARD_SESSION_ID &> /dev/null
                    set -e VECTRAGUARD_SESSION_ID
                end
            end
            
            if not set -q VECTRAGUARD_SESSION_ID
                set -gx VECTRAGUARD_SESSION_ID (vectra-guard session start --agent "$USER-fish" --workspace $HOME 2>/dev/null | tail -n 1 || echo "")
                if test -n "$VECTRAGUARD_SESSION_ID"
                    echo $VECTRAGUARD_SESSION_ID > ~/.vectra-guard-session
                end
            end
        end
    end
    
    # Command logging
    function _vectra_guard_preexec --on-event fish_preexec
        set -g VECTRA_LAST_CMD $argv
    end
    
    function _vectra_guard_postexec --on-event fish_postexec
        if set -q VECTRA_LAST_CMD; and set -q VECTRAGUARD_SESSION_ID
            # Log command synchronously to avoid background job notifications
            fish -c "vectra-guard session record --session $VECTRAGUARD_SESSION_ID --command \"$VECTRA_LAST_CMD\" --exit-code $status &> /dev/null"
        end
    end
    
    # Initialize
    _vectra_guard_init
    
    # Convenience alias
    alias vg='vectra-guard'
end

# End Vectra Guard Integration
# ============================================================================
FISH_EOF
    
    echo "  ✅ Fish integration installed"
}

# Install for each detected shell
for shell in "${SELECTED_SHELLS[@]}"; do
    case $shell in
        bash) install_bash ;;
        zsh) install_zsh ;;
        fish) install_fish ;;
    esac
done
echo ""

# Initialize configuration
echo "Step 3/5: Initializing vectra-guard..."
if [ ! -f ~/.config/vectra-guard/config.yaml ] && [ ! -f vectra-guard.yaml ]; then
    if vectra-guard init &>/dev/null; then
        echo "  ✅ Created ~/.config/vectra-guard/config.yaml"
    else
        echo "  ℹ️  Config initialization skipped (optional)"
    fi
else
    echo "  ℹ️  Config already exists"
fi
echo ""

# Optional: IDE integration setup (Cursor/VS Code)
echo "Step 4/5: IDE integration (optional)..."
if [ -t 0 ] && [ -c /dev/tty ]; then
    read -p "Configure Cursor/VS Code integration? [y/N] " -n 1 -r < /dev/tty || REPLY="n"
    echo
    if [[ "${REPLY:-n}" =~ ^[Yy]$ ]]; then
        if [ -f "$(dirname "$0")/setup-cursor-protection.sh" ]; then
            read -p "Workspace path for IDE integration [$(pwd)]: " -r < /dev/tty
            WORKSPACE_PATH="${REPLY:-$(pwd)}"
            if [ ! -d "$WORKSPACE_PATH" ]; then
                echo "❌ Workspace path does not exist: $WORKSPACE_PATH"
                echo "   Skipping IDE integration."
            else
                echo "  ✅ Running Cursor/VS Code setup for: $WORKSPACE_PATH"
                WORKSPACE="$WORKSPACE_PATH" "$(dirname "$0")/setup-cursor-protection.sh"
            fi
        else
            echo "  ⚠️  setup-cursor-protection.sh not found, skipping"
        fi
    fi
else
    echo "  ℹ️  Skipping IDE integration in non-interactive mode"
fi
echo ""

# Optional: Install command aliases
echo "Step 5/5: Setting up safety aliases (optional)..."
# Use /dev/tty to read from terminal when piped through curl | bash
if [ -t 0 ] && [ -c /dev/tty ]; then
    read -p "Install safety aliases (wrap dangerous commands)? [y/N] " -n 1 -r < /dev/tty || REPLY="n"
    echo
else
    # Non-interactive mode - skip aliases
    REPLY="n"
    echo "  ℹ️  Skipping in non-interactive mode"
fi
if [[ "${REPLY:-n}" =~ ^[Yy]$ ]]; then
    for shell in "${SELECTED_SHELLS[@]}"; do
        case $shell in
            bash|zsh)
                cat >> ~/."${shell}rc" << 'ALIAS_EOF'

# Vectra Guard Safety Aliases
if command -v vectra-guard &> /dev/null; then
    alias rm='vectra-guard exec -- rm'
    alias sudo='vectra-guard exec --interactive -- sudo'
    alias curl='vectra-guard exec -- curl'
    alias wget='vectra-guard exec -- wget'
fi
ALIAS_EOF
                ;;
            fish)
                cat >> ~/.config/fish/config.fish << 'ALIAS_EOF'

# Vectra Guard Safety Aliases
if command -v vectra-guard > /dev/null
    alias rm='vectra-guard exec -- rm'
    alias sudo='vectra-guard exec --interactive -- sudo'
    alias curl='vectra-guard exec -- curl'
    alias wget='vectra-guard exec -- wget'
end
ALIAS_EOF
                ;;
        esac
    done
    echo "  ✅ Safety aliases installed"
fi
echo ""

# Summary
echo "=========================================="
echo "✅ Shell Tracker Installed!"
echo "=========================================="
echo ""
echo "Tracked shells: ${SELECTED_SHELLS[*]}"
echo "Convenience alias: vg (shorthand for vectra-guard)"
echo ""
echo "Next steps:"
if [[ " ${SELECTED_SHELLS[*]} " =~ " bash " ]]; then
    echo "1. Restart your terminal (or run: source ~/.bashrc)"
elif [[ " ${SELECTED_SHELLS[*]} " =~ " zsh " ]]; then
    echo "1. Restart your terminal (or run: source ~/.zshrc)"
elif [[ " ${SELECTED_SHELLS[*]} " =~ " fish " ]]; then
    echo "1. Restart your terminal (fish will auto-load config)"
fi
echo "2. Verify: echo \$VECTRAGUARD_SESSION_ID"
echo "3. Test: echo 'hello world'"
echo "4. Check logs: vg session show \$VECTRAGUARD_SESSION_ID"
echo ""
echo "Now works in:"
echo "  ✅ Terminal"
echo "  ✅ Cursor"
echo "  ✅ VSCode"
echo "  ✅ Any IDE"
echo "  ✅ SSH sessions"
echo "  ✅ Anywhere!"
echo ""
echo "To uninstall: Restore from backups"
if [[ " ${SELECTED_SHELLS[*]} " =~ " bash " ]]; then
    echo "  mv ~/.bashrc.vectra-backup ~/.bashrc"
fi
if [[ " ${SELECTED_SHELLS[*]} " =~ " zsh " ]]; then
    echo "  mv ~/.zshrc.vectra-backup ~/.zshrc"
fi
if [[ " ${SELECTED_SHELLS[*]} " =~ " fish " ]]; then
    echo "  mv ~/.config/fish/config.fish.vectra-backup ~/.config/fish/config.fish"
fi
echo ""
