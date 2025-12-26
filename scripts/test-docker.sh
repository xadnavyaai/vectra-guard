#!/bin/bash
# Dockerized Testing Script for Vectra Guard
#
# This script runs tests in a Docker container for consistent, isolated testing.
#
# Usage:
#   ./scripts/test-docker.sh [options] [test-target]
#
# Options:
#   --quick          Run only critical tests
#   --security       Run only security tests
#   --all            Run all tests (default)
#   --coverage       Generate coverage report
#   --rebuild        Rebuild test image
#   --shell          Drop into shell instead of running tests
#   --clean          Clean up test containers and images
#
# Examples:
#   ./scripts/test-docker.sh                    # Run all tests
#   ./scripts/test-docker.sh --security         # Run security tests only
#   ./scripts/test-docker.sh --quick            # Run quick tests
#   ./scripts/test-docker.sh --shell           # Interactive shell
#   ./scripts/test-docker.sh --clean            # Clean up

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Options
QUICK_MODE=false
SECURITY_ONLY=false
DESTRUCTIVE=false
ALL_TESTS=true
COVERAGE=false
REBUILD=false
SHELL_MODE=false
CLEAN=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --quick)
            QUICK_MODE=true
            ALL_TESTS=false
            shift
            ;;
        --security)
            SECURITY_ONLY=true
            ALL_TESTS=false
            shift
            ;;
        --destructive)
            DESTRUCTIVE=true
            ALL_TESTS=false
            shift
            ;;
        --all)
            ALL_TESTS=true
            shift
            ;;
        --coverage)
            COVERAGE=true
            shift
            ;;
        --rebuild)
            REBUILD=true
            shift
            ;;
        --shell)
            SHELL_MODE=true
            shift
            ;;
        --clean)
            CLEAN=true
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
    echo -e "\n${BLUE}╔═══════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║  $1${NC}"
    echo -e "${BLUE}╚═══════════════════════════════════════════════════════════╝${NC}\n"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check Docker
check_docker() {
    # Check if we're inside a container (Docker-in-Docker scenario)
    if [ -f /.dockerenv ] || [ -n "${VECTRAGUARD_CONTAINER:-}" ]; then
        print_info "Running inside container - Docker check skipped"
        return 0
    fi
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed or not in PATH"
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker daemon is not running"
        exit 1
    fi
    
    print_success "Docker is available"
}

# Clean up function
cleanup() {
    print_header "Cleaning Up Test Environment"
    
    # Stop and remove containers
    docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true
    
    # Remove test image
    docker rmi vectra-guard-test:latest 2>/dev/null || true
    
    # Remove dangling images
    docker image prune -f 2>/dev/null || true
    
    print_success "Cleanup complete"
}

# Build test image
build_image() {
    print_header "Building Test Image"
    
    cd "$PROJECT_ROOT"
    
    # Detect architecture
    local arch=$(uname -m)
    print_info "Detected architecture: $arch"
    
    # Docker will automatically build for the host architecture
    # The Dockerfile uses golang:1.21-alpine which supports both ARM and x86
    print_info "Building test image (will match host architecture)..."
    
    docker-compose -f docker-compose.test.yml build --no-cache
    
    print_success "Test image built successfully"
}

# Run tests
run_tests() {
    print_header "Running Tests in Docker"
    
    cd "$PROJECT_ROOT"
    
    # Determine test command
    local test_cmd=""
    
    if [ "$SHELL_MODE" = true ]; then
        test_cmd="bash"
    elif [ "$DESTRUCTIVE" = true ]; then
        test_cmd="./scripts/test-destructive.sh"
    elif [ "$SECURITY_ONLY" = true ]; then
        test_cmd="go test -v ./internal/analyzer/... -run TestEnhancedDestructiveCommandDetection && go test -v ./internal/sandbox/... -run TestMandatorySandboxingForCriticalCommands && go test -v ./cmd/... -run TestSecurityImprovementsRegression"
    elif [ "$QUICK_MODE" = true ]; then
        test_cmd="go test -v ./cmd/... -run TestSecurityImprovementsRegression"
    elif [ "$COVERAGE" = true ]; then
        test_cmd="go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html"
    else
        test_cmd="make test"
    fi
    
    print_info "Test command: $test_cmd"
    
    # Run tests - override the command from docker-compose
    # Use --no-deps to avoid starting other services
    if docker-compose -f docker-compose.test.yml run --rm --no-deps test sh -c "$test_cmd"; then
        print_success "All tests passed!"
        return 0
    else
        print_error "Some tests failed"
        return 1
    fi
}

# Check if running inside container
is_inside_container() {
    [ -f /.dockerenv ] || [ -n "${VECTRAGUARD_CONTAINER:-}" ] || [ -n "${GO_ENV:-}" ]
}

# Main execution
main() {
    print_header "Vectra Guard - Dockerized Testing"
    
    # If inside container, just run tests directly
    if is_inside_container; then
        print_info "Detected: Running inside container"
        print_info "Running tests directly..."
        
        if [ "$SHELL_MODE" = true ]; then
            print_info "You're already in the test container. Run 'make test' or any test command."
            exec bash
        elif [ "$DESTRUCTIVE" = true ]; then
            ./scripts/test-destructive.sh
        elif [ "$SECURITY_ONLY" = true ]; then
            go test -v ./internal/analyzer/... -run TestEnhancedDestructiveCommandDetection && \
            go test -v ./internal/sandbox/... -run TestMandatorySandboxingForCriticalCommands && \
            go test -v ./cmd/... -run TestSecurityImprovementsRegression
        elif [ "$QUICK_MODE" = true ]; then
            go test -v ./cmd/... -run TestSecurityImprovementsRegression
        elif [ "$COVERAGE" = true ]; then
            go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html
        else
            make test
        fi
        exit $?
    fi
    
    # Running on host - need Docker
    check_docker
    
    # Handle clean
    if [ "$CLEAN" = true ]; then
        cleanup
        exit 0
    fi
    
    # Rebuild if requested
    if [ "$REBUILD" = true ] || ! docker images | grep -q vectra-guard-test; then
        build_image
    fi
    
    # Run tests or shell
    if [ "$SHELL_MODE" = true ]; then
        print_header "Starting Interactive Shell"
        print_info "You're now in the test container. Run 'make test' or any test command."
        cd "$PROJECT_ROOT"
        docker-compose -f docker-compose.test.yml run --rm --no-deps test bash
    else
        run_tests
    fi
}

# Run main
main "$@"

