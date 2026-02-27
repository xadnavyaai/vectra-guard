# Vectra Guard Agent Guidance (Claude)

## Use Vectra Guard for execution
- Run shell commands via `vectra-guard exec -- <command>` when changes are involved.
- Avoid destructive commands or anything targeting system paths.

## CVE Scanning Workflow
- **Before installing dependencies, scan for vulnerabilities:**
  ```bash
  vg cve sync --path .
  vg cve scan --path .
  vg cve explain <package>@<version> --ecosystem <npm|pypi|go>
  ```
- Enable in config: `cve: { enabled: true, sources: ["osv"] }`
- **Example workflow:**
  ```bash
  # Sync CVE database
  vg cve sync --path .
  
  # Scan manifest files (package.json, requirements.txt, go.mod)
  vg cve scan --path .
  
  # If clean, install safely
  vg exec -- npm install
  ```

## Soft Delete (Safe File Deletion & Recovery)
- **Files deleted via `rm` are automatically backed up** when soft delete is enabled
- **Complete workflow:**
  ```bash
  # Delete files - automatically backed up (not permanently deleted)
  vg exec -- rm -rf old-files/
  vg exec -- rm file.txt
  
  # List all backups with details
  vg restore list
  
  # Show detailed information about a backup
  vg restore show <backup-id>
  
  # Restore deleted files to original location
  vg restore <backup-id>
  
  # Restore to a different location
  vg restore <backup-id> --to /path/to/restore
  
  # Clean old backups (rotation based on age/count/size)
  vg restore clean
  vg restore clean --older-than 7  # Clean backups older than 7 days
  
  # Manually trigger auto-delete (if enabled)
  vg restore auto-delete
  
  # View backup statistics
  vg restore stats
  
  # Permanently delete a backup
  vg restore delete <backup-id>
  ```
- **Git Protection**: 
  - `.git` directory and git config files (`.gitignore`, `.gitattributes`, etc.) get enhanced protection
  - Git backups are kept longer (2x threshold) when auto-delete is enabled
  - Multiple backup copies for git files
- **Auto-Delete Feature**:
  - Automatically permanently delete backups older than N days (configurable)
  - Runs automatically after each backup operation
  - Git backups get extra protection (2x threshold)
- **Configuration:**
  ```yaml
  soft_delete:
    enabled: true
    max_age_days: 30           # Keep backups for 30 days
    max_backups: 100           # Maximum 100 backups
    max_size_mb: 1024          # Maximum 1GB total
    auto_cleanup: true         # Auto-rotate old backups
    auto_delete: false         # Auto-delete old backups (disabled by default)
    auto_delete_after_days: 90  # Delete backups older than 90 days (if enabled)
    protect_git: true          # Enhanced git protection
    git_backup_copies: 3       # Extra copies for git files
    rotation_policy: "age_and_count"  # age, count, size, age_and_count
  ```
- **Important**: Critical deletions (like `rm -rf /`) are still blocked. Soft delete only applies to safe deletions.

## Config & Sandbox
- Config lives in `vectra-guard.yaml` (or `.vectra-guard/config.yaml` with `--local`).
- Sandbox is enabled by default. Prefer `mode: always` for risky commands.
- Cache-optimized sandbox:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Setup helpers
- **macOS/Linux** install (no sudo): `curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash`
- **Windows** install (PowerShell, no admin): `irm https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-windows.ps1 | iex`
- Ensure install dir is on `PATH`
- `vectra-guard init --local`
- `vectra-guard sandbox deps install`
- `vectra-guard roadmap add --title "..." --summary "..." --tags "agent,plan"`

## Security Practices (Lightweight)
- Prefer user-space installs and avoid `sudo`.
- Avoid `curl | bash`; download and review scripts first.
- Keep secrets out of logs and outputs.
