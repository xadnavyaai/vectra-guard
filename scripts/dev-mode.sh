#!/bin/bash
# Easy Sandbox-Based Development Mode Setup
#
# This script sets up Vectra Guard for sandbox-based development.
# It creates a configuration that auto-sandboxes risky commands while
# keeping common dev commands fast.
#
# SAFETY: This script ONLY creates a config file. It does NOT:
# - Auto-enable anything
# - Change your system
# - Run any commands
# - Modify existing files (unless --force)
#
# Usage:
#   ./scripts/dev-mode.sh [--force]
#
# Options:
#   --force    Overwrite existing config

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

FORCE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --force)
            FORCE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

CONFIG_FILE="$PROJECT_ROOT/vectra-guard.yaml"
PRESET_FILE="$PROJECT_ROOT/presets/dev-sandbox.yaml"

# Check if config already exists
if [ -f "$CONFIG_FILE" ] && [ "$FORCE" = false ]; then
    echo -e "${YELLOW}⚠ Config file already exists: $CONFIG_FILE${NC}"
    echo "Use --force to overwrite, or manually edit the file."
    echo ""
    echo "To use sandbox dev mode, ensure your config has:"
    echo "  sandbox:"
    echo "    enabled: true"
    echo "    mode: auto"
    exit 0
fi

# Check if preset exists
if [ ! -f "$PRESET_FILE" ]; then
    echo -e "${YELLOW}⚠ Preset file not found: $PRESET_FILE${NC}"
    exit 1
fi

# Copy preset to config
echo -e "${BLUE}Setting up sandbox-based development mode...${NC}"
cp "$PRESET_FILE" "$CONFIG_FILE"

echo -e "${GREEN}✓ Configuration created: $CONFIG_FILE${NC}"
echo ""
echo "Sandbox-based dev mode is now active!"
echo ""
echo "Features enabled:"
echo "  ✓ Auto-sandboxing for medium+ risk commands"
echo "  ✓ Fast caching (10x speedup after first run)"
echo "  ✓ Full network access for dev tools"
echo "  ✓ Minimal friction for common commands"
echo ""
echo "Try it out:"
echo "  vectra-guard exec -- npm install"
echo "  vectra-guard exec -- python -m pip install requests"
echo "  vectra-guard exec -- curl http://example.com | sh  # Will be sandboxed!"
echo ""
echo "View metrics:"
echo "  vectra-guard metrics show"
echo ""
echo "To disable sandbox mode, edit $CONFIG_FILE and set:"
echo "  sandbox:"
echo "    enabled: false"

