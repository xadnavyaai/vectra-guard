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

func TestGuardLevelDefaults(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.GuardLevel.Level != GuardLevelAuto {
		t.Errorf("expected default guard level to be auto, got %s", cfg.GuardLevel.Level)
	}
	
	if !cfg.GuardLevel.AllowUserBypass {
		t.Error("expected user bypass to be allowed by default")
	}
	
	if cfg.GuardLevel.BypassEnvVar != "VECTRAGUARD_BYPASS" {
		t.Errorf("expected default bypass env var to be VECTRAGUARD_BYPASS, got %s", cfg.GuardLevel.BypassEnvVar)
	}
}

func TestPolicyDefaults(t *testing.T) {
	cfg := DefaultConfig()
	
	if !cfg.Policies.MonitorGitOps {
		t.Error("expected MonitorGitOps to be true by default")
	}
	
	if !cfg.Policies.BlockForceGit {
		t.Error("expected BlockForceGit to be true by default")
	}
	
	if !cfg.Policies.DetectProdEnv {
		t.Error("expected DetectProdEnv to be true by default")
	}
	
	if !cfg.Policies.OnlyDestructiveSQL {
		t.Error("expected OnlyDestructiveSQL to be true by default")
	}
	
	expectedPatterns := []string{"prod", "production", "prd", "live", "staging", "stg"}
	if len(cfg.Policies.ProdEnvPatterns) != len(expectedPatterns) {
		t.Errorf("expected %d prod env patterns, got %d", len(expectedPatterns), len(cfg.Policies.ProdEnvPatterns))
	}
}

func TestGuardLevelParsing(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected GuardLevel
	}{
		{"auto level", "guard_level:\n  level: auto\n", GuardLevelAuto},
		{"low level", "guard_level:\n  level: low\n", GuardLevelLow},
		{"medium level", "guard_level:\n  level: medium\n", GuardLevelMedium},
		{"high level", "guard_level:\n  level: high\n", GuardLevelHigh},
		{"paranoid level", "guard_level:\n  level: paranoid\n", GuardLevelParanoid},
		{"off level", "guard_level:\n  level: off\n", GuardLevelOff},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := decodeYAML([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("decode yaml: %v", err)
			}
			
			if cfg.GuardLevel.Level != tt.expected {
				t.Errorf("expected guard level %s, got %s", tt.expected, cfg.GuardLevel.Level)
			}
		})
	}
}

func TestGuardLevelValidation(t *testing.T) {
	validLevels := []GuardLevel{
		GuardLevelOff,
		GuardLevelLow,
		GuardLevelMedium,
		GuardLevelHigh,
		GuardLevelParanoid,
	}
	
	for _, level := range validLevels {
		t.Run(string(level), func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.GuardLevel.Level = level
			
			// Should not panic or error
			_ = cfg.GuardLevel.Level
		})
	}
}

func TestPolicyConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()
	
	tests := []struct {
		name     string
		value    bool
		expected bool
	}{
		{"MonitorGitOps", cfg.Policies.MonitorGitOps, true},
		{"BlockForceGit", cfg.Policies.BlockForceGit, true},
		{"DetectProdEnv", cfg.Policies.DetectProdEnv, true},
		{"OnlyDestructiveSQL", cfg.Policies.OnlyDestructiveSQL, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("expected %s to be %v, got %v", tt.name, tt.expected, tt.value)
			}
		})
	}
}

func TestCompleteYAMLConfig(t *testing.T) {
	yaml := `
logging:
  format: json

guard_level:
  level: high
  allow_user_bypass: false
  bypass_env_var: CUSTOM_BYPASS
  require_approval_above: medium

policies:
  monitor_git_ops: true
  block_force_git: true
  detect_prod_env: true
  only_destructive_sql: false
  prod_env_patterns:
    - prod
    - production
    - uat
  allowlist:
    - "echo *"
    - "git status"
  denylist:
    - "rm -rf /"
`
	
	cfg, err := decodeYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("decode yaml: %v", err)
	}
	
	// Verify all fields
	if cfg.Logging.Format != "json" {
		t.Errorf("expected format json, got %s", cfg.Logging.Format)
	}
	
	if cfg.GuardLevel.Level != GuardLevelHigh {
		t.Errorf("expected guard level high, got %s", cfg.GuardLevel.Level)
	}
	
	if cfg.GuardLevel.AllowUserBypass != false {
		t.Error("expected allow_user_bypass to be false")
	}
	
	if cfg.GuardLevel.BypassEnvVar != "CUSTOM_BYPASS" {
		t.Errorf("expected bypass env var CUSTOM_BYPASS, got %s", cfg.GuardLevel.BypassEnvVar)
	}
	
	if !cfg.Policies.MonitorGitOps {
		t.Error("expected monitor_git_ops to be true")
	}
	
	if !cfg.Policies.BlockForceGit {
		t.Error("expected block_force_git to be true")
	}
	
	if !cfg.Policies.DetectProdEnv {
		t.Error("expected detect_prod_env to be true")
	}
	
	if cfg.Policies.OnlyDestructiveSQL {
		t.Error("expected only_destructive_sql to be false")
	}
	
	expectedPatterns := []string{"prod", "production", "uat"}
	if len(cfg.Policies.ProdEnvPatterns) != len(expectedPatterns) {
		t.Errorf("expected %d patterns, got %d", len(expectedPatterns), len(cfg.Policies.ProdEnvPatterns))
	}
	
	if len(cfg.Policies.Allowlist) != 2 {
		t.Errorf("expected 2 allowlist items, got %d", len(cfg.Policies.Allowlist))
	}
	
	if len(cfg.Policies.Denylist) != 1 {
		t.Errorf("expected 1 denylist item, got %d", len(cfg.Policies.Denylist))
	}
}

func TestConfigMerging(t *testing.T) {
	base := Config{
		GuardLevel: GuardLevelConfig{
			Level: GuardLevelLow,
			AllowUserBypass: true,
			BypassEnvVar: "BASE_VAR",
		},
		Policies: PolicyConfig{
			MonitorGitOps: false,
			BlockForceGit: false,
		},
	}
	
	override := Config{
		GuardLevel: GuardLevelConfig{
			Level: GuardLevelHigh,
			AllowUserBypass: false, // Explicitly set to false
			// BypassEnvVar not set (empty string)
		},
		Policies: PolicyConfig{
			MonitorGitOps: true,
			BlockForceGit: false, // Explicitly set to false
		},
	}
	
	merge(&base, override)
	
	if base.GuardLevel.Level != GuardLevelHigh {
		t.Error("guard level should be overridden to high")
	}
	
	// Note: boolean values are always merged from source, even if false
	// This is by design since we can't distinguish between "not set" and "set to false" for booleans
	if base.GuardLevel.AllowUserBypass != false {
		t.Error("allow_user_bypass should be overridden to false")
	}
	
	// BypassEnvVar should keep base value since override is empty
	if base.GuardLevel.BypassEnvVar != "BASE_VAR" {
		t.Error("bypass_env_var should keep base value when override is empty")
	}
	
	if !base.Policies.MonitorGitOps {
		t.Error("monitor_git_ops should be overridden to true")
	}
	
	// BlockForceGit is merged from override (false)
	if base.Policies.BlockForceGit {
		t.Error("block_force_git should be overridden to false")
	}
}

func TestInvalidGuardLevel(t *testing.T) {
	yaml := `
guard_level:
  level: invalid_level
`
	
	cfg, err := decodeYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("decode yaml: %v", err)
	}
	
	// Should parse but with the invalid value
	if cfg.GuardLevel.Level != GuardLevel("invalid_level") {
		t.Errorf("expected invalid_level, got %s", cfg.GuardLevel.Level)
	}
}

func TestPartialConfig(t *testing.T) {
	// Test that partial configs work and fill in defaults
	yaml := `
guard_level:
  level: paranoid
`
	
	cfg, err := decodeYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("decode yaml: %v", err)
	}
	
	if cfg.GuardLevel.Level != GuardLevelParanoid {
		t.Errorf("expected paranoid, got %s", cfg.GuardLevel.Level)
	}
	
	// Other fields should be empty/default
	if cfg.GuardLevel.BypassEnvVar != "" {
		t.Error("bypass env var should be empty when not specified")
	}
}

func TestBooleanParsing(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected bool
	}{
		{
			name:     "true value",
			yaml:     "policies:\n  monitor_git_ops: true",
			expected: true,
		},
		{
			name:     "false value",
			yaml:     "policies:\n  monitor_git_ops: false",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := decodeYAML([]byte(tt.yaml))
			if err != nil {
				t.Fatalf("decode yaml: %v", err)
			}
			
			if cfg.Policies.MonitorGitOps != tt.expected {
				t.Errorf("expected %v, got %v (yaml: %q)", tt.expected, cfg.Policies.MonitorGitOps, tt.yaml)
			}
		})
	}
}

func TestEmptyProdPatternsConfig(t *testing.T) {
	yaml := `
policies:
  detect_prod_env: true
  prod_env_patterns: []
`
	
	cfg, err := decodeYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("decode yaml: %v", err)
	}
	
	if len(cfg.Policies.ProdEnvPatterns) != 0 {
		t.Errorf("expected empty prod patterns, got %d", len(cfg.Policies.ProdEnvPatterns))
	}
}

func TestConfigContextHelpers(t *testing.T) {
	cfg := Config{
		GuardLevel: GuardLevelConfig{
			Level: GuardLevelHigh,
		},
	}
	
	ctx := WithConfig(context.Background(), cfg)
	retrieved := FromContext(ctx)
	
	if retrieved.GuardLevel.Level != GuardLevelHigh {
		t.Error("config not properly stored/retrieved from context")
	}
	
	// Test with nil context
	defaultCfg := FromContext(nil)
	if defaultCfg.GuardLevel.Level != GuardLevelAuto {
		t.Errorf("should return default config (auto) for nil context, got %s", defaultCfg.GuardLevel.Level)
	}
}

func TestProductionIndicatorsParsing(t *testing.T) {
	yaml := `
production_indicators:
  branches:
    - main
    - master
    - production
  keywords:
    - prod
    - staging
`
	
	cfg, err := decodeYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("decode yaml: %v", err)
	}
	
	expectedBranches := []string{"main", "master", "production"}
	if len(cfg.ProductionIndicators.Branches) != len(expectedBranches) {
		t.Errorf("expected %d branches, got %d", len(expectedBranches), len(cfg.ProductionIndicators.Branches))
	}
	
	for i, branch := range expectedBranches {
		if cfg.ProductionIndicators.Branches[i] != branch {
			t.Errorf("expected branch %s, got %s", branch, cfg.ProductionIndicators.Branches[i])
		}
	}
	
	expectedKeywords := []string{"prod", "staging"}
	if len(cfg.ProductionIndicators.Keywords) != len(expectedKeywords) {
		t.Errorf("expected %d keywords, got %d", len(expectedKeywords), len(cfg.ProductionIndicators.Keywords))
	}
}

func TestDetectGuardLevelGitBranch(t *testing.T) {
	cfg := DefaultConfig()
	
	tests := []struct {
		name       string
		branch     string
		expected   GuardLevel
		description string
	}{
		{
			name:        "main branch",
			branch:      "main",
			expected:    GuardLevelParanoid,
			description: "main branch should trigger paranoid mode",
		},
		{
			name:        "master branch",
			branch:      "master",
			expected:    GuardLevelParanoid,
			description: "master branch should trigger paranoid mode",
		},
		{
			name:        "production branch",
			branch:      "production",
			expected:    GuardLevelParanoid,
			description: "production branch should trigger paranoid mode",
		},
		{
			name:        "feature branch",
			branch:      "feature/new-feature",
			expected:    GuardLevelMedium,
			description: "feature branch should use medium (default)",
		},
		{
			name:        "branch with prod in name",
			branch:      "hotfix-prod-issue",
			expected:    GuardLevelHigh,
			description: "branch containing 'prod' should trigger high",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := DetectionContext{
				GitBranch: tt.branch,
			}
			
			level := DetectGuardLevel(cfg, ctx)
			
			if level != tt.expected {
				t.Errorf("%s: expected %s, got %s", tt.description, tt.expected, level)
			}
		})
	}
}

func TestDetectGuardLevelCommand(t *testing.T) {
	cfg := DefaultConfig()
	
	tests := []struct {
		name     string
		command  string
		expected GuardLevel
	}{
		{
			name:     "deploy to production",
			command:  "kubectl apply -f prod-config.yaml",
			expected: GuardLevelHigh,
		},
		{
			name:     "production database",
			command:  "psql -h prod-db.company.com",
			expected: GuardLevelHigh,
		},
		{
			name:     "deploy command",
			command:  "npm run deploy",
			expected: GuardLevelHigh,
		},
		{
			name:     "staging environment",
			command:  "ssh user@staging.server.com",
			expected: GuardLevelHigh,
		},
		{
			name:     "local development",
			command:  "npm test",
			expected: GuardLevelMedium,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := DetectionContext{
				Command: tt.command,
			}
			
			level := DetectGuardLevel(cfg, ctx)
			
			if level != tt.expected {
				t.Errorf("command %q: expected %s, got %s", tt.command, tt.expected, level)
			}
		})
	}
}

func TestDetectGuardLevelNonAutoReturnsConfigured(t *testing.T) {
	cfg := DefaultConfig()
	cfg.GuardLevel.Level = GuardLevelHigh
	
	ctx := DetectionContext{
		GitBranch: "main", // Would normally trigger paranoid
		Command:   "deploy to prod", // Would normally trigger high
	}
	
	level := DetectGuardLevel(cfg, ctx)
	
	// Should return configured level, not auto-detected
	if level != GuardLevelHigh {
		t.Errorf("when not auto, should return configured level (high), got %s", level)
	}
}

func TestDetectGuardLevelPriorityMostDangerous(t *testing.T) {
	cfg := DefaultConfig()
	
	// Main branch (paranoid) + production keyword (high) = paranoid wins
	ctx := DetectionContext{
		GitBranch: "main",
		Command:   "deploy to prod-server",
	}
	
	level := DetectGuardLevel(cfg, ctx)
	
	if level != GuardLevelParanoid {
		t.Errorf("most dangerous context should win: expected paranoid, got %s", level)
	}
}

func TestIsInMeaningfulContext(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		keyword  string
		expected bool
	}{
		{
			name:     "keyword in URL",
			text:     "https://api.prod.company.com",
			keyword:  "prod",
			expected: true,
		},
		{
			name:     "keyword in path",
			text:     "/var/www/production/app",
			keyword:  "production",
			expected: true,
		},
		{
			name:     "keyword with dash",
			text:     "deploy-prod-config",
			keyword:  "prod",
			expected: true,
		},
		{
			name:     "keyword with underscore",
			text:     "db_prod_users",
			keyword:  "prod",
			expected: true,
		},
		{
			name:     "keyword at start",
			text:     "prod-database",
			keyword:  "prod",
			expected: true,
		},
		{
			name:     "keyword at end",
			text:     "database-prod",
			keyword:  "prod",
			expected: true,
		},
		{
			name:     "keyword as substring without delimiter",
			text:     "reproduction",
			keyword:  "prod",
			expected: false,
		},
		{
			name:     "keyword in word",
			text:     "products",
			keyword:  "prod",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isInMeaningfulContext(tt.text, tt.keyword)
			
			if result != tt.expected {
				t.Errorf("text %q with keyword %q: expected %v, got %v", 
					tt.text, tt.keyword, tt.expected, result)
			}
		})
	}
}

func TestDetectGuardLevelEnvironmentVars(t *testing.T) {
	cfg := DefaultConfig()
	
	tests := []struct {
		name     string
		env      map[string]string
		expected GuardLevel
	}{
		{
			name: "production environment variable",
			env: map[string]string{
				"ENV": "production",
			},
			expected: GuardLevelHigh,
		},
		{
			name: "staging environment",
			env: map[string]string{
				"ENVIRONMENT": "staging",
			},
			expected: GuardLevelHigh,
		},
		{
			name: "development environment",
			env: map[string]string{
				"ENV": "development",
			},
			expected: GuardLevelMedium,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := DetectionContext{
				Environment: tt.env,
			}
			
			level := DetectGuardLevel(cfg, ctx)
			
			if level != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, level)
			}
		})
	}
}

func TestDetectGuardLevelWorkingDirectory(t *testing.T) {
	cfg := DefaultConfig()
	
	tests := []struct {
		name       string
		workingDir string
		expected   GuardLevel
	}{
		{
			name:       "production path",
			workingDir: "/var/www/production/app",
			expected:   GuardLevelHigh,
		},
		{
			name:       "staging path",
			workingDir: "/home/user/projects/staging-app",
			expected:   GuardLevelHigh,
		},
		{
			name:       "dev path",
			workingDir: "/home/user/projects/my-app",
			expected:   GuardLevelMedium,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := DetectionContext{
				WorkingDir: tt.workingDir,
			}
			
			level := DetectGuardLevel(cfg, ctx)
			
			if level != tt.expected {
				t.Errorf("workdir %q: expected %s, got %s", tt.workingDir, tt.expected, level)
			}
		})
	}
}

func TestProductionIndicatorsDefaults(t *testing.T) {
	cfg := DefaultConfig()
	
	expectedBranches := []string{"main", "master", "production", "release"}
	if len(cfg.ProductionIndicators.Branches) != len(expectedBranches) {
		t.Errorf("expected %d default branches, got %d", len(expectedBranches), len(cfg.ProductionIndicators.Branches))
	}
	
	expectedKeywords := []string{"prod", "production", "prd", "live", "staging", "stg"}
	if len(cfg.ProductionIndicators.Keywords) != len(expectedKeywords) {
		t.Errorf("expected %d default keywords, got %d", len(expectedKeywords), len(cfg.ProductionIndicators.Keywords))
	}
}

func TestCompleteConfigWithProductionIndicators(t *testing.T) {
	yaml := `
guard_level:
  level: auto

production_indicators:
  branches:
    - main
    - production
  keywords:
    - prod
    - live

policies:
  monitor_git_ops: true
`
	
	cfg, err := decodeYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("decode yaml: %v", err)
	}
	
	if cfg.GuardLevel.Level != GuardLevelAuto {
		t.Errorf("expected auto guard level, got %s", cfg.GuardLevel.Level)
	}
	
	if len(cfg.ProductionIndicators.Branches) != 2 {
		t.Errorf("expected 2 branches, got %d", len(cfg.ProductionIndicators.Branches))
	}
	
	if len(cfg.ProductionIndicators.Keywords) != 2 {
		t.Errorf("expected 2 keywords, got %d", len(cfg.ProductionIndicators.Keywords))
	}
}

func writeFile(t *testing.T, path string, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
