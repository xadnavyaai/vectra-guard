# Security Testing Guide

## Overview

This document describes the comprehensive test suite for verifying security improvements that prevent destructive command mishandling.

> **Quick Start:** For local testing without Docker, see [Local Testing](#local-testing) section below. For Dockerized testing, see [DOCKER_TESTING.md](DOCKER_TESTING.md).

## Test Structure

### 1. Unit Tests (Go)

Located in test files alongside implementation:
- `internal/analyzer/analyzer_test.go` - Pattern detection tests
- `internal/sandbox/sandbox_test.go` - Sandboxing logic tests
- `cmd/exec_test.go` - Execution flow tests

### 2. Integration Test Script

Located at: `scripts/test-security.sh`

A bash script that runs incremental tests to verify security improvements.

## Running Tests

### Run All Go Tests

```bash
go test ./...
```

### Run Specific Test Suites

```bash
# Test enhanced destructive command detection
go test ./internal/analyzer/... -run TestEnhancedDestructiveCommandDetection

# Test mandatory sandboxing
go test ./internal/sandbox/... -run TestMandatorySandboxingForCriticalCommands

# Test pre-execution assessment
go test ./cmd/... -run TestPreExecutionAssessment

# Test regression (incident scenario)
go test ./cmd/... -run TestSecurityImprovementsRegression
```

### Run Security Test Script

```bash
# Full test suite
./scripts/test-security.sh

# Quick mode (only critical tests)
./scripts/test-security.sh --quick

# Verbose output
./scripts/test-security.sh --verbose
```

## Test Coverage

### 1. Enhanced Destructive Command Detection

**Tests:** `TestEnhancedDestructiveCommandDetection`, `TestHomeDirectoryDeletionDetection`

**What it tests:**
- Detection of `rm -r /*` (the incident scenario)
- Detection of `rm -rf /*`
- Detection of `rm -r /`
- Detection of system directory targets (`/bin`, `/usr`, `/etc`, etc.)
- Detection of home directory wildcards (`~/*`, `$HOME/*`)
- Safe operations should NOT trigger (relative paths, specific files)

**Key Assertions:**
- ✅ All variations of destructive root deletion are detected
- ✅ System directory targets are detected
- ✅ Safe operations are not flagged
- ✅ Severity is correctly set to "critical"

### 2. Mandatory Sandboxing

**Tests:** `TestMandatorySandboxingForCriticalCommands`

**What it tests:**
- Critical commands cannot be bypassed by trust store
- Critical commands cannot be bypassed by allowlist
- Critical commands cannot be bypassed by "never" mode
- Critical commands still attempt sandbox even if disabled
- Non-critical commands can still use allowlist

**Key Assertions:**
- ✅ Critical commands ALWAYS sandbox (cannot bypass)
- ✅ Trust store does NOT bypass critical commands
- ✅ Allowlist does NOT bypass critical commands
- ✅ Configuration does NOT bypass critical commands

### 3. Pre-Execution Assessment

**Tests:** `TestPreExecutionAssessment`

**What it tests:**
- Critical commands are blocked if sandbox unavailable
- Critical commands proceed if sandbox available
- Non-critical commands can proceed without sandbox
- Fork bombs are blocked if sandbox unavailable

**Key Assertions:**
- ✅ Critical commands blocked when sandbox disabled
- ✅ No fallback to host execution for critical commands
- ✅ Proper error messages for blocked commands

### 4. Regression Tests

**Tests:** `TestSecurityImprovementsRegression`

**What it tests:**
- The incident scenario (`rm -r /*`) is now detected
- All variations are detected
- Proper severity levels assigned

**Key Assertions:**
- ✅ Incident command is detected
- ✅ All variations are detected
- ✅ No regressions introduced

## Test Scenarios

### Scenario 1: The Incident (rm -r /*)

**Command:** `rm -r /*`

**Expected Behavior:**
1. ✅ Detected by analyzer as `DANGEROUS_DELETE_ROOT` (critical)
2. ✅ Pre-execution assessment blocks if sandbox unavailable
3. ✅ Mandatory sandboxing enforced (cannot bypass)
4. ✅ Runs in isolated container

**Test:** `TestSecurityImprovementsRegression`

### Scenario 2: System Directory Deletion

**Command:** `rm -rf /bin`

**Expected Behavior:**
1. ✅ Detected as `DANGEROUS_DELETE_ROOT` (critical)
2. ✅ Mandatory sandboxing enforced
3. ✅ Cannot bypass with trust store

**Test:** `TestEnhancedDestructiveCommandDetection`

### Scenario 3: Trust Store Bypass Attempt

**Command:** `rm -r /*` (previously trusted)

**Expected Behavior:**
1. ✅ Detected as critical
2. ✅ Trust store entry ignored
3. ✅ Still requires mandatory sandboxing

**Test:** `TestMandatorySandboxingForCriticalCommands`

### Scenario 4: Sandbox Disabled

**Command:** `rm -r /*` (sandbox disabled in config)

**Expected Behavior:**
1. ✅ Detected as critical
2. ✅ Pre-execution assessment blocks execution
3. ✅ Error: "Sandbox required for critical commands"

**Test:** `TestPreExecutionAssessment`

## Continuous Testing

### Pre-Commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
go test ./internal/analyzer/... -run TestEnhancedDestructiveCommandDetection
go test ./internal/sandbox/... -run TestMandatorySandboxingForCriticalCommands
go test ./cmd/... -run TestSecurityImprovementsRegression
```

### CI/CD Integration

```yaml
test:
  script:
    - go test -v ./...
    - ./scripts/test-security.sh --quick
```

## Test Maintenance

### Adding New Tests

1. **For new patterns:** Add to `TestEnhancedDestructiveCommandDetection`
2. **For new bypass attempts:** Add to `TestMandatorySandboxingForCriticalCommands`
3. **For new assessment logic:** Add to `TestPreExecutionAssessment`

### Updating Tests

When security improvements are made:
1. Update existing tests to reflect new behavior
2. Add regression tests for fixed scenarios
3. Update test documentation

## Troubleshooting

### Tests Failing

1. **Check Go version:** Requires Go 1.19+
2. **Check dependencies:** Run `go mod tidy`
3. **Check test output:** Run with `-v` flag for verbose output

### Script Not Running

1. **Check permissions:** `chmod +x scripts/test-security.sh`
2. **Check binary:** Ensure `vectra-guard` is built
3. **Check paths:** Script should be run from project root

## Test Results Interpretation

### All Tests Pass ✅

- Security improvements are working correctly
- Incident scenario is prevented
- No regressions detected

### Some Tests Fail ❌

- Review failure messages
- Check if new patterns need to be added
- Verify configuration is correct

### Regression Detected 🚨

- **CRITICAL:** Incident scenario is not prevented
- Review recent changes
- Fix immediately before deployment

## Best Practices

1. **Run tests before committing** - Catch issues early
2. **Run full suite before release** - Ensure all improvements work
3. **Add tests for new patterns** - Maintain coverage
4. **Document test scenarios** - Help others understand

---

## Local Testing

### Prerequisites

1. **Go installed** (1.25+)
   ```bash
   go version
   ```

2. **Docker installed** (optional, for sandbox testing)
   ```bash
   docker --version
   ```

### Build the Binary

```bash
# Build the binary
go build -o vectra-guard .

# Or build with version (recommended for releases)
go build -ldflags "-X github.com/vectra-guard/vectra-guard/cmd.Version=v0.0.2" -o vectra-guard .

# Verify it built
ls -lh vectra-guard

# Check version
./vectra-guard version
```

### Test the Incident Scenario

```bash
# Test that rm -r /* is now detected (the incident scenario)
echo "rm -r /*" | ./vectra-guard validate /dev/stdin
```

**Expected:** Should show `DANGEROUS_DELETE_ROOT` (critical) finding.

### Test Execution Blocking

```bash
# Create test config with sandbox disabled
cat > /tmp/test-config.yaml <<EOF
guard_level:
  level: medium
sandbox:
  enabled: false
EOF

# Try to execute critical command (should be BLOCKED)
./vectra-guard exec --config /tmp/test-config.yaml "rm -r /*" 2>&1
```

**Expected:** Should error saying sandbox is required for critical commands.

### Run Security Test Script

```bash
# Quick test
./scripts/test-security.sh --quick

# Full test suite
./scripts/test-security.sh
```

For Dockerized testing, see [DOCKER_TESTING.md](DOCKER_TESTING.md).

---

**Stay Safe. Test Thoroughly.** 🛡️🧪

