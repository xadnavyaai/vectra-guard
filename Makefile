.PHONY: build build-version test clean version

# Default version for development
VERSION ?= dev

# Build without version (shows "dev")
build:
	go build -o vectra-guard .

# Build with version
build-version:
	@if [ "$(VERSION)" = "dev" ]; then \
		echo "Usage: make build-version VERSION=v0.0.2"; \
		exit 1; \
	fi
	go build -ldflags "-X github.com/vectra-guard/vectra-guard/cmd.Version=$(VERSION)" -o vectra-guard .

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	rm -f vectra-guard
	rm -rf dist

# Show current version (if binary exists)
version:
	@./vectra-guard version 2>/dev/null || echo "Binary not built. Run 'make build' first."

# Dockerized testing
test-docker:
	./scripts/test-docker.sh

test-docker-quick:
	./scripts/test-docker.sh --quick

test-docker-security:
	./scripts/test-docker.sh --security

test-docker-destructive:
	./scripts/test-docker.sh --destructive

test-docker-shell:
	./scripts/test-docker.sh --shell

test-docker-clean:
	./scripts/test-docker.sh --clean

# Destructive testing (attack vectors)
test-destructive:
	./scripts/test-destructive.sh

test-destructive-quick:
	./scripts/test-destructive.sh --quick

test-destructive-docker:
	docker-compose -f docker-compose.test.yml run --rm --no-deps test-destructive

# Dev mode setup
dev-mode:
	./scripts/dev-mode.sh

dev-mode-force:
	./scripts/dev-mode.sh --force

# Help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  build          - Build binary (version: dev)"
	@echo "  build-version  - Build with version (requires VERSION=v0.0.2)"
	@echo ""
	@echo "Testing:"
	@echo "  test          - Run all tests"
	@echo "  test-docker   - Run tests in Docker container"
	@echo "  test-docker-quick - Run quick tests in Docker"
	@echo "  test-docker-security - Run security tests in Docker"
	@echo "  test-docker-destructive - Run destructive tests in Docker"
	@echo "  test-docker-shell - Interactive shell in test container"
	@echo "  test-docker-clean - Clean up test containers/images"
	@echo ""
	@echo "Destructive Testing (Attack Vectors):"
	@echo "  test-destructive - Run destructive test suite locally"
	@echo "  test-destructive-quick - Run quick destructive tests"
	@echo "  test-destructive-docker - Run destructive tests in Docker"
	@echo ""
	@echo "Development:"
	@echo "  dev-mode      - Setup sandbox-based dev mode (easy setup)"
	@echo "  dev-mode-force - Force overwrite existing config"
	@echo ""
	@echo "Utilities:"
	@echo "  clean         - Remove build artifacts"
	@echo "  version       - Show current version"
	@echo ""
	@echo "Examples:"
	@echo "  make build"
	@echo "  make build-version VERSION=v0.0.3"
	@echo "  make test"
	@echo "  make test-docker"
	@echo "  make test-docker-security"
	@echo "  make test-destructive-docker"
	@echo "  make dev-mode"

