# VectraGuard Security Instructions for OpenClaw

Use these guidelines for all shell commands, file operations, and dependency management.

## Safe Execution
- Run shell commands via `vg exec -- <command>` when the command could modify system state.
- Destructive commands (`rm -rf /`, `mkfs`, `dd if=`, `sudo`) are blocked by design.
- If unsure about a command's safety, validate first: `vg validate <script>`

## CVE Scanning (Before Installing Dependencies)
- **Always scan before installing packages:**
  ```bash
  vg cve sync --path .
  vg cve scan --path .

  # If clean, proceed safely
  vg exec -- npm install
  ```
- Explain a specific vulnerability: `vg cve explain <package>@<version> --ecosystem <npm|pypi|go>`

## Soft Delete (Safe File Deletion)
- Files deleted via `rm` are automatically backed up when soft delete is enabled:
  ```bash
  vg exec -- rm -rf old-files/    # Backed up, not permanently deleted
  vg restore list                  # List all backups
  vg restore <backup-id>           # Restore deleted files
  ```
- `.git` directory and git config files get enhanced protection.

## Secret Detection
- Scan for exposed secrets before committing:
  ```bash
  vg scan-secrets --path .
  ```
- Never log, echo, or expose API keys, tokens, or credentials.

## Session Tracking
- Track agent activity in auditable sessions:
  ```bash
  SESSION=$(vg session start --agent "openclaw")
  export VECTRAGUARD_SESSION_ID=$SESSION

  # All commands are now tracked
  vg exec -- npm test

  # End session
  vg session end $SESSION
  ```

## Security Practices
- Prefer user-space installs — avoid `sudo`.
- Avoid `curl | bash` — download and review scripts first.
- Keep secrets out of logs, command history, and outputs.
- Use sandboxed execution for untrusted code:
  ```yaml
  sandbox:
    enabled: true
    mode: always
    enable_cache: true
  ```

## Quick Reference
| Task | Command |
|------|---------|
| Execute safely | `vg exec -- <command>` |
| Validate script | `vg validate <script>` |
| CVE sync | `vg cve sync --path .` |
| CVE scan | `vg cve scan --path .` |
| Scan secrets | `vg scan-secrets --path .` |
| Delete safely | `vg exec -- rm <file>` |
| Restore file | `vg restore <backup-id>` |
| Start session | `vg session start --agent "openclaw"` |
| End session | `vg session end <id>` |
