package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestRunExplainHandlesFindings(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "danger.sh")
	if err := os.WriteFile(script, []byte("sudo rm -rf /\n"), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", os.Stdout))

	if err := runExplain(ctx, script); err != nil {
		t.Fatalf("explain should not fail even with findings: %v", err)
	}
}
