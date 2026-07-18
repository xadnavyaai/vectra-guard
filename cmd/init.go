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

func runInit(ctx context.Context, force bool, tomlFormat bool, local bool, global bool) error {
	logger := logging.FromContext(ctx)
	cfg := config.DefaultConfig()
	cfg.Policies.Allowlist = []string{"echo \"safe\"", "touch /tmp/ok"}
	cfg.Policies.Denylist = []string{"rm -rf /", "sudo ", "mkfs", "dd if="}

	workdir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve cwd: %w", err)
	}

	// Default to global when neither --local nor --global is specified
	useGlobal := global || !local

	var target string
	if local {
		target = filepath.Join(workdir, ".vectra-guard", "config.yaml")
		if tomlFormat {
			target = filepath.Join(workdir, ".vectra-guard", "config.toml")
		}
		cacheDir := filepath.Join(workdir, ".vectra-guard", "cache")
		cfg.Sandbox.CacheDir = cacheDir
		cfg.Sandbox.WorkspaceDir = workdir
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create local config directory: %w", err)
		}
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return fmt.Errorf("create local cache directory: %w", err)
		}
	} else if useGlobal {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolve home directory: %w", err)
		}
		configDir := filepath.Join(home, ".config", "vectra-guard")
		target = filepath.Join(configDir, "config.yaml")
		if tomlFormat {
			target = filepath.Join(configDir, "config.toml")
		}
		if err := os.MkdirAll(configDir, 0o755); err != nil {
			return fmt.Errorf("create global config directory: %w", err)
		}
	} else {
		target = filepath.Join(workdir, "vectra-guard.yaml")
		if tomlFormat {
			target = filepath.Join(workdir, "vectra-guard.toml")
		}
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
	builder.WriteString("\n[sandbox]\n")
	builder.WriteString(fmt.Sprintf("cache_dir = \"%s\"\n", cfg.Sandbox.CacheDir))
	builder.WriteString(fmt.Sprintf("workspace_dir = \"%s\"\n", cfg.Sandbox.WorkspaceDir))
	builder.WriteString("\n[cve]\n")
	builder.WriteString(fmt.Sprintf("enabled = %t\n", cfg.CVE.Enabled))
	builder.WriteString(fmt.Sprintf("cache_dir = \"%s\"\n", cfg.CVE.CacheDir))
	builder.WriteString(fmt.Sprintf("update_interval_hours = %d\n", cfg.CVE.UpdateIntervalHours))
	builder.WriteString(formatArray("sources", cfg.CVE.Sources))
	builder.WriteString("\n[env_protection]\n")
	builder.WriteString(fmt.Sprintf("enabled = %t\n", cfg.EnvProtection.Enabled))
	builder.WriteString(fmt.Sprintf("masking_mode = \"%s\"  # Options: full, partial, hash, fake\n", cfg.EnvProtection.MaskingMode))
	builder.WriteString(fmt.Sprintf("block_env_access = %t  # Block printenv, env commands\n", cfg.EnvProtection.BlockEnvAccess))
	builder.WriteString(fmt.Sprintf("block_dotenv_read = %t  # Block reading .env files\n", cfg.EnvProtection.BlockDotenvRead))
	builder.WriteString(formatArray("allow_read_vars", cfg.EnvProtection.AllowReadVars))
	builder.WriteString("\n[llmtrace]\n")
	builder.WriteString(fmt.Sprintf("enabled = %t\n", cfg.LLMTrace.Enabled))
	builder.WriteString(fmt.Sprintf("host = \"%s\"\n", cfg.LLMTrace.Host))
	builder.WriteString("# public_key = \"pk-lf-...\"\n")
	builder.WriteString("# secret_key = \"sk-lf-...\"\n")
	builder.WriteString(fmt.Sprintf("poll_interval_seconds = %d\n", cfg.LLMTrace.PollInterval))
	builder.WriteString(fmt.Sprintf("batch_size = %d\n", cfg.LLMTrace.BatchSize))
	builder.WriteString(fmt.Sprintf("score_prefix = \"%s\"\n", cfg.LLMTrace.ScorePrefix))
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
	builder.WriteString("sandbox:\n")
	builder.WriteString(fmt.Sprintf("  cache_dir: %s\n", cfg.Sandbox.CacheDir))
	builder.WriteString(fmt.Sprintf("  workspace_dir: %s\n", cfg.Sandbox.WorkspaceDir))
	builder.WriteString("cve:\n")
	builder.WriteString(fmt.Sprintf("  enabled: %t\n", cfg.CVE.Enabled))
	builder.WriteString(fmt.Sprintf("  cache_dir: %s\n", cfg.CVE.CacheDir))
	builder.WriteString(fmt.Sprintf("  update_interval_hours: %d\n", cfg.CVE.UpdateIntervalHours))
	builder.WriteString("  sources:\n")
	for _, item := range cfg.CVE.Sources {
		builder.WriteString(fmt.Sprintf("    - %s\n", item))
	}
	builder.WriteString("env_protection:\n")
	builder.WriteString(fmt.Sprintf("  enabled: %t\n", cfg.EnvProtection.Enabled))
	builder.WriteString(fmt.Sprintf("  masking_mode: %s  # Options: full, partial, hash, fake\n", cfg.EnvProtection.MaskingMode))
	builder.WriteString(fmt.Sprintf("  block_env_access: %t  # Block printenv, env commands\n", cfg.EnvProtection.BlockEnvAccess))
	builder.WriteString(fmt.Sprintf("  block_dotenv_read: %t  # Block reading .env files\n", cfg.EnvProtection.BlockDotenvRead))
	builder.WriteString("  allow_read_vars:\n")
	for _, item := range cfg.EnvProtection.AllowReadVars {
		builder.WriteString(fmt.Sprintf("    - %s\n", item))
	}
	builder.WriteString("llmtrace:\n")
	builder.WriteString(fmt.Sprintf("  enabled: %t\n", cfg.LLMTrace.Enabled))
	builder.WriteString(fmt.Sprintf("  host: %s\n", cfg.LLMTrace.Host))
	builder.WriteString("  # public_key: pk-lf-...\n")
	builder.WriteString("  # secret_key: sk-lf-...\n")
	builder.WriteString(fmt.Sprintf("  poll_interval_seconds: %d\n", cfg.LLMTrace.PollInterval))
	builder.WriteString(fmt.Sprintf("  batch_size: %d\n", cfg.LLMTrace.BatchSize))
	builder.WriteString(fmt.Sprintf("  score_prefix: %s\n", cfg.LLMTrace.ScorePrefix))
	return builder.String(), nil
}

func formatArray(key string, values []string) string {
	var quoted []string
	for _, v := range values {
		quoted = append(quoted, fmt.Sprintf("\"%s\"", v))
	}
	return fmt.Sprintf("%s = [%s]\n", key, strings.Join(quoted, ", "))
}
