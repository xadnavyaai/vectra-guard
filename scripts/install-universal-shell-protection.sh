#!/bin/bash
# Vectra Guard: Universal Shell Protection Installer
# Integrates with bash, zsh, and fish to protect ALL tools

set -e

echo "🛡️  Vectra Guard - Universal Shell Protection Installer"
echo "======================================================="
echo ""

# Check if vectra-guard is installed
if ! command -v vectra-guard &> /dev/null; then
    echo "❌ Error: vectra-guard not found in PATH"
    echo "   Please install vectra-guard first"
    exit 1
fi

# Detect available shells
SHELLS=()
[[ -f ~/.bashrc ]] && SHELLS+=("bash")
[[ -f ~/.zshrc ]] && SHELLS+=("zsh")
[[ -d ~/.config/fish ]] && SHELLS+=("fish")

if [ ${#SHELLS[@]} -eq 0 ]; then
    echo "❌ No shell configuration files found"
    echo "   Please create ~/.bashrc or ~/.zshrc first"
    exit 1
fi

echo "Detected shells: ${SHELLS[*]}"
echo ""

# Backup existing configs
echo "Step 1/4: Backing up existing configurations..."
for shell in "${SHELLS[@]}"; do
    case $shell in
        bash)
            cp ~/.bashrc ~/.bashrc.vectra-backup
            echo "  ✅ Backed up ~/.bashrc"
            ;;
        zsh)
            cp ~/.zshrc ~/.zshrc.vectra-backup
            echo "  ✅ Backed up ~/.zshrc"
            ;;
        fish)
            cp ~/.config/fish/config.fish ~/.config/fish/config.fish.vectra-backup 2>/dev/null || true
            echo "  ✅ Backed up ~/.config/fish/config.fish"
            ;;
    esac
done
echo ""

# Install shell integrations
echo "Step 2/4: Installing shell integrations..."

# Function to add bash integration
install_bash() {
    cat >> ~/.bashrc << 'BASH_EOF'

# ============================================================================
# Vectra Guard Integration (Auto-generated)
# ============================================================================

if command -v vectra-guard &> /dev/null; then
    # Initialize session
    _vectra_guard_init() {
        if [ -z "$VECTRAGUARD_SESSION_ID" ]; then
            if [ -f ~/.vectra-guard-session ]; then
                export VECTRAGUARD_SESSION_ID=$(cat ~/.vectra-guard-session)
                # Verify session is still valid
                if ! vectra-guard session show "$VECTRAGUARD_SESSION_ID" &>/dev/null; then
                    # Session expired, start new one
                    unset VECTRAGUARD_SESSION_ID
                fi
            fi
            
            if [ -z "$VECTRAGUARD_SESSION_ID" ]; then
                SESSION=$(vectra-guard session start --agent "${USER}-bash" --workspace "$HOME" 2>/dev/null)
                if [ -n "$SESSION" ]; then
                    export VECTRAGUARD_SESSION_ID=$SESSION
                    echo $SESSION > ~/.vectra-guard-session
                    echo "🛡️  Vectra Guard session: $SESSION"
                fi
            fi
        fi
    }
    
    # Command logging hook
    _vectra_guard_preexec() {
        VECTRA_LAST_CMD="$BASH_COMMAND"
        VECTRA_START_TIME=$SECONDS
    }
    
    _vectra_guard_precmd() {
        local exit_code=$?
        if [ -n "$VECTRA_LAST_CMD" ] && [ -n "$VECTRAGUARD_SESSION_ID" ]; then
            # Log command asynchronously (don't block prompt)
            (
                vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "logged: $VECTRA_LAST_CMD" &>/dev/null
            ) &
        fi
        unset VECTRA_LAST_CMD
    }
    
    # Set up hooks
    trap '_vectra_guard_preexec' DEBUG
    PROMPT_COMMAND="_vectra_guard_precmd${PROMPT_COMMAND:+; $PROMPT_COMMAND}"
    
    # Initialize on shell start
    _vectra_guard_init
fi

# End Vectra Guard Integration
# ============================================================================
BASH_EOF
    
    echo "  ✅ Bash integration installed"
}

# Function to add zsh integration
install_zsh() {
    cat >> ~/.zshrc << 'ZSH_EOF'

# ============================================================================
# Vectra Guard Integration (Auto-generated)
# ============================================================================

if command -v vectra-guard &> /dev/null; then
    # Initialize session
    _vectra_guard_init() {
        if [[ -z "$VECTRAGUARD_SESSION_ID" ]]; then
            if [[ -f ~/.vectra-guard-session ]]; then
                export VECTRAGUARD_SESSION_ID=$(cat ~/.vectra-guard-session)
                # Verify session is still valid
                if ! vectra-guard session show "$VECTRAGUARD_SESSION_ID" &>/dev/null; then
                    unset VECTRAGUARD_SESSION_ID
                fi
            fi
            
            if [[ -z "$VECTRAGUARD_SESSION_ID" ]]; then
                SESSION=$(vectra-guard session start --agent "${USER}-zsh" --workspace "$HOME" 2>/dev/null)
                if [[ -n "$SESSION" ]]; then
                    export VECTRAGUARD_SESSION_ID=$SESSION
                    echo $SESSION > ~/.vectra-guard-session
                    echo "🛡️  Vectra Guard session: $SESSION"
                fi
            fi
        fi
    }
    
    # Command hooks
    _vectra_guard_preexec() {
        VECTRA_LAST_CMD="$1"
    }
    
    _vectra_guard_precmd() {
        local exit_code=$?
        if [[ -n "$VECTRA_LAST_CMD" && -n "$VECTRAGUARD_SESSION_ID" ]]; then
            # Log asynchronously
            (
                vectra-guard exec --session "$VECTRAGUARD_SESSION_ID" -- echo "logged: $VECTRA_LAST_CMD" &>/dev/null
            ) &
        fi
        unset VECTRA_LAST_CMD
    }
    
    # Register hooks
    autoload -Uz add-zsh-hook
    add-zsh-hook preexec _vectra_guard_preexec
    add-zsh-hook precmd _vectra_guard_precmd
    
    # Initialize
    _vectra_guard_init
fi

# End Vectra Guard Integration
# ============================================================================
ZSH_EOF
    
    echo "  ✅ Zsh integration installed"
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
                set -gx VECTRAGUARD_SESSION_ID (cat ~/.vectra-guard-session)
                # Verify session
                if not vectra-guard session show $VECTRAGUARD_SESSION_ID &> /dev/null
                    set -e VECTRAGUARD_SESSION_ID
                end
            end
            
            if not set -q VECTRAGUARD_SESSION_ID
                set -gx VECTRAGUARD_SESSION_ID (vectra-guard session start --agent "$USER-fish" --workspace $HOME 2>/dev/null)
                if test -n "$VECTRAGUARD_SESSION_ID"
                    echo $VECTRAGUARD_SESSION_ID > ~/.vectra-guard-session
                    echo "🛡️  Vectra Guard session: $VECTRAGUARD_SESSION_ID"
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
            # Log asynchronously
            fish -c "vectra-guard exec --session $VECTRAGUARD_SESSION_ID -- echo 'logged: $VECTRA_LAST_CMD' &> /dev/null" &
        end
    end
    
    # Initialize
    _vectra_guard_init
end

# End Vectra Guard Integration
# ============================================================================
FISH_EOF
    
    echo "  ✅ Fish integration installed"
}

# Install for each detected shell
for shell in "${SHELLS[@]}"; do
    case $shell in
        bash) install_bash ;;
        zsh) install_zsh ;;
        fish) install_fish ;;
    esac
done
echo ""

# Initialize configuration
echo "Step 3/4: Initializing vectra-guard..."
if [ ! -f vectra-guard.yaml ]; then
    vectra-guard init
    echo "  ✅ Created vectra-guard.yaml"
else
    echo "  ℹ️  vectra-guard.yaml already exists"
fi
echo ""

# Optional: Install command aliases
echo "Step 4/4: Setting up safety aliases (optional)..."
read -p "Install safety aliases (wrap dangerous commands)? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    for shell in "${SHELLS[@]}"; do
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
echo "✅ Universal Shell Protection Installed!"
echo "=========================================="
echo ""
echo "Protected shells: ${SHELLS[*]}"
echo ""
echo "Next steps:"
echo "1. Restart your terminal (or run: source ~/.bashrc)"
echo "2. Verify: echo \$VECTRAGUARD_SESSION_ID"
echo "3. Test: echo 'hello world'"
echo "4. Check logs: vectra-guard session show \$VECTRAGUARD_SESSION_ID"
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
echo "  mv ~/.bashrc.vectra-backup ~/.bashrc"
echo "  mv ~/.zshrc.vectra-backup ~/.zshrc"
echo ""

