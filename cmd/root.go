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
	default:
		return usageError()
	}
}

func usageError() error {
	exe, _ := os.Executable()
	name := filepath.Base(exe)
	return fmt.Errorf("usage: %s [--config FILE] [--output text|json] <init|validate|explain> [args]", name)
}
