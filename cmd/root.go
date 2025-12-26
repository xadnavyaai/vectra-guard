package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
	cfg, _, err := config.Load(*configPath, workdir)
	if err != nil {
		return err
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, cfg)
	ctx = logging.WithLogger(ctx, logging.NewLogger(*outputFormat, os.Stdout))

	subcommand := root.Arg(0)
	subArgs := root.Args()[1:]

	switch subcommand {
	case "init":
		subFlags := flag.NewFlagSet("init", flag.ContinueOnError)
		force := subFlags.Bool("force", false, "Overwrite existing config file")
		asTOML := subFlags.Bool("toml", false, "Write config as TOML instead of YAML")
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		return runInit(ctx, *force, *asTOML)
	case "validate":
		subFlags := flag.NewFlagSet("validate", flag.ContinueOnError)
		if err := subFlags.Parse(subArgs); err != nil {
			return err
		}
		if subFlags.NArg() != 1 {
			return usageError()
		}
		return runValidate(ctx, subFlags.Arg(0))
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
		default:
			return usageError()
		}
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
  init                         Initialize configuration file
  validate <script>            Validate a shell script for security issues
  explain <script>             Explain security risks in a script
  exec [--interactive] <cmd>   Execute command with security validation
  session start                Start an agent session
  session end <id>             End an agent session
  session list                 List all sessions
  session show <id>            Show session details
  trust list                   List trusted commands
  trust add <cmd>              Add command to trust store
  trust remove <cmd>           Remove command from trust store
  trust clean                  Clean expired entries
  metrics show [--json]        Show sandbox metrics
  metrics reset                Reset metrics
  version                      Show version information
`, name)
	return fmt.Errorf("%s", usage)
}
