# Vectra Guard Configuration Guide

This document provides detailed configuration options and preset examples for Vectra Guard.

## Table of Contents

- [Quick Start Presets](#quick-start-presets)
- [How Protection Levels Work](#how-protection-levels-work)
- [Configuration File Locations](#configuration-file-locations)
- [Basic Configuration](#basic-configuration)
- [Guard Level (Auto-Detection)](#guard-level-auto-detection)
- [Production Indicators](#production-indicators)
- [Security Policies](#security-policies)
- [Sandbox Configuration](#sandbox-configuration)
- [Advanced Configuration](#advanced-configuration)

---

## Quick Start Presets

For most users, copying one of these presets into your `vectra-guard.yaml` is all you need.

### 1. Developer Preset (Recommended) 👩‍💻
*Best for local development. Fast, unobtrusive, but safe.*

```yaml
guard_level:
  level: auto             # Auto-detect context (low in dev, high in prod)
  allow_user_bypass: true # Allow overriding with env var

sandbox:
  enabled: true
  mode: auto              # Only sandbox risky commands (like npm install)
  security_level: balanced # Good isolation, allows outbound network
  enable_cache: true      # 10x faster subsequent runs
```

### 2. CI/CD Pipeline 🤖
*For automated testing and builds.*

```yaml
guard_level:
  level: high             # Block critical/high risks without approval

sandbox:
  enabled: true
  mode: always            # Run EVERYTHING in sandbox for isolation
  security_level: strict  # Stricter isolation
  enable_cache: true      # Speed up builds
```

### 3. Production / Zero Trust 🔒
*Maximum security for production environments.*

```yaml
guard_level:
  level: paranoid         # Require explicit approval for everything

sandbox:
  enabled: true
  mode: always
  security_level: paranoid # No network, read-only root, minimal caps
  network_mode: none       # Completely offline
```

---

## How Protection Levels Work

Vectra Guard has three main "knobs" to control security. Understanding how they interact helps you tune it perfectly.

### 1. Guard Level (`guard_level.level`)
**Controls "What do we ask the user about?"**
- `low`: Only block Critical risks.
- `medium`: Block Critical + High risks.
- `high`: Block Critical + High + Medium risks.
- `paranoid`: Everything requires approval.
- `auto`: Starts at `medium`, but bumps to `paranoid` if it detects you are on a `production` branch or handling sensitive data.

### 2. Sandbox Mode (`sandbox.mode`)
**Controls "Where does the command run?"**
- `auto`: Safe commands -> Host. Risky commands -> Sandbox.
- `always`: Everything -> Sandbox.
- `risky`: Only High/Critical risk -> Sandbox.

### 3. Sandbox Security Level (`sandbox.security_level`)
**Controls "How locked down is the sandbox?"**
- `permissive`: Shares host network, some capabilities. (Fastest)
- `balanced`: Own network namespace (outbound allowed), standard caps. (Default)
- `strict`: Restricted network, dropped caps.
- `paranoid`: No network, read-only filesystem, no caps.

**Example Scenario:**
If you run `npm install`:
- `guard_level: auto` sees "npm install" is medium risk -> Approves execution.
- `sandbox.mode: auto` sees "npm install" modifies files/network -> Sends to Sandbox.
- `sandbox.security_level: balanced` -> Sandbox allows downloading packages.

---

## Configuration File Locations

Vectra Guard looks for configuration in the following order (last one wins):

1. **User config**: `~/.config/vectra-guard/config.yaml`
2. **Project config**: `./vectra-guard.yaml` (in project root)
3. **Explicit path**: Via `--config` flag

```bash
# Use project config
vectra-guard exec npm test

# Use specific config file
vectra-guard exec --config /path/to/config.yaml npm test
```

---

## Basic Configuration

The minimal configuration file:

```yaml
# vectra-guard.yaml

guard_level:
  level: auto  # Recommended: auto-detect context

logging:
  format: json  # or: text

policies:
  monitor_git_ops: true
  detect_prod_env: true
```

That's it! Everything else uses smart defaults.

---

## Guard Level (Auto-Detection)

### What is Auto-Detection?

When `level: auto`, Vectra Guard intelligently analyzes:

1. **Git branch** (main/master → paranoid, feature/* → low)
2. **Command content** (deploy commands → high)
3. **URLs and hostnames** (api.prod.company.com → high)
4. **Database names** (prod_db → high)
5. **File paths** (/var/www/production → high)
6. **Environment variables** (ENV=production → high)

**Priority Rule**: When multiple signals conflict, **the most dangerous context wins** (safety first!).

### Guard Levels Explained

| Level | Behavior | Use Case |
|-------|----------|----------|
| `auto` | **Smart detection** (recommended) | Automatically adjusts based on context |
| `low` | Only critical issues blocked | Local development, trusted environments |
| `medium` | Critical + high issues blocked | Team collaboration, general use |
| `high` | Critical + high + medium blocked | Staging environments, careful operations |
| `paranoid` | Everything requires approval | Production, untrusted code, maximum safety |
| `off` | No protection | Testing only (not recommended) |

### Runtime Override

```bash
# Temporarily lower protection
VECTRA_GUARD_LEVEL=low vg exec "risky command"

# Force paranoid mode
VECTRA_GUARD_LEVEL=paranoid vg exec "production deploy"
```

---

## Production Indicators

Teach Vectra Guard your organization's patterns:

```yaml
production_indicators:
  # Git branches that indicate production
  branches:
    - main
    - master
    - production
    - release
    - live
    
  # Keywords in URLs, hostnames, database names, paths
  keywords:
    - prod
    - production
    - prd
    - live
    - staging
    - stg
    - uat
    - preprod
```

### Detection Examples

When auto-detection encounters these patterns:

```bash
# Git branch: main → PARANOID
git checkout main
vg exec npm run deploy  # Requires approval

# Command with "prod" → HIGH
vg exec "kubectl apply -f prod-config.yaml"  # Warning + increased scrutiny

# URL with "production" → HIGH
vg exec "curl https://api.production.company.com/deploy"  # Warning

# Database name with "prod" → HIGH
vg exec "psql prod_database -c 'SELECT * FROM users'"  # Warning
```

---

## Security Policies

### Git Operations

```yaml
policies:
  # Monitor risky git operations
  monitor_git_ops: true
  
  # Block force push (git push --force)
  block_force_git: true
```

**What gets detected:**
- `git push --force` / `git push -f` (blocked if `block_force_git: true`)
- `git reset --hard` (warning)
- `git clean -fd` (warning)
- `git filter-branch` (critical warning)

### Production Environment Detection

```yaml
policies:
  # Detect production/staging operations
  detect_prod_env: true
  
  # SQL detection mode
  only_destructive_sql: true  # Only flag DROP/DELETE/TRUNCATE
```

### Allow/Deny Lists

```yaml
policies:
  # Commands always allowed (safe operations)
  allowlist:
    - "echo *"
    - "ls *"
    - "cat *"
    - "git status"
    - "git log"
    - "npm install"
    - "npm test"
  
  # Commands blocked or requiring approval
  denylist:
    - "rm -rf /"
    - "sudo rm"
    - ":(){ :|:& };:"  # fork bomb
    - "curl * | sh"
    - "wget * | bash"
    - "DROP DATABASE"
```

**Wildcard matching**: Use `*` for flexible patterns.

---

## Advanced Configuration

### Environment Variable Protection

```yaml
env_protection:
  enabled: true
  
  # Masking mode: full, partial, hash, fake
  masking_mode: partial
  
  # Additional sensitive variables
  protected_vars:
    - MY_SECRET_KEY
    - COMPANY_API_KEY
  
  # Variables allowed to read
  allow_read_vars:
    - HOME
    - USER
    - PATH
  
  # Custom fake values for testing
  fake_values:
    DATABASE_URL: "postgresql://user:pass@localhost:5432/dev_db"
    API_KEY: "dev_key_1234567890abcdef"
  
  # Block environment access commands
  block_env_access: false
  
  # Block .env file reading
  block_dotenv_read: true
```

### Approval Thresholds

```yaml
guard_level:
  level: medium
  require_approval_above: medium  # Require approval for medium+ severity
```

---

## Environment Variable Overrides

Override any configuration at runtime:

```bash
# Override guard level
VECTRA_GUARD_LEVEL=low vg exec "command"

# Bypass protection (if allowed)
export VECTRAGUARD_BYPASS="i-am-human-$(whoami)"
vg exec "command"

# Combine with other tools
VECTRA_GUARD_LEVEL=paranoid docker-compose run vectra-guard
```

---

## Best Practices

### 1. **Use Auto-Detection**

```yaml
guard_level:
  level: auto  # Let Vectra Guard be smart
```

### 2. **Commit Config to Git**

Share protection across your team:

```bash
git add vectra-guard.yaml
git commit -m "Add Vectra Guard security policies"
```

### 3. **Layer Your Protection**

- **Project config**: General team policies
- **User config**: Personal preferences
- **Environment override**: Situation-specific adjustments

```bash
# Team config in project
cat vectra-guard.yaml
guard_level:
  level: auto

# Personal override when needed
VECTRA_GUARD_LEVEL=low vg exec "trusted script"
```

### 4. **Test Your Configuration**

```bash
# Dry run to see what would be detected
vg validate script.sh

# Explain detected risks
vg explain script.sh

# Test execution with logging
vg exec --interactive "risky command"
```

### 5. **Review Sessions Regularly**

```bash
# See what your agents have been doing
vg session list

# Review specific session
vg session show $SESSION_ID
```

---

## Troubleshooting

### "Too many false positives"

**Solution**: Lower the guard level or add to allowlist

```yaml
guard_level:
  level: low  # or medium

policies:
  allowlist:
    - "your safe command *"
```

### "Not detecting production environment"

**Solution**: Add your org's patterns

```yaml
production_indicators:
  keywords:
    - your-prod-pattern
    - your-env-name
```

### "Need temporary bypass"

**Solution**: Use environment variable override

```bash
VECTRA_GUARD_LEVEL=low vg exec "command"
```

---

## Getting Help

- **View current config**: `vg init --show-config`
- **Validate config**: `vg validate vectra-guard.yaml`
- **Documentation**: [README.md](README.md)
- **Issues**: [GitHub Issues](https://github.com/xadnavyaai/vectra-guard/issues)

---

**Remember**: Vectra Guard's `auto` level is designed to be smart. Start there, then customize as needed! 🛡️
