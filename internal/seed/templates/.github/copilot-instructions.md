# Vectra Guard Copilot Instructions

## Execution Safety
- Prefer `vectra-guard exec -- <command>` for commands that modify files or system state.
- Do not suggest destructive commands (e.g., `rm -rf /`, `mkfs`, `dd if=`). These are blocked and unsafe.

## Configuration
- Global config: `~/.config/vectra-guard/config.yaml` (created by `vectra-guard init`, the default).
- Repo-local config: `.vectra-guard/config.yaml` (created by `vectra-guard init --local`).

## Sandboxing
- Default sandboxing is enabled. Use `auto` or `always` modes.
- Cache-optimized sandbox example:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Useful Commands
- `vectra-guard init --local`
- `vectra-guard sandbox deps install`
- `vectra-guard exec -- <command>`
- `vectra-guard roadmap add --title "..." --summary "..." --tags "agent,plan"`

