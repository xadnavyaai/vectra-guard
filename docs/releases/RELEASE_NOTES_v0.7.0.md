# Vectra Guard v0.7.0 Release Notes

Release Date: 2026-07-18

## Major Features

This release focuses on **AI-agent-aware security** — moving beyond command
gating into the LLM and agent runtime surface, plus a "post-Mythos" default of
treating trust boundaries and observability data as first-class audit targets.

### LLM Trace Security Analysis (`vg llmtrace`)
A Langfuse-compatible client that pulls LLM traces and scores them for attack
patterns, then writes risk scores back to the observability provider.

- `vg llmtrace connect` — configure host + public/secret key (`LLMTRACE_*` or `LANGFUSE_*` env)
- `vg llmtrace sync [--from TS] [--dry-run]` — analyze traces since last sync
- `vg llmtrace scan --trace <id> [--write-scores]` — analyze a single trace
- `vg llmtrace watch` — continuous poll loop
- Five checks per trace: **prompt injection** (reuses the prompt firewall),
  **cost anomaly** (mean + 2σ baseline), **tool abuse** (shell/exec/network
  tool calls), **agent loop** (repeated observations), and **data exfiltration**.

### AI Agent Posture Audit (`vg audit agent-posture`)
Discovers installed AI agents in a workspace and audits their security posture,
emitting a 0–100 score.

- Agent inventory with VectraGuard-integration detection
- OAuth/API scope analysis (broad scope, committed secrets, unscoped tokens, gitignore coverage)
- Behavioral baselines from session history
- Security hygiene checks (sandbox, env protection, CVE scanning, git monitoring)

### Trust-Boundary Scanner (`vg scan boundaries`)
Scans a repository for FFI / unsafe trust-boundary crossings — the code that
heavily-reviewed-but-still-risky post-Mythos guidance says to audit first.

- Go (`unsafe.Pointer`, `reflect.*Header`, cgo `import "C"`), Rust (`unsafe`
  block/fn/impl/trait), Java (JNI `native`), C (`dlopen`/`dlsym`/manual mapping),
  Python (`ctypes`/`cffi`), and Node (ffi-napi / N-API bindings)
- Severity-ranked findings with directed-review guidance

### Behavioral Session Profiling (`vg behavioral profile`)
Builds a diffable action-graph (nodes/edges) from a session's audit records so
an anomalous run can be compared against prior runs on similar tasks.

### CVE Cache Freshness (`vg cve freshness`)
Reports the age and sources of the local CVE cache so CI can fail on stale data
(`vg cve scan --max-age-days N`).

### Expanded Prompt Firewall
The `prompt-firewall` engine gains encoded-payload, obfuscation, and
sandwich-attack detection, backed by a labeled benign/malicious corpus for
regression testing.

## Testing

- New smoke/unit tests for the previously untested `llmtrace` and `agentposture`
  engines, plus command-level tests for scan-boundaries, behavioral, and cve
  freshness.
- Full suite verified green on a clean Linux VM prior to release (the only
  environment-dependent failure, `TestRuntimeSelection/explicit_docker`,
  requires Docker and passes in CI).

## Documentation

- `docs/ai-agent-threat-model.md` — stage-by-stage AI agent threat mapping
- `docs/prompt-firewall.md` — prompt firewall usage and detection classes
- Seed templates (Claude, Agents, Codex) refreshed with the new command workflows
- OpenClaw plugin manifest and narrative blog posts

## Installation

**macOS & Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/xadnavyaai/vectra-guard/main/scripts/install-windows.ps1 | iex
```

**Quick start (AI-agent security):**
```bash
vg scan boundaries --path .          # audit FFI / unsafe trust boundaries
vg audit agent-posture               # score installed AI agents
vg llmtrace connect --host <url>     # wire up LLM trace analysis
vg cve freshness                     # check CVE cache age
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

- **5 new security modules** (llmtrace, agentposture, boundaries, behavioral, cve-freshness)
- **9 new test files**; ~9,400 insertions across 66 files since v0.6.0
- **6 platform binaries** (3 OS × 2 architectures)

---

**Stay Safe. Code Fearlessly.**

[Report Bug](https://github.com/xadnavyaai/vectra-guard/issues) · [Request Feature](https://github.com/xadnavyaai/vectra-guard/issues) · [Documentation](https://github.com/xadnavyaai/vectra-guard)
