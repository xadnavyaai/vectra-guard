# Getting Started with Vectra Guard

> **Simple guide to start protecting your development environment in 5 minutes**

---

## 🎯 What You'll Achieve

After following this guide (critical outcomes first):
- ✅ Commands run with `vectra-guard exec` will be protected
- ✅ Risky commands will be caught or sandboxed automatically
- ✅ Full audit trail of everything executed (sessions)
- ✅ CVE scanning for known vulnerable dependencies
- ✅ Secret and code scanning before deploy (`scan-secrets`, `scan-security`)
- ✅ Optional agent helpers: context summaries + roadmap planning

**Supported platforms:** macOS and Debian Linux (x86_64, arm64).

**Prerequisites:** `git` and `go` (install via Homebrew on macOS or apt on Debian/Ubuntu).

---

## ⚡ Quick Start (30 Seconds)

### Step 1: Install Vectra Guard

**Recommended – one command:**

```bash
# Downloads the latest release binary and installs locally
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

The installer defaults to user-space (`$HOME/.local/bin`). Ensure `~/.local/bin` is on `PATH`.

**One-line uninstall:**

```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/uninstall.sh | bash
```

**Prereqs:** `curl` or `wget` must be installed.

For alternative installation methods (Go install, build from source), see the **Installation Options** section in `README.md`.

**One-command full setup (deps + tool):**
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-all.sh | bash
```

### Step 2: Start a Session (Recommended)

```bash
SESSION=$(vectra-guard session start --agent "manual")
export VECTRAGUARD_SESSION_ID=$SESSION
```

### Step 3: Run a Protected Command

```bash
vectra-guard exec -- echo "Hello, Vectra Guard!"
```

### Step 4: Audit the session

```bash
vectra-guard audit session
```

### Step 4b: CVE Scan (recommended)

```bash
vectra-guard cve sync --path .
vectra-guard cve scan --path .
```

### Step 5: Validate a script and/or scan for secrets

```bash
# Validate a script (never executes)
vectra-guard validate scripts/deploy.sh

# Scan for secrets and risky code before deploy
vectra-guard scan-secrets --path .
vectra-guard scan-security --path . --languages go,python,c,config
```

**That's it!** Vectra Guard is now protecting commands run with `vectra-guard exec`.

### Optional: Agentic setup (Cursor/IDE/OpenClaw)

For agent workflows: seed agent instructions, install workflow examples. See [More features](#-more-features) and [README](README.md#-features-by-impact).

```bash
# Seed agent instructions into the current repo
vectra-guard seed agents --target . --targets "agents,cursor"

# For OpenClaw users: seed the VectraGuard plugin config
vectra-guard seed agents --target . --targets "openclaw"

# Or seed everything at once
vectra-guard seed agents --target . --targets "agents,claude,cursor,openclaw"
```

---

## ✅ Verify It's Working

### Test 1: Check Session

```bash
# Should show a session ID
echo $VECTRAGUARD_SESSION_ID
```

**Expected output**: `session-1234567890...`

---

## 🛡️ Features in order (by impact)

What you just did, ordered by impact:

1. **Execution protection** — Commands run with `vectra-guard exec` are sandboxed or blocked by risk. Example: `vectra-guard exec -- npm install`
2. **Script validation** — Analyze scripts without executing: `vectra-guard validate scripts/deploy.sh`
3. **Session and audit** — Traceability: `vectra-guard session start`, then `vectra-guard audit session`
4. **CVE scanning** — Flag vulnerable dependencies: `vectra-guard cve sync --path .` and `vectra-guard cve scan --path .`
5. **Secret and code scanning** — Predetect before deploy: `vectra-guard scan-secrets --path .`, `vectra-guard scan-security --path .`. See [Control panel & deployment security](docs/control-panel-security.md).

---

## 🎯 More features

Additional features that improve workflow, ordered by impact: **seed agents** (Cursor/IDE/OpenClaw), **explain** (why a script is risky), **trust store**, **OpenClaw plugin** (runtime security for AI agents), **context summaries**, **roadmap planning**, **shell tracker** (logging only), **lockdown**, **prompt firewall**, **validate-agent**, **container mode**, **git pre-commit hook**, **IDE integration**. See [README](README.md#-more-features) or [FEATURES.md](FEATURES.md) for details.

---

### Test 2: Run a Command

```bash
# Run any command through the guard
vectra-guard exec -- echo "Hello, Vectra Guard!"
```

---

### Test 3: Package Audits (npm/python)

```bash
# npm audit (auto-installs npm if missing)
vectra-guard audit npm --path .

# python audit (auto-installs pip-audit if missing)
vectra-guard audit python --path .
```

---

### Test 4: Check if Logged

```bash
# View what was logged
vectra-guard session show $VECTRAGUARD_SESSION_ID
```

**Expected output**: Should show the echo command you just ran

---

### Test 5: Try a Risky Command

```bash
# This will show a warning
vectra-guard exec -- rm -rf /tmp/test-file
```

**Expected**: You'll see risk warnings, but command still logged

---

## 🎓 How to Use Daily

Once critical setup is done, use these patterns.

### Just Use Your Tools Normally!

**In Cursor**:
```bash
# Open Cursor terminal
vectra-guard exec -- npm install
# ✅ Protected and logged
```

**In VSCode**:
```bash
# Open VSCode terminal
vectra-guard exec -- git push
# ✅ Protected and logged
```

**In Regular Terminal**:
```bash
# Open Terminal app
vectra-guard exec -- sudo apt update
# ✅ Shows risk warning, requires approval (if configured)
```

**Everything just works!** Use `vectra-guard exec` to protect commands.

---

## 📊 View Your Activity

### See What Commands Were Run

```bash
# Show current session
vectra-guard session show $VECTRAGUARD_SESSION_ID
```

### List All Sessions

```bash
# See all your sessions
vectra-guard session list
```

### View Specific Session

```bash
# If you have a session ID
vectra-guard session show session-1234567890
```

---

## 🛡️ Common Use Cases

### Use Case 1: Working with AI (Cursor/Copilot)

**Scenario**: AI suggests a command

```bash
# AI suggests in Cursor terminal:
rm -rf node_modules && npm install
```

**What happens**:
1. ✅ Command validated automatically
2. ✅ Risk assessed (medium risk for rm -rf)
3. ✅ Logged to your session
4. ✅ Executes normally
5. ✅ You can review later what AI did

**Check what happened**:
```bash
vectra-guard session show $VECTRAGUARD_SESSION_ID
```

---

### Use Case 2: Validating Scripts Before Running

**Scenario**: You have a script to run

```bash
# First, check if it's safe
vectra-guard validate deploy.sh

# Get detailed explanation
vectra-guard explain deploy.sh

# If safe, run it
vectra-guard exec -- ./deploy.sh
```

---

### Use Case 3: Interactive Approval for Sudo

**Scenario**: You want approval prompts for sudo commands

**Setup** (one time):
```bash
# Add to your shell config (~/.bashrc or ~/.zshrc)
alias sudo='vectra-guard exec --interactive -- sudo'
```

**Now when you run**:
```bash
sudo apt install something
```

**You'll get**:
```
⚠️  Command requires approval
Command: sudo apt install something
Risk Level: MEDIUM

Do you want to proceed? [y/N]:
```

---

### Use Case 4: Team Collaboration

**Share protection with your team**:

```bash
# In your project directory
vectra-guard init

# Edit vectra-guard.yaml with team policies
# Example:
```

```yaml
policies:
  allowlist:
    - "npm install"
    - "npm test"
    - "git status"
  denylist:
    - "rm -rf /"
    - "sudo rm"
```

```bash
# Commit to git
git add vectra-guard.yaml
git commit -m "Add security policies"
git push

# Team members pull and get same protection
```

---

## 🔧 Customization

### Configure Policies

**Edit** `vectra-guard.yaml` in your project or `~/.config/vectra-guard/config.yaml`.

**Quick Start Preset:**
```yaml
guard_level:
  level: auto

sandbox:
  enabled: true
  mode: always  # Default: maximum security with caching
  security_level: balanced
  enable_cache: true
```
See [CONFIGURATION.md](CONFIGURATION.md) for full details.

### Configure Policies

```yaml
logging:
  format: text  # or "json" for structured logs

policies:
  # Always allow these
  allowlist:
    - "echo *"
    - "ls *"
    - "npm install"
    - "npm test"
    
  # Block or warn about these
  denylist:
    - "rm -rf /"
    - "sudo rm"
    - "curl * | sh"
    - "dd if="
```

---

## 🎯 Real Examples

### Example 1: Daily Development

**Morning**:
```bash
# Open your terminal (session starts automatically)
cd ~/my-project

# Work normally
vectra-guard exec -- npm install
vectra-guard exec -- npm test
git commit -m "Fix bug"
vectra-guard exec -- git push

# Everything logged automatically! ✅
```

**End of day**:
```bash
# Review what you did
vectra-guard session show $VECTRAGUARD_SESSION_ID

# Output shows all your commands with timestamps
```

---

### Example 2: Running Untrusted Scripts

**Scenario**: Downloaded a script from internet

```bash
# First, validate it
vectra-guard validate suspicious-script.sh

# Get detailed risk analysis
vectra-guard explain suspicious-script.sh

# If you trust it, run it
./suspicious-script.sh  # Still protected
```

---

### Example 3: AI Code Review

**Scenario**: AI generated a script

```bash
# AI created deploy.sh
# Check it first
vectra-guard validate deploy.sh

# Might show:
# ⚠️ [SUDO_USAGE] Line 5: sudo systemctl restart
# ⚠️ [DANGEROUS_DELETE] Line 12: rm -rf /tmp/*

# Review and fix before running
```

---

## 🐳 Advanced: Container Mode (Maximum Security)

**For critical work or untrusted code**:

```bash
# Run in isolated container
docker-compose up agent-prod
```

**Inside container**:
- ✅ Completely isolated
- ✅ Cannot access host system
- ✅ Read-only filesystem
- ✅ No network access
- ✅ Maximum security

**Use when**:
- Running unknown scripts
- Testing malware samples
- Production deployments
- High-security requirements

---

## 📱 IDE Integration

### Cursor IDE

Use `vectra-guard exec` in the Cursor terminal to protect commands.

**Optional**: Add tasks to `.vscode/tasks.json`:

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "🛡️ Install (Protected)",
      "type": "shell",
      "command": "vectra-guard exec -- npm install"
    }
  ]
}
```

Then: `Cmd/Ctrl + Shift + P` → "Run Task"

---

### Git Pre-commit Hook

**Automatically validate scripts before commit**:

```bash
# Create .git/hooks/pre-commit
cat > .git/hooks/pre-commit << 'EOF'
#!/bin/bash
SCRIPTS=$(git diff --cached --name-only | grep '\.sh$')
for script in $SCRIPTS; do
  vectra-guard validate "$script" || exit 1
done
EOF

chmod +x .git/hooks/pre-commit
```

Now scripts are checked automatically on every commit!

---

## 🆘 Troubleshooting

### Problem: Session not starting

**Check**:
```bash
which vectra-guard
# Should show: /Users/<you>/.local/bin/vectra-guard
```

**Fix**:
```bash
# Reinstall the binary
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
# Start a new session
SESSION=$(vectra-guard session start --agent "manual")
export VECTRAGUARD_SESSION_ID=$SESSION
```

---

### Problem: Commands not being logged

**Check**:
```bash
echo $VECTRAGUARD_SESSION_ID
# Should show session ID
```

**Fix**:
```bash
# Manually start session
source ~/.bashrc  # or ~/.zshrc
# Or
./.vectra-guard/start-session.sh
```

---

### Problem: Too many false positives

**Solution**: Adjust policies

```yaml
# Add to vectra-guard.yaml
policies:
  allowlist:
    - "rm -rf node_modules"  # Known safe
    - "sudo npm install -g"  # Trusted operation
```

---

## 💡 Pro Tips

### Tip 1: Project-Specific Policies

```bash
# Each project can have its own policies
cd ~/project-a
vectra-guard init  # Creates vectra-guard.yaml

# Different policies for different projects
```

### Tip 2: Review Sessions Regularly

```bash
# Add to your shell config
alias vectra-today='vectra-guard session list | grep $(date +%Y-%m-%d)'
```

### Tip 3: Export for Compliance

```bash
# Generate audit report
vectra-guard session show $SESSION_ID --output json > audit.json
```

### Tip 4: Safety Aliases

```bash
# Add to ~/.bashrc or ~/.zshrc
alias rm='vectra-guard exec -- rm'
alias sudo='vectra-guard exec --interactive -- sudo'
```

### Tip 5: Quick Session Info

```bash
# Add to your prompt
PS1='🛡️[$VECTRAGUARD_SESSION_ID] \w $ '
```

---

## 📚 Learn More

### Available Commands

```bash
# Initialize config
vectra-guard init

# Validate script
vectra-guard validate script.sh

# Explain risks
vectra-guard explain script.sh

# Execute with protection
vectra-guard exec -- command

# Session management
vectra-guard session start --agent NAME
vectra-guard session end SESSION_ID
vectra-guard session list
vectra-guard session show SESSION_ID
```

### Get Help

```bash
# Show usage
vectra-guard --help

# See examples in README
cat README.md

# Check your config
cat vectra-guard.yaml
```

---

## 🎯 Quick Reference Card

| Task | Command |
|------|---------|
| **Check protection is active** | `echo $VECTRAGUARD_SESSION_ID` |
| **View current activity** | `vectra-guard session show $VECTRAGUARD_SESSION_ID` |
| **List all sessions** | `vectra-guard session list` |
| **Validate a script** | `vectra-guard validate script.sh` |
| **Explain risks** | `vectra-guard explain script.sh` |
| **Run with approval** | `vectra-guard exec --interactive -- command` |
| **Initialize config** | `vectra-guard init` |

---

## ✅ Success Checklist

After setup, you should have:
- [x] `vectra-guard` binary installed
- [x] Session ID in `$VECTRAGUARD_SESSION_ID`
- [x] Commands being logged
- [x] Risky commands showing warnings

**If all checked, you're fully protected!** 🛡️

---

## 🎓 Next Steps

1. **Use your tools normally** - Everything is automatic
2. **Check logs occasionally** - `vectra-guard session show $VECTRAGUARD_SESSION_ID`
3. **Customize policies** - Edit `vectra-guard.yaml`
4. **Share with team** - Commit config to git
5. **Enable container mode** - For maximum security
6. **Scan for secrets and risky code** - Run `vectra-guard scan-secrets --path .` and `vectra-guard scan-security --path . --languages go,python,c,config` in CI or before deploy. See [Control panel & deployment security](docs/control-panel-security.md) (checklist, rule reference, and [detection behavior](docs/control-panel-security.md#detection-behavior-scan-secrets-and-scan-security)).

---

## 🎉 You're Ready!

**Vectra Guard is now protecting your environment.**

Just use Cursor, VSCode, or Terminal as normal. Everything is automatically:
- ✅ Validated
- ✅ Logged
- ✅ Risk-assessed
- ✅ Protected

**No workflow changes needed. Work fearlessly!** 🛡️

---

## 🆘 Need Help?

- **Documentation**: [README.md](README.md)
- **Issues**: https://github.com/xadnavyaai/vectra-guard/issues
- **Examples**: This file!

---

<div align="center">

**Happy Secure Coding!** 🚀

*Remember: The best security is the security you don't have to think about.*

</div>
