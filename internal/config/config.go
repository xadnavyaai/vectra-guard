package config

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config represents the merged configuration for vectra-guard.
type Config struct {
	Logging              LoggingConfig              `yaml:"logging" toml:"logging" json:"logging"`
	Policies             PolicyConfig               `yaml:"policies" toml:"policies" json:"policies"`
	EnvProtection        EnvProtectionConfig        `yaml:"env_protection" toml:"env_protection" json:"env_protection"`
	GuardLevel           GuardLevelConfig           `yaml:"guard_level" toml:"guard_level" json:"guard_level"`
	ProductionIndicators ProductionIndicatorsConfig `yaml:"production_indicators" toml:"production_indicators" json:"production_indicators"`
	Sandbox              SandboxConfig              `yaml:"sandbox" toml:"sandbox" json:"sandbox"`
	CVE                  CVEConfig                  `yaml:"cve" toml:"cve" json:"cve"`
	SoftDelete           SoftDeleteConfig           `yaml:"soft_delete" toml:"soft_delete" json:"soft_delete"`
}

// LoggingConfig controls output formatting.
type LoggingConfig struct {
	Format string `yaml:"format" toml:"format" json:"format"`
}

// GuardLevel determines the aggressiveness of protection
type GuardLevel string

const (
	GuardLevelAuto     GuardLevel = "auto"     // Auto-detect based on context (recommended)
	GuardLevelOff      GuardLevel = "off"      // No protection
	GuardLevelLow      GuardLevel = "low"      // Only critical issues
	GuardLevelMedium   GuardLevel = "medium"   // Critical + high issues (default)
	GuardLevelHigh     GuardLevel = "high"     // Critical + high + medium issues
	GuardLevelParanoid GuardLevel = "paranoid" // Everything requires approval
)

// GuardLevelConfig controls the overall security posture
type GuardLevelConfig struct {
	Level                GuardLevel `yaml:"level" toml:"level" json:"level"`
	AllowUserBypass      bool       `yaml:"allow_user_bypass" toml:"allow_user_bypass" json:"allow_user_bypass"`
	BypassEnvVar         string     `yaml:"bypass_env_var" toml:"bypass_env_var" json:"bypass_env_var"`
	RequireApprovalAbove string     `yaml:"require_approval_above" toml:"require_approval_above" json:"require_approval_above"`
}

// PolicyConfig captures simple allow/deny rules applied during analysis.
type PolicyConfig struct {
	Allowlist            []string `yaml:"allowlist" toml:"allowlist" json:"allowlist"`
	Denylist             []string `yaml:"denylist" toml:"denylist" json:"denylist"`
	ProtectedDirectories []string `yaml:"protected_directories" toml:"protected_directories" json:"protected_directories"`
	MonitorGitOps        bool     `yaml:"monitor_git_ops" toml:"monitor_git_ops" json:"monitor_git_ops"`
	BlockForceGit        bool     `yaml:"block_force_git" toml:"block_force_git" json:"block_force_git"`
	DetectProdEnv        bool     `yaml:"detect_prod_env" toml:"detect_prod_env" json:"detect_prod_env"`
	ProdEnvPatterns      []string `yaml:"prod_env_patterns" toml:"prod_env_patterns" json:"prod_env_patterns"`
	OnlyDestructiveSQL   bool     `yaml:"only_destructive_sql" toml:"only_destructive_sql" json:"only_destructive_sql"`
}

// EnvProtectionConfig controls environment variable protection and masking.
type EnvProtectionConfig struct {
	Enabled         bool              `yaml:"enabled" toml:"enabled" json:"enabled"`
	MaskingMode     string            `yaml:"masking_mode" toml:"masking_mode" json:"masking_mode"` // full, partial, hash, fake
	ProtectedVars   []string          `yaml:"protected_vars" toml:"protected_vars" json:"protected_vars"`
	AllowReadVars   []string          `yaml:"allow_read_vars" toml:"allow_read_vars" json:"allow_read_vars"`
	FakeValues      map[string]string `yaml:"fake_values" toml:"fake_values" json:"fake_values"`
	BlockEnvAccess  bool              `yaml:"block_env_access" toml:"block_env_access" json:"block_env_access"`
	BlockDotenvRead bool              `yaml:"block_dotenv_read" toml:"block_dotenv_read" json:"block_dotenv_read"`
}

// ProductionIndicatorsConfig defines patterns to detect production environments.
type ProductionIndicatorsConfig struct {
	Branches []string `yaml:"branches" toml:"branches" json:"branches"`
	Keywords []string `yaml:"keywords" toml:"keywords" json:"keywords"`
}

// SandboxMode determines when to use sandboxing
type SandboxMode string

const (
	SandboxModeAuto   SandboxMode = "auto"   // Auto-detect based on risk
	SandboxModeAlways SandboxMode = "always" // Always sandbox
	SandboxModeRisky  SandboxMode = "risky"  // Only high/critical risk
	SandboxModeNever  SandboxMode = "never"  // Never sandbox
)

// SandboxSecurityLevel controls isolation strength
type SandboxSecurityLevel string

const (
	SandboxSecurityPermissive SandboxSecurityLevel = "permissive" // Minimal isolation
	SandboxSecurityBalanced   SandboxSecurityLevel = "balanced"   // Good balance (default)
	SandboxSecurityStrict     SandboxSecurityLevel = "strict"     // Strong isolation
	SandboxSecurityParanoid   SandboxSecurityLevel = "paranoid"   // Maximum isolation
)

// SandboxRuntime specifies the sandbox runtime to use
type SandboxRuntime string

const (
	SandboxRuntimeAuto       SandboxRuntime = "auto"       // Auto-detect best runtime
	SandboxRuntimeBubblewrap SandboxRuntime = "bubblewrap" // Use bubblewrap (fast)
	SandboxRuntimeNamespace  SandboxRuntime = "namespace"  // Use custom namespaces
	SandboxRuntimeDocker     SandboxRuntime = "docker"     // Use Docker (compatible)
	SandboxRuntimePodman     SandboxRuntime = "podman"     // Use Podman
)

// SandboxConfig controls sandbox execution behavior
type SandboxConfig struct {
	// Core settings
	Enabled       bool                 `yaml:"enabled" toml:"enabled" json:"enabled"`
	Mode          SandboxMode          `yaml:"mode" toml:"mode" json:"mode"`
	SecurityLevel SandboxSecurityLevel `yaml:"security_level" toml:"security_level" json:"security_level"`

	// Runtime configuration
	Runtime string `yaml:"runtime" toml:"runtime" json:"runtime"` // auto, bubblewrap, namespace, docker, podman
	Image   string `yaml:"image" toml:"image" json:"image"`       // Docker/Podman image
	Timeout int    `yaml:"timeout" toml:"timeout" json:"timeout"` // seconds

	// Environment detection
	AutoDetectEnv bool `yaml:"auto_detect_env" toml:"auto_detect_env" json:"auto_detect_env"` // Auto-detect dev/CI
	PreferFast    bool `yaml:"prefer_fast" toml:"prefer_fast" json:"prefer_fast"`             // Prefer fast runtimes in dev

	// Cache configuration
	EnableCache bool     `yaml:"enable_cache" toml:"enable_cache" json:"enable_cache"`
	CacheDirs   []string `yaml:"cache_dirs" toml:"cache_dirs" json:"cache_dirs"`
	CacheDir    string   `yaml:"cache_dir" toml:"cache_dir" json:"cache_dir"` // Cache directory for namespace runtime

	// Network configuration
	NetworkMode  string `yaml:"network_mode" toml:"network_mode" json:"network_mode"`    // none, restricted, full
	AllowNetwork bool   `yaml:"allow_network" toml:"allow_network" json:"allow_network"` // Allow network access

	// Filesystem configuration
	ReadOnlyPaths []string `yaml:"read_only_paths" toml:"read_only_paths" json:"read_only_paths"` // Read-only filesystem paths
	WorkspaceDir  string   `yaml:"workspace_dir" toml:"workspace_dir" json:"workspace_dir"`       // Workspace directory

	// Security profiles
	SeccompProfile string `yaml:"seccomp_profile" toml:"seccomp_profile" json:"seccomp_profile"` // strict, moderate, minimal, none
	CapabilitySet  string `yaml:"capability_set" toml:"capability_set" json:"capability_set"`    // none, minimal, normal
	UseOverlayFS   bool   `yaml:"use_overlayfs" toml:"use_overlayfs" json:"use_overlayfs"`       // Use OverlayFS (namespace only)

	// Environment
	EnvWhitelist []string `yaml:"env_whitelist" toml:"env_whitelist" json:"env_whitelist"`

	// Custom mounts
	BindMounts []BindMountConfig `yaml:"bind_mounts" toml:"bind_mounts" json:"bind_mounts"`

	// Observability
	EnableMetrics   bool `yaml:"enable_metrics" toml:"enable_metrics" json:"enable_metrics"`
	LogOutput       bool `yaml:"log_output" toml:"log_output" json:"log_output"`
	ShowRuntimeInfo bool `yaml:"show_runtime_info" toml:"show_runtime_info" json:"show_runtime_info"` // Show runtime selection

	// Trust store
	TrustStorePath string `yaml:"trust_store_path" toml:"trust_store_path" json:"trust_store_path"`
}

// CVEConfig controls CVE awareness and caching.
type CVEConfig struct {
	Enabled             bool     `yaml:"enabled" toml:"enabled" json:"enabled"`
	Sources             []string `yaml:"sources" toml:"sources" json:"sources"`
	UpdateIntervalHours int      `yaml:"update_interval_hours" toml:"update_interval_hours" json:"update_interval_hours"`
	CacheDir            string   `yaml:"cache_dir" toml:"cache_dir" json:"cache_dir"`
}

// SoftDeleteConfig controls soft delete (backup) behavior.
type SoftDeleteConfig struct {
	Enabled             bool   `yaml:"enabled" toml:"enabled" json:"enabled"`
	BackupDir           string `yaml:"backup_dir" toml:"backup_dir" json:"backup_dir"`
	MaxAgeDays          int    `yaml:"max_age_days" toml:"max_age_days" json:"max_age_days"`
	MaxBackups          int    `yaml:"max_backups" toml:"max_backups" json:"max_backups"`
	MaxSizeMB           int    `yaml:"max_size_mb" toml:"max_size_mb" json:"max_size_mb"`
	AutoCleanup         bool   `yaml:"auto_cleanup" toml:"auto_cleanup" json:"auto_cleanup"`
	AutoDelete          bool   `yaml:"auto_delete" toml:"auto_delete" json:"auto_delete"`                                  // Auto-delete old backups (permanent deletion)
	AutoDeleteAfterDays int    `yaml:"auto_delete_after_days" toml:"auto_delete_after_days" json:"auto_delete_after_days"` // Auto-delete backups older than N days
	ProtectGit          bool   `yaml:"protect_git" toml:"protect_git" json:"protect_git"`
	GitBackupCopies     int    `yaml:"git_backup_copies" toml:"git_backup_copies" json:"git_backup_copies"`
	RotationPolicy      string `yaml:"rotation_policy" toml:"rotation_policy" json:"rotation_policy"` // age, count, size, age_and_count
}

// BindMountConfig represents a bind mount configuration
type BindMountConfig struct {
	HostPath      string `yaml:"host_path" toml:"host_path" json:"host_path"`
	ContainerPath string `yaml:"container_path" toml:"container_path" json:"container_path"`
	ReadOnly      bool   `yaml:"read_only" toml:"read_only" json:"read_only"`
}

// DefaultConfig returns the in-process defaults.
func DefaultConfig() Config {
	return Config{
		Logging: LoggingConfig{Format: "text"},
		Policies: PolicyConfig{
			Allowlist: []string{},
			Denylist:  []string{"rm -rf /", "sudo ", ":(){ :|:& };:", "mkfs", "dd if="},
			ProtectedDirectories: []string{
				"/",          // Root directory
				"/bin",       // System binaries
				"/sbin",      // System admin binaries
				"/usr",       // User programs
				"/usr/bin",   // User binaries
				"/usr/sbin",  // User admin binaries
				"/usr/local", // Local programs
				"/etc",       // System configuration
				"/var",       // Variable data
				"/lib",       // Libraries
				"/lib64",     // 64-bit libraries
				"/opt",       // Optional software
				"/boot",      // Boot files
				"/root",      // Root home
				"/sys",       // System files
				"/proc",      // Process files
				"/dev",       // Device files
				"/home",      // User homes
				"/srv",       // Service data
			},
			MonitorGitOps:      true,
			BlockForceGit:      true,
			DetectProdEnv:      true,
			ProdEnvPatterns:    []string{"prod", "production", "prd", "live", "staging", "stg"},
			OnlyDestructiveSQL: true, // Only flag destructive SQL operations
		},
		EnvProtection: EnvProtectionConfig{
			Enabled:         true,
			MaskingMode:     "partial", // partial, full, hash, fake
			ProtectedVars:   []string{},
			AllowReadVars:   []string{"HOME", "USER", "PATH", "SHELL", "TERM", "LANG", "PWD"},
			FakeValues:      make(map[string]string),
			BlockEnvAccess:  false, // Warn but don't block by default
			BlockDotenvRead: true,  // Block .env file reads by default
		},
		GuardLevel: GuardLevelConfig{
			Level:                GuardLevelAuto, // Auto-detect by default (was GuardLevelMedium)
			AllowUserBypass:      true,
			BypassEnvVar:         "VECTRAGUARD_BYPASS",
			RequireApprovalAbove: "medium",
		},
		ProductionIndicators: ProductionIndicatorsConfig{
			Branches: []string{"main", "master", "production", "release"},
			Keywords: []string{"prod", "production", "prd", "live", "staging", "stg"},
		},
		Sandbox: SandboxConfig{
			Enabled:       true,
			Mode:          SandboxModeAlways, // Always sandbox for maximum security
			SecurityLevel: SandboxSecurityBalanced,
			Runtime:       "auto", // Auto-detect best runtime (bubblewrap > docker > podman)
			Image:         "ubuntu:22.04",
			Timeout:       600,  // 10 minutes (longer for builds)
			AutoDetectEnv: true, // Auto-detect dev/CI
			PreferFast:    true, // Prefer fast runtimes (bubblewrap/namespace) when available
			EnableCache:   true, // Enable caching for 10x speedup
			CacheDirs: []string{
				// Comprehensive cache directories for all major package managers
				"~/.npm",            // Node.js npm
				"~/.yarn",           // Yarn
				"~/.pnpm",           // pnpm
				"~/.cache/pip",      // Python pip
				"~/.cache/pip3",     // Python pip3
				"~/.cargo",          // Rust cargo
				"~/.rustup",         // Rust toolchain
				"~/go/pkg",          // Go modules
				"~/.m2",             // Maven
				"~/.gradle",         // Gradle
				"~/.gem",            // Ruby gems
				"~/.cache/go-build", // Go build cache
				"~/.cache/npm",      // npm cache
				"~/.cache/yarn",     // Yarn cache
				"~/.cache/pip",      // pip cache
				"~/.cache/pip3",     // pip3 cache
				"~/.cache/cargo",    // Cargo cache
				"~/.cache/composer", // PHP Composer
				"~/.cache/bower",    // Bower
				"~/.cache/nuget",    // .NET NuGet
			},
			CacheDir:       "",           // Will use ~/.cache/vectra-guard by default
			NetworkMode:    "restricted", // Restricted network (allows package managers)
			AllowNetwork:   true,         // Allow network for package installs
			ReadOnlyPaths:  []string{},   // Will use defaults if empty
			WorkspaceDir:   "",           // Will use current directory by default
			SeccompProfile: "moderate",   // strict, moderate, minimal, none
			CapabilitySet:  "minimal",    // none, minimal, normal
			UseOverlayFS:   true,         // Use OverlayFS for /tmp (performance)
			EnvWhitelist: []string{
				"HOME", "USER", "PATH", "SHELL", "TERM",
				"LANG", "LC_ALL", "PWD", "OLDPWD",
				// Common dev environment variables
				"NODE_ENV", "NPM_TOKEN", "GITHUB_TOKEN",
				"DOCKER_HOST", "CI", "TRAVIS", "CIRCLE", "JENKINS",
			},
			BindMounts:      []BindMountConfig{},
			EnableMetrics:   true,  // Track performance metrics
			LogOutput:       false, // Don't log output by default
			ShowRuntimeInfo: false, // Don't spam user by default
			TrustStorePath:  "",
		},
		CVE: CVEConfig{
			Enabled:             true,
			Sources:             []string{"osv"},
			UpdateIntervalHours: 24,
			CacheDir:            "",
		},
		SoftDelete: SoftDeleteConfig{
			Enabled:             true,            // Enabled by default for safety
			BackupDir:           "",              // Will use ~/.vectra-guard/backups by default
			MaxAgeDays:          30,              // Keep backups for 30 days
			MaxBackups:          100,             // Maximum 100 backups
			MaxSizeMB:           1024,            // Maximum 1GB total
			AutoCleanup:         true,            // Auto-cleanup old backups (move to archive)
			AutoDelete:          false,           // Auto-delete old backups (permanent deletion) - disabled by default
			AutoDeleteAfterDays: 90,              // Auto-delete backups older than 90 days (if auto_delete enabled)
			ProtectGit:          true,            // Protect git files
			GitBackupCopies:     3,               // Keep 3 copies of git files
			RotationPolicy:      "age_and_count", // Use both age and count limits
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
	// Check HOME environment variable first (works on all platforms for testing)
	// Then fall back to os.UserHomeDir()
	var home string
	if homeEnv := os.Getenv("HOME"); homeEnv != "" {
		home = homeEnv
	} else if homeDir, err := os.UserHomeDir(); err == nil {
		home = homeDir
	}

	if home != "" {
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
		localYAML := filepath.Join(workdir, ".vectra-guard", "config.yaml")
		localTOML := filepath.Join(workdir, ".vectra-guard", "config.toml")
		if exists(projectYAML) {
			paths = append(paths, projectYAML)
		}
		if exists(projectTOML) {
			paths = append(paths, projectTOML)
		}
		if exists(localYAML) {
			paths = append(paths, localYAML)
		}
		if exists(localTOML) {
			paths = append(paths, localTOML)
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

	// Merge policy booleans (false is a valid value, so we don't check for zero)
	dst.Policies.MonitorGitOps = src.Policies.MonitorGitOps
	dst.Policies.BlockForceGit = src.Policies.BlockForceGit
	dst.Policies.DetectProdEnv = src.Policies.DetectProdEnv
	dst.Policies.OnlyDestructiveSQL = src.Policies.OnlyDestructiveSQL

	if len(src.Policies.ProdEnvPatterns) > 0 {
		dst.Policies.ProdEnvPatterns = src.Policies.ProdEnvPatterns
	}

	// Merge guard level
	if src.GuardLevel.Level != "" {
		dst.GuardLevel.Level = src.GuardLevel.Level
	}
	if src.GuardLevel.BypassEnvVar != "" {
		dst.GuardLevel.BypassEnvVar = src.GuardLevel.BypassEnvVar
	}
	if src.GuardLevel.RequireApprovalAbove != "" {
		dst.GuardLevel.RequireApprovalAbove = src.GuardLevel.RequireApprovalAbove
	}
	dst.GuardLevel.AllowUserBypass = src.GuardLevel.AllowUserBypass

	// Merge production indicators
	if len(src.ProductionIndicators.Branches) > 0 {
		dst.ProductionIndicators.Branches = src.ProductionIndicators.Branches
	}
	if len(src.ProductionIndicators.Keywords) > 0 {
		dst.ProductionIndicators.Keywords = src.ProductionIndicators.Keywords
	}

	// Merge sandbox configuration
	if src.Sandbox.Mode != "" {
		dst.Sandbox.Mode = src.Sandbox.Mode
	}
	if src.Sandbox.SecurityLevel != "" {
		dst.Sandbox.SecurityLevel = src.Sandbox.SecurityLevel
	}
	if src.Sandbox.Runtime != "" {
		dst.Sandbox.Runtime = src.Sandbox.Runtime
	}
	if src.Sandbox.Image != "" {
		dst.Sandbox.Image = src.Sandbox.Image
	}
	if src.Sandbox.Timeout > 0 {
		dst.Sandbox.Timeout = src.Sandbox.Timeout
	}
	if src.Sandbox.NetworkMode != "" {
		dst.Sandbox.NetworkMode = src.Sandbox.NetworkMode
	}
	if src.Sandbox.CacheDir != "" {
		dst.Sandbox.CacheDir = src.Sandbox.CacheDir
	}
	if src.Sandbox.WorkspaceDir != "" {
		dst.Sandbox.WorkspaceDir = src.Sandbox.WorkspaceDir
	}
	if src.Sandbox.SeccompProfile != "" {
		dst.Sandbox.SeccompProfile = src.Sandbox.SeccompProfile
	}
	if src.Sandbox.CapabilitySet != "" {
		dst.Sandbox.CapabilitySet = src.Sandbox.CapabilitySet
	}
	if src.Sandbox.TrustStorePath != "" {
		dst.Sandbox.TrustStorePath = src.Sandbox.TrustStorePath
	}
	if len(src.Sandbox.EnvWhitelist) > 0 {
		dst.Sandbox.EnvWhitelist = src.Sandbox.EnvWhitelist
	}
	if len(src.Sandbox.CacheDirs) > 0 {
		dst.Sandbox.CacheDirs = src.Sandbox.CacheDirs
	}
	if len(src.Sandbox.ReadOnlyPaths) > 0 {
		dst.Sandbox.ReadOnlyPaths = src.Sandbox.ReadOnlyPaths
	}
	if len(src.Sandbox.BindMounts) > 0 {
		dst.Sandbox.BindMounts = src.Sandbox.BindMounts
	}
	// Copy boolean fields
	dst.Sandbox.Enabled = src.Sandbox.Enabled
	dst.Sandbox.AutoDetectEnv = src.Sandbox.AutoDetectEnv
	dst.Sandbox.PreferFast = src.Sandbox.PreferFast
	dst.Sandbox.EnableCache = src.Sandbox.EnableCache
	dst.Sandbox.AllowNetwork = src.Sandbox.AllowNetwork
	dst.Sandbox.UseOverlayFS = src.Sandbox.UseOverlayFS
	dst.Sandbox.EnableMetrics = src.Sandbox.EnableMetrics
	dst.Sandbox.LogOutput = src.Sandbox.LogOutput
	dst.Sandbox.ShowRuntimeInfo = src.Sandbox.ShowRuntimeInfo

	// Merge CVE configuration
	dst.CVE.Enabled = src.CVE.Enabled
	if len(src.CVE.Sources) > 0 {
		dst.CVE.Sources = src.CVE.Sources
	}
	if src.CVE.UpdateIntervalHours > 0 {
		dst.CVE.UpdateIntervalHours = src.CVE.UpdateIntervalHours
	}
	if src.CVE.CacheDir != "" {
		dst.CVE.CacheDir = src.CVE.CacheDir
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
			key := strings.TrimSuffix(line, ":")
			switch key {
			case "logging":
				mode = "logging"
				listTarget = nil
			case "policies":
				mode = "policies"
				listTarget = nil
			case "guard_level":
				mode = "guard_level"
				listTarget = nil
			case "production_indicators":
				mode = "production_indicators"
				listTarget = nil
			case "sandbox":
				mode = "sandbox"
				listTarget = nil
			case "cve":
				mode = "cve"
				listTarget = nil
			case "allowlist":
				if mode == "policies" {
					listTarget = &cfg.Policies.Allowlist
				}
			case "denylist":
				if mode == "policies" {
					listTarget = &cfg.Policies.Denylist
				}
			case "protected_directories":
				if mode == "policies" {
					listTarget = &cfg.Policies.ProtectedDirectories
				}
			case "prod_env_patterns":
				if mode == "policies" {
					listTarget = &cfg.Policies.ProdEnvPatterns
				}
			case "branches":
				if mode == "production_indicators" {
					listTarget = &cfg.ProductionIndicators.Branches
				}
			case "keywords":
				if mode == "production_indicators" {
					listTarget = &cfg.ProductionIndicators.Keywords
				}
			case "sources":
				if mode == "cve" {
					listTarget = &cfg.CVE.Sources
				}
			}
			continue
		}

		if strings.HasPrefix(line, "- ") {
			if listTarget != nil {
				item := strings.TrimSpace(strings.TrimPrefix(line, "- "))
				item = strings.Trim(item, `"'`)
				*listTarget = append(*listTarget, item)
			}
			continue
		}

		// Parse key: value lines
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			// Strip inline comments (everything after #)
			if commentIdx := strings.Index(value, "#"); commentIdx >= 0 {
				value = strings.TrimSpace(value[:commentIdx])
			}

			switch mode {
			case "logging":
				if key == "format" {
					cfg.Logging.Format = value
				}
			case "guard_level":
				switch key {
				case "level":
					cfg.GuardLevel.Level = GuardLevel(value)
				case "allow_user_bypass":
					cfg.GuardLevel.AllowUserBypass = value == "true"
				case "bypass_env_var":
					cfg.GuardLevel.BypassEnvVar = value
				case "require_approval_above":
					cfg.GuardLevel.RequireApprovalAbove = value
				}
			case "policies":
				switch key {
				case "monitor_git_ops":
					cfg.Policies.MonitorGitOps = value == "true"
				case "block_force_git":
					cfg.Policies.BlockForceGit = value == "true"
				case "detect_prod_env":
					cfg.Policies.DetectProdEnv = value == "true"
				case "only_destructive_sql":
					cfg.Policies.OnlyDestructiveSQL = value == "true"
				}
			case "sandbox":
				switch key {
				case "enabled":
					cfg.Sandbox.Enabled = value == "true"
				case "mode":
					cfg.Sandbox.Mode = SandboxMode(value)
				case "security_level":
					cfg.Sandbox.SecurityLevel = SandboxSecurityLevel(value)
				case "runtime":
					cfg.Sandbox.Runtime = value
				case "image":
					cfg.Sandbox.Image = value
				case "network_mode":
					cfg.Sandbox.NetworkMode = value
				case "enable_cache":
					cfg.Sandbox.EnableCache = value == "true"
				case "enable_metrics":
					cfg.Sandbox.EnableMetrics = value == "true"
				case "log_output":
					cfg.Sandbox.LogOutput = value == "true"
				case "trust_store_path":
					cfg.Sandbox.TrustStorePath = value
				case "seccomp_profile":
					cfg.Sandbox.SeccompProfile = value
				}
			case "cve":
				switch key {
				case "enabled":
					cfg.CVE.Enabled = value == "true"
				case "cache_dir":
					cfg.CVE.CacheDir = value
				case "update_interval_hours":
					if parsed, err := strconv.Atoi(value); err == nil {
						cfg.CVE.UpdateIntervalHours = parsed
					}
				}
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
		case "cve":
			switch key {
			case "enabled":
				cfg.CVE.Enabled = trimQuotes(value) == "true"
			case "cache_dir":
				cfg.CVE.CacheDir = trimQuotes(value)
			case "update_interval_hours":
				if parsed, err := strconv.Atoi(trimQuotes(value)); err == nil {
					cfg.CVE.UpdateIntervalHours = parsed
				}
			case "sources":
				cfg.CVE.Sources = parseStringArray(value)
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

// DetectionContext holds information about the execution context
type DetectionContext struct {
	Command     string
	GitBranch   string
	WorkingDir  string
	Environment map[string]string
}

// DetectGuardLevel analyzes the context and returns the appropriate guard level
// If the configured level is not "auto", it returns the configured level as-is.
// Otherwise, it intelligently detects based on the context.
func DetectGuardLevel(cfg Config, ctx DetectionContext) GuardLevel {
	// If not auto, return as configured
	if cfg.GuardLevel.Level != GuardLevelAuto {
		return cfg.GuardLevel.Level
	}

	// Auto-detection logic: be safe, choose most protective level when in doubt
	indicators := cfg.ProductionIndicators
	if len(indicators.Branches) == 0 && len(indicators.Keywords) == 0 {
		// Use defaults if not configured
		indicators = DefaultConfig().ProductionIndicators
	}

	// Check git branch first (strongest signal for production)
	if ctx.GitBranch != "" {
		branchLower := strings.ToLower(ctx.GitBranch)
		for _, prodBranch := range indicators.Branches {
			if branchLower == strings.ToLower(prodBranch) {
				return GuardLevelParanoid // Production branch = paranoid
			}
		}
		// Check if branch contains production keywords
		for _, keyword := range indicators.Keywords {
			if strings.Contains(branchLower, strings.ToLower(keyword)) {
				return GuardLevelHigh // Branch name has prod indicator = high
			}
		}
	}

	// Check command string for production indicators
	if ctx.Command != "" {
		cmdLower := strings.ToLower(ctx.Command)
		highRiskKeywords := []string{}

		for _, keyword := range indicators.Keywords {
			if strings.Contains(cmdLower, strings.ToLower(keyword)) {
				// Check if in meaningful context
				if isInMeaningfulContext(cmdLower, keyword) {
					highRiskKeywords = append(highRiskKeywords, keyword)
				}
			}
		}

		if len(highRiskKeywords) > 0 {
			return GuardLevelHigh // Production in command = high
		}

		// Check for deployment-related commands
		deploymentKeywords := []string{"deploy", "release", "publish", "ship"}
		for _, keyword := range deploymentKeywords {
			if strings.Contains(cmdLower, keyword) {
				return GuardLevelHigh // Deployment command = high
			}
		}
	}

	// Check working directory for indicators
	if ctx.WorkingDir != "" {
		dirLower := strings.ToLower(ctx.WorkingDir)
		for _, keyword := range indicators.Keywords {
			if isInMeaningfulContext(dirLower, keyword) {
				return GuardLevelHigh // Production path = high
			}
		}
	}

	// Check environment variables
	for key, value := range ctx.Environment {
		keyLower := strings.ToLower(key)
		valueLower := strings.ToLower(value)

		// Check for environment indicators in variable names or values
		envIndicators := []string{"env", "environment", "stage", "tier"}
		isEnvVar := false
		for _, indicator := range envIndicators {
			if strings.Contains(keyLower, indicator) {
				isEnvVar = true
				break
			}
		}

		if isEnvVar {
			for _, keyword := range indicators.Keywords {
				if strings.Contains(valueLower, strings.ToLower(keyword)) {
					return GuardLevelHigh // Production env var = high
				}
			}
		}
	}

	// Default: medium (safe default, not too restrictive, not too permissive)
	return GuardLevelMedium
}

// isInMeaningfulContext checks if a keyword appears in a meaningful context
// (not just as a substring in unrelated words)
func isInMeaningfulContext(text, keyword string) bool {
	lower := strings.ToLower(text)
	keyLower := strings.ToLower(keyword)

	// Check if keyword appears with typical delimiters (either side)
	contexts := []string{
		"/" + keyLower + "/",
		"/" + keyLower + "-",
		"/" + keyLower + "_",
		"/" + keyLower + ".",
		"-" + keyLower + "-",
		"-" + keyLower + "/",
		"-" + keyLower + "_",
		"-" + keyLower + ".",
		"_" + keyLower + "_",
		"_" + keyLower + "/",
		"_" + keyLower + "-",
		"_" + keyLower + ".",
		"." + keyLower + ".",
		"." + keyLower + "/",
		"." + keyLower + "-",
		"@" + keyLower,
		"=" + keyLower,
		" " + keyLower + " ",
		" " + keyLower + "-",
		" " + keyLower + "/",
		":" + keyLower,
	}

	for _, ctx := range contexts {
		if strings.Contains(lower, ctx) {
			return true
		}
	}

	// Also check if at start or end with delimiter
	if strings.HasPrefix(lower, keyLower+" ") ||
		strings.HasPrefix(lower, keyLower+"-") ||
		strings.HasPrefix(lower, keyLower+"_") ||
		strings.HasPrefix(lower, keyLower+".") ||
		strings.HasPrefix(lower, keyLower+"/") ||
		strings.HasSuffix(lower, " "+keyLower) ||
		strings.HasSuffix(lower, "-"+keyLower) ||
		strings.HasSuffix(lower, "_"+keyLower) ||
		strings.HasSuffix(lower, "."+keyLower) ||
		strings.HasSuffix(lower, "/"+keyLower) {
		return true
	}

	return false
}

// GetCurrentGitBranch attempts to detect the current git branch
func GetCurrentGitBranch(workdir string) string {
	// Try reading .git/HEAD
	gitHeadPath := filepath.Join(workdir, ".git", "HEAD")
	data, err := os.ReadFile(gitHeadPath)
	if err != nil {
		return ""
	}

	content := strings.TrimSpace(string(data))
	// Format: ref: refs/heads/branch-name
	if strings.HasPrefix(content, "ref: refs/heads/") {
		return strings.TrimPrefix(content, "ref: refs/heads/")
	}

	return ""
}

// IsGitRepo returns true if workdir contains a .git directory.
func IsGitRepo(workdir string) bool {
	info, err := os.Stat(filepath.Join(workdir, ".git"))
	return err == nil && info.IsDir()
}

// GetGitRemoteURL reads the origin remote URL from .git/config.
func GetGitRemoteURL(workdir string) string {
	data, err := os.ReadFile(filepath.Join(workdir, ".git", "config"))
	if err != nil {
		return ""
	}
	lines := strings.Split(string(data), "\n")
	inOrigin := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == `[remote "origin"]` {
			inOrigin = true
			continue
		}
		if inOrigin {
			if strings.HasPrefix(trimmed, "[") {
				break // next section
			}
			if strings.HasPrefix(trimmed, "url = ") {
				return strings.TrimSpace(strings.TrimPrefix(trimmed, "url = "))
			}
		}
	}
	return ""
}

// ParseRepoName extracts "org/repo" from an HTTPS or SSH git remote URL.
func ParseRepoName(remoteURL string) string {
	if remoteURL == "" {
		return ""
	}
	// SSH: git@github.com:org/repo.git
	if strings.Contains(remoteURL, ":") && strings.Contains(remoteURL, "@") {
		parts := strings.SplitN(remoteURL, ":", 2)
		if len(parts) == 2 {
			name := parts[1]
			name = strings.TrimSuffix(name, ".git")
			return name
		}
	}
	// HTTPS: https://github.com/org/repo.git
	remoteURL = strings.TrimSuffix(remoteURL, ".git")
	parts := strings.Split(remoteURL, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return ""
}

type configPathsKey struct{}

// WithConfigPaths stores the loaded config file paths on the context.
func WithConfigPaths(ctx context.Context, paths []string) context.Context {
	return context.WithValue(ctx, configPathsKey{}, paths)
}

// ConfigPathsFromContext retrieves the loaded config file paths from context.
func ConfigPathsFromContext(ctx context.Context) []string {
	if ctx == nil {
		return nil
	}
	if paths, ok := ctx.Value(configPathsKey{}).([]string); ok {
		return paths
	}
	return nil
}
