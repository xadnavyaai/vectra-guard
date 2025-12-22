package config

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the merged configuration for vectra-guard.
type Config struct {
	Logging  LoggingConfig `yaml:"logging" toml:"logging" json:"logging"`
	Policies PolicyConfig  `yaml:"policies" toml:"policies" json:"policies"`
}

// LoggingConfig controls output formatting.
type LoggingConfig struct {
	Format string `yaml:"format" toml:"format" json:"format"`
}

// PolicyConfig captures simple allow/deny rules applied during analysis.
type PolicyConfig struct {
	Allowlist []string `yaml:"allowlist" toml:"allowlist" json:"allowlist"`
	Denylist  []string `yaml:"denylist" toml:"denylist" json:"denylist"`
}

// DefaultConfig returns the in-process defaults.
func DefaultConfig() Config {
	return Config{
		Logging: LoggingConfig{Format: "text"},
		Policies: PolicyConfig{
			Allowlist: []string{},
			Denylist:  []string{"rm -rf /", "sudo ", ":(){ :|:& };:", "mkfs", "dd if="},
		},
	}
}

// Load discovers and merges configuration from the provided path, project-local config, and user config.
// Precedence (highest wins): explicit flag > project > user > defaults.
func Load(configPath string, workdir string) (Config, []string, error) {
	cfg := DefaultConfig()
	var loaded []string

	paths, err := resolvePaths(configPath, workdir)
	if err != nil {
		return cfg, loaded, err
	}

	for _, path := range paths {
		if err := applyFile(&cfg, path); err != nil {
			return cfg, loaded, err
		}
		loaded = append(loaded, path)
	}

	return cfg, loaded, nil
}

func resolvePaths(explicit, workdir string) ([]string, error) {
	var paths []string
	// Lowest precedence first.
	if home, err := os.UserHomeDir(); err == nil {
		candidateYAML := filepath.Join(home, ".config", "vectra-guard", "config.yaml")
		candidateTOML := filepath.Join(home, ".config", "vectra-guard", "config.toml")
		if exists(candidateYAML) {
			paths = append(paths, candidateYAML)
		}
		if exists(candidateTOML) {
			paths = append(paths, candidateTOML)
		}
	}

	if workdir != "" {
		projectYAML := filepath.Join(workdir, "vectra-guard.yaml")
		projectTOML := filepath.Join(workdir, "vectra-guard.toml")
		if exists(projectYAML) {
			paths = append(paths, projectYAML)
		}
		if exists(projectTOML) {
			paths = append(paths, projectTOML)
		}
	}

	if explicit != "" {
		if !exists(explicit) {
			return nil, fmt.Errorf("config file not found: %s", explicit)
		}
		paths = append(paths, explicit)
	}

	return paths, nil
}

func applyFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}

	var next Config
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		next, err = decodeYAML(data)
	case ".toml":
		next, err = decodeTOML(data)
	default:
		err = errors.New("unsupported config extension; use .yaml or .toml")
	}
	if err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}

	merge(cfg, next)
	return nil
}

func merge(dst *Config, src Config) {
	if src.Logging.Format != "" {
		dst.Logging.Format = src.Logging.Format
	}

	if len(src.Policies.Allowlist) > 0 {
		dst.Policies.Allowlist = src.Policies.Allowlist
	}
	if len(src.Policies.Denylist) > 0 {
		dst.Policies.Denylist = src.Policies.Denylist
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func decodeYAML(data []byte) (Config, error) {
	var cfg Config
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	mode := ""
	var listTarget *[]string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasSuffix(line, ":") {
			switch strings.TrimSuffix(line, ":") {
			case "logging":
				mode = "logging"
				listTarget = nil
			case "policies":
				mode = "policies"
				listTarget = nil
			case "allowlist":
				if mode == "policies" {
					listTarget = &cfg.Policies.Allowlist
				}
			case "denylist":
				if mode == "policies" {
					listTarget = &cfg.Policies.Denylist
				}
			}
			continue
		}

		if strings.HasPrefix(line, "- ") {
			if listTarget != nil {
				*listTarget = append(*listTarget, strings.TrimSpace(strings.TrimPrefix(line, "- ")))
			}
			continue
		}

		if mode == "logging" && strings.HasPrefix(line, "format:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				cfg.Logging.Format = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			}
		}
	}

	return cfg, scanner.Err()
}

func decodeTOML(data []byte) (Config, error) {
	var cfg Config
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	section := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.Trim(line, "[]")
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch section {
		case "logging":
			if key == "format" {
				cfg.Logging.Format = trimQuotes(value)
			}
		case "policies":
			switch key {
			case "allowlist":
				cfg.Policies.Allowlist = parseStringArray(value)
			case "denylist":
				cfg.Policies.Denylist = parseStringArray(value)
			}
		}
	}

	return cfg, scanner.Err()
}

func parseStringArray(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if value == "" {
		return nil
	}
	raw := strings.Split(value, ",")
	var cleaned []string
	for _, item := range raw {
		item = trimQuotes(strings.TrimSpace(item))
		if item != "" {
			cleaned = append(cleaned, item)
		}
	}
	return cleaned
}

func trimQuotes(s string) string {
	return strings.Trim(s, `"`)
}

type ctxKey struct{}

// WithConfig stores the config on the context.
func WithConfig(ctx context.Context, cfg Config) context.Context {
	return context.WithValue(ctx, ctxKey{}, cfg)
}

// FromContext extracts the config or returns defaults.
func FromContext(ctx context.Context) Config {
	if ctx == nil {
		return DefaultConfig()
	}
	if cfg, ok := ctx.Value(ctxKey{}).(Config); ok {
		return cfg
	}
	return DefaultConfig()
}
