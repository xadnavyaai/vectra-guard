# Findings Analysis: Six AI Agent Repos

This document draws patterns from the Vectra Guard scan of six similar AI agent repos (Moltbot, Open Interpreter, AutoGPT, Huginn, LocalAGI, Aider) and discusses them in depth. Use it together with the stored artifacts for reference.

**Stored artifacts (Vectra Guard tool output format)**

| Artifact | Purpose |
|----------|---------|
| [similar-agent-findings.json](similar-agent-findings.json) | Findings in **tool format**: `{ "repos": [ ... ] }`; each repo has `path`, `code_findings` (array of `{ "file", "line", "severity", "code", "description", "remediation" }`), `code_by_severity`, `secrets_total`, `package_audits`. Same schema as `vg audit repo --output json`. |
| [similar-agent-scan-raw.txt](similar-agent-scan-raw.txt) | Full text stdout of `vg scan-security` and `vg audit repo` for all repos. Finding lines: `file:line  CODE  description  → remediation`. |
| [findings-analysis.md](findings-analysis.md) | This document: pattern discussion and recommendations; examples use tool format. |

**How to regenerate:**  
- Text: `VG_CMD="./vectra-guard" ./scripts/test-similar-agent-repos.sh 2>&1 | tee docs/reports/similar-agent-scan-raw.txt`  
- Tool-format JSON: `VG_CMD="./vectra-guard" VG_OUTPUT=json ./scripts/test-similar-agent-repos.sh` → writes `similar-agent-findings.json`.

---

## Executive Summary

Across 6 repos we saw **2,149** static code findings. The dominant pattern is **external HTTP usage** (non-localhost URLs and remote HTTP calls), followed by **environment/API access**, then **subprocess use** and **Go equivalents** in the one Go-heavy repo (LocalAGI). **Bind-all-interfaces** and **unauthenticated-access** (config) appear in a minority of locations but map directly to the kind of incidents that made Moltbot headlines. High-severity items (e.g. `PY_EXEC`, `PY_EVAL`) are rare but concentrated in code paths that execute or evaluate dynamic content.

---

## Pattern 1: External and Remote HTTP (PY_EXTERNAL_HTTP, PY_REMOTE_HTTP, GO_*)

**Counts (from scan):** PY_EXTERNAL_HTTP 1,196 | PY_REMOTE_HTTP 171 | GO_NET_HTTP 20 | GO_EXTERNAL_HTTP 12.

**What it is:** Code that uses non-localhost HTTP(S) URLs or makes remote HTTP requests (e.g. `requests.get`, `urllib`, `net/http`). The scanner flags both the presence of external URLs and generic remote HTTP usage.

**Why it matters:** Agents routinely call APIs (LLM providers, plugins, webhooks). The risk is **SSRF** (server-side request forgery) and **data exfiltration** if URLs or responses are influenced by untrusted input. Logging full URLs or response bodies can also leak tokens or PII.

**Observed patterns:**  
- Config and auth modules that read API base URLs from env or config.  
- Chat/LLM services that call external endpoints.  
- Tests and scripts that hit real or mock HTTP endpoints.  
- Browser/display components that fetch remote resources.

**Recommendations:**  
- Never build request URLs from user or agent content without allowlisting or validation.  
- Prefer fixed, well-known base URLs from config; validate redirects and response size.  
- Avoid logging request/response bodies that may contain secrets.  
- In CI, run `vg scan-security` and treat PY_EXTERNAL_HTTP / PY_REMOTE_HTTP as review points, not auto-fail, and fix or document exceptions.

---

## Pattern 2: Environment and Secret Access (PY_ENV_ACCESS, GO_ENV_READ)

**Counts:** PY_ENV_ACCESS 430 | GO_ENV_READ 37.

**What it is:** Access to environment variables or `.env`-style config (e.g. `os.getenv`, `os.environ`, `godotenv`). The scanner does not distinguish “safe” vars (e.g. `LOG_LEVEL`) from secrets (e.g. `API_KEY`).

**Why it matters:** Env vars are the standard way to inject API keys and credentials. The risk is **leaking them** via logs, error messages, debug endpoints, or responses. Moltbot-style incidents included exposed credentials and API keys in config or logs.

**Observed patterns:**  
- Auth/config modules reading API keys, auth URLs, and feature flags.  
- Test files that set or assert on env vars (often benign but noisy).  
- Core runtime and chat services that need API keys and model settings.

**Recommendations:**  
- Use a secrets manager or restricted env in production; avoid logging any variable that might hold a secret.  
- In agent code, never echo env values to the user or to external channels.  
- Allowlist known-safe vars in scanners where supported; for the rest, treat as “review for exposure.”

---

## Pattern 3: Subprocess and Shell Execution (PY_SUBPROCESS, GO_EXEC_COMMAND)

**Counts:** PY_SUBPROCESS 172 | GO_EXEC_COMMAND 2.

**What it is:** Use of `subprocess`, `os.system`, `exec.Command`, or similar to run external commands or shells.

**Why it matters:** Agents that “run code” or “execute tasks” often do so via subprocess. Unvalidated or partially validated input can lead to **command injection** and **arbitrary code execution** on the host—the kind of capability that, when combined with an exposed control plane, leads to full compromise.

**Observed patterns:**  
- Running system tools (browsers, calendar, display, terminal).  
- Scripts that invoke LLM or helper binaries.  
- Test and dev scripts (e.g. running servers or migrations).

**Recommendations:**  
- Prefer allowlisted commands and arguments; avoid passing raw user/agent text into shell.  
- Use `shell=False` and explicit argument lists; sanitize or reject dynamic parts.  
- Where possible, run agent-triggered commands in a sandbox (e.g. `vg exec`) or isolated environment.

---

## Pattern 4: Dynamic Code Execution (PY_EXEC, PY_EVAL)

**Counts:** PY_EXEC 3 | PY_EVAL 2.

**What it is:** Use of `exec()` or `eval()` on dynamic strings. The scanner flags their presence, not whether the input is trusted.

**Why it matters:** Even a small number of findings can be high impact. If the executed string is influenced by user or agent output, you get **remote code execution**. Common in “run code in a sandbox” or “evaluate expression” features.

**Observed patterns:**  
- Open Interpreter’s computer/runtime code that executes generated code.  
- Similar “run snippet” or expression-eval paths in agent runtimes.

**Recommendations:**  
- Treat every PY_EXEC/PY_EVAL as critical until proven safe (e.g. fully sandboxed, no user input).  
- Prefer restricted interpreters, AST validation, or dedicated sandbox processes over raw exec/eval.  
- Document and optionally allowlist the few justified uses.

---

## Pattern 5: Binding to All Interfaces (BIND_ALL_INTERFACES)

**Count:** 22 (Python and Go; some from config).

**What it is:** Listening on `0.0.0.0` (all interfaces) instead of `127.0.0.1`. The scanner flags this in code and in config (YAML/JSON).

**Why it matters:** This was a direct factor in Moltbot: dashboards bound to 0.0.0.0 behind reverse proxies, with “localhost” trust, ended up reachable from the internet without auth. Result: exposed control panels, credentials, and conversation data.

**Observed patterns:**  
- Core server or API entrypoints (e.g. Open Interpreter’s async_core, archived servers).  
- Local skills or plugins that start a small HTTP server.  
- Config files that set host to 0.0.0.0 for “easy LAN access.”

**Recommendations:**  
- Default to `127.0.0.1` for dev and single-user; use 0.0.0.0 only when deliberately exposing the service.  
- If binding to 0.0.0.0: require authentication and TLS; do not rely only on “we’re behind a proxy” or “only localhost connects.”  
- In CI, fail or require justification for every BIND_ALL_INTERFACES in production code paths.

---

## Pattern 6: Unauthenticated Access (UNAUTHENTICATED_ACCESS)

**Count:** 3 (config).

**What it is:** Config that disables or weakens auth (e.g. `auth: false`, `secure: false`). The scanner looks for these patterns in YAML/JSON.

**Why it matters:** Combined with bind-all or proxy trust, this leads to **unauthenticated access** to admin or API surfaces—the worst-case “unauthenticated command execution” reported in Moltbot-style incidents.

**Observed patterns:**  
- Lockfiles or config snippets that contain `auth: false` or similar (sometimes false positives).  
- Explicit “disable auth for local dev” settings that might be copied to production.

**Recommendations:**  
- Never disable auth on interfaces reachable beyond localhost.  
- Use separate configs or env flags for “local dev” vs “deployed”; CI should scan the deployed config.  
- Review every UNAUTHENTICATED_ACCESS; allowlist only documented, non-deployment configs.

---

## Cross-Cutting Themes

1. **Concentration in auth, config, and “run” paths** — Most findings sit in auth/config, chat/LLM integration, and code that runs subprocesses or dynamic code. Hardening these areas has outsized impact.

2. **Tests and scripts are noisy but not safe to ignore** — Many PY_ENV_ACCESS and PY_REMOTE_HTTP hits are in tests or one-off scripts. They still can leak secrets or normalize risky patterns; review or exclude by path, but don’t disable the rules globally.

3. **Go vs Python** — The only Go-heavy repo (LocalAGI) shows the same conceptual issues (env read, net/http, external HTTP, exec) under different rule codes. Same mitigations apply.

4. **Deployment and “localhost” trust** — The most dangerous combination is bind-all + unauthenticated or proxy-trust. Scans that include `--languages config` catch config-side issues; code-side BIND_ALL_INTERFACES completes the picture.

---

## How to Use This for Reference

- **When writing or reviewing agent code:** Use the pattern sections above as a checklist (HTTP, env, subprocess, exec/eval, bind, auth).
- **When triaging scan output:** Use [similar-agent-findings.json](similar-agent-findings.json) (tool format: `code_findings[]` with `file`, `line`, `severity`, `code`, `description`, `remediation`); use [similar-agent-scan-raw.txt](similar-agent-scan-raw.txt) to grep for specific files or codes. Text finding line format: `file:line  CODE  description  → remediation`.
- **When updating the blog or docs:** Pull totals from `similar-agent-findings.json` (each repo’s `code_findings.length`, `code_by_severity`, etc.); link to this analysis for “why it matters” and recommendations.
- **When adding or tuning rules:** Compare new rule codes against the patterns here to keep naming and severity consistent.

---

*Generated from Vectra Guard v0.4.0 scan. Findings stored in vectra-guard tool output format (audit repo JSON schema). Rule reference: [Control panel & deployment security](../control-panel-security.md#scan-security-rule-reference).*
