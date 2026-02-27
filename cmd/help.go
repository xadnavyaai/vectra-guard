package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
)

func runHelp(ctx context.Context, topic string) error {
	topic = strings.ToLower(strings.TrimSpace(topic))
	switch topic {
	case "", "all":
		return printHelp(`Vectra Guard help:

  vg help [topic]

Topics:
  context   Summarize code/docs for navigation
  roadmap   Track repo-specific planning items
  init      Initialize config (including repo-local)
  sandbox   Install sandbox dependencies
  seed      Seed agent instructions into a repo
  audit     Audit npm/pip packages for vulnerabilities
  cve       Scan manifests using local CVE cache
`)
	case "context":
		return printHelp(`Context summaries:

  vg context summarize code <file|directory> [--max 5] [--output text|json] [--since <commit|date>]
  vg context summarize docs <file|directory> [--max 3] [--output text|json] [--since <commit|date>]
  vg context summarize advanced <file|directory> [--max 3] [--output text|json] [--since <commit|date>]

Summarize a single file or entire repository. When given a directory, processes all
relevant files and groups results by file. Results are cached in .vectra-guard/cache/
for faster subsequent runs.

Advanced mode parses Go files and surfaces high-impact functions with call-graph signals.

Options:
  --max N          Maximum number of summary lines per file (default: 5)
  --output FORMAT  Output format: text (default) or json
  --since REF      Only process files changed since commit/date (e.g., HEAD~1, 2024-01-01)

Examples:
  vg context summarize code cmd/root.go
  vg context summarize code . --output json              # JSON output for agents
  vg context summarize code . --since HEAD~1             # Only changed files
  vg context summarize code . --since 2024-01-01         # Since date
  vg context summarize advanced internal/ --max 10        # More summary lines
`)
	case "roadmap":
		return printHelp(`Roadmap planning:

  vg roadmap add --title "Title" --summary "Details" --tags "agent,plan"
  vg roadmap list [--status planned]
  vg roadmap show <id>
  vg roadmap status <id> <status>
  vg roadmap log <id> --note "Update" --session <session-id>

Roadmaps are stored per workspace under ~/.vectra-guard/roadmaps.
`)
	case "init":
		return printHelp(`Initialization:

  vg init            # writes ~/.config/vectra-guard/config.yaml (global, default)
  vg init --local    # writes .vectra-guard/config.yaml in cwd (repo-scoped)
  vg init --toml     # writes config as TOML instead of YAML
  vg init --force    # overwrite existing config

Global init (default) writes to ~/.config/vectra-guard/config.yaml.
Local init creates a repo-scoped cache directory at .vectra-guard/cache.
Config is loaded in order: global -> project -> local (later files override).
`)
	case "sandbox":
		return printHelp(`Sandbox dependencies:

  vg sandbox deps install [--force] [--dry-run]

Install Docker/Podman + bubblewrap for sandboxing. Use --force to remove
conflicting binaries (e.g., /usr/local/bin/hub-tool) on macOS. Use --dry-run
or DRY_RUN=1 to preview commands.
`)
	case "seed":
		return printHelp(`Seed agent instructions:

  vg seed agents [--target .] [--force] [--targets "agents,cursor"]
  vg seed agents --list

Defaults to seeding only AGENTS.md unless --targets is provided.
Use --list to see available targets.

Available targets:
  agents    AGENTS.md (universal agent instructions)
  claude    CLAUDE.md (Claude Code / Claude)
  codex     CODEX.md (OpenAI Codex)
  copilot   .github/copilot-instructions.md
  cursor    .cursor/rules/vectra-guard.md
  windsurf  .windsurf/rules.md
  vscode    .vscode/vectra-guard.instructions.md
  openclaw  .openclaw/plugins/vectraguard.yaml (OpenClaw plugin config)

Examples:
  vg seed agents --target . --targets "agents,claude"
  vg seed agents --target . --targets "openclaw"
  vg seed agents --target . --targets "agents,cursor,openclaw"
  vg seed agents --force --targets "openclaw"   # Overwrite existing
`)
	case "audit":
		return printHelp(`Package auditing:

  vg audit npm [--path .] [--fail] [--no-install]
  vg audit python [--path .] [--fail] [--no-install]

Uses npm audit and pip-audit to surface known vulnerabilities. If --fail is
set, exits non-zero when findings exist. By default, missing tools are
auto-installed; use --no-install to disable.
`)
	case "cve":
		return printHelp(`CVE awareness:

  vg cve sync [--path .] [--force]
  vg cve scan [--path .] [--refresh]
  vg cve explain <name[@version]> [--ecosystem npm] [--refresh]

Sync fetches CVE data into a local cache. Scan analyzes manifests/lockfiles and
reports known vulnerabilities. Explain shows cached advisories for a package.
`)
	default:
		return printHelp(fmt.Sprintf("Unknown help topic: %s\n", topic))
	}
}

func printHelp(message string) error {
	_, err := fmt.Fprint(os.Stdout, message)
	return err
}
