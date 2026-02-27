package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestRunInitCreatesGlobalConfig(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	ctx := logging.WithLogger(context.Background(), logging.NewLogger("text", os.Stdout))
	// Default (no --local, no --global) should write to ~/.config/vectra-guard/config.yaml
	if err := runInit(ctx, false, false, false, false); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	path := filepath.Join(homeDir, ".config", "vectra-guard", "config.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected global config at %s: %v", path, err)
	}
}

func TestRunInitCreatesLocalConfig(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	ctx := logging.WithLogger(context.Background(), logging.NewLogger("text", os.Stdout))
	if err := runInit(ctx, false, false, true, false); err != nil {
		t.Fatalf("runInit --local: %v", err)
	}

	path := filepath.Join(dir, ".vectra-guard", "config.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected local config at %s: %v", path, err)
	}
}

func TestRunInitRespectsTomlFlag(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	ctx := logging.WithLogger(context.Background(), logging.NewLogger("text", os.Stdout))
	if err := runInit(ctx, false, true, false, false); err != nil {
		t.Fatalf("runInit toml: %v", err)
	}

	path := filepath.Join(homeDir, ".config", "vectra-guard", "config.toml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected global TOML config at %s: %v", path, err)
	}
}

func TestRunInitRequiresForce(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	configDir := filepath.Join(homeDir, ".config", "vectra-guard")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	ctx := logging.WithLogger(context.Background(), logging.NewLogger("text", os.Stdout))
	if err := runInit(ctx, false, false, false, false); err == nil {
		t.Fatalf("expected error when file exists without --force")
	}
	if err := runInit(ctx, true, false, false, false); err != nil {
		t.Fatalf("force overwrite: %v", err)
	}
}

func chdir(t *testing.T, dir string) func() {
	t.Helper()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	return func() { _ = os.Chdir(prev) }
}
