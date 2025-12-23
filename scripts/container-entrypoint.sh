#!/bin/bash
# Vectra Guard Container Entrypoint
# Starts monitoring daemon and executes command with protection

set -e

AGENT_NAME="${AGENT_NAME:-container-agent}"
WORKSPACE="${WORKSPACE:-/workspace}"

echo "🛡️  Vectra Guard Container Environment"
echo "======================================"
echo "Agent: $AGENT_NAME"
echo "Workspace: $WORKSPACE"
echo ""

# Initialize vectra-guard config if not exists
if [ ! -f "$WORKSPACE/vectra-guard.yaml" ]; then
    echo "📝 Initializing vectra-guard configuration..."
    vectra-guard init
fi

# Start session
echo "🚀 Starting protected session..."
SESSION_ID=$(vectra-guard session start --agent "$AGENT_NAME" --workspace "$WORKSPACE")
export VECTRAGUARD_SESSION_ID="$SESSION_ID"

echo "✅ Session started: $SESSION_ID"
echo ""

# Setup cleanup handler
cleanup() {
    echo ""
    echo "🔒 Ending protected session..."
    vectra-guard session show "$SESSION_ID"
    vectra-guard session end "$SESSION_ID"
    echo "✅ Session ended"
}
trap cleanup EXIT INT TERM

# If no command specified, start interactive shell
if [ $# -eq 0 ]; then
    echo "Starting interactive protected shell..."
    echo "All commands will be monitored and validated."
    echo "Type 'exit' to end the session."
    echo ""
    exec /bin/bash
else
    # Execute provided command with protection
    echo "Executing: $@"
    echo ""
    exec vectra-guard exec --session "$SESSION_ID" -- "$@"
fi

