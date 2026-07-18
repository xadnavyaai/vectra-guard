# Vectra Guard v0.7.1 Release Notes

Release Date: 2026-07-18

Patch release on top of [v0.7.0](RELEASE_NOTES_v0.7.0.md).

## Bug Fixes

### Windows: session IDs could collide and clobber each other
`generateSessionID()` derived the ID solely from `time.Now().UnixNano()`. On
platforms with a low-resolution wall clock (notably Windows), two sessions
started within the same clock tick received **identical IDs** and overwrote
each other's session file on disk.

- Symptom: back-to-back sessions on Windows could lose data, and behavioral
  diffs between two such sessions came back empty (the target session had
  clobbered the base).
- Fix: session IDs now append 6 random bytes, so uniqueness no longer depends
  on clock granularity. No code parses the numeric portion of the ID, so the
  format change is backward compatible.

This also restores a green `windows-latest` leg in the multi-platform CI build.

## Installation

**macOS & Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-windows.ps1 | iex
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

---

**Stay Safe. Code Fearlessly.**

[Report Bug](https://github.com/xadnavyaai/vectra-guard/issues) · [Request Feature](https://github.com/xadnavyaai/vectra-guard/issues) · [Documentation](https://github.com/xadnavyaai/vectra-guard)
