# vectra-guard
Bash Guard for Coding Agents part of VectraHub

## Go Engineering Standards
See [GO_PRACTICES.md](GO_PRACTICES.md) for the Go engineering standards and required enforcement steps used in this repository.

## CLI usage (feature set 1)
The CLI provides fast linting and explanations for shell scripts.

### Installation
```bash
go install github.com/vectra-guard/vectra-guard@latest
```

### Commands
- `vectra-guard init` – scaffold a `vectra-guard.yaml` (or `--toml`) with starter allow/deny rules. Use `--force` to overwrite.
- `vectra-guard validate <script>` – lint a script for risky patterns and fail with a non-zero exit code on findings.
- `vectra-guard explain <script>` – summarize detected risks and suggestions without failing the command.

### Configuration discovery
- User config: `~/.config/vectra-guard/config.yaml|config.toml`
- Project config: `vectra-guard.yaml|vectra-guard.toml` in the current working directory
- Flag override: `--config /path/to/config`

Later entries override earlier ones. The `--output json` flag enables structured logging for CI pipelines.
