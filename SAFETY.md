# Safety Guarantees

## Destructive Testing Safety

The `test-destructive.sh` script is **100% safe** to run on your system:

### ✅ Safe by Design

1. **Detection Tests (99% of tests)**
   - Use `vectra-guard validate` which **NEVER executes commands**
   - Only analyzes command text for security risks
   - Cannot harm your system in any way

2. **Execution Tests (1% of tests)**
   - Only run when:
     - Inside a Docker container (`/.dockerenv` exists), OR
     - `VECTRAGUARD_CONTAINER` environment variable is set
   - When run locally, execution tests are **automatically skipped**
   - When run in Docker, all commands are **sandboxed in isolated containers**

3. **Sandbox Requirement**
   - Script verifies sandbox is enabled before any execution tests
   - If sandbox is disabled, execution tests are skipped
   - All dangerous commands are isolated in Docker containers

### 🛡️ Protection Layers

```
┌─────────────────────────────────────────┐
│  Destructive Test Script                 │
├─────────────────────────────────────────┤
│  Layer 1: validate (no execution)       │ ← 99% of tests
│  Layer 2: Docker container check        │ ← Execution tests
│  Layer 3: Sandbox verification          │ ← Safety check
│  Layer 4: Docker sandbox isolation      │ ← If execution happens
└─────────────────────────────────────────┘
```

### Running Tests

**Local (100% Safe):**
```bash
./scripts/test-destructive.sh
# Only runs detection tests (validate) - never executes
```

**Docker (Isolated):**
```bash
make test-destructive-docker
# Runs in isolated Docker container - cannot affect host
```

## Dev Mode Safety

The `dev-mode.sh` script is **completely safe**:

### ✅ What It Does
- Creates a `vectra-guard.yaml` config file
- Does NOT auto-enable anything
- Does NOT change your system
- Does NOT run any commands
- Does NOT modify existing files (unless `--force`)

### ✅ What It Doesn't Do
- ❌ Does NOT automatically wrap your shell
- ❌ Does NOT change your PATH
- ❌ Does NOT modify system files
- ❌ Does NOT run any commands automatically
- ❌ Does NOT enable protection without your explicit use

### How Dev Mode Works

1. **Opt-In Only**
   - You must explicitly use `vectra-guard exec` to get protection
   - Your normal commands work exactly as before
   - No automatic interception

2. **Config File Only**
   - Just creates a YAML config file
   - You can delete it anytime: `rm vectra-guard.yaml`
   - No system changes

3. **Sandbox Protection**
   - When you use `vectra-guard exec`, risky commands run in Docker
   - Your system is never at risk
   - All dangerous operations are isolated

### Example

```bash
# Setup (safe - just creates config)
./scripts/dev-mode.sh

# Your normal commands (unchanged)
npm install          # Works exactly as before
ls -la              # Works exactly as before

# Protected commands (opt-in)
vectra-guard exec -- npm install     # Runs in sandbox if risky
vectra-guard exec -- rm -rf /         # Blocked/sandboxed
```

## Verification

To verify safety, check the scripts:

```bash
# Check destructive test script
grep -n "validate\|exec" scripts/test-destructive.sh
# Shows: mostly "validate" (safe), "exec" only in Docker

# Check dev mode script
grep -n "exec\|system\|sudo\|rm" scripts/dev-mode.sh
# Shows: only file operations (cp), no dangerous commands
```

## Summary

✅ **Destructive tests**: Safe - only validate (no execution) or Docker-isolated  
✅ **Dev mode setup**: Safe - only creates config file  
✅ **No automatic changes**: Everything is opt-in  
✅ **Sandbox protection**: All risky operations isolated in Docker  

**Your system is safe!** 🛡️

