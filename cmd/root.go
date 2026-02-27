package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

// Version is set at build time using -ldflags
// Example: go build -ldflags "-X github.com/vectra-guard/vectra-guard/cmd.Version=v0.0.2"
var Version = "dev" // Default version for development builds

// Execute parses arguments and runs the requested subcommand.
func Execute() {
	if err := execute(os.Args[1:]); err != nil {
		code := 1
		if exitErr, ok := err.(*exitError); ok {
			code = exitErr.code
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}

func execute(args []string) error {
	root := flag.NewFlagSet("vectra-guard", flag.ContinueOnError)
	root.SetOutput(os.Stdout)
	configPath := root.String("config", "", "Path to config file (overrides auto-discovery)")
	outputFormat := root.String("output", "text", "Output format: text or json")

	if err := root.Parse(args); err != nil {
		return err
	}

	if root.NArg() < 1 {
		return usageError()
	}

	workdir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve working directory: %w", err)
	}
	cfg, loadedPaths, err := config.Load(*configPath, workdir)
	if err != nil {
		return err
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, cfg)
	ctx = config.WithConfigPaths(ctx, loadedPaths)
	ctx = logging.WithLogger(ctx, logging.NewLogger(*outputFormat, os.Stdout))

	subcommand := root.Arg(0)
	subArgs := root.Args()[1:]

	switch subcommand {
	case "scan-secrets":
		subFlags := flag.NewFlagSet("scan-secrets", flag.ContinueOnError)
		target := subFlags.String("path", ".", "Target directory or file to scan for secrets")
		allowlist := subFlags.String("allowlist", "", "Path to allowlist file of known-safe secrets")
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		return runScanSecrets(ctx, *target, *allowlist)
	case "scan-security":
		subFlags := flag.NewFlagSet("scan-security", flag.ContinueOnError)
		target := subFlags.String("path", ".", "Target directory or file to scan for security issues")
		languages := subFlags.String("languages", "", "Comma-separated languages to scan (go,python,c,config). Default: go,python,c. Use config for YAML/JSON deployment checks.")
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		return runScanSecurity(ctx, *target, *languages)
	case "help":
		topic := ""
		if len(subArgs) > 0 {
			topic = subArgs[0]
		}
		return runHelp(ctx, topic)
	case "init":
		subFlags := flag.NewFlagSet("init", flag.ContinueOnError)
		force := subFlags.Bool("force", false, "Overwrite existing config file")
		asTOML := subFlags.Bool("toml", false, "Write config as TOML instead of YAML")
		local := subFlags.Bool("local", false, "Write config under .vectra-guard/ with repo-local cache")
		global := subFlags.Bool("global", false, "Write config under ~/.config/vectra-guard/ (default)")
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		return runInit(ctx, *force, *asTOML, *local, *global)
	case "validate":
		subFlags := flag.NewFlagSet("validate", flag.ContinueOnError)
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		if subFlags.NArg() != 1 {
			return usageError()
		}
		return runValidate(ctx, subFlags.Arg(0))
	case "validate-agent":
		subFlags := flag.NewFlagSet("validate-agent", flag.ContinueOnError)
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		if subFlags.NArg() != 1 {
			return usageError()
		}
		return runValidateAgent(ctx, subFlags.Arg(0))
	case "prompt-firewall":
		subFlags := flag.NewFlagSet("prompt-firewall", flag.ContinueOnError)
		file := subFlags.String("file", "", "Read prompt text from file instead of stdin")
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		return runPromptFirewall(ctx, *file)
	case "explain":
		subFlags := flag.NewFlagSet("explain", flag.ContinueOnError)
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		if subFlags.NArg() != 1 {
			return usageError()
		}
		return runExplain(ctx, subFlags.Arg(0))
	case "exec":
		subFlags := flag.NewFlagSet("exec", flag.ContinueOnError)
		interactive := subFlags.Bool("interactive", false, "Prompt for approval on risky commands")
		sessionID := subFlags.String("session", "", "Track execution in session")
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		if subFlags.NArg() < 1 {
			return usageError()
		}
		return runExec(ctx, subFlags.Args(), *interactive, *sessionID)
	case "cve":
		if len(subArgs) < 1 {
			return usageError()
		}
		cveCmd := subArgs[0]
		cveArgs := subArgs[1:]
		switch cveCmd {
		case "sync":
			subFlags := flag.NewFlagSet("cve-sync", flag.ContinueOnError)
			target := subFlags.String("path", ".", "Target directory for manifests")
			force := subFlags.Bool("force", false, "Refresh cached entries even if fresh")
			if err := subFlags.Parse(cveArgs); err != nil {
				return err
			}
			return runCVESync(ctx, *target, *force)
		case "scan":
			subFlags := flag.NewFlagSet("cve-scan", flag.ContinueOnError)
			target := subFlags.String("path", ".", "Target directory for manifests")
			refresh := subFlags.Bool("refresh", false, "Refresh cached entries before scanning")
			if err := subFlags.Parse(cveArgs); err != nil {
				return err
			}
			return runCVEScan(ctx, *target, *refresh)
		case "explain":
			subFlags := flag.NewFlagSet("cve-explain", flag.ContinueOnError)
			ecosystem := subFlags.String("ecosystem", "npm", "Package ecosystem (npm, Go, etc.)")
			refresh := subFlags.Bool("refresh", false, "Refresh cache for the specified package")
			if err := subFlags.Parse(cveArgs); err != nil {
				return err
			}
			if subFlags.NArg() < 1 {
				return usageError()
			}
			return runCVEExplain(ctx, subFlags.Arg(0), *ecosystem, *refresh)
		default:
			return usageError()
		}
	case "restore":
		return runRestore(ctx, subArgs)
	case "lockdown":
		if len(subArgs) < 1 {
			return usageError()
		}
		lockCmd := subArgs[0]
		lockArgs := subArgs[1:]
		switch lockCmd {
		case "enable":
			reason := ""
			if len(lockArgs) > 0 {
				reason = strings.Join(lockArgs, " ")
			}
			return runLockdownEnable(ctx, reason)
		case "disable":
			return runLockdownDisable(ctx)
		case "status":
			return runLockdownStatus(ctx)
		default:
			return usageError()
		}
	case "audit":
		if len(subArgs) < 1 {
			return usageError()
		}
		auditTool := subArgs[0]
		subFlags := flag.NewFlagSet("audit", flag.ContinueOnError)
		target := subFlags.String("path", ".", "Target directory for audit")
		failOn := subFlags.Bool("fail", false, "Exit non-zero if findings exist")
		noInstall := subFlags.Bool("no-install", false, "Disable auto-install of audit dependencies")
		sessionID := subFlags.String("session", "", "Session ID for session audit")
		allSessions := subFlags.Bool("all", false, "Audit across all sessions (session tool only)")
		outputFormat := subFlags.String("output", "text", "Report format: text, markdown, or json (repo only)")
		allowlistPath := subFlags.String("allowlist", "", "Path to allowlist file for secrets (repo only)")
		ignoreGlobs := subFlags.String("ignore", "", "Comma-separated path globs to ignore in secret scan (repo only)")
		if err := subFlags.Parse(subArgs[1:]); err != nil {
			return err
		}
		var repoOpts *repoAuditOptions
		if strings.ToLower(auditTool) == "repo" {
			repoOpts = &repoAuditOptions{
				OutputFormat:  *outputFormat,
				AllowlistPath: *allowlistPath,
				IgnoreGlobs:   *ignoreGlobs,
			}
		}
		return runAudit(ctx, auditTool, *target, *failOn, !*noInstall, *sessionID, *allSessions, repoOpts)
	case "sandbox":
		if len(subArgs) < 1 {
			return usageError()
		}
		sandboxCmd := subArgs[0]
		sandboxArgs := subArgs[1:]

		switch sandboxCmd {
		case "deps":
			if len(sandboxArgs) < 1 {
				return usageError()
			}
			depsCmd := sandboxArgs[0]
			depsArgs := sandboxArgs[1:]

			switch depsCmd {
			case "install":
				subFlags := flag.NewFlagSet("sandbox-deps-install", flag.ContinueOnError)
				forceDefault := envBool("VG_FORCE")
				dryRunDefault := envBool("DRY_RUN")
				force := subFlags.Bool("force", forceDefault, "Remove conflicting binaries if needed")
				dryRun := subFlags.Bool("dry-run", dryRunDefault, "Print commands without executing")
				if err := subFlags.Parse(depsArgs); err != nil {
					return err
				}
				return runSandboxDepsInstall(ctx, *force, *dryRun)
			default:
				return usageError()
			}
		default:
			return usageError()
		}
	case "session":
		if len(subArgs) < 1 {
			return usageError()
		}
		sessionCmd := subArgs[0]
		sessionArgs := subArgs[1:]

		switch sessionCmd {
		case "start":
			subFlags := flag.NewFlagSet("session-start", flag.ContinueOnError)
			agent := subFlags.String("agent", "unknown", "Agent name")
			workspace := subFlags.String("workspace", "", "Workspace path")
			if err := subFlags.Parse(sessionArgs); err != nil {
				return err
			}
			return runSessionStart(ctx, *agent, *workspace)
		case "end":
			if len(sessionArgs) < 1 {
				return usageError()
			}
			return runSessionEnd(ctx, sessionArgs[0])
		case "list":
			return runSessionList(ctx)
		case "show":
			if len(sessionArgs) < 1 {
				return usageError()
			}
			return runSessionShow(ctx, sessionArgs[0])
		case "record":
			subFlags := flag.NewFlagSet("session-record", flag.ContinueOnError)
			command := subFlags.String("command", "", "Command string to record")
			exitCode := subFlags.Int("exit-code", 0, "Exit code of the command")
			sessionID := subFlags.String("session", "", "Session ID (optional)")
			if err := subFlags.Parse(sessionArgs); err != nil {
				return err
			}
			return runSessionRecord(ctx, *sessionID, *command, *exitCode)
		default:
			return usageError()
		}
	case "session-diff":
		subFlags := flag.NewFlagSet("session-diff", flag.ContinueOnError)
		rootPath := subFlags.String("path", "", "Limit diff to a specific root path")
		jsonOutput := subFlags.Bool("json", false, "Output diff in JSON format")
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		if subFlags.NArg() < 1 {
			return usageError()
		}
		return runSessionDiff(ctx, subFlags.Arg(0), *rootPath, *jsonOutput)
	case "serve":
		subFlags := flag.NewFlagSet("serve", flag.ContinueOnError)
		port := subFlags.Int("port", 8000, "Port to bind the local dashboard to")
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		return runServe(ctx, *port)
	case "trust":
		if len(subArgs) < 1 {
			return usageError()
		}
		trustCmd := subArgs[0]
		trustArgs := subArgs[1:]

		switch trustCmd {
		case "list":
			return runTrustList(ctx)
		case "add":
			if len(trustArgs) < 1 {
				return usageError()
			}
			subFlags := flag.NewFlagSet("trust-add", flag.ContinueOnError)
			note := subFlags.String("note", "", "Note about why this command is trusted")
			duration := subFlags.String("duration", "", "Trust duration (e.g., 24h, 7d)")
			if err := subFlags.Parse(trustArgs[1:]); err != nil {
				return err
			}
			return runTrustAdd(ctx, trustArgs[0], *note, *duration)
		case "remove":
			if len(trustArgs) < 1 {
				return usageError()
			}
			return runTrustRemove(ctx, trustArgs[0])
		case "clean":
			return runTrustClean(ctx)
		default:
			return usageError()
		}
	case "metrics":
		if len(subArgs) < 1 {
			return usageError()
		}
		metricsCmd := subArgs[0]
		metricsArgs := subArgs[1:]

		switch metricsCmd {
		case "show":
			subFlags := flag.NewFlagSet("metrics-show", flag.ContinueOnError)
			jsonOutput := subFlags.Bool("json", false, "Output in JSON format")
			if err := subFlags.Parse(metricsArgs); err != nil {
				return err
			}
			return runMetricsShow(ctx, *jsonOutput)
		case "reset":
			return runMetricsReset(ctx)
		default:
			return usageError()
		}
	case "roadmap":
		if len(subArgs) < 1 {
			return usageError()
		}
		roadmapCmd := subArgs[0]
		roadmapArgs := subArgs[1:]

		switch roadmapCmd {
		case "add":
			subFlags := flag.NewFlagSet("roadmap-add", flag.ContinueOnError)
			title := subFlags.String("title", "", "Roadmap item title")
			summary := subFlags.String("summary", "", "Roadmap item summary")
			status := subFlags.String("status", "planned", "Roadmap item status")
			tags := subFlags.String("tags", "", "Comma-separated tags")
			if err := subFlags.Parse(roadmapArgs); err != nil {
				return err
			}
			return runRoadmapAdd(ctx, *title, *summary, *status, splitCSV(*tags))
		case "list":
			subFlags := flag.NewFlagSet("roadmap-list", flag.ContinueOnError)
			status := subFlags.String("status", "", "Filter by status")
			if err := subFlags.Parse(roadmapArgs); err != nil {
				return err
			}
			return runRoadmapList(ctx, *status)
		case "show":
			if len(roadmapArgs) < 1 {
				return usageError()
			}
			return runRoadmapShow(ctx, roadmapArgs[0])
		case "status":
			if len(roadmapArgs) < 2 {
				return usageError()
			}
			return runRoadmapStatus(ctx, roadmapArgs[0], roadmapArgs[1])
		case "log":
			if len(roadmapArgs) < 1 {
				return usageError()
			}
			subFlags := flag.NewFlagSet("roadmap-log", flag.ContinueOnError)
			note := subFlags.String("note", "", "Log note")
			sessionID := subFlags.String("session", "", "Session ID to reference")
			source := subFlags.String("source", "manual", "Log source")
			if err := subFlags.Parse(roadmapArgs[1:]); err != nil {
				return err
			}
			return runRoadmapLog(ctx, roadmapArgs[0], *note, *sessionID, *source)
		default:
			return usageError()
		}
	case "context":
		if len(subArgs) < 1 {
			return usageError()
		}
		contextCmd := subArgs[0]
		contextArgs := subArgs[1:]

		switch contextCmd {
		case "summarize":
			subFlags := flag.NewFlagSet("context-summarize", flag.ContinueOnError)
			maxItems := subFlags.Int("max", 5, "Maximum number of lines or sentences")
			outputFormat := subFlags.String("output", "text", "Output format: text or json")
			since := subFlags.String("since", "", "Only process files changed since commit/time (e.g., HEAD~1, 2024-01-01)")
			if err := subFlags.Parse(contextArgs); err != nil {
				return err
			}
			if subFlags.NArg() < 2 {
				return usageError()
			}
			return runContextSummarize(ctx, subFlags.Arg(0), subFlags.Arg(1), *maxItems, *outputFormat, *since)
		default:
			return usageError()
		}
	case "seed":
		if len(subArgs) < 1 {
			return usageError()
		}
		seedCmd := subArgs[0]
		seedArgs := subArgs[1:]

		switch seedCmd {
		case "agents":
			subFlags := flag.NewFlagSet("seed-agents", flag.ContinueOnError)
			target := subFlags.String("target", ".", "Target repository directory")
			force := subFlags.Bool("force", false, "Overwrite existing files")
			targets := subFlags.String("targets", "", "Comma/space-separated targets (e.g., agents,claude,cursor)")
			list := subFlags.Bool("list", false, "List available targets and exit")
			yes := subFlags.Bool("yes", false, "Auto-confirm prompts (non-interactive mode)")
			if err := subFlags.Parse(seedArgs); err != nil {
				return err
			}
			selected := parseSeedTargets(*targets)
			return runSeedAgents(ctx, *target, *force, selected, *list, *yes)
		default:
			return usageError()
		}
	case "version":
		return runVersion(ctx, *outputFormat)
	default:
		return usageError()
	}
}

func runVersion(ctx context.Context, outputFormat string) error {
	if outputFormat == "json" {
		fmt.Printf(`{"version":"%s","name":"vectra-guard"}`+"\n", Version)
	} else {
		fmt.Printf("vectra-guard version %s\n", Version)
	}
	return nil
}

func usageError() error {
	exe, _ := os.Executable()
	name := filepath.Base(exe)
	usage := fmt.Sprintf(`usage: %s [--config FILE] [--output text|json] <command> [args]

Commands:
  scan-secrets                 Scan files for exposed secrets (regex + entropy)
  scan-security                Scan source code (Go/Python/C) for risky patterns
  init                         Initialize configuration file
  validate <script>            Validate a shell script for security issues
  validate-agent <path>        Validate agent scripts (file or directory)
  prompt-firewall [--file]     Analyze prompts for injection and jailbreak attempts
  lockdown <enable|disable|status>  Control global lockdown mode
  explain <script>             Explain security risks in a script
  exec [--interactive] <cmd>   Execute command with security validation
  audit <npm|python>           Audit package vulnerabilities (npm/pip-audit)
  sandbox deps install         Install sandbox dependencies (Docker/Podman + bubblewrap)
  session start                Start an agent session
  session end <id>             End an agent session
  session list                 List all sessions
  session show <id>            Show session details
  session-diff <id>            Show added/modified/deleted files for a session
  serve [--port]               Serve local dashboard on http://127.0.0.1:PORT
  trust list                   List trusted commands
  trust add <cmd>              Add command to trust store
  trust remove <cmd>           Remove command from trust store
  trust clean                  Clean expired entries
  metrics show [--json]        Show sandbox metrics
  metrics reset                Reset metrics
  roadmap add                  Add a roadmap item
  roadmap list                 List roadmap items
  roadmap show <id>            Show a roadmap item
  roadmap status <id> <status> Update roadmap item status
  roadmap log <id>             Append a log entry to a roadmap item
  context summarize <mode> <path>  Summarize file or repo (code, docs, advanced)
  seed agents                  Seed agent instructions into a repo
  help [topic]                 Show help for a command or topic
  version                      Show version information
`, name)
	return fmt.Errorf("%s", usage)
}
