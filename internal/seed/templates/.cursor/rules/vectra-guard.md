# Vectra Guard Cursor Rules

## Command Execution
- Use `vectra-guard exec -- <command>` for commands that change files or state.
- Never propose destructive commands (`rm -rf /`, `mkfs`, `dd if=`).

## Config
- Global config: `vectra-guard init` → `~/.config/vectra-guard/config.yaml` (default).
- Repo-local config: `vectra-guard init --local` → `.vectra-guard/config.yaml`.

## Sandboxing
- Default sandboxing is enabled. Use `mode: always` for high-risk operations.
- Cache-optimized sandbox example:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Shortcuts
- `vg` is an alias for `vectra-guard` when shell integration is enabled.

## Roadmap
- Capture plans and progress with `vectra-guard roadmap add/list/status/log`.
