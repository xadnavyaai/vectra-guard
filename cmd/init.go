package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func runInit(ctx context.Context, force bool, tomlFormat bool) error {
	logger := logging.FromContext(ctx)
	cfg := config.DefaultConfig()
	cfg.Policies.Allowlist = []string{"echo \"safe\"", "touch /tmp/ok"}
	cfg.Policies.Denylist = []string{"rm -rf /", "sudo ", "mkfs", "dd if="}

	workdir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve cwd: %w", err)
	}
	target := filepath.Join(workdir, "vectra-guard.yaml")
	if tomlFormat {
		target = filepath.Join(workdir, "vectra-guard.toml")
	}

	if _, err := os.Stat(target); err == nil && !force {
		return fmt.Errorf("config already exists at %s (use --force to overwrite)", target)
	}

	var content string
	if tomlFormat {
		content, err = encodeTOML(cfg)
	} else {
		content, err = encodeYAML(cfg)
	}
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	logger.Info("config initialized", map[string]any{"path": target})
	return nil
}

func encodeTOML(cfg config.Config) (string, error) {
	builder := &strings.Builder{}
	builder.WriteString("[logging]\n")
	builder.WriteString(fmt.Sprintf("format = \"%s\"\n\n", cfg.Logging.Format))
	builder.WriteString("[policies]\n")
	builder.WriteString(formatArray("allowlist", cfg.Policies.Allowlist))
	builder.WriteString(formatArray("denylist", cfg.Policies.Denylist))
	return builder.String(), nil
}

func encodeYAML(cfg config.Config) (string, error) {
	builder := &strings.Builder{}
	builder.WriteString("logging:\n")
	builder.WriteString(fmt.Sprintf("  format: %s\n", cfg.Logging.Format))
	builder.WriteString("policies:\n")
	builder.WriteString("  allowlist:\n")
	for _, item := range cfg.Policies.Allowlist {
		builder.WriteString(fmt.Sprintf("    - %s\n", item))
	}
	builder.WriteString("  denylist:\n")
	for _, item := range cfg.Policies.Denylist {
		builder.WriteString(fmt.Sprintf("    - %s\n", item))
	}
	return builder.String(), nil
}

func formatArray(key string, values []string) string {
	var quoted []string
	for _, v := range values {
		quoted = append(quoted, fmt.Sprintf("\"%s\"", v))
	}
	return fmt.Sprintf("%s = [%s]\n", key, strings.Join(quoted, ", "))
}
