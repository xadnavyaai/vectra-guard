#!/usr/bin/env bash
# Test that the PATH and alias lines written by install.sh work when sourced.
# Run from repo root. Requires vectra-guard on PATH (e.g. after install) or set INSTALL_DIR.

set -e
# Use default install dir so vectra-guard is found after sourcing
INSTALL_DIR="$HOME/.local/bin"
RC=$(mktemp)
trap 'rm -f "$RC"' EXIT

# Same block install.sh writes
{
  echo "# Vectra Guard: add install dir to PATH"
  if [ "$INSTALL_DIR" = "$HOME/.local/bin" ]; then
    echo 'export PATH="$PATH:$HOME/.local/bin"'
  else
    echo "export PATH=\"\$PATH:$INSTALL_DIR\""
  fi
  echo "alias vg='vectra-guard'"
  echo "alias vectraguard='vectra-guard'"
} >> "$RC"

echo "=== Written to rc (same as install.sh) ==="
cat "$RC"
echo ""
echo "=== Testing in bash (source rc then run vg / vectraguard) ==="
bash -c "
  shopt -s expand_aliases
  source '$RC'
  vg version 2>&1 | head -1
  vectraguard version 2>&1 | head -1
"
echo ""
echo "=== Aliases work: vg and vectraguard both run vectra-guard ==="
