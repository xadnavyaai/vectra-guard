# vectra-guard Feature Roadmap

This roadmap outlines incremental, developer-friendly milestones for building **vectra-guard**, starting with simple CLI capabilities and progressing toward sandbox orchestration and local LLM guard rails. Each phase is scoped to fit typical development workflows: small, testable steps with minimal surprises.

## 1) CLI Foundations (fast wins)
- **Command parser skeleton**: Introduce a Go `main` package with `cobra` or `urfave/cli` to standardize flags and subcommands.
- **Init command**: Scaffold a `vectra-guard init` to create a local config file (YAML/TOML), default policies, and example allow/deny rules.
- **Validate command**: Provide `vectra-guard validate <script>` to lint bash scripts for obvious red flags (e.g., `rm -rf /`, unguarded `sudo`).
- **Explain command**: Add `vectra-guard explain <script>` to surface a concise risk summary and suggested mitigations.
- **Config awareness**: Respect `~/.config/vectra-guard/config.(yaml|toml)` and a project-local config, merging with clear precedence.
- **Logging & exit codes**: Structured JSON logs (human-readable option), consistent non-zero exit codes on violations.

## 2) Sandbox-Oriented Execution (fast/easy orchestration)
- **Dry-run sandbox flag**: Add `--sandbox` to execute scripts inside a lightweight environment (e.g., `firejail`/`bwrap`/`chroot` depending on availability) with minimal setup.
- **File system guards**: Mount a temp overlay with read-only host paths and writable scratch space; block device and proc writes by default.
- **Network guards**: Default to no egress; allow opt-in host allowlist for specific domains/ports.
- **Resource ceilings**: Set quick limits (CPU, memory, wall clock) using OS primitives (`ulimit`/`cgroups` where available) to keep orchestration simple.
- **Audit trail**: Emit a per-run report (JSON) capturing commands executed, blocked operations, and resource usage.
- **Graceful fallback**: Detect missing sandbox tooling and fall back to simulation mode with clear warnings instead of failing hard.

## 3) Local LLM Guard Rails (Ollama small models)
- **Local model choice**: Integrate a small Ollama-served model (e.g., `llama2:7b` or similar) for latency-friendly, offline analysis.
- **Policy prompt templates**: Maintain prompt templates in version control to explain rules, risky patterns, and desired guard-rail behavior.
- **Pre-execution review**: Use the LLM to critique scripts before running, returning a structured risk report with confidence scores.
- **Inline suggestions**: Provide actionable rewrites or command substitutions (e.g., replace destructive commands with safer alternatives).
- **Post-run reflection**: Summarize sandbox audit logs to highlight anomalies and propose tightened policies.
- **Opt-out & privacy**: Default to local inference only; no external calls. Make LLM usage a flag (`--llm-review`) and document data handling.

## 4) Testing & Developer Workflow Alignment
- **Unit tests first**: Add table-driven tests for CLI parsing, config merging, and policy checks.
- **Integration harness**: Minimal test scripts that run in sandbox mode to verify guard rails without heavy setup.
- **CI-friendly**: Fast checks (lint, unit tests) under a few minutes; sandbox tests marked so they can be skipped or run in containers.
- **Docs as code**: Keep usage examples in README snippets and ensure they are exercised via smoke tests where possible.
- **Contribution guide**: Document the standard workflow (format, lint, test, sandbox smoke) to keep contributors aligned.

---
**Principles**: Prefer small, observable iterations; prioritize defaults that fail safe; and keep tooling lightweight to match common developer workflows.
