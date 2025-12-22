package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/vectra-guard/vectra-guard/internal/analyzer"
	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func runValidate(ctx context.Context, scriptPath string) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("read script: %w", err)
	}

	findings := analyzer.AnalyzeScript(scriptPath, content, cfg.Policies)
	if len(findings) == 0 {
		logger.Info("script validated successfully", map[string]any{"path": scriptPath})
		return nil
	}

	// Surface findings and fail the command.
	for _, f := range findings {
		logger.Warn("finding", map[string]any{
			"path":           scriptPath,
			"line":           f.Line,
			"code":           f.Code,
			"severity":       f.Severity,
			"description":    f.Description,
			"recommendation": f.Recommendation,
		})
	}

	return &exitError{message: "violations detected", code: 2}
}

type exitError struct {
	message string
	code    int
}

func (e *exitError) Error() string {
	return e.message
}

func (e *exitError) Is(target error) bool {
	var other *exitError
	if errors.As(target, &other) {
		return e.code == other.code
	}
	return false
}
