# Reports and Findings

This directory stores scan outputs and analysis from Vectra Guard runs on similar AI agent repos. **Findings are stored in Vectra Guard tool output format** (same schema as `vg audit repo --output json`).

## Vectra Guard output format

- **Text (default):** `vg scan-security` emits `[WARN] security finding file=... line=... language=... severity=... code=... description=...`. `vg audit repo` emits a summary line then one line per finding: `file:line  CODE  description  → remediation`.
- **JSON:** `vg audit repo --path <dir> --output json` emits a single object:
  - `path` (string)
  - `code_findings` (array of `{ "file", "line", "severity", "code", "description", "remediation" }`)
  - `code_by_severity` (object)
  - `secrets_total` (number)
  - `package_audits` (array)

Stored findings use this schema so tools and scripts can consume them the same way as live `vg` output.

## Stored artifacts

| File | Description |
|------|-------------|
| **similar-agent-findings.json** | Findings in **vectra-guard tool format**: `{ "repos": [ ... ] }` where each element is one `vg audit repo --output json` object (path, code_findings, code_by_severity, secrets_total, package_audits). Generated with `VG_OUTPUT=json`. |
| **similar-agent-scan-raw.txt** | Full stdout of `vg scan-security` and `vg audit repo` (text) for all repos. Grep by `code=`, repo path, or severity. |
| **findings-summary.json** | Placeholder/schema for combined tool-format output; use **similar-agent-findings.json** when generated. |
| **findings-analysis.md** | Pattern-by-pattern discussion; examples use tool format (text and JSON keys). |

## Cross-validation (rule-code counts)

Canonical counts for the six-repo scan come from **similar-agent-scan-raw.txt** (grep by `code=...`). Validated totals:

| Rule | Count | Source |
|------|-------|--------|
| PY_EXTERNAL_HTTP | 1,196 | similar-agent-scan-raw.txt |
| PY_REMOTE_HTTP | 171 | similar-agent-scan-raw.txt |
| BIND_ALL_INTERFACES | 22 | similar-agent-scan-raw.txt |

Secrets totals in the blog table are from `vg audit repo` with improved detection: lockfiles skipped, and ENTROPY_CANDIDATE only when the line has secret context (token/api_key/secret etc.) plus FP filters (paths, slugs, UUIDs, identifiers), so counts are lower and more actionable (~7.5K total across six repos).

## Scripts

- **Verify secret findings:** From repo root, `go run scripts/verify-secret-findings.go` samples findings per repo, verifies the match on the reported line, and classifies TRUE_ISSUE vs FP. Requires `test-workspaces/` with cloned repos.
- **Binary E2E:** `./scripts/test-binary-e2e.sh ./vectra-guard` runs the built binary through scan-secrets (known patterns, context, lockfile skip), scan-security (fixture, comment skip), audit repo, validate, and init. See [Control panel security — Detection behavior](../control-panel-security.md#detection-behavior-scan-secrets-and-scan-security).

## How to refer to them

- **From docs or blog:** Link to `findings-analysis.md` for narrative; link to [Secret findings examples](../blog/secret-findings-examples.md) for what secret findings mean and how detection was improved; link to `similar-agent-findings.json` for machine-readable findings in tool format.
- **From code or CI:** Read `similar-agent-findings.json` (same schema as `vg audit repo --output json`); parse `similar-agent-scan-raw.txt` for raw text.
- **Rule reference:** [Control panel & deployment security](../control-panel-security.md#scan-security-rule-reference).

## Regenerating

**Text report (human-readable):**
```bash
./scripts/test-similar-agent-repos.sh clone-all
VG_CMD="./vectra-guard" ./scripts/test-similar-agent-repos.sh 2>&1 | tee docs/reports/similar-agent-scan-raw.txt
```

**Findings in tool format (JSON):**
```bash
VG_CMD="./vectra-guard" VG_OUTPUT=json ./scripts/test-similar-agent-repos.sh
# Writes docs/reports/similar-agent-findings.json (requires jq to merge repos)
```
