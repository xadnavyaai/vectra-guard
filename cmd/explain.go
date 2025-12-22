package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/vectra-guard/vectra-guard/internal/analyzer"
	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func runExplain(ctx context.Context, scriptPath string) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}

	findings := analyzer.AnalyzeScript(scriptPath, content, cfg.Policies)
	if len(findings) == 0 {
		logger.Info("no obvious risks detected", map[string]any{"path": scriptPath})
		return nil
	}

	for _, f := range findings {
		logger.Info("risk summary", map[string]any{
			"path":           scriptPath,
			"line":           f.Line,
			"code":           f.Code,
			"severity":       f.Severity,
			"description":    f.Description,
			"recommendation": f.Recommendation,
		})
	}
	return nil
}
