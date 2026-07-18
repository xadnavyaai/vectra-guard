package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/boundaries"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func runScanBoundaries(ctx context.Context, targetPath string, languagesCSV string) error {
	logger := logging.FromContext(ctx)

	opts := boundaries.Options{
		Languages: make(map[string]bool),
	}
	if languagesCSV != "" {
		for _, part := range strings.Split(languagesCSV, ",") {
			lang := strings.TrimSpace(strings.ToLower(part))
			if lang == "" {
				continue
			}
			opts.Languages[lang] = true
		}
	}

	findings, err := boundaries.ScanPath(targetPath, opts)
	if err != nil {
		return err
	}

	if len(findings) == 0 {
		logger.Info("boundary scan completed (no findings)", map[string]any{
			"path": targetPath,
		})
		return nil
	}

	for _, f := range findings {
		logger.Warn("boundary finding", map[string]any{
			"file":        f.File,
			"line":        f.Line,
			"language":    f.Language,
			"severity":    f.Severity,
			"code":        f.Code,
			"description": f.Description,
		})
	}

	// Boundary findings are informational by default — they flag audit
	// targets, not bugs. We still exit non-zero so CI can gate.
	return &exitError{
		message: fmt.Sprintf("%d boundary crossing(s) require directed review", len(findings)),
		code:    2,
	}
}
