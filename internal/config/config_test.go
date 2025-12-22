package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeYAMLParsesPolicies(t *testing.T) {
	body := `
logging:
  format: json
policies:
  allowlist:
    - echo ok
  denylist:
    - rm -rf /
`
	cfg, err := decodeYAML([]byte(body))
	if err != nil {
		t.Fatalf("decode yaml: %v", err)
	}

	if cfg.Logging.Format != "json" {
		t.Fatalf("expected logging format json, got %s", cfg.Logging.Format)
	}
	if len(cfg.Policies.Allowlist) != 1 || cfg.Policies.Allowlist[0] != "echo ok" {
		t.Fatalf("unexpected allowlist: %+v", cfg.Policies.Allowlist)
	}
	if len(cfg.Policies.Denylist) != 1 || cfg.Policies.Denylist[0] != "rm -rf /" {
		t.Fatalf("unexpected denylist: %+v", cfg.Policies.Denylist)
	}
}

func TestDecodeTOMLParsesPolicies(t *testing.T) {
	body := `
[logging]
format = "json"

[policies]
allowlist = ["echo ok"]
denylist = ["rm -rf /"]
`
	cfg, err := decodeTOML([]byte(body))
	if err != nil {
		t.Fatalf("decode toml: %v", err)
	}

	if cfg.Logging.Format != "json" {
		t.Fatalf("expected logging format json, got %s", cfg.Logging.Format)
	}
	if len(cfg.Policies.Allowlist) != 1 || cfg.Policies.Allowlist[0] != "echo ok" {
		t.Fatalf("unexpected allowlist: %+v", cfg.Policies.Allowlist)
	}
	if len(cfg.Policies.Denylist) != 1 || cfg.Policies.Denylist[0] != "rm -rf /" {
		t.Fatalf("unexpected denylist: %+v", cfg.Policies.Denylist)
	}
}

func TestLoadRespectsPrecedence(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, ".config", "vectra-guard")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	homeCfg := filepath.Join(configDir, "config.yaml")
	projectCfg := filepath.Join(tmp, "vectra-guard.yaml")
	explicitCfg := filepath.Join(tmp, "explicit.yaml")

	writeFile(t, homeCfg, "logging:\n  format: json\n")
	writeFile(t, projectCfg, "logging:\n  format: text\npolicies:\n  allowlist:\n    - project-rule\n")
	writeFile(t, explicitCfg, "logging:\n  format: verbose\npolicies:\n  denylist:\n    - explicit-rule\n")

	prevHome := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", prevHome) })
	_ = os.Setenv("HOME", tmp)

	cfg, loaded, err := Load(explicitCfg, tmp)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if cfg.Logging.Format != "verbose" {
		t.Fatalf("explicit config should win: %s", cfg.Logging.Format)
	}
	if len(cfg.Policies.Allowlist) != 1 || cfg.Policies.Allowlist[0] != "project-rule" {
		t.Fatalf("project allowlist should be applied, got: %+v", cfg.Policies.Allowlist)
	}
	if len(cfg.Policies.Denylist) == 0 || cfg.Policies.Denylist[0] != "explicit-rule" {
		t.Fatalf("explicit denylist should override: %+v", cfg.Policies.Denylist)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected three loaded configs, got %d", len(loaded))
	}
}

func TestContextHelpers(t *testing.T) {
	cfg := Config{Logging: LoggingConfig{Format: "json"}}
	ctx := WithConfig(context.Background(), cfg)
	got := FromContext(ctx)
	if got.Logging.Format != "json" {
		t.Fatalf("expected json format from context, got %s", got.Logging.Format)
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
