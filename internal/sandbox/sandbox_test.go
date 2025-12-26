package sandbox

import (
	"context"
	"os"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/analyzer"
	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestDecideExecutionMode(t *testing.T) {
	logger := logging.NewLogger("text", os.Stderr)
	
	tests := []struct {
		name           string
		config         config.Config
		cmdArgs        []string
		riskLevel      string
		expectedMode   ExecutionMode
		expectedReason string
	}{
		{
			name: "sandboxing disabled",
			config: config.Config{
				Sandbox: config.SandboxConfig{Enabled: false},
			},
			cmdArgs:        []string{"echo", "test"},
			riskLevel:      "medium",
			expectedMode:   ExecutionModeHost,
			expectedReason: "sandboxing disabled in config",
		},
		{
			name: "low risk with auto mode",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					Mode:          config.SandboxModeAuto,
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs:        []string{"echo", "test"},
			riskLevel:      "low",
			expectedMode:   ExecutionModeHost,
			expectedReason: "low risk, no isolation needed",
		},
		{
			name: "medium risk with auto mode",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					Mode:          config.SandboxModeAuto,
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs:      []string{"curl", "http://example.com"},
			riskLevel:    "medium",
			expectedMode: ExecutionModeSandbox,
		},
		{
			name: "high risk with auto mode",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					Mode:          config.SandboxModeAuto,
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs:      []string{"rm", "-rf", "/tmp/test"},
			riskLevel:    "high",
			expectedMode: ExecutionModeSandbox,
		},
		{
			name: "always mode",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					Mode:          config.SandboxModeAlways,
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs:        []string{"echo", "test"},
			riskLevel:      "low",
			expectedMode:   ExecutionModeSandbox,
			expectedReason: "always-sandbox mode enabled",
		},
		{
			name: "never mode",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					Mode:          config.SandboxModeNever,
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs:        []string{"rm", "-rf", "/tmp/test"},
			riskLevel:      "high",
			expectedMode:   ExecutionModeHost,
			expectedReason: "sandboxing disabled by mode",
		},
		{
			name: "networked install detected",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					Mode:          config.SandboxModeAuto,
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs:      []string{"npm", "install", "express"},
			riskLevel:    "low",
			expectedMode: ExecutionModeSandbox,
		},
		{
			name: "allowlist match",
			config: config.Config{
				Policies: config.PolicyConfig{
					Allowlist: []string{"echo", "ls"},
				},
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					Mode:          config.SandboxModeAuto,
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs:        []string{"echo", "test"},
			riskLevel:      "medium",
			expectedMode:   ExecutionModeHost,
			expectedReason: "matches allowlist pattern",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := NewExecutor(tt.config, logger)
			if err != nil {
				t.Fatalf("NewExecutor() error = %v", err)
			}
			
			decision := executor.DecideExecutionMode(context.Background(), tt.cmdArgs, tt.riskLevel, nil)
			
			if decision.Mode != tt.expectedMode {
				t.Errorf("DecideExecutionMode() mode = %v, want %v", decision.Mode, tt.expectedMode)
			}
			
			if tt.expectedReason != "" && decision.Reason != tt.expectedReason {
				t.Errorf("DecideExecutionMode() reason = %v, want %v", decision.Reason, tt.expectedReason)
			}
		})
	}
}

func TestIsNetworkedInstall(t *testing.T) {
	logger := logging.NewLogger("text", os.Stderr)
	cfg := config.Config{
		Sandbox: config.SandboxConfig{
			Enabled: true,
			Mode:    config.SandboxModeAuto,
		},
	}
	
	executor, err := NewExecutor(cfg, logger)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}
	
	tests := []struct {
		command  string
		expected bool
	}{
		{"npm install express", true},
		{"yarn install", true},
		{"pip install requests", true},
		{"cargo install ripgrep", true},
		{"go get github.com/spf13/cobra", true},
		{"apt-get install vim", true},
		{"echo hello", false},
		{"ls -la", false},
		{"git status", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			result := executor.isNetworkedInstall(tt.command)
			if result != tt.expected {
				t.Errorf("isNetworkedInstall(%q) = %v, want %v", tt.command, result, tt.expected)
			}
		})
	}
}

func TestShouldEnableCache(t *testing.T) {
	logger := logging.NewLogger("text", os.Stderr)
	cfg := config.Config{
		Sandbox: config.SandboxConfig{
			Enabled:     true,
			EnableCache: true,
		},
	}
	
	executor, err := NewExecutor(cfg, logger)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}
	
	tests := []struct {
		cmdArgs  []string
		expected bool
	}{
		{[]string{"npm", "install"}, true},
		{[]string{"yarn", "build"}, true},
		{[]string{"pip", "install", "-r", "requirements.txt"}, true},
		{[]string{"cargo", "build"}, true},
		{[]string{"go", "test"}, true},
		{[]string{"echo", "hello"}, false},
		{[]string{"ls", "-la"}, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.cmdArgs[0], func(t *testing.T) {
			result := executor.shouldEnableCache(tt.cmdArgs)
			if result != tt.expected {
				t.Errorf("shouldEnableCache(%v) = %v, want %v", tt.cmdArgs, result, tt.expected)
			}
		})
	}
}

func TestBuildSandboxConfig(t *testing.T) {
	logger := logging.NewLogger("text", os.Stderr)
	
	tests := []struct {
		name          string
		securityLevel config.SandboxSecurityLevel
		checkFunc     func(*testing.T, SandboxConfig)
	}{
		{
			name:          "permissive",
			securityLevel: config.SandboxSecurityPermissive,
			checkFunc: func(t *testing.T, cfg SandboxConfig) {
				if cfg.NetworkMode != "full" {
					t.Errorf("NetworkMode = %v, want full", cfg.NetworkMode)
				}
				if cfg.ReadOnlyRoot {
					t.Error("ReadOnlyRoot should be false for permissive")
				}
			},
		},
		{
			name:          "balanced",
			securityLevel: config.SandboxSecurityBalanced,
			checkFunc: func(t *testing.T, cfg SandboxConfig) {
				if cfg.ReadOnlyRoot {
					t.Error("ReadOnlyRoot should be false for balanced")
				}
				if len(cfg.CapDrop) == 0 {
					t.Error("CapDrop should not be empty for balanced")
				}
			},
		},
		{
			name:          "strict",
			securityLevel: config.SandboxSecurityStrict,
			checkFunc: func(t *testing.T, cfg SandboxConfig) {
				if !cfg.ReadOnlyRoot {
					t.Error("ReadOnlyRoot should be true for strict")
				}
				if cfg.NetworkMode != "restricted" {
					t.Errorf("NetworkMode = %v, want restricted", cfg.NetworkMode)
				}
			},
		},
		{
			name:          "paranoid",
			securityLevel: config.SandboxSecurityParanoid,
			checkFunc: func(t *testing.T, cfg SandboxConfig) {
				if !cfg.ReadOnlyRoot {
					t.Error("ReadOnlyRoot should be true for paranoid")
				}
				if cfg.NetworkMode != "none" {
					t.Errorf("NetworkMode = %v, want none", cfg.NetworkMode)
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					SecurityLevel: tt.securityLevel,
					Runtime:       "docker",
					Image:         "ubuntu:22.04",
				},
			}
			
			executor, err := NewExecutor(cfg, logger)
			if err != nil {
				t.Fatalf("NewExecutor() error = %v", err)
			}
			
			decision := ExecutionDecision{
				Mode:        ExecutionModeSandbox,
				ShouldCache: false,
			}
			
			sandboxCfg := executor.buildSandboxConfig(decision)
			tt.checkFunc(t, sandboxCfg)
		})
	}
}

func TestBuildDockerArgs(t *testing.T) {
	logger := logging.NewLogger("text", os.Stderr)
	cfg := config.Config{
		Sandbox: config.SandboxConfig{
			Enabled:       true,
			SecurityLevel: config.SandboxSecurityBalanced,
			Runtime:       "docker",
			Image:         "ubuntu:22.04",
		},
	}
	
	executor, err := NewExecutor(cfg, logger)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}
	
	sandboxCfg := SandboxConfig{
		Runtime:         "docker",
		Image:           "ubuntu:22.04",
		WorkDir:         "/test",
		NetworkMode:     "none",
		ReadOnlyRoot:    true,
		NoNewPrivileges: true,
		MemoryLimit:     "512m",
		CPULimit:        "0.5",
	}
	
	cmdArgs := []string{"echo", "test"}
	args := executor.buildDockerArgs(sandboxCfg, cmdArgs)
	
	// Check for essential Docker flags
	containsFlag := func(flag string) bool {
		for _, arg := range args {
			if arg == flag {
				return true
			}
		}
		return false
	}
	
	if !containsFlag("--rm") {
		t.Error("Missing --rm flag")
	}
	
	if !containsFlag("--read-only") {
		t.Error("Missing --read-only flag")
	}
	
	if !containsFlag("--memory") {
		t.Error("Missing --memory flag")
	}
	
	// Check image and command are at the end
	if args[len(args)-3] != "ubuntu:22.04" {
		t.Errorf("Image not in correct position: %v", args)
	}
	
	if args[len(args)-2] != "echo" || args[len(args)-1] != "test" {
		t.Errorf("Command not in correct position: %v", args)
	}
}

func TestGenerateCacheKey(t *testing.T) {
	logger := logging.NewLogger("text", os.Stderr)
	cfg := config.Config{
		Sandbox: config.SandboxConfig{Enabled: true},
	}
	
	executor, err := NewExecutor(cfg, logger)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}
	
	key1 := executor.generateCacheKey([]string{"npm", "install"})
	key2 := executor.generateCacheKey([]string{"npm", "install"})
	
	if key1 != key2 {
		t.Error("Same command should generate same cache key")
	}
	
	key3 := executor.generateCacheKey([]string{"yarn", "install"})
	if key1 == key3 {
		t.Error("Different commands should generate different cache keys")
	}
}

func TestExecutionDecisionCaching(t *testing.T) {
	logger := logging.NewLogger("text", os.Stderr)
	cfg := config.Config{
		Sandbox: config.SandboxConfig{
			Enabled:     true,
			Mode:        config.SandboxModeAuto,
			EnableCache: true,
		},
	}
	
	executor, err := NewExecutor(cfg, logger)
	if err != nil {
		t.Fatalf("NewExecutor() error = %v", err)
	}
	
	// npm install should enable caching
	decision := executor.DecideExecutionMode(context.Background(), []string{"npm", "install"}, "low", nil)
	if !decision.ShouldCache {
		t.Error("npm install should enable caching")
	}
	
	// echo should not enable caching
	decision = executor.DecideExecutionMode(context.Background(), []string{"echo", "test"}, "low", nil)
	if decision.ShouldCache {
		t.Error("echo should not enable caching")
	}
}

// Benchmark tests
func BenchmarkDecideExecutionMode(b *testing.B) {
	logger := logging.NewLogger("text", os.Stderr)
	cfg := config.Config{
		Sandbox: config.SandboxConfig{
			Enabled:       true,
			Mode:          config.SandboxModeAuto,
			SecurityLevel: config.SandboxSecurityBalanced,
		},
	}
	
	executor, _ := NewExecutor(cfg, logger)
	cmdArgs := []string{"npm", "install", "express"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.DecideExecutionMode(context.Background(), cmdArgs, "medium", nil)
	}
}

func BenchmarkBuildDockerArgs(b *testing.B) {
	logger := logging.NewLogger("text", os.Stderr)
	cfg := config.Config{
		Sandbox: config.SandboxConfig{
			Enabled:       true,
			SecurityLevel: config.SandboxSecurityBalanced,
			Runtime:       "docker",
			Image:         "ubuntu:22.04",
		},
	}
	
	executor, _ := NewExecutor(cfg, logger)
	sandboxCfg := SandboxConfig{
		Runtime:      "docker",
		Image:        "ubuntu:22.04",
		WorkDir:      "/test",
		NetworkMode:  "restricted",
		MemoryLimit:  "512m",
		CPULimit:     "0.5",
	}
	cmdArgs := []string{"echo", "test"}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		executor.buildDockerArgs(sandboxCfg, cmdArgs)
	}
}

// TestMandatorySandboxingForCriticalCommands tests that critical commands
// cannot be bypassed by trust store, allowlist, or configuration
func TestMandatorySandboxingForCriticalCommands(t *testing.T) {
	logger := logging.NewLogger("text", os.Stderr)
	
	criticalFindings := []analyzer.Finding{
		{
			Code:     "DANGEROUS_DELETE_ROOT",
			Severity: "critical",
		},
	}
	
	tests := []struct {
		name           string
		config         config.Config
		cmdArgs        []string
		riskLevel      string
		findings       []analyzer.Finding
		expectedMode   ExecutionMode
		expectedReason string
		description    string
	}{
		{
			name: "critical command with trust store bypass attempt",
			config: config.Config{
				Policies: config.PolicyConfig{
					Allowlist: []string{"rm"}, // Even in allowlist
				},
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					Mode:          config.SandboxModeAuto,
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs:        []string{"rm", "-r", "/*"},
			riskLevel:      "critical",
			findings:       criticalFindings,
			expectedMode:   ExecutionModeSandbox,
			expectedReason: "CRITICAL: Mandatory sandbox required for system-destructive command",
			description:    "Critical command should sandbox even if in allowlist",
		},
		{
			name: "critical command with never mode attempt",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					Mode:          config.SandboxModeNever, // Even with never mode
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs:        []string{"rm", "-r", "/*"},
			riskLevel:      "critical",
			findings:       criticalFindings,
			expectedMode:   ExecutionModeSandbox,
			expectedReason: "CRITICAL: Mandatory sandbox required for system-destructive command",
			description:    "Critical command should sandbox even with never mode",
		},
		{
			name: "critical command with sandbox disabled",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Enabled:       false, // Sandbox disabled
					Mode:          config.SandboxModeAuto,
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs:        []string{"rm", "-r", "/*"},
			riskLevel:      "critical",
			findings:       criticalFindings,
			expectedMode:   ExecutionModeSandbox, // Still should try to sandbox
			expectedReason: "CRITICAL: Mandatory sandbox required for system-destructive command",
			description:    "Critical command should attempt sandbox even if disabled (will fail later)",
		},
		{
			name: "non-critical command can use allowlist",
			config: config.Config{
				Policies: config.PolicyConfig{
					Allowlist: []string{"echo"},
				},
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					Mode:          config.SandboxModeAuto,
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs:        []string{"echo", "test"},
			riskLevel:      "low",
			findings:       []analyzer.Finding{},
			expectedMode:   ExecutionModeHost,
			expectedReason: "matches allowlist pattern",
			description:    "Non-critical commands can still use allowlist",
		},
		{
			name: "critical fork bomb detection",
			config: config.Config{
				Sandbox: config.SandboxConfig{
					Enabled:       true,
					Mode:          config.SandboxModeAuto,
					SecurityLevel: config.SandboxSecurityBalanced,
				},
			},
			cmdArgs: []string{":(){", ":|:&", "};:"},
			riskLevel: "critical",
			findings: []analyzer.Finding{
				{
					Code:     "FORK_BOMB",
					Severity: "critical",
				},
			},
			expectedMode:   ExecutionModeSandbox,
			expectedReason: "CRITICAL: Mandatory sandbox required for system-destructive command",
			description:    "Fork bomb should be mandatory sandboxed",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, err := NewExecutor(tt.config, logger)
			if err != nil {
				t.Fatalf("NewExecutor() error = %v", err)
			}
			
			// Add to trust store to test bypass attempt
			if tt.name == "critical command with trust store bypass attempt" {
				cmdString := "rm -r /*"
				_ = executor.trust.Add(cmdString, 0, "test")
			}
			
			decision := executor.DecideExecutionMode(
				context.Background(),
				tt.cmdArgs,
				tt.riskLevel,
				tt.findings,
			)
			
			if decision.Mode != tt.expectedMode {
				t.Errorf("DecideExecutionMode() mode = %v, want %v (%s)",
					decision.Mode, tt.expectedMode, tt.description)
			}
			
			if tt.expectedReason != "" && !containsSubstring(decision.Reason, tt.expectedReason) {
				t.Errorf("DecideExecutionMode() reason = %v, want to contain %v (%s)",
					decision.Reason, tt.expectedReason, tt.description)
			}
		})
	}
}

// containsSubstring checks if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && findSubstringInString(s, substr)))
}

func findSubstringInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

