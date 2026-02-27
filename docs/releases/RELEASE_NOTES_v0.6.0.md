# Vectra Guard v0.6.0 Release Notes

Release Date: 2026-02-28

## Major Features

### Full Windows Platform Support
Vectra Guard now runs natively on Windows with feature parity to macOS and Linux.

- **One-line installer** (PowerShell, no admin required):
  ```powershell
  irm https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-windows.ps1 | iex
  ```
- Installs to `%LOCALAPPDATA%\VectraGuard` (user-space, no admin)
- SHA256 checksum verification, upgrade detection, `vg`/`vectraguard` aliases
- CI non-interactive mode (`$env:CI` / `$env:VECTRAGUARD_YES`)
- Windows added to GitHub Actions CI and release build matrix

### Critical Security Fix: Windows System Directory Blocking
Previously, commands like `rm -rf C:\Windows` bypassed the critical command blocker because the directory validator only recognized Unix-style paths. This is now fully blocked.

- Added `isWindowsAbsPath` and `normalizeWindowsPath` detection helpers
- 26 Windows system path patterns added to `rootDeletePatterns` (both `\` and `/` styles)
- Fallback substring matching for paths with spaces (e.g. `C:\Program Files`)
- `ValidateProtectedDirectory` and `IsProtectedDirectory` now handle Windows paths
- Comprehensive test coverage: 23 new Windows-specific test cases

### Global Config Default for `vg init`
`vg init` now writes to `~/.config/vectra-guard/config.yaml` by default instead of `vectra-guard.yaml` in the current directory.

- `vg init` — global config at `~/.config/vectra-guard/config.yaml` (new default)
- `vg init --local` — repo-scoped config at `.vectra-guard/config.yaml` (unchanged)
- `vg init --global` — explicit global flag
- Config loading order: global → project → local (later files override)

### Security Dashboard with Live Events
New web-based admin dashboard via `vg serve`.

- **Live event feed** via Server-Sent Events (SSE)
- **Risk distribution** charts and execution history
- **Session management** panel with agent tracking
- **Agent coverage** with OpenClaw detection status
- **CVE scanner** integration showing vulnerability data
- **Trust store** viewer for pre-approved commands
- Works on all platforms (TCP binding, embedded HTML)

### OpenClaw Integration for `vg seed agents`
Smart detection, confirmation, and merge for OpenClaw agent instructions.

- Auto-detects OpenClaw state directory (`~/.openclaw/`, env vars, legacy paths)
- Marker-based merge (`<!-- vectraguard:begin/end -->`) — idempotent updates
- Interactive confirmation with custom path support
- `--yes` flag for non-interactive/CI usage

## Bug Fixes

- **Windows path blocking**: `rm -rf C:\Windows`, `rm -rf C:\Program Files`, etc. now hard-blocked
- **Global config init**: `vg init` writes to `~/.config/vectra-guard/` instead of cwd
- **Cross-platform tests**: Fixed hardcoded Unix paths in tests (`USERPROFILE` on Windows, `filepath.Join`)
- **gofmt compliance**: Fixed formatting in openclaw.go, server.go, server_test.go
- **Docker e2e tests**: Updated init checks for new global config path

## Documentation

- 28+ files updated for Windows install instructions and global config path
- All 7 seed templates updated (Claude, Agents, Codex, Cursor, Windsurf, VSCode, Copilot)
- Install scripts updated (bash, PowerShell, shell tracker, Homebrew)
- Release notes, checklists, and CI workflow docs updated

## Installation

**macOS & Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-windows.ps1 | iex
```

**Quick start:**
```bash
vg init                    # Global config (~/.config/vectra-guard/)
vg exec -- npm install     # Safe execution
vg serve                   # Admin dashboard at http://127.0.0.1:8000
```

## Supported Platforms

| Platform | Architecture | Binary |
|----------|-------------|--------|
| macOS | x86_64 | `vectra-guard-darwin-amd64` |
| macOS | ARM64 | `vectra-guard-darwin-arm64` |
| Linux | x86_64 | `vectra-guard-linux-amd64` |
| Linux | ARM64 | `vectra-guard-linux-arm64` |
| Windows | x86_64 | `vectra-guard-windows-amd64.exe` |
| Windows | ARM64 | `vectra-guard-windows-arm64.exe` |

## Stats

- **17 commits** since v0.5.0
- **23 new test cases** for Windows path blocking
- **28+ documentation files** updated
- **6 platform binaries** (3 OS x 2 architectures)

---

**Stay Safe. Code Fearlessly.**

[Report Bug](https://github.com/xadnavyaai/vectra-guard/issues) · [Request Feature](https://github.com/xadnavyaai/vectra-guard/issues) · [Documentation](https://github.com/xadnavyaai/vectra-guard)
