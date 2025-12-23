#!/bin/bash
# Vectra Guard Updater
# Quick upgrade to latest version

set -e

echo "⚡ Vectra Guard Updater"
echo "======================"
echo ""

# Check if installed
if ! command -v vectra-guard &> /dev/null; then
    echo "❌ Vectra Guard is not installed"
    echo "   Install with: curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash"
    exit 1
fi

CURRENT=$(vectra-guard --help 2>&1 | head -1 || echo "unknown")
echo "Current version: $CURRENT"
echo "Location: $(which vectra-guard)"
echo ""
echo "Fetching latest version..."
echo ""

# Run installer (it will handle upgrade)
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash

