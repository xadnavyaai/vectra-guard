# Refined comment for Claude Cowork / agent data-loss incident

Use this when responding to posts about AI agents (e.g. Claude Cowork) accidentally deleting irreplaceable user data (e.g. family photos).

---

## Short version (recommended for Reddit/social)

> Really sorry to hear that — losing years of family photos like that is devastating.
>
> This is exactly why we need better guardrails. At NavyaAI we built [Vectra Guard](https://vectraguard.dev) so agents run destructive actions through a safety layer: risky commands get blocked or sandboxed, and with soft delete enabled, `rm` is backed up so you can restore.
>
> Two commands to add guardrails:
>
> ```bash
> curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
> vectra-guard seed agents --targets "claude"
> ```
>
> We open-sourced it so anyone can audit the behavior. [vectraguard.dev](https://vectraguard.dev) · [GitHub](https://github.com/xadnavyaai/vectra-guard)

---

## Slightly longer (if you want to add “can we prevent this?”)

> Really sorry to hear that — losing years of family photos like that is devastating.
>
> This is exactly why we need better guardrails. At NavyaAI we built [Vectra Guard](https://vectraguard.dev) for this: agents are instructed to run shell commands through `vectra-guard exec`, so we can block critical deletes (e.g. `rm -rf /`, wiping home), sandbox risky ones, and with soft delete enabled, even allowed `rm` is backed up so you can restore with `vectra-guard restore`.
>
> Two commands to guardrail your system:
>
> ```bash
> curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
> vectra-guard seed agents --targets "claude"
> ```
>
> Caveat: this helps when the agent actually uses Vectra Guard (e.g. in Cursor after seeding, or when your shell routes commands through it). We can’t force third-party products like Claude Cowork to use it; we open-sourced it so vendors can adopt it and users can audit it.
>
> [vectraguard.dev](https://vectraguard.dev) · [GitHub](https://github.com/xadnavyaai/vectra-guard)

---

## Can we prevent this with Vectra Guard?

**When Vectra Guard *does* help**

- **Agent runs commands through Vectra Guard**  
  After you run `vectra-guard seed agents --targets "claude"` (or cursor, etc.), the seeded instructions tell the agent to run shell commands that change system state via `vectra-guard exec -- <command>`.
- **Then:**
  - **Blocked:** Critical patterns are always blocked (e.g. `rm -rf /`, `rm -rf /etc`, `rm -rf ~/*`, and other dangerous deletes in `exec` and analyzer).
  - **Soft delete (optional):** If soft delete is enabled in config, any `rm` that is allowed (e.g. `rm -rf ~/Desktop/some-folder`) is **backed up** before deletion. You can list backups with `vectra-guard restore list` and restore with `vectra-guard restore <backup-id>`. So even a mistaken delete of a photo folder could be recovered from backup. See [How to restore deleted files](#how-to-restore-deleted-files-soft-delete) and [How to test that soft delete is working](#how-to-test-that-soft-delete-is-working) below.
  - **Audit:** You get session and audit logs of what ran and what was blocked.

**When Vectra Guard *cannot* prevent it**

- **Agent runs raw terminal commands**  
  If the agent (e.g. Claude Cowork) runs commands directly in the system shell and does **not** use `vectra-guard exec`, Vectra Guard never sees those commands, so it cannot block or back them up. Protection only applies when commands are executed via `vectra-guard exec` (or when the user’s shell is configured to route commands through Vectra Guard).
- **Third-party products**  
  We cannot force products like Claude Cowork to integrate Vectra Guard. Seeding helps in environments where the agent reads project/IDE rules (e.g. Cursor). For Cowork specifically, protection would require either Anthropic integrating such guardrails or the user running in an environment where all shell execution is proxied through Vectra Guard.

**Summary**

- **Yes, we can reduce or prevent this class of incident** when:
  - The agent is instructed (e.g. via seed) to use `vectra-guard exec` for commands that modify the system, and
  - The user enables soft delete so that allowed `rm` operations are backed up and restorable.
- **We cannot prevent it** when the agent bypasses Vectra Guard and runs destructive commands directly in the shell. Open-sourcing Vectra Guard allows the community and vendors to adopt and audit it so more agents run through this safety layer.

---

## How to restore deleted files (soft delete)

When soft delete is enabled and a delete was run through `vectra-guard exec`, files are backed up instead of removed. Use the restore commands to list, inspect, and restore them.

### 1. Enable soft delete

Ensure your config has soft delete enabled. If you use `vectra-guard init`, you can add:

```yaml
# vectra-guard.yaml (or .vectra-guard/config.yaml with --local)
soft_delete:
  enabled: true
  max_age_days: 30
  max_backups: 100
  max_size_mb: 1024
  auto_cleanup: true
  protect_git: true
```

(Soft delete can be enabled by default in some setups; if `vg restore list` works, it’s on.)

### 2. List backups

```bash
vectra-guard restore list
# or
vg restore list
```

Example output:

```
ID       Timestamp              Command              Files   Size
a1b2c3d4 2026-02-08 10:30:15   rm -rf my-photos/     42      15.3 MB
e5f6g7h8 2026-02-08 09:15:00   rm old-file.txt       1       4.2 KB
```

### 3. Inspect a backup (optional)

```bash
vectra-guard restore show <backup-id>
```

Example:

```bash
vg restore show a1b2c3d4
```

This prints the backup ID, timestamp, original command, session, and the list of files (paths and sizes).

### 4. Restore to original location

```bash
vectra-guard restore <backup-id>
# or
vg restore a1b2c3d4
```

Files are put back where they were before the delete. Use the **full** backup ID. Right after a soft delete, the CLI prints something like `Restore with: vg restore <full-id>` — that ID is the one to use. The ID column in `vg restore list` may show a shortened form; if `vg restore <id>` says "backup not found", try the full ID from that post-delete message.

### 5. Restore to a different location

To restore somewhere else (e.g. to avoid overwriting):

```bash
vectra-guard restore <backup-id> --to /path/to/safe/dir
# or
vg restore a1b2c3d4 --to ~/Restored/my-photos
```

### 6. Other restore commands

```bash
vg restore stats          # Backup counts and sizes
vg restore clean         # Remove old backups (respects rotation policy)
vg restore clean --older-than 7   # Remove backups older than 7 days
vg restore delete <id>   # Permanently remove one backup from the list
```

---

## How to test that soft delete is working

Run a safe end-to-end test so you can see backup and restore in action.

### 1. Enable soft delete

Ensure `soft_delete.enabled: true` in your config (see [How to restore](#how-to-restore-deleted-files-soft-delete) above).

### 2. Create a test directory and file

Use a path **under your current directory** (e.g. in a project folder). Paths like `/tmp/...` or other system paths may be blocked by default.

```bash
mkdir -p vectra-guard-test
echo "safe to delete" > vectra-guard-test/hello.txt
cat vectra-guard-test/hello.txt   # should print: safe to delete
```

### 3. “Delete” it via Vectra Guard (soft delete)

```bash
vectra-guard exec -- rm -rf vectra-guard-test
```

You should see:

- A line like: **Restore with: vg restore &lt;backup-id&gt;**  
  Copy that `<backup-id>` for the next step.
- The directory should be gone from the filesystem:

```bash
ls vectra-guard-test
# ls: vectra-guard-test: No such file or directory
```

### 4. List backups and confirm the backup exists

```bash
vectra-guard restore list
```

You should see an entry for `rm -rf vectra-guard-test` with 1 or 2 files and a small size.

### 5. Restore using the backup ID

Use the **full** backup ID from the “Restore with” message (or the ID from `vg restore list` if your build supports matching):

```bash
vectra-guard restore <backup-id>
# Example:
# vectra-guard restore a1b2c3d4e5f6
```

You should see something like:

```
Restoring backup a1b2c3d4e5f6...
✅ Restored 2 files from backup a1b2c3d4e5f6
   Restored to original locations
```

### 6. Verify the files are back

```bash
ls vectra-guard-test
cat vectra-guard-test/hello.txt
```

You should see `hello.txt` again and its content: `safe to delete`.

### 7. Clean up the test backup (optional)

```bash
vectra-guard restore delete <backup-id>
rm -rf vectra-guard-test
```

If at step 3 you don’t see “Restore with: vg restore …” and the directory is really deleted (not in `vg restore list`), then either soft delete is disabled or the command didn’t go through `vectra-guard exec`. Double-check config and that you ran `vectra-guard exec -- rm -rf ...`.
