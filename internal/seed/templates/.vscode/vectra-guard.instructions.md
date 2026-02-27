# Vectra Guard VS Code Instructions

## Execution Safety
- Prefer `vectra-guard exec -- <command>` for commands that modify files or system state.
- Avoid destructive commands (`rm -rf /`, `mkfs`, `dd if=`).

## Config & Sandbox
- Global config: `~/.config/vectra-guard/config.yaml` (created by `vectra-guard init`).
- Repo-local config: `.vectra-guard/config.yaml` (created by `vectra-guard init --local`).
- Sandbox is enabled by default; use `auto` or `always`.
- Cache-optimized sandbox example:
  - `sandbox: { enabled: true, mode: always, enable_cache: true }`

## Helpful Commands
- `vectra-guard init` (global) or `vectra-guard init --local` (repo-scoped)
- `vectra-guard sandbox deps install`
- `vectra-guard exec -- <command>`
- `vectra-guard roadmap add --title "..." --summary "..." --tags "agent,plan"`

