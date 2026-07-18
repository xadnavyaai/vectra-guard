# AI Agent Threat Model — Vectra Guard Reference

This page maps the stages of an AI agent's execution envelope to the
threats that live at each stage, the Vectra Guard feature that addresses
each threat, the CLI command to run, and the config knob that governs it.

It is a long-lived reference page, not a blog post. The blog posts in
[`docs/blog/`](blog/) link back here for context; update this page when
the stage list or the feature set changes.

## How to use this page

Read the table top-to-bottom. Each row represents a stage in the agent's
execution envelope — from untrusted text coming in, through tool calls,
execution, filesystem changes, network activity, and finally the audit
trail. Pick the rows that match your agent's shape, and wire the
corresponding command into your development loop or CI.

The post-Mythos mental model: **treat every stage as an independent
layer**. No single stage is a complete defense, and no single stage is
optional. The prompt firewall does not replace the sandbox. The sandbox
does not replace CVE scanning. The session audit does not replace any of
them — it tells you what happened after the fact so you can diff it.

## Threat model table

| Stage                      | What can go wrong                                                                                                                                                   | Vectra Guard feature                  | CLI command                                                                 | Config knob                                                                 |
|----------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------|-----------------------------------------------------------------------------|-----------------------------------------------------------------------------|
| **Input — untrusted text** | Prompt injection embedded in web pages, files, tool results, or email bodies redirects an otherwise well-aligned agent to the attacker's goals.                    | `prompt-firewall`                     | `vg prompt-firewall --file <path>` / `cat … \| vg prompt-firewall`          | `prompt_firewall: { enabled: true, threshold: high }`                       |
| **Tool call — intent**     | The model decides to invoke a tool (shell, HTTP, filesystem) for the wrong reason, or with attacker-controlled arguments smuggled through a previous tool result.   | `exec` risk analyzer                  | `vg exec -- <command>`                                                      | `exec: { block_on_critical: true }`                                         |
| **Exec — blast radius**    | A malicious or mistaken command runs against the host filesystem, network, or system state. Memory-safe languages still reach unsafe code inside `unsafe {}` / FFI. | Sandbox + seccomp, `scan boundaries`  | `vg exec -- <command>` (sandboxed) / `vg scan boundaries --path .`          | `sandbox: { enabled: true, mode: always, enable_cache: true }`              |
| **Filesystem — writes**    | The agent deletes, overwrites, or rewrites files you did not intend it to touch. Mistakes accelerate when model capability increases.                               | Soft delete (`vg restore`)            | `vg restore list` / `vg restore <id>`                                       | `soft_delete: { enabled: true, max_age_days: 30, protect_git: true }`       |
| **Dependencies — install** | The agent runs `npm install` / `pip install` / `go get` and pulls a known-vulnerable package that shipped an advisory days ago.                                     | CVE scanner + cache freshness         | `vg cve sync --path .` / `vg cve scan --path . --max-age-days 7`            | `cve: { enabled: true, sources: ["osv"], update_interval_hours: 24 }`       |
| **Network — egress**       | An injected payload drives the agent to exfiltrate data via a Markdown image tag, a `curl` call, or a `fetch` embedded in its output.                               | `prompt-firewall` (exfil detectors), sandbox egress rules | `vg prompt-firewall --file <path>`                                          | `prompt_firewall: { detectors: { exfiltration_channel: true } }`            |
| **Session — behavior**     | The agent's overall action sequence drifts from what similar runs normally look like. No single step is malicious; the shape of the run is the signal.              | `behavioral profile` (session action graph) | `vg behavioral profile --session <id> --output json`                        | `behavioral: { enabled: true }` *(v1: always on when a session exists)*     |
| **Audit — after the fact** | You need to answer "what did my agent actually do" weeks later, for compliance, debugging, or a post-incident review.                                                | Session audit + session-diff          | `vg session show <id>` / `vg session-diff <a> <b>` / `vg audit session`    | `session: { enabled: true, persist: true }`                                 |
| **Repo — trust boundaries**| Memory-safe language gives a false sense of security. Real bugs live at FFI hand-offs (`unsafe {}`, cgo, ctypes, N-API, JNI, `dlsym`) that the language cannot audit for you. | `scan boundaries`                     | `vg scan boundaries --path .`                                               | *(no config knob; run as a scheduled check in CI)*                          |

## Layering order

If you can only adopt the layers one at a time, adopt them in this
order. Earlier layers are cheaper, have a smaller blast radius, and
block more attacks per unit of developer friction.

1. **`prompt-firewall`** at the input gateway. Cheapest integration,
   highest per-dollar leverage. Blocks the attack before it reaches the
   model's context window.
2. **`sandbox: { mode: always }`** in your config. Free to turn on,
   silently contains blast radius the first time it matters.
3. **`vg cve scan --max-age-days 7`** in CI, before every dependency
   install step. Fails the run if the cache is stale, so you cannot
   silently skip the check.
4. **`vg scan boundaries`** scheduled as a weekly or per-PR job. Not
   about finding bugs — about knowing where the audit has to happen if
   one is reported.
5. **`vg session`** + **`vg behavioral profile`** on every non-trivial
   agent run. The audit trail does not need to be watched in real time
   to be useful; you need it when something looks off a week later.
6. **`vg restore`** turned on by default. You will not notice it until
   an agent deletes the wrong file, and then you will notice it very
   quickly.

## What this page does not cover

Vectra Guard is deliberately narrow. It does not replace:

- **Network-layer controls** (firewall rules, egress proxies, DNS
  filtering). Those belong in your infrastructure layer, not in an
  agent wrapper.
- **Code review of agent-authored changes.** The audit trail shows you
  what happened; a human still has to decide whether it was right.
- **Model alignment.** A well-aligned model with a bad input still
  produces a bad output. That is what the prompt firewall is for — but
  the prompt firewall is a keyword + heuristic layer, not a substitute
  for model-side safety.
- **Vulnerability research.** Vectra Guard does not find bugs in your
  code. It surfaces the places (`unsafe {}`, FFI, `unsafe.Pointer`)
  where bugs, if they exist, will live. Directed semantic search for
  unknown bugs is what Project Glasswing is for.
- **Secret scanning at scale.** `vg scan-secrets` handles the common
  developer-laptop cases; for full-org secret management, use a
  dedicated secret scanner and a secret broker.

## Related reading

- [`docs/prompt-firewall.md`](prompt-firewall.md) — full detector list
  and risk-score thresholds.
- [`docs/cve-awareness.md`](cve-awareness.md) — CVE sync + scan design
  notes, including cache freshness.
- [`docs/control-panel-security.md`](control-panel-security.md) — CI
  integration recipes for `scan-secrets` and `scan-security`.
- [`docs/soft-delete.md`](soft-delete.md) — soft delete / restore
  design.
- [`docs/blog/vectra-guard-in-a-mythos-world.md`](blog/vectra-guard-in-a-mythos-world.md)
  — narrative context for why each of these layers matters more in
  April 2026 than it did a month ago.
- [`docs/blog/prompt-firewall-after-mythos.md`](blog/prompt-firewall-after-mythos.md)
  — technical deep-dive on the v2 detector suite.
