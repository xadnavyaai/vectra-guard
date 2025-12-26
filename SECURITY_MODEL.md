# Vectra Guard - Security Model for Dev & Prod Cycles

## Overview

This document outlines the security model for protecting development and production cycles from destructive commands, with a focus on **mandatory pre-execution assessment** and **true sandboxing** that cannot be bypassed.

## The Problem: What Happened

During development, a tool executed `rm -r /*` which:
- Was **not detected** by the analyzer (only checked for `rm -rf /`)
- **Bypassed sandboxing** (trust store or config allowed host execution)
- **Corrupted the entire OS** because it ran directly on the host

## Core Security Principles

### 1. **Mandatory Pre-Execution Assessment**

**Every command must be assessed BEFORE execution**, not after. The assessment includes:

- **Pattern Analysis**: Detects dangerous command patterns (not just exact matches)
- **Path Analysis**: Identifies operations targeting root, system directories, or user home
- **Risk Classification**: Categorizes commands as low/medium/high/critical
- **Permission Check**: Determines if command requires sandboxing or approval

**Key Rule**: If assessment fails or is inconclusive, **default to BLOCK**.

### 2. **True Sandboxing (Cannot Be Bypassed)**

For **critical** commands, sandboxing is **mandatory** and **cannot be bypassed** by:
- Trust store entries
- Allowlist patterns
- Configuration settings
- User bypass mechanisms

**Critical commands that require mandatory sandboxing:**
- `DANGEROUS_DELETE_ROOT` - Any rm command targeting root or system directories
- `DANGEROUS_DELETE_HOME` - Recursive delete of home directory
- `FORK_BOMB` - Commands that can crash the system
- `SENSITIVE_ENV_ACCESS` - Access to credentials/secrets
- `DOTENV_FILE_READ` - Reading .env files

### 3. **Enhanced Pattern Detection**

The analyzer now detects **all variations** of destructive commands:

```bash
# All of these are now detected:
rm -rf /          # Original pattern
rm -r /           # Without force flag
rm -rf /*         # With wildcard
rm -r /*          # Without force, with wildcard
rm -rf / *        # Space between / and *
rm -rf /bin       # System directories
rm -rf /usr       # System directories
# ... and many more variations
```

### 4. **Defense in Depth**

Multiple layers of protection:
1. **Pattern Detection** - Catches dangerous patterns
2. **Path Analysis** - Identifies risky file operations
3. **Risk Classification** - Assigns severity levels
4. **Mandatory Sandboxing** - Enforces isolation for critical commands
5. **Pre-Execution Block** - Prevents execution if sandbox unavailable

---

## Development Cycle Security Model

### Goals
- **Productivity**: Don't block legitimate development work
- **Safety**: Prevent catastrophic mistakes
- **Learning**: Allow developers to understand risks

### Configuration

```yaml
guard_level:
  level: medium  # Blocks critical + high severity
  allow_user_bypass: true  # Developers can bypass with proper auth
  require_approval_above: medium

policies:
  monitor_git_ops: true
  block_force_git: false  # Allow in dev (but warn)
  detect_prod_env: false  # Not needed in dev
  only_destructive_sql: true
  
  allowlist:
    - "echo *"
    - "ls *"
    - "cat *"
    - "git status"
    - "git diff"
    - "npm install"
    - "npm test"
    - "npm run build"
  
  denylist:
    - "rm -rf /"
    - "rm -r /*"
    - "rm -rf /*"
    - "sudo rm"
    - ":(){ :|:& };:"

sandbox:
  enabled: true
  mode: auto  # Auto-sandbox medium+ risk
  security_level: balanced  # Good balance of security and usability
  runtime: docker
  image: ubuntu:22.04
  network_mode: restricted  # Allow network but restrict egress
```

### Dev Cycle Behavior

1. **Low Risk Commands** (e.g., `ls`, `cat`, `git status`)
   - ✅ Run on host
   - ✅ No approval needed
   - ✅ Fast execution

2. **Medium Risk Commands** (e.g., `npm install`, `git push`)
   - ⚠️ Run in sandbox
   - ⚠️ May require approval if interactive
   - ✅ Cached for performance

3. **High Risk Commands** (e.g., `git push --force`, destructive SQL)
   - 🔒 **MANDATORY sandbox**
   - 🔒 **REQUIRES approval** (interactive mode)
   - 🔒 Cannot bypass with trust store

4. **Critical Commands** (e.g., `rm -r /*`, `rm -rf /`)
   - 🚫 **BLOCKED** - Cannot execute
   - 🚫 **MANDATORY sandbox** if somehow approved
   - 🚫 **Cannot bypass** - Even with user bypass
   - 🚫 **Sandbox must be available** - No fallback to host

### Dev Cycle Best Practices

1. **Start with `medium` guard level** - Good balance
2. **Enable sandboxing** - Even in dev, it prevents accidents
3. **Use interactive mode** for risky operations
4. **Review findings** - Understand why commands are flagged
5. **Test in sandbox first** - Verify commands work before trusting

---

## Production Cycle Security Model

### Goals
- **Maximum Protection**: Prevent any destructive operations
- **Zero Tolerance**: No bypasses, no exceptions
- **Audit Trail**: Log everything for compliance

### Configuration

```yaml
guard_level:
  level: paranoid  # Everything requires approval
  allow_user_bypass: false  # NO bypasses in production
  require_approval_above: low  # Even low-risk needs approval

policies:
  monitor_git_ops: true
  block_force_git: true  # BLOCK force push in prod
  detect_prod_env: true  # Detect production indicators
  only_destructive_sql: true
  
  prod_env_patterns:
    - prod
    - production
    - prd
    - live
    - staging
    - stg
  
  allowlist:
    - "git status"
    - "git diff"
    - "git log"
    - "ls *"
    - "cat *"
    # Minimal allowlist - only read-only operations
  
  denylist:
    - "rm *"
    - "sudo *"
    - "dd if="
    - "mkfs"
    - "git push --force"
    - "DROP DATABASE"
    - "TRUNCATE"
    - "DELETE FROM"

sandbox:
  enabled: true
  mode: always  # ALWAYS sandbox in production
  security_level: paranoid  # Maximum isolation
  runtime: docker
  image: ubuntu:22.04
  network_mode: none  # No network access
  read_only_root: true  # Read-only filesystem
  seccomp_profile: seccomp-profile.json  # Strict syscall filtering
```

### Production Cycle Behavior

1. **ALL Commands**
   - 🔒 **MANDATORY sandbox** - No exceptions
   - 🔒 **REQUIRES approval** - Even low-risk commands
   - 🔒 **No network access** - Isolated from internet
   - 🔒 **Read-only root** - Cannot modify system

2. **Critical Commands**
   - 🚫 **BLOCKED** - Cannot execute at all
   - 🚫 **No approval possible** - Even interactive mode won't help
   - 🚫 **Logged and alerted** - Security team notified

3. **High Risk Commands**
   - 🔒 **MANDATORY sandbox**
   - 🔒 **REQUIRES multiple approvals** - May need team sign-off
   - 🔒 **Audit logged** - All attempts recorded

4. **Production Environment Detection**
   - 🔒 Commands targeting production are **elevated to critical**
   - 🔒 Force push to production = **BLOCKED**
   - 🔒 Destructive SQL in production = **BLOCKED**

### Production Cycle Best Practices

1. **Use `paranoid` guard level** - Maximum protection
2. **Disable all bypasses** - No exceptions
3. **Always sandbox** - Even "safe" commands
4. **Enable audit logging** - Track all operations
5. **Review logs regularly** - Monitor for suspicious activity
6. **Test changes in dev first** - Never test in production
7. **Use staging environment** - Validate before production

---

## Security Layers Explained

### Layer 1: Pattern Detection (Analyzer)

**What it does:**
- Scans command strings for dangerous patterns
- Detects variations (not just exact matches)
- Identifies risky operations (rm, sudo, etc.)

**When it runs:**
- BEFORE command execution
- On every command
- Cannot be skipped

**Example:**
```bash
Command: rm -r /*
Detection: DANGEROUS_DELETE_ROOT (critical)
Action: BLOCKED
```

### Layer 2: Path Analysis

**What it does:**
- Analyzes file paths in commands
- Identifies operations on root, system dirs, or home
- Checks for wildcard usage in dangerous contexts

**When it runs:**
- As part of pattern detection
- Before execution

**Example:**
```bash
Command: rm -rf /usr/bin/*
Detection: System directory deletion (critical)
Action: BLOCKED
```

### Layer 3: Risk Classification

**What it does:**
- Assigns severity levels (low/medium/high/critical)
- Combines multiple findings
- Determines overall risk

**When it runs:**
- After pattern detection
- Before execution decision

**Example:**
```bash
Findings:
  - DANGEROUS_DELETE_ROOT (critical)
  - PRODUCTION_ENVIRONMENT (high)
Risk Level: CRITICAL
Action: MANDATORY SANDBOX + BLOCK
```

### Layer 4: Mandatory Sandboxing

**What it does:**
- Enforces sandbox execution for critical commands
- Cannot be bypassed by trust store or config
- Requires sandbox to be available

**When it runs:**
- For critical commands
- Before execution

**Example:**
```bash
Command: rm -r /*
Risk: CRITICAL
Sandbox: MANDATORY (cannot bypass)
Result: Runs in isolated container
```

### Layer 5: Pre-Execution Block

**What it does:**
- Prevents execution if sandbox unavailable
- Blocks critical commands if sandbox disabled
- No fallback to host execution

**When it runs:**
- Before execution
- For critical commands only

**Example:**
```bash
Command: rm -r /*
Sandbox: Disabled in config
Action: BLOCKED - Cannot execute without sandbox
```

---

## Comparison: Dev vs Prod

| Feature | Development | Production |
|---------|------------|------------|
| **Guard Level** | `medium` | `paranoid` |
| **User Bypass** | ✅ Allowed | ❌ Disabled |
| **Sandbox Mode** | `auto` | `always` |
| **Security Level** | `balanced` | `paranoid` |
| **Network Access** | `restricted` | `none` |
| **Approval Required** | Medium+ risk | All commands |
| **Trust Store** | ✅ Can bypass | ❌ Cannot bypass |
| **Allowlist** | ✅ Can bypass | ⚠️ Limited |
| **Critical Commands** | 🔒 Mandatory sandbox | 🚫 BLOCKED |
| **Audit Logging** | ⚠️ Optional | ✅ Required |

---

## Migration Guide

### From Current Setup to Secure Dev

1. **Update config:**
   ```yaml
   guard_level:
     level: medium
     allow_user_bypass: true
   
   sandbox:
     enabled: true
     mode: auto
     security_level: balanced
   ```

2. **Test commands:**
   ```bash
   vectra-guard exec rm -r /*  # Should be BLOCKED
   vectra-guard exec npm install  # Should sandbox
   ```

3. **Review findings:**
   - Check what's being flagged
   - Adjust allowlist if needed
   - Understand risk levels

### From Dev to Production

1. **Update config:**
   ```yaml
   guard_level:
     level: paranoid
     allow_user_bypass: false
   
   sandbox:
     mode: always
     security_level: paranoid
     network_mode: none
   ```

2. **Test in staging first:**
   - Validate all commands work in sandbox
   - Check performance impact
   - Review audit logs

3. **Deploy to production:**
   - Monitor for issues
   - Review logs regularly
   - Adjust as needed

---

## Troubleshooting

### Command Being Blocked Unnecessarily

**Dev:**
- Lower guard level to `low`
- Add to allowlist (with review)
- Use bypass for one-time operations

**Prod:**
- Review why it's blocked
- Consider if it's truly safe
- May need to adjust pattern detection

### Sandbox Not Available

**Dev:**
- Install Docker/Podman
- Check sandbox config
- Fallback to host (for non-critical)

**Prod:**
- **CRITICAL commands cannot execute without sandbox**
- Must fix sandbox before proceeding
- No fallback allowed

### Performance Issues

**Dev:**
- Enable caching
- Use `balanced` security level
- Allowlist frequently used safe commands

**Prod:**
- Accept performance cost for security
- Use optimized sandbox images
- Consider dedicated sandbox infrastructure

---

## Best Practices Summary

### Development
1. ✅ Use `medium` guard level
2. ✅ Enable sandboxing with `auto` mode
3. ✅ Allow user bypass for emergencies
4. ✅ Review findings to learn
5. ✅ Test commands in sandbox first

### Production
1. ✅ Use `paranoid` guard level
2. ✅ Disable all bypasses
3. ✅ Always sandbox (`always` mode)
4. ✅ Enable audit logging
5. ✅ Review logs regularly
6. ✅ Test in staging first

---

## Security Guarantees

### What We Guarantee

1. **Critical commands** (`rm -r /*`, etc.) are **ALWAYS blocked** or **MANDATORY sandboxed**
2. **No bypass possible** for critical commands (even with user bypass)
3. **Sandbox required** for critical commands (no fallback to host)
4. **Enhanced detection** catches all variations of destructive commands
5. **Pre-execution assessment** happens before any command runs

### What We Don't Guarantee

1. **100% detection** - New attack patterns may not be caught
2. **Sandbox escape prevention** - Advanced attackers may escape
3. **Performance** - Security comes with overhead
4. **False positive elimination** - Some safe commands may be flagged

---

**Stay Safe. Code Fearlessly.** 🛡️

