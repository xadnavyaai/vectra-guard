# Vectra Guard v0.2.0 Release Notes

Release Date: 2026-01-19

## 🎉 Major Features

### CVE Awareness & Vulnerability Scanning
Vectra Guard now includes built-in CVE scanning and vulnerability detection for your dependencies!

- **Local CVE Database**: Sync and cache vulnerability data from OSV (Open Source Vulnerabilities)
- **Manifest Scanning**: Automatically scans `package.json`, `package-lock.json`, `go.mod`, and other dependency manifests
- **Package Risk Analysis**: Explain specific packages with `vg cve explain <package>@<version>`
- **Pre-Install Protection**: Scan dependencies before installing to catch vulnerabilities early

#### New Commands
```bash
# Sync vulnerability database
vg cve sync --path .

# Scan project dependencies
vg cve scan --path .

# Explain specific package
vg cve explain lodash@4.17.20 --ecosystem npm
```

#### Configuration
```yaml
cve:
  enabled: true
  sources: ["osv"]
  update_interval_hours: 24
  cache_dir: "~/.cache/vectra-guard/cve"
```

## 📚 Documentation Updates

### New Documentation
- **FEATURES.md**: Comprehensive feature guide with examples, workflows, and use cases
- **docs/cve-awareness.md**: Technical design document for CVE integration
- CVE scanning added to README.md, GETTING_STARTED.md, and CONFIGURATION.md

### Agent Integration
- Updated agent seed templates (AGENTS.md, CLAUDE.md, CODEX.md) with CVE workflow
- Agents now automatically check for vulnerabilities before installing dependencies
- One-line agent integration: `vg seed agents --target . --targets "agents,cursor"`

## 🔧 Technical Improvements

### CVE Infrastructure
- New `internal/cve/` package with modular design:
  - `store.go`: Local CVE cache with persistence
  - `osv.go`: OSV API client with retries
  - `scanner.go`: Manifest discovery and parsing
  - `types.go`: Core CVE data structures
- Comprehensive test coverage for CVE functionality
- Go module dependency: `golang.org/x/mod` for version parsing

### Makefile Enhancements
- Added `test-cve`: Run CVE-specific tests
- Updated `test-internal` to include CVE tests
- Extended test-docker and test-local-* targets

## 📦 Installation

**macOS & Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-windows.ps1 | iex
```

**With full setup (macOS/Linux):**
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-all.sh | bash
```

**Enable CVE scanning:**
```bash
vg init --local
# Edit .vectra-guard/config.yaml and add:
# cve:
#   enabled: true
```

## 🚀 Quick Start with CVE Scanning

```bash
# 1. Initialize with local config
vg init --local

# 2. Enable CVE in config
cat >> .vectra-guard/config.yaml <<EOF
cve:
  enabled: true
  sources: ["osv"]
EOF

# 3. Sync vulnerability database
vg cve sync --path .

# 4. Scan your project
vg cve scan --path .

# 5. If clean, install safely
vg exec -- npm install
```

## 🎯 Roadmap Updates

### Completed (Free Tier)
- ✅ Basic CVE awareness + dependency scanning
- ✅ Local CVE cache + periodic sync (NVD/MITRE/OSV)
- ✅ CLI CVE scan + explain output for manifests/lockfiles

### Coming Soon (Paid Tier)
- Advanced CVE prioritization + risk scoring automation
- Enterprise dashboards + reporting
- Premium vulnerability feeds + SLAs
- CI/CD integrations

## 🔒 Security Note

- CVE scanning is **opt-in** by default (set `cve.enabled: true`)
- All critical command protections remain **non-bypassable**
- CVE data is cached locally for offline scanning
- No data is sent to external services except OSV API calls

## 📊 Stats

- **6 new Go source files** for CVE functionality
- **500+ lines of test coverage** for CVE features
- **3 agent templates updated** with CVE workflows
- **4 documentation files** created/updated

## 🙏 Acknowledgments

Thanks to the open-source community for:
- [OSV](https://osv.dev/) - Open Source Vulnerabilities database
- All contributors and testers

---

**Stay Safe. Code Fearlessly.** 🛡️

[Report Bug](https://github.com/xadnavyaai/vectra-guard/issues) · [Request Feature](https://github.com/xadnavyaai/vectra-guard/issues) · [Documentation](https://github.com/xadnavyaai/vectra-guard)
