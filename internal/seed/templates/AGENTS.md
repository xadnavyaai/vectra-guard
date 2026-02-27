# Vectra Guard Agent Instructions

Use these guidelines for agentic coding tools and IDE assistants.

## Safe Execution
- Run shell commands via `vectra-guard exec -- <command>` when the command could modify system state.
- Avoid destructive commands (e.g., `rm -rf /`, `rm -rf /bin`, `mkfs`, `dd if=`). These are blocked by design.
- If you need a sandboxed environment, use `vectra-guard sandbox deps install` and keep sandbox enabled.

## CVE Scanning (Before Installing Dependencies)
- **Always scan dependencies before installing:**
  1. Sync vulnerability database: `vectra-guard cve sync --path .`
  2. Scan project: `vectra-guard cve scan --path .`
  3. If vulnerabilities found, check: `vectra-guard cve explain <package>@<version> --ecosystem <npm|pypi|go>`
- Enable CVE scanning in config:
  ```yaml
  cve:
    enabled: true
    sources: ["osv"]
    update_interval_hours: 24
  ```
- **Workflow example:**
  ```bash
  # Before npm install, pip install, go get, etc.
  vg cve sync --path .
  vg cve scan --path .
  
  # If clean, proceed
  vg exec -- npm install
  ```

## Soft Delete (Safe File Deletion)
- **Files deleted via `rm` are automatically backed up** when soft delete is enabled
- **Safe deletion workflow:**
  ```bash
  # Delete files - they're automatically backed up
  vg exec -- rm -rf old-files/
  
  # List all backups
  vg restore list
  
  # Restore a deleted file/directory
  vg restore <backup-id>
  
  # Show backup details
  vg restore show <backup-id>
  
  # Clean old backups (rotation)
  vg restore clean
  
  # Manually trigger auto-delete (if enabled)
  vg restore auto-delete
  ```
- **Git Protection**: `.git` directory and git config files get extra protection
- **Auto-Delete**: Optionally automatically delete old backups permanently
- **Enable soft delete in config:**
  ```yaml
  soft_delete:
    enabled: true
    max_age_days: 30
    max_backups: 100
    auto_cleanup: true      # Auto-rotate old backups
    auto_delete: false      # Auto-delete backups older than N days (disabled by default)
    auto_delete_after_days: 90  # Delete backups older than 90 days (if auto_delete enabled)
    protect_git: true
  ```
- **Important**: Critical deletions (like `rm -rf /`) are still blocked. Soft delete only applies to safe deletions.

## Recommended Setup
- **macOS/Linux** — install locally (no sudo):
  - `curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash`
  - Ensure `~/.local/bin` is on `PATH`
- **Windows** — install via PowerShell (no admin):
  - `irm https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-windows.ps1 | iex`
- Initialize a repo-local config and cache:
  - `vectra-guard init --local`
- Default config file: `vectra-guard.yaml` (or `.vectra-guard/config.yaml` for local mode).

## Security Practices (Lightweight)
- Prefer user-space installs and avoid `sudo`.
- Avoid `curl | bash`; download and review scripts first.
- Keep secrets out of logs, command history, and outputs.

## Sandboxing
- Default behavior is safe-by-default; sandbox can be set to `auto` or `always`.
- Use cache-optimized sandboxing for builds:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Shortcuts
- `vg` is an alias for `vectra-guard` when shell integration is installed.

## Roadmap (Plan + Handoff)
- Capture plans and decisions so humans and agents stay aligned:
  - `vectra-guard roadmap add --title "..." --summary "..." --tags "agent,plan"`
  - `vectra-guard roadmap list`
  - `vectra-guard roadmap status <id> in-progress`
  - `vectra-guard roadmap log <id> --note "..." --session $VECTRAGUARD_SESSION_ID`
