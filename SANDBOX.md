# Vectra Guard Sandbox Execution

**Enterprise-grade sandboxing for AI coding agents**

Vectra Guard's sandbox system provides transparent, fast, and secure isolation for risky commands. It intelligently decides when to sandbox based on risk level and context, with minimal friction for developers.

---

## 🎯 Core Capabilities

### Phase 1: Core Execution Engine
**Policy-driven executor chooses between host and sandbox execution**

```yaml
sandbox:
  enabled: true
  mode: auto  # auto, always, risky, never
```

**Decision Logic:**
- ✅ Low-risk commands → host execution
- ⚠️ Medium/high-risk → sandbox execution
- 🔒 Networked installs → automatic sandboxing
- 📝 Trusted commands → remembered and run on host

### Phase 2: Sandbox Runtime & Isolation
**Multiple runtime options with transparent execution**

```yaml
sandbox:
  runtime: docker  # docker, podman, process
  image: ubuntu:22.04
  timeout: 300
```

**Supported Runtimes:**
- **Docker** - Most compatible, works everywhere
- **Podman** - Rootless alternative, same CLI
- **Process** - Linux namespaces (fastest, Linux-only)

### Phase 3: Cache Strategy
**Reuse dependency caches across sandbox runs**

```yaml
sandbox:
  enable_cache: true
  cache_dirs:
    - ~/.npm
    - ~/.cargo
    - ~/go/pkg
```

**Automatic Cache Mounts:**
- Node.js: `~/.npm`, `~/.yarn`, `~/.pnpm`
- Python: `~/.cache/pip`
- Go: `~/go/pkg`
- Rust: `~/.cargo`
- Ruby: `~/.gem`

**Benefits:**
- 🚀 Fast subsequent installs
- 💾 Shared across sandbox instances
- 🔄 Persistent between runs

### Phase 4: Security Posture Controls
**Tune isolation without sacrificing speed**

```yaml
sandbox:
  security_level: balanced  # permissive, balanced, strict, paranoid
  network_mode: restricted   # none, restricted, full
```

**Security Levels:**

| Level | Network | Root FS | Memory | CPU | Use Case |
|-------|---------|---------|--------|-----|----------|
| **Permissive** | Full | R/W | 2GB | 2.0 | Development |
| **Balanced** | Restricted | R/W | 1GB | 1.0 | Default |
| **Strict** | Restricted | RO | 512MB | 0.5 | CI/CD |
| **Paranoid** | None | RO | 256MB | 0.25 | Production |

### Phase 5: Policy Learning & Trust
**"Approve and remember" reduces friction**

```bash
# Interactive approval with remember option
⚠️  Command requires approval
Command: npm install suspicious-package
Risk Level: MEDIUM

Options:
  y  - Yes, run once
  r  - Yes, and remember (trust permanently)
  n  - No, cancel

Choose [y/r/N]: r
✅ Approved and remembered
```

**Trust Management:**
```bash
# List trusted commands
vg trust list

# Add command to trust store
vg trust add "npm install express" --note "Common package"

# Remove trusted command
vg trust remove "npm install express"

# Clean expired entries
vg trust clean
```

### Phase 6: Developer Experience
**"Just works" mode with minimal friction**

```yaml
# Developer preset - copy to vectra-guard.yaml
guard_level:
  level: low

sandbox:
  enabled: true
  mode: auto
  security_level: permissive
  enable_cache: true
  network_mode: full
```

**Quick Start:**
```bash
# Use developer preset
cp presets/developer.yaml vectra-guard.yaml

# Install and run - no friction
npm install express
npm run dev
```

### Phase 7: Observability & Analytics
**Track sandbox usage and performance**

```bash
# View metrics
vg metrics show

# Output:
Vectra Guard Sandbox Metrics
===============================
Total Executions:    142
  - Host:            89 (62.7%)
  - Sandbox:         53 (37.3%)
  - Cached:          41 (28.9%)

Average Duration:    1.2s

By Risk Level:
  - low: 89 (62.7%)
  - medium: 42 (29.6%)
  - high: 11 (7.7%)
```

**Metrics Tracking:**
- Total/host/sandbox execution counts
- Cache hit rates
- Average execution duration
- Breakdown by risk level
- Breakdown by runtime
- Last 100 execution history

---

## 🎨 UX Enhancements

### Single-line Notices
```bash
📦 Running in sandbox (cached).
   Why: medium risk + networked install
```

### Explain the "Why"
Every sandbox decision includes clear reasoning:
- "Sandbox chosen: medium risk + networked install"
- "Running on host: low risk, no isolation needed"
- "Running on host: command previously approved and trusted"

### No Prompts Unless Needed
- Low-risk: silent execution
- Medium-risk: automatic sandboxing, no prompt
- High-risk: approval only if interactive
- Trusted: remembered approvals skip prompts

### Remembered Approvals
```bash
✅ Approved and remembered

# Next time - no prompt!
npm install express
# → runs directly, remembered from previous approval
```

### Consistent Output
Same format whether running on host or in sandbox:
```bash
# Host execution
$ vg exec "echo hello"
hello

# Sandbox execution
$ vg exec "npm install"
📦 Running in sandbox (cached).
   Why: medium risk + networked install
added 142 packages in 2.3s
```

---

## 📋 Configuration Reference

### Complete Sandbox Config

```yaml
sandbox:
  # Core settings
  enabled: true
  mode: auto              # auto, always, risky, never
  security_level: balanced # permissive, balanced, strict, paranoid
  
  # Runtime
  runtime: docker         # docker, podman, process
  image: ubuntu:22.04
  timeout: 300           # seconds
  
  # Caching
  enable_cache: true
  cache_dirs:
    - ~/.npm
    - ~/.cargo
    - ~/go/pkg
  
  # Network
  network_mode: restricted # none, restricted, full
  
  # Security
  seccomp_profile: /path/to/seccomp.json
  
  # Environment
  env_whitelist:
    - HOME
    - USER
    - PATH
    - SHELL
    - TERM
    - PWD
  
  # Custom mounts
  bind_mounts:
    - host_path: /path/on/host
      container_path: /path/in/container
      read_only: false
  
  # Observability
  enable_metrics: true
  log_output: false
  
  # Trust store
  trust_store_path: ~/.vectra-guard/trust.json
```

---

## 🎯 Presets

### Developer Preset
**Optimized for local development**

```bash
cp presets/developer.yaml vectra-guard.yaml
```

Features:
- Low guard level (minimal friction)
- Auto sandboxing for risky commands
- Permissive security (fast)
- Full network access
- Caching enabled

### CI/CD Preset
**Balanced for automated pipelines**

```bash
cp presets/ci-cd.yaml vectra-guard.yaml
```

Features:
- High guard level (strong protection)
- Sandbox only high-risk commands
- Balanced security
- Restricted network
- Caching enabled for speed

### Production Preset
**Maximum protection for production**

```bash
cp presets/production.yaml vectra-guard.yaml
```

Features:
- Paranoid guard level (all require approval)
- Always sandbox
- Maximum isolation
- No network access
- No caching (reproducibility)

---

## 🔧 Advanced Usage

### Custom Security Profiles

```yaml
sandbox:
  security_level: strict
  seccomp_profile: /etc/vectra-guard/seccomp-custom.json
  network_mode: none
  bind_mounts:
    - host_path: /readonly/data
      container_path: /data
      read_only: true
```

### Environment Variable Control

```yaml
sandbox:
  env_whitelist:
    - HOME
    - USER
    - PATH
    # Custom vars
    - COMPANY_API_TOKEN
    - BUILD_NUMBER
```

### Selective Sandboxing

```yaml
# Allowlist trusted commands
policies:
  allowlist:
    - "npm test"
    - "npm run build"
    - "git status"

# Always sandbox these
policies:
  denylist:
    - "rm -rf"
    - "curl * | sh"
    - "DROP DATABASE"
```

---

## 🚀 Performance Tips

### 1. Enable Caching
```yaml
sandbox:
  enable_cache: true
```
**Impact:** 10x faster subsequent installs

### 2. Use Process Sandbox (Linux)
```yaml
sandbox:
  runtime: process
```
**Impact:** 50% faster than Docker

### 3. Permissive Mode for Dev
```yaml
sandbox:
  security_level: permissive
```
**Impact:** Minimal overhead

### 4. Trust Common Commands
```bash
vg trust add "npm install"
vg trust add "npm test"
```
**Impact:** Skip sandboxing entirely

---

## 🔍 Troubleshooting

### Docker Not Available
```bash
# Use podman instead
sandbox:
  runtime: podman

# Or process sandbox (Linux only)
sandbox:
  runtime: process
```

### Slow Execution
```bash
# Enable caching
sandbox:
  enable_cache: true

# Or trust frequently-used commands
vg trust add "npm install"
```

### Network Issues
```bash
# Allow full network access
sandbox:
  network_mode: full
```

### Permission Errors
```bash
# Use permissive security level
sandbox:
  security_level: permissive

# Or bind mount with write access
sandbox:
  bind_mounts:
    - host_path: /path/to/data
      container_path: /data
      read_only: false
```

---

## 📊 Metrics & Analytics

### View Metrics
```bash
# Human-readable summary
vg metrics show

# JSON format
vg metrics show --json
```

### Reset Metrics
```bash
vg metrics reset
```

### Metrics File Location
```
~/.vectra-guard/metrics.json
```

---

## 🔐 Security Considerations

### Isolation Guarantees

| Runtime | Process | Network | Filesystem | Performance |
|---------|---------|---------|------------|-------------|
| **Docker** | ✅ Full | ✅ Full | ✅ Full | Good |
| **Podman** | ✅ Full | ✅ Full | ✅ Full | Good |
| **Process** | ⚠️ Partial | ⚠️ Partial | ⚠️ Partial | Excellent |

### Trust Store Security
- Stored at `~/.vectra-guard/trust.json`
- File permissions: `0600` (owner read/write only)
- SHA256 hash-based indexing
- Optional expiration times

### Seccomp Profiles
Custom seccomp profiles restrict system calls:
```yaml
sandbox:
  seccomp_profile: /etc/vectra-guard/seccomp-strict.json
```

Example profile provided: `seccomp-profile.json`

---

## 🎓 Examples

### Example 1: Development Workflow
```bash
# Clone and setup
git clone https://github.com/myproject/app.git
cd app

# Use developer preset
cp /path/to/vectra-guard/presets/developer.yaml vectra-guard.yaml

# Install dependencies (auto-sandboxed, cached)
vg exec "npm install"

# Run tests (trusted command, runs on host)
vg trust add "npm test"
vg exec "npm test"

# Build (sandboxed first time, then trusted)
vg exec "npm run build"
vg trust add "npm run build"
```

### Example 2: CI/CD Pipeline
```yaml
# .github/workflows/ci.yml
steps:
  - name: Setup Vectra Guard
    run: |
      curl -sSL https://vectra-guard.dev/install.sh | bash
      cp presets/ci-cd.yaml vectra-guard.yaml

  - name: Install dependencies
    run: vg exec "npm ci"

  - name: Run tests
    run: vg exec "npm test"

  - name: Build
    run: vg exec "npm run build"
```

### Example 3: Production Deployment
```bash
# Production server
cp presets/production.yaml vectra-guard.yaml

# All commands require approval
vg exec "kubectl apply -f deployment.yaml"
# → Prompts for approval with security details

# Critical operations blocked
vg exec "rm -rf /data"
# → Blocked immediately
```

---

## 🤝 Integration

### Shell Wrapper
```bash
# Install shell wrapper
./scripts/install-shell-wrapper.sh

# Now all commands protected
npm install  # → automatically uses vg exec
```

### VS Code / Cursor Integration
```bash
# Setup Cursor protection
./scripts/setup-cursor-protection.sh
```

### API Usage
```go
import "github.com/vectra-guard/vectra-guard/internal/sandbox"

executor, _ := sandbox.NewExecutor(cfg, logger)
decision := executor.DecideExecutionMode(ctx, cmdArgs, riskLevel, findings)
err := executor.Execute(ctx, cmdArgs, decision)
```

---

## 📚 Further Reading

- [Configuration Guide](CONFIGURATION.md)
- [Getting Started](GETTING_STARTED.md)
- [Advanced Features](ADVANCED_FEATURES.md)
- [API Documentation](https://pkg.go.dev/github.com/vectra-guard/vectra-guard)

---

## 💡 Best Practices

1. **Start with Auto Mode**
   ```yaml
   sandbox:
     mode: auto
   ```
   Let Vectra Guard make smart decisions.

2. **Enable Caching**
   ```yaml
   sandbox:
     enable_cache: true
   ```
   Dramatically improves performance.

3. **Trust Common Commands**
   ```bash
   vg trust add "npm test"
   vg trust add "npm run build"
   ```
   Reduce friction for safe operations.

4. **Monitor Metrics**
   ```bash
   vg metrics show
   ```
   Understand your usage patterns.

5. **Use Presets**
   ```bash
   cp presets/developer.yaml vectra-guard.yaml
   ```
   Start with proven configurations.

---

## 🎉 Summary

Vectra Guard's sandbox system provides:

✅ **Transparent** - Works like normal execution  
✅ **Fast** - Caching makes it nearly instant  
✅ **Secure** - Multiple isolation levels  
✅ **Smart** - Auto-detects when to sandbox  
✅ **Flexible** - Multiple runtimes and presets  
✅ **Observable** - Full metrics and analytics  
✅ **Learnable** - Remembers trusted commands  
✅ **Developer-friendly** - Minimal friction  

**Get started in 60 seconds:**
```bash
cp presets/developer.yaml vectra-guard.yaml
vg exec "npm install express"
```

That's it! 🚀

