package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestRunValidateReturnsExitErrorOnFindings(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "danger.sh")
	if err := os.WriteFile(script, []byte("rm -rf /\n"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	err := runValidate(ctx, script)
	if err == nil {
		t.Fatalf("expected exit error for findings")
	}
	if exitErr, ok := err.(*exitError); !ok || exitErr.code != 2 {
		t.Fatalf("expected exit error code 2, got %#v", err)
	}
}

func TestRunValidateSucceedsWhenClean(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "ok.sh")
	if err := os.WriteFile(script, []byte("echo safe\n"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	if err := runValidate(ctx, script); err != nil {
		t.Fatalf("expected clean validation, got %v", err)
	}
}
