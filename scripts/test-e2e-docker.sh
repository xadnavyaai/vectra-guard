#!/bin/bash
# End-to-End Test Suite for Vectra Guard
# Comprehensive feature testing in Docker

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Global binary path (set once, used everywhere)
VECTRA_BINARY=""

# Options
VERBOSE=false
QUICK_MODE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --verbose) VERBOSE=true; shift ;;
        --quick) QUICK_MODE=true; shift ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

print_header() {
    echo -e "\n${MAGENTA}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${MAGENTA}║  $1${NC}"
    echo -e "${MAGENTA}╚═══════════════════════════════════════════════════════════╝${NC}\n"
}

print_success() { echo -e "${GREEN}✓ PASS${NC}: $1"; ((TESTS_PASSED++)); }
print_failure() { echo -e "${RED}✗ FAIL${NC}: $1"; ((TESTS_FAILED++)); }
print_skip() { echo -e "${YELLOW}⊘ SKIP${NC}: $1"; ((TESTS_SKIPPED++)); }
print_info() { echo -e "${CYAN}ℹ${NC} $1"; }

# Get and build binary (SAFE: doesn't overwrite local binary)
setup_binary() {
    cd "$PROJECT_ROOT" || exit 1
    
    # Check if existing binary works (if running in Docker, it should be Linux)
    if [ -f "$PROJECT_ROOT/vectra-guard" ]; then
        if "$PROJECT_ROOT/vectra-guard" version >/dev/null 2>&1; then
            VECTRA_BINARY="$PROJECT_ROOT/vectra-guard"
            print_info "Using existing binary: $VECTRA_BINARY"
            return 0
        fi
    fi
    
    # Build in a safe temp location to avoid overwriting local binary
    local temp_binary
    temp_binary=$(mktemp /tmp/vectra-guard-test.XXXXXX) || {
        print_failure "Failed to create temp file for binary"
        exit 1
    }
    
    print_info "Building vectra-guard binary in temp location (safe, won't overwrite local)..."
    if ! go build -buildvcs=false -o "$temp_binary" .; then
        rm -f "$temp_binary"
        print_failure "Failed to build binary"
        exit 1
    fi
    
    chmod +x "$temp_binary" 2>/dev/null || true
    
    if [ ! -f "$temp_binary" ]; then
        print_failure "Binary not found after build"
        exit 1
    fi
    
    # Test if it can run
    if ! "$temp_binary" version >/dev/null 2>&1; then
        print_info "Binary may have architecture issues, but will try..."
    fi
    
    VECTRA_BINARY="$temp_binary"
    print_info "Using binary: $VECTRA_BINARY (temp location, safe)"
    
    # Cleanup temp binary on exit
    trap "rm -f '$temp_binary'" EXIT
}

test_command() {
    local test_name="$1"
    local cmd="$2"
    local expected_exit="${3:-0}"
    
    local output
    local exit_code=0
    output=$(eval "$cmd" 2>&1) || exit_code=$?
    
    if [ $exit_code -ne $expected_exit ]; then
        print_failure "$test_name (expected exit $expected_exit, got $exit_code)"
        [ "$VERBOSE" = true ] && echo "  Output: $output"
        return 1
    fi
    
    print_success "$test_name"
    return 0
}

# Test suites
test_initialization() {
    print_header "Testing Initialization (init)"
    cd "$TEST_WORKSPACE" || return 1
    
    test_command "Initialize config" "$VECTRA_BINARY init" 0
    [ -f "$HOME/.config/vectra-guard/config.yaml" ] && print_success "Config file created" || print_failure "Config file not created"
    test_command "Initialize with --force" "$VECTRA_BINARY init --force" 0
    test_command "Initialize as TOML" "$VECTRA_BINARY init --toml --force" 0
    [ -f "$HOME/.config/vectra-guard/config.toml" ] && print_success "TOML config created" || print_failure "TOML config not created"
}

test_validation() {
    print_header "Testing Validation (validate)"
    cd "$TEST_WORKSPACE" || return 1
    
    echo 'echo "safe"' > safe.sh
    echo 'rm -rf /' > dangerous.sh
    
    test_command "Validate safe script" "$VECTRA_BINARY validate safe.sh" 0
    
    local output
    local exit_code=0
    output=$($VECTRA_BINARY validate dangerous.sh 2>&1) || exit_code=$?
    if [ $exit_code -eq 1 ] || [ $exit_code -eq 2 ]; then
        print_success "Validate dangerous script"
    else
        print_failure "Validate dangerous script (got exit $exit_code)"
    fi
}

test_execution() {
    print_header "Testing Protected Execution (exec)"
    cd "$TEST_WORKSPACE" || return 1
    
    test_command "Execute safe command" "$VECTRA_BINARY exec echo hello" 0
    test_command "Execute with args" "$VECTRA_BINARY exec ls -la" 0
}

test_session() {
    print_header "Testing Session Management"
    cd "$TEST_WORKSPACE" || return 1
    
    # Use sed instead of tail for compatibility
    local session_output
    session_output=$($VECTRA_BINARY session start --agent "e2e-test" 2>&1)
    local session_id=$(echo "$session_output" | sed -n '$p' | grep -o 'session-[^ ]*' || echo "")
    
    if [ -n "$session_id" ]; then
        print_success "Session started: $session_id"
        test_command "Show session" "$VECTRA_BINARY session show $session_id" 0
        test_command "List sessions" "$VECTRA_BINARY session list" 0
        test_command "End session" "$VECTRA_BINARY session end $session_id" 0
    else
        print_failure "Failed to start session"
    fi
}

test_trust() {
    print_header "Testing Trust Management"
    cd "$TEST_WORKSPACE" || return 1
    
    test_command "List trusted" "$VECTRA_BINARY trust list" 0
    test_command "Add trusted" "$VECTRA_BINARY trust add 'echo test' --note 'Test'" 0
    test_command "Remove trusted" "$VECTRA_BINARY trust remove 'echo test'" 0
}

test_metrics() {
    print_header "Testing Metrics"
    cd "$TEST_WORKSPACE" || return 1
    
    test_command "Show metrics" "$VECTRA_BINARY metrics show" 0
    
    local output
    local exit_code=0
    output=$($VECTRA_BINARY metrics show --json 2>&1) || exit_code=$?
    if [ $exit_code -eq 0 ]; then
        print_success "Show metrics as JSON"
    else
        print_failure "Show metrics as JSON (exit: $exit_code)"
    fi
    
    test_command "Reset metrics" "$VECTRA_BINARY metrics reset" 0
}

main() {
    echo -e "${MAGENTA}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${MAGENTA}║     Vectra Guard - End-to-End Test Suite                  ║${NC}"
    echo -e "${MAGENTA}╚═══════════════════════════════════════════════════════════╝${NC}\n"
    
    [ -f /.dockerenv ] && print_info "Running inside Docker container" || print_info "Running on host"
    
    setup_binary
    
    TEST_WORKSPACE=$(mktemp -d)
    export TEST_WORKSPACE
    
    test_initialization || true
    test_validation || true
    test_execution || true
    test_session || true
    test_trust || true
    test_metrics || true
    
    echo -e "\n${MAGENTA}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${MAGENTA}║                    Test Summary                            ║${NC}"
    echo -e "${MAGENTA}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo -e "${GREEN}Tests Passed:${NC} $TESTS_PASSED"
    echo -e "${RED}Tests Failed:${NC} $TESTS_FAILED"
    echo -e "${YELLOW}Tests Skipped:${NC} $TESTS_SKIPPED"
    
    local total=$((TESTS_PASSED + TESTS_FAILED))
    local rate=0
    [ $total -gt 0 ] && rate=$((TESTS_PASSED * 100 / total))
    echo -e "\n${CYAN}Success Rate:${NC} ${rate}% (${TESTS_PASSED}/${total})"
    
    # Cleanup test workspace (safe - only removes temp directory)
    if [ -n "${TEST_WORKSPACE:-}" ] && [ -d "$TEST_WORKSPACE" ]; then
        print_info "Cleaning up test workspace..."
        rm -rf "$TEST_WORKSPACE"
    fi
    
    [ $TESTS_FAILED -eq 0 ] && exit 0 || exit 1
}

main "$@"
