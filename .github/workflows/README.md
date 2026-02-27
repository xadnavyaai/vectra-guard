# GitHub Actions CI/CD Workflows

This directory contains the CI/CD workflows for Vectra Guard. All workflows are automated via GitHub Actions.

## 📋 Workflows Overview

### 1. **PR Quick Check** (`pr-quick-check.yml`)
- **Trigger**: Runs on every PR (open, update, reopen)
- **Purpose**: Fast feedback for developers
- **Duration**: ~2-5 minutes
- **Checks**:
  - Go code formatting (`go fmt`)
  - Static analysis (`go vet`)
  - Build verification
  - Fast unit tests (with `-short` flag)

### 2. **CI/CD Pipeline** (`ci.yml`)
- **Trigger**: Runs on PRs and pushes to `main` branch
- **Purpose**: Comprehensive testing before merge
- **Duration**: ~10-20 minutes
- **Jobs**:
  1. **Unit Tests**: All Go unit tests, internal tests, CVE tests
  2. **Build Verification**: Ensures binary builds successfully
  3. **Docker Tests**: Runs Docker-based integration tests
  4. **Multi-platform Build**: Tests builds on Ubuntu, macOS, and Windows
  5. **Lint & Code Quality**: Formatting, vet, staticcheck
  6. **Test Coverage**: Generates coverage reports

### 3. **Release** (`release.yml`)
- **Trigger**: When a version tag is pushed (e.g., `v0.2.1`)
- **Purpose**: Automated release creation
- **Produces**:
  - Multi-platform binaries (Linux/macOS/Windows, amd64/arm64)
  - SHA256 checksums for all binaries
  - GitHub Release with installation instructions
- **Platforms**:
  - `linux-amd64`
  - `linux-arm64`
  - `darwin-amd64` (Intel Mac)
  - `darwin-arm64` (Apple Silicon)
  - `windows-amd64`
  - `windows-arm64`

## 🚀 Workflow Execution Order

### For Pull Requests:
1. **PR Quick Check** runs first (fast feedback)
2. **CI/CD Pipeline** runs in parallel (comprehensive tests)
3. Both must pass before merge is allowed

### For Main Branch:
1. **CI/CD Pipeline** runs on every push to validate main branch integrity

### For Release Tags:
1. **Release** workflow runs when tag matches `v*.*.*` pattern
2. Builds and publishes release binaries

## 📊 Workflow Dependencies

```
Pull Request
├── PR Quick Check (fast)
│   └── Formatting
│   └── Build Check
│   └── Fast Tests
│
└── CI/CD Pipeline (comprehensive)
    ├── Unit Tests
    ├── Build Verification
    ├── Docker Tests
    ├── Multi-platform Build
    ├── Lint & Code Quality
    ├── Test Coverage
    └── CI Success (summary)

Main Branch Push
└── CI/CD Pipeline (comprehensive)

Version Tag (v*.*.*)
└── Release
    ├── Pre-release Tests
    ├── Build Release Binaries (6 platforms)
    ├── Create GitHub Release
    └── Update Homebrew (optional)
```

## 🔧 Local Testing

Before pushing, you can run the same tests locally:

### Quick Checks (same as PR Quick Check)
```bash
# Check formatting
gofmt -s -l .

# Run go vet
go vet ./...

# Build
go build -v ./...

# Fast tests
go test -short -v ./...
```

### Full CI Pipeline (same as CI/CD Pipeline)
```bash
# Unit tests
make test

# Internal tests
make test-internal

# CVE tests
make test-cve

# Docker tests
make test-docker-pr

# Build
make build

# Test coverage
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

### Comprehensive Test Suite
```bash
# Run all tests (internal + docker + quick + extensive)
make test-all-comprehensive

# Or quick test suite
make test-all-quick
```

## 📦 Creating a Release

To create a new release:

1. **Ensure main branch is stable**:
   ```bash
   # All CI checks should pass on main
   ```

2. **Create and push a version tag**:
   ```bash
   git tag -a v0.3.0 -m "Release v0.3.0"
   git push origin v0.3.0
   ```

3. **GitHub Actions will automatically**:
   - Run pre-release tests
   - Build binaries for all platforms
   - Create GitHub Release
   - Upload binaries and checksums

4. **Verify the release**:
   - Check GitHub Releases page
   - Download and test a binary
   - Verify checksums

## ⚙️ Configuration

### Go Version
All workflows use Go version defined in `env.GO_VERSION`. Currently: **1.25.1**

To update Go version, modify the `GO_VERSION` environment variable in all workflow files.

### Caching
Workflows use GitHub Actions cache for:
- Go build cache (`~/.cache/go-build`)
- Go modules (`~/go/pkg/mod`)

This speeds up subsequent runs by ~2-3 minutes.

### Required Secrets
No secrets are required for basic CI/CD. The workflows use:
- `GITHUB_TOKEN` (automatically provided by GitHub Actions)

## 🛠️ Troubleshooting

### Workflow Fails on Go Version
- Ensure `go.mod` specifies the correct Go version
- Update `GO_VERSION` in workflow files if needed

### Docker Tests Fail
- Check Docker Compose configuration in `docker-compose.test.yml`
- Verify Docker service is available in GitHub Actions

### Release Build Fails
- Ensure tag follows semantic versioning: `v0.0.0`
- Check that all tests pass before tagging

### Lint Failures
- Run `go fmt ./...` locally
- Run `go vet ./...` to catch issues early
- Install and run `staticcheck ./...` for additional checks

## 📈 Monitoring

### Check Workflow Status
- Visit: `https://github.com/<your-org>/vectra-guard/actions`
- View individual workflow runs
- Download artifacts (coverage reports, binaries)

### Badge for README
Add this badge to your README.md:

```markdown
![CI Status](https://github.com/<your-org>/vectra-guard/workflows/CI%2FCD%20Pipeline/badge.svg)
```

## 🤝 Contributing

When contributing:

1. **Before opening PR**: Run quick checks locally
2. **PR opened**: Quick Check runs automatically
3. **PR updated**: All checks re-run
4. **PR approved**: CI/CD Pipeline must pass
5. **Merged to main**: CI/CD runs again to validate main branch

## 📝 Notes

- **PR Quick Check** provides fast feedback (~2-5 min)
- **CI/CD Pipeline** is comprehensive (~10-20 min)
- Both must pass for PR to be mergeable
- Release workflow is automatic on tag push
- All binaries are cross-compiled with CGO disabled

## 🔗 Related Files

- Makefile: Test targets used by workflows
- `docker-compose.test.yml`: Docker test configuration
- `go.mod`: Go version and dependencies
- `scripts/`: Test scripts referenced by Makefile

---

**Last Updated**: February 2026
**Go Version**: 1.25.1
**Supported Platforms**: Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64, arm64)
