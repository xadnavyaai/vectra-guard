# Security Dashboard (`vg serve`)

## Overview

Vectra Guard includes a built-in local security dashboard that provides real-time visibility into agent sessions, sandbox execution metrics, CVE scan results, and trust store management. The dashboard runs as a local HTTP server bound exclusively to `127.0.0.1` — it is never exposed to the network.

## Quick Start

```bash
# Start the dashboard on default port 8000
vg serve

# Start on a custom port
vg serve --port 9090
```

Open `http://127.0.0.1:8000` in your browser.

## Features

### Real-Time Live Feed (SSE)

The dashboard uses Server-Sent Events to push updates to the browser in real time. Events include:

| Event Type | Description |
|---|---|
| `session_started` | A new agent session was created |
| `session_ended` | An agent session ended (includes risk score) |
| `command_executed` | A command ran through `vg exec` |
| `command_blocked` | A high/critical risk command was blocked |
| `metric_recorded` | A new sandbox metric was recorded |

The event hub polls the sessions directory and metrics file every 2 seconds for changes and broadcasts them to all connected dashboard clients.

### Session Monitoring

View all agent sessions with details:

- Session ID, agent name, workspace path
- Start/end time, risk score
- Full command history with risk levels, exit codes, and approval status

### Sandbox Metrics

Live charts and statistics for sandbox execution:

- Total executions (host vs. sandbox vs. cached)
- Execution duration averages
- Risk level distribution
- Execution history timeline

Metrics require `sandbox.enable_metrics: true` in your config:

```yaml
sandbox:
  enabled: true
  mode: always
  enable_metrics: true
```

### CVE Vulnerability Summary

Aggregated view of CVE scan results:

- Total packages scanned and vulnerabilities found
- Severity breakdown (critical, high, medium, low)
- Per-package vulnerability details with CVE IDs, CVSS scores, and summaries

Run `vg cve sync --path . && vg cve scan --path .` before starting the dashboard to populate CVE data.

### Trust Store Management

View all trusted commands:

- Command string, creation date, expiration
- Trust notes and duration

### System Status

The `/api/status` endpoint (shown in the dashboard header) reports:

- Vectra Guard version
- Guard level (auto, off, low, medium, high, paranoid)
- Sandbox configuration (enabled, mode, security level, runtime)
- Feature flags (sandbox, metrics, CVE, soft delete)
- Server uptime

## API Endpoints

The dashboard exposes a JSON API on the same port:

| Endpoint | Method | Description |
|---|---|---|
| `/` | GET | Dashboard HTML (embedded single-page app) |
| `/api/status` | GET | System status, version, feature flags |
| `/api/sessions` | GET | List all sessions |
| `/api/sessions/{id}` | GET | Get session details by ID |
| `/api/metrics` | GET | Sandbox execution metrics |
| `/api/trust` | GET | Trust store entries |
| `/api/cve` | GET | CVE scan summary |
| `/api/events` | GET | SSE stream for real-time events |

All endpoints return JSON (except `/` and `/api/events`).

## Security

The dashboard is designed with security in mind:

- **Local-only binding**: Listens on `127.0.0.1` only, never `0.0.0.0`
- **Security headers**: `X-Content-Type-Options`, `X-Frame-Options: DENY`, `Referrer-Policy: no-referrer`, strict `Content-Security-Policy`, `Permissions-Policy`
- **No CORS**: Same-origin only, no cross-origin access
- **Session ID validation**: Regex-validated to prevent path traversal (`^[a-zA-Z0-9][a-zA-Z0-9\-]*$`, max 64 chars)
- **SSE client limit**: Maximum 32 concurrent SSE connections to prevent goroutine exhaustion
- **Read timeouts**: `ReadHeaderTimeout: 5s`, `ReadTimeout: 30s`, `IdleTimeout: 120s`

## Configuration

The dashboard reads its configuration from `vectra-guard.yaml` (or your active config file). Key settings that affect the dashboard:

```yaml
# Sandbox metrics (required for metrics panel)
sandbox:
  enabled: true
  enable_metrics: true

# CVE scanning (required for CVE panel)
cve:
  enabled: true
  sources: ["osv"]

# Soft delete (shown in status)
soft_delete:
  enabled: true
```

## Architecture

```
cmd/serve.go              CLI command: parses --port, loads trust/CVE stores, starts server
internal/serve/server.go  HTTP server: routing, API handlers, security headers
internal/serve/dashboard.go  Embedded HTML dashboard (go:embed)
internal/serve/events.go  SSE event hub: polls sessions/metrics, broadcasts changes
internal/serve/static/    Static assets (dashboard.html)
```

The dashboard HTML is embedded into the binary at build time using Go's `embed` package, so no external files are needed at runtime.

## Tips

- Run `vg serve` in the background while working: `vg serve &`
- Use `vg seed agents` to see the dashboard tip in the seed output
- Combine with session tracking: start a session, run commands, then view them live in the dashboard
- Use the JSON API endpoints for programmatic access or CI integration

## Example Workflow

```bash
# 1. Enable metrics in config
# (add sandbox.enable_metrics: true to vectra-guard.yaml)

# 2. Start dashboard in background
vg serve --port 8000 &

# 3. Start a session
SESSION=$(vg session start --agent "cursor-ai")

# 4. Run commands (they appear in the live feed)
vg exec -- npm install
vg exec -- npm test

# 5. View session in dashboard at http://127.0.0.1:8000

# 6. End session
vg session end $SESSION
```
