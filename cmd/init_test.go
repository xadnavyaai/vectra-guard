package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestRunInitCreatesConfig(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	ctx := logging.WithLogger(context.Background(), logging.NewLogger("text", os.Stdout))
	if err := runInit(ctx, false, false); err != nil {
		t.Fatalf("runInit: %v", err)
	}

	path := filepath.Join(dir, "vectra-guard.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file at %s: %v", path, err)
	}
}

func TestRunInitRespectsTomlFlag(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	ctx := logging.WithLogger(context.Background(), logging.NewLogger("text", os.Stdout))
	if err := runInit(ctx, false, true); err != nil {
		t.Fatalf("runInit toml: %v", err)
	}

	path := filepath.Join(dir, "vectra-guard.toml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected TOML config at %s: %v", path, err)
	}
}

func TestRunInitRequiresForce(t *testing.T) {
	dir := t.TempDir()
	defer chdir(t, dir)()

	path := filepath.Join(dir, "vectra-guard.yaml")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	ctx := logging.WithLogger(context.Background(), logging.NewLogger("text", os.Stdout))
	if err := runInit(ctx, false, false); err == nil {
		t.Fatalf("expected error when file exists without --force")
	}
	if err := runInit(ctx, true, false); err != nil {
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
