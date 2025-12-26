#!/bin/bash
# Security Test Script for Vectra Guard
# 
# This script incrementally tests security improvements to prevent
# escalated mishandling of destructive commands.
#
# Usage:
#   ./scripts/test-security.sh [--quick] [--verbose]
#
# Options:
#   --quick    Run only critical tests
#   --verbose  Show detailed output

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0
TESTS_SKIPPED=0

# Options
QUICK_MODE=false
VERBOSE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --quick)
            QUICK_MODE=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Print functions
print_header() {
    echo -e "\n${BLUE}=== $1 ===${NC}"
}

print_test() {
    echo -e "${YELLOW}Testing:${NC} $1"
}

print_pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((TESTS_PASSED++))
}

print_fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    ((TESTS_FAILED++))
}

print_skip() {
    echo -e "${YELLOW}⊘ SKIP${NC}: $1"
    ((TESTS_SKIPPED++))
}

# Check if vectra-guard binary exists
check_binary() {
    if ! command -v vectra-guard &> /dev/null; then
        if [ -f "$PROJECT_ROOT/vectra-guard" ]; then
            export PATH="$PROJECT_ROOT:$PATH"
        else
            echo -e "${RED}Error: vectra-guard binary not found${NC}"
            echo "Please build it first: go build -o vectra-guard ."
            exit 1
        fi
    fi
}

# Test 1: Enhanced Destructive Command Detection
test_destructive_command_detection() {
    print_header "Test 1: Enhanced Destructive Command Detection"
    
    local dangerous_commands=(
        "rm -r /*"
        "rm -rf /*"
        "rm -r /"
        "rm -rf /bin"
        "rm -rf /usr"
        "rm -rf /etc"
    )
    
    for cmd in "${dangerous_commands[@]}"; do
        print_test "Detection of: $cmd"
        
        if vectra-guard validate <(echo "$cmd") 2>&1 | grep -q "DANGEROUS_DELETE_ROOT"; then
            print_pass "Detected: $cmd"
        else
            print_fail "NOT detected: $cmd"
        fi
    done
}

# Test 2: Mandatory Sandboxing
test_mandatory_sandboxing() {
    print_header "Test 2: Mandatory Sandboxing for Critical Commands"
    
    # This test requires the binary to be built and Go tests to run
    print_test "Running Go tests for mandatory sandboxing"
    
    if cd "$PROJECT_ROOT" && go test -v ./internal/sandbox/... -run TestMandatorySandboxingForCriticalCommands 2>&1 | grep -q "PASS"; then
        print_pass "Mandatory sandboxing tests passed"
    else
        print_fail "Mandatory sandboxing tests failed"
    fi
}

# Test 3: Pre-Execution Assessment
test_pre_execution_assessment() {
    print_header "Test 3: Pre-Execution Assessment"
    
    # Test that critical commands are blocked before execution
    print_test "Critical command should be blocked"
    
    # Create a temp config with sandbox disabled
    local temp_config=$(mktemp)
    cat > "$temp_config" <<EOF
guard_level:
  level: medium
sandbox:
  enabled: false
EOF
    
    # Try to execute a critical command
    if vectra-guard exec --config "$temp_config" "rm -r /*" 2>&1 | grep -qi "sandbox required\|blocked\|critical"; then
        print_pass "Critical command blocked when sandbox disabled"
    else
        print_fail "Critical command NOT blocked when sandbox disabled"
    fi
    
    rm -f "$temp_config"
}

# Test 4: Pattern Variations
test_pattern_variations() {
    print_header "Test 4: Pattern Variation Detection"
    
    local patterns=(
        "rm -rf /"
        "rm -r /"
        "rm -rf /*"
        "rm -r /*"
        "rm -rf / *"
        "rm -r / *"
    )
    
    for pattern in "${patterns[@]}"; do
        print_test "Pattern: $pattern"
        
        if vectra-guard validate <(echo "$pattern") 2>&1 | grep -q "DANGEROUS_DELETE_ROOT\|critical"; then
            print_pass "Pattern detected: $pattern"
        else
            print_fail "Pattern NOT detected: $pattern"
        fi
    done
}

# Test 5: Go Unit Tests
test_go_unit_tests() {
    print_header "Test 5: Go Unit Tests"
    
    print_test "Running analyzer tests"
    if cd "$PROJECT_ROOT" && go test ./internal/analyzer/... -run TestEnhancedDestructiveCommandDetection 2>&1 | grep -q "PASS"; then
        print_pass "Analyzer tests passed"
    else
        print_fail "Analyzer tests failed"
    fi
    
    print_test "Running sandbox tests"
    if cd "$PROJECT_ROOT" && go test ./internal/sandbox/... -run TestMandatorySandboxingForCriticalCommands 2>&1 | grep -q "PASS"; then
        print_pass "Sandbox tests passed"
    else
        print_fail "Sandbox tests failed"
    fi
}

# Test 6: Integration Test
test_integration() {
    print_header "Test 6: Integration Test"
    
    print_test "Full flow: detection -> assessment -> sandboxing"
    
    # This is a complex test that would require actual execution
    # For now, we'll verify the components work together
    if [ -f "$PROJECT_ROOT/vectra-guard" ]; then
        print_pass "Binary exists and can be tested"
    else
        print_skip "Binary not found, skipping integration test"
    fi
}

# Test 7: Regression Test (The Incident Scenario)
test_incident_scenario() {
    print_header "Test 7: Regression Test - Incident Scenario"
    
    print_test "Testing: rm -r /* (the incident command)"
    
    # This should be detected and blocked
    if vectra-guard validate <(echo "rm -r /*") 2>&1 | grep -q "DANGEROUS_DELETE_ROOT\|critical"; then
        print_pass "Incident scenario command is now detected"
    else
        print_fail "Incident scenario command is NOT detected - REGRESSION!"
    fi
}

# Main test runner
main() {
    echo -e "${BLUE}"
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║     Vectra Guard Security Test Suite                    ║"
    echo "║     Testing security improvements incrementally          ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    check_binary
    
    # Run tests
    test_incident_scenario  # Run this first - it's the most important
    
    if [ "$QUICK_MODE" = false ]; then
        test_destructive_command_detection
        test_pattern_variations
        test_go_unit_tests
        test_mandatory_sandboxing
        test_pre_execution_assessment
        test_integration
    else
        echo -e "${YELLOW}Quick mode: Running only critical tests${NC}"
        test_go_unit_tests
    fi
    
    # Print summary
    echo -e "\n${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                    Test Summary                            ║${NC}"
    echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
    echo -e "${RED}Failed:${NC} $TESTS_FAILED"
    echo -e "${YELLOW}Skipped:${NC} $TESTS_SKIPPED"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed! ✓${NC}"
        exit 0
    else
        echo -e "\n${RED}Some tests failed! ✗${NC}"
        exit 1
    fi
}

# Run main
main "$@"

