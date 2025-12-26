# Dockerized Testing Guide

## Overview

Dockerized testing provides a consistent, isolated environment for running Vectra Guard tests. This ensures tests run the same way across different machines and CI/CD pipelines.

## Quick Start

### Run All Tests in Docker

```bash
# Using the script
./scripts/test-docker.sh

# Or using Makefile
make test-docker
```

### Run Quick Tests

```bash
./scripts/test-docker.sh --quick
# or
make test-docker-quick
```

### Run Security Tests Only

```bash
./scripts/test-docker.sh --security
# or
make test-docker-security
```

## Prerequisites

1. **Docker installed** and running
   ```bash
   docker --version
   docker info
   ```

2. **Docker Compose** (usually comes with Docker Desktop)
   ```bash
   docker-compose --version
   ```

## Test Script Options

### Basic Usage

```bash
# Run all tests
./scripts/test-docker.sh

# Run only critical/quick tests
./scripts/test-docker.sh --quick

# Run only security tests
./scripts/test-docker.sh --security

# Generate coverage report
./scripts/test-docker.sh --coverage

# Interactive shell in test container
./scripts/test-docker.sh --shell

# Rebuild test image
./scripts/test-docker.sh --rebuild

# Clean up test containers and images
./scripts/test-docker.sh --clean
```

### Options Explained

| Option | Description |
|--------|-------------|
| `--quick` | Run only critical regression tests |
| `--security` | Run only security-related tests |
| `--all` | Run all tests (default) |
| `--coverage` | Generate test coverage report |
| `--rebuild` | Rebuild test image before running |
| `--shell` | Drop into interactive shell instead of running tests |
| `--clean` | Clean up test containers and images |

## Test Environment

### Docker Image

The test environment (`Dockerfile.test`) includes:
- Go 1.21
- Docker and Docker Compose (for sandbox testing)
- All build tools (make, git, etc.)
- Test dependencies

### Docker Compose

The `docker-compose.test.yml` provides:
- Isolated test container
- Docker-in-Docker support (for sandbox tests)
- Volume mounts for source code
- Go module cache for faster builds

## Usage Examples

### Example 1: Run All Tests

```bash
./scripts/test-docker.sh
```

**Output:**
```
╔═══════════════════════════════════════════════════════════╗
║  Vectra Guard - Dockerized Testing                       ║
╚═══════════════════════════════════════════════════════════╝

✓ Docker is available
╔═══════════════════════════════════════════════════════════╗
║  Building Test Image                                      ║
╚═══════════════════════════════════════════════════════════╝

...
✓ Test image built successfully

╔═══════════════════════════════════════════════════════════╗
║  Running Tests in Docker                                 ║
╚═══════════════════════════════════════════════════════════╝

ℹ Test command: make test
...
✓ All tests passed!
```

### Example 2: Quick Security Tests

```bash
./scripts/test-docker.sh --quick
```

Runs only the critical regression test that verifies the incident scenario is prevented.

### Example 3: Interactive Shell

```bash
./scripts/test-docker.sh --shell
```

Drops you into a bash shell inside the test container where you can:
- Run tests manually
- Debug issues
- Explore the test environment

```bash
# Inside the container
make test
go test -v ./internal/analyzer/... -run TestEnhancedDestructiveCommandDetection
./vectra-guard version
```

### Example 4: Generate Coverage Report

```bash
./scripts/test-docker.sh --coverage
```

Generates `coverage.html` in the project root that you can open in a browser.

### Example 5: Clean Up

```bash
./scripts/test-docker.sh --clean
```

Removes:
- Test containers
- Test images
- Dangling images

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Docker Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      
      - name: Run tests in Docker
        run: |
          chmod +x scripts/test-docker.sh
          ./scripts/test-docker.sh
      
      - name: Run security tests
        run: ./scripts/test-docker.sh --security
      
      - name: Upload coverage
        if: success()
        uses: codecov/codecov-action@v2
        with:
          file: ./coverage.out
```

### GitLab CI Example

```yaml
test:
  stage: test
  image: docker:latest
  services:
    - docker:dind
  before_script:
    - apk add --no-cache docker-compose
  script:
    - ./scripts/test-docker.sh
```

## Troubleshooting

### Docker Not Running

```bash
# Check Docker status
docker info

# Start Docker (macOS)
open -a Docker

# Start Docker (Linux)
sudo systemctl start docker
```

### Permission Denied

```bash
# Make script executable
chmod +x scripts/test-docker.sh

# Or use Makefile
make test-docker
```

### Test Image Out of Date

```bash
# Rebuild test image
./scripts/test-docker.sh --rebuild
```

### Docker Socket Issues

If you get Docker socket permission errors:

```bash
# Add user to docker group (Linux)
sudo usermod -aG docker $USER
newgrp docker

# Or run with sudo (not recommended)
sudo ./scripts/test-docker.sh
```

### Out of Disk Space

```bash
# Clean up Docker
docker system prune -a

# Clean test environment
./scripts/test-docker.sh --clean
```

## Advantages of Dockerized Testing

1. **Consistency** - Same environment everywhere
2. **Isolation** - Tests don't affect your system
3. **Reproducibility** - Easy to reproduce test failures
4. **CI/CD Ready** - Works the same locally and in CI
5. **Clean State** - Fresh environment for each test run
6. **Docker-in-Docker** - Can test sandbox functionality properly

## Comparison: Local vs Docker

| Feature | Local Testing | Docker Testing |
|---------|---------------|----------------|
| Speed | ⚡ Faster | 🐢 Slower (container overhead) |
| Consistency | ⚠️ Varies by machine | ✅ Same everywhere |
| Isolation | ⚠️ Uses your system | ✅ Fully isolated |
| CI/CD | ⚠️ May differ | ✅ Same as CI |
| Sandbox Tests | ⚠️ Requires Docker | ✅ Docker-in-Docker |

## Best Practices

1. **Use Docker for CI/CD** - Ensures consistency
2. **Use local for development** - Faster iteration
3. **Run Docker tests before commit** - Catch issues early
4. **Clean up regularly** - Free disk space
5. **Rebuild when dependencies change** - Keep image fresh

## Next Steps

1. ✅ Run tests: `./scripts/test-docker.sh`
2. ✅ Verify security tests: `./scripts/test-docker.sh --security`
3. ✅ Integrate into CI/CD pipeline
4. ✅ Set up pre-commit hooks

---

## Architecture Support (ARM & x86)

Vectra Guard's Dockerized testing setup supports both **ARM64** (Apple Silicon, ARM servers) and **x86_64** (Intel/AMD) architectures automatically.

### Automatic Architecture Detection

Docker automatically builds images for the **host architecture**:
- **ARM64** machines (Apple Silicon M1/M2/M3, ARM servers) → builds ARM64 images
- **x86_64** machines (Intel/AMD) → builds x86_64 images

### Base Image Support

The `golang:1.25-alpine` base image supports both architectures:
- ✅ **linux/amd64** (x86_64)
- ✅ **linux/arm64** (ARM64)

Docker automatically pulls the correct architecture variant.

### Verify Architecture

```bash
# Check your host architecture
uname -m
# Output: arm64 or x86_64

# Check Docker architecture
docker version --format '{{.Server.Arch}}'

# Check test image architecture
docker inspect vectra-guard-test:latest | grep Architecture
```

### Cross-Platform Testing

To test on a different architecture than your host:

```bash
# Build for x86_64 on ARM machine
docker buildx build --platform linux/amd64 -f Dockerfile.test -t vectra-guard-test:amd64 .

# Build for ARM64 on x86 machine
docker buildx build --platform linux/arm64 -f Dockerfile.test -t vectra-guard-test:arm64 .
```

---

**Test Consistently. Deploy Confidently.** 🐳🧪

