package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

// ExecutionMode determines where and how a command runs
type ExecutionMode string

const (
	ExecutionModeHost    ExecutionMode = "host"    // Direct execution on host
	ExecutionModeSandbox ExecutionMode = "sandbox" // Isolated container/process
)

// ExecutionDecision contains the decision logic for execution mode
type ExecutionDecision struct {
	Mode          ExecutionMode
	Reason        string
	RiskLevel     string
	ShouldCache   bool
	CacheKey      string
	SecurityLevel string
}

// SandboxConfig controls sandbox behavior and isolation
type SandboxConfig struct {
	// Runtime configuration
	Runtime          string        // docker, podman, process
	Image            string        // Container image to use
	Timeout          time.Duration // Execution timeout
	WorkDir          string        // Working directory to mount
	
	// Cache configuration
	EnableCache      bool     // Enable dependency caching
	CacheMounts      []string // Paths to mount for caching
	SharedCachePaths []string // System-wide cache paths
	
	// Security posture
	NetworkMode      string   // none, restricted, full
	ReadOnlyRoot     bool     // Make root filesystem read-only
	NoNewPrivileges  bool     // Prevent privilege escalation
	CapDrop          []string // Linux capabilities to drop
	CapAdd           []string // Linux capabilities to add
	SeccompProfile   string   // Path to seccomp profile
	
	// Resource limits
	MemoryLimit      string // Memory limit (e.g., "512m")
	CPULimit         string // CPU limit (e.g., "1.0")
	PidsLimit        int    // Max number of processes
	
	// Environment
	EnvWhitelist     []string          // Environment variables to pass through
	EnvOverrides     map[string]string // Environment variable overrides
	
	// Bind mounts
	BindMounts       []BindMount // Additional bind mounts
	
	// Observability
	EnableMetrics    bool // Collect execution metrics
	LogOutput        bool // Log command output
}

// BindMount represents a host path mounted into the sandbox
type BindMount struct {
	HostPath      string
	ContainerPath string
	ReadOnly      bool
}

// Executor handles command execution with sandbox support
type Executor struct {
	config config.Config
	logger *logging.Logger
	trust  *TrustStore
}

// NewExecutor creates a new sandbox executor
func NewExecutor(cfg config.Config, logger *logging.Logger) (*Executor, error) {
	trustStore, err := NewTrustStore(cfg.Sandbox.TrustStorePath)
	if err != nil {
		return nil, fmt.Errorf("initialize trust store: %w", err)
	}
	
	return &Executor{
		config: cfg,
		logger: logger,
		trust:  trustStore,
	}, nil
}

// DecideExecutionMode determines whether to run in host or sandbox
func (e *Executor) DecideExecutionMode(ctx context.Context, cmdArgs []string, riskLevel string, findings interface{}) ExecutionDecision {
	cmdString := strings.Join(cmdArgs, " ")
	sandboxCfg := e.config.Sandbox
	
	decision := ExecutionDecision{
		Mode:          ExecutionModeHost,
		RiskLevel:     riskLevel,
		ShouldCache:   false,
		SecurityLevel: string(sandboxCfg.SecurityLevel),
	}
	
	// Rule 1: Check if sandboxing is disabled
	if !sandboxCfg.Enabled {
		decision.Reason = "sandboxing disabled in config"
		return decision
	}
	
	// Rule 2: Check trust store - if approved and remembered, run on host
	if e.trust.IsTrusted(cmdString) {
		decision.Reason = "command previously approved and trusted"
		return decision
	}
	
	// Rule 3: Check mode-specific behavior first
	if sandboxCfg.Mode == config.SandboxModeAlways {
		// Always mode sandboxes everything
		decision.Mode = ExecutionModeSandbox
		decision.Reason = "always-sandbox mode enabled"
		decision.ShouldCache = e.shouldEnableCache(cmdArgs)
		decision.CacheKey = e.generateCacheKey(cmdArgs)
		return decision
	}
	
	if sandboxCfg.Mode == config.SandboxModeNever {
		// Never mode never sandboxes
		decision.Reason = "sandboxing disabled by mode"
		return decision
	}
	
	// Rule 4: Check allowlist patterns first - they take precedence
	if e.matchesAllowlist(cmdString) {
		decision.Reason = "matches allowlist pattern"
		return decision
	}
	
	// Rule 5: Check for networked installs (before low-risk check)
	isNetworkedInstall := e.isNetworkedInstall(cmdString)
	
	// Rule 6: Determine if command should run in sandbox based on risk and policy
	shouldSandbox := false
	reason := ""
	
	switch sandboxCfg.Mode {
	case config.SandboxModeAuto:
		// Auto mode: sandbox medium+ risk or when networked install detected
		if riskLevel == "medium" || riskLevel == "high" || riskLevel == "critical" {
			shouldSandbox = true
			reason = fmt.Sprintf("%s risk detected", riskLevel)
		}
		
		// Check for networked install operations
		if isNetworkedInstall {
			shouldSandbox = true
			if reason != "" {
				reason += " + networked install"
			} else {
				reason = "networked install detected"
			}
		}
		
	case config.SandboxModeRisky:
		// Only sandbox high/critical risk
		if riskLevel == "high" || riskLevel == "critical" {
			shouldSandbox = true
			reason = fmt.Sprintf("%s risk - requires isolation", riskLevel)
		}
	}
	
	// If sandbox not triggered, check if we can run on host
	if !shouldSandbox {
		// Low-risk commands run on host
		if riskLevel == "low" && sandboxCfg.SecurityLevel != config.SandboxSecurityParanoid {
			decision.Reason = "low risk, no isolation needed"
			return decision
		}
		
		decision.Reason = "no sandbox triggers matched"
		return decision
	}
	
	// Sandbox triggered
	decision.Mode = ExecutionModeSandbox
	decision.Reason = reason
	decision.ShouldCache = e.shouldEnableCache(cmdArgs)
	decision.CacheKey = e.generateCacheKey(cmdArgs)
	
	return decision
}

// Execute runs a command in the determined execution mode
func (e *Executor) Execute(ctx context.Context, cmdArgs []string, decision ExecutionDecision) error {
	if decision.Mode == ExecutionModeHost {
		return e.executeOnHost(ctx, cmdArgs)
	}
	
	return e.executeInSandbox(ctx, cmdArgs, decision)
}

// executeOnHost runs command directly on the host
func (e *Executor) executeOnHost(ctx context.Context, cmdArgs []string) error {
	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
	}
	
	cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = os.Environ()
	
	return cmd.Run()
}

// executeInSandbox runs command in an isolated sandbox
func (e *Executor) executeInSandbox(ctx context.Context, cmdArgs []string, decision ExecutionDecision) error {
	sandboxCfg := e.buildSandboxConfig(decision)
	
	start := time.Now()
	e.logger.Info("executing in sandbox", map[string]any{
		"command":    strings.Join(cmdArgs, " "),
		"runtime":    sandboxCfg.Runtime,
		"cache":      decision.ShouldCache,
		"network":    sandboxCfg.NetworkMode,
		"reason":     decision.Reason,
	})
	
	var err error
	switch sandboxCfg.Runtime {
	case "docker":
		err = e.executeDocker(ctx, cmdArgs, sandboxCfg)
	case "podman":
		err = e.executePodman(ctx, cmdArgs, sandboxCfg)
	case "process":
		err = e.executeProcess(ctx, cmdArgs, sandboxCfg)
	default:
		return fmt.Errorf("unsupported sandbox runtime: %s", sandboxCfg.Runtime)
	}
	
	duration := time.Since(start)
	
	if err == nil {
		e.logger.Info("sandbox execution completed", map[string]any{
			"duration": duration.String(),
			"cached":   decision.ShouldCache,
		})
	}
	
	return err
}

// buildSandboxConfig creates runtime configuration based on security level
func (e *Executor) buildSandboxConfig(decision ExecutionDecision) SandboxConfig {
	cfg := e.config.Sandbox
	workDir, _ := os.Getwd()
	
	sandboxCfg := SandboxConfig{
		Runtime:         cfg.Runtime,
		Image:           cfg.Image,
		Timeout:         time.Duration(cfg.Timeout) * time.Second,
		WorkDir:         workDir,
		EnableCache:     decision.ShouldCache,
		NetworkMode:     cfg.NetworkMode,
		NoNewPrivileges: true,
		EnableMetrics:   cfg.EnableMetrics,
		LogOutput:       cfg.LogOutput,
		EnvWhitelist:    cfg.EnvWhitelist,
		EnvOverrides:    make(map[string]string),
	}
	
	// Configure security posture based on level
	switch cfg.SecurityLevel {
	case config.SandboxSecurityPermissive:
		sandboxCfg.NetworkMode = "full"
		sandboxCfg.ReadOnlyRoot = false
		sandboxCfg.MemoryLimit = "2g"
		sandboxCfg.CPULimit = "2.0"
		
	case config.SandboxSecurityBalanced:
		sandboxCfg.NetworkMode = cfg.NetworkMode
		if sandboxCfg.NetworkMode == "" {
			sandboxCfg.NetworkMode = "restricted"
		}
		sandboxCfg.ReadOnlyRoot = false
		sandboxCfg.MemoryLimit = "1g"
		sandboxCfg.CPULimit = "1.0"
		sandboxCfg.CapDrop = []string{"ALL"}
		sandboxCfg.CapAdd = []string{"CHOWN", "DAC_OVERRIDE", "FOWNER", "SETGID", "SETUID"}
		
	case config.SandboxSecurityStrict:
		sandboxCfg.NetworkMode = "restricted"
		sandboxCfg.ReadOnlyRoot = true
		sandboxCfg.MemoryLimit = "512m"
		sandboxCfg.CPULimit = "0.5"
		sandboxCfg.PidsLimit = 100
		sandboxCfg.CapDrop = []string{"ALL"}
		sandboxCfg.CapAdd = []string{"CHOWN", "DAC_OVERRIDE"}
		sandboxCfg.SeccompProfile = cfg.SeccompProfile
		
	case config.SandboxSecurityParanoid:
		sandboxCfg.NetworkMode = "none"
		sandboxCfg.ReadOnlyRoot = true
		sandboxCfg.MemoryLimit = "256m"
		sandboxCfg.CPULimit = "0.25"
		sandboxCfg.PidsLimit = 50
		sandboxCfg.CapDrop = []string{"ALL"}
		sandboxCfg.SeccompProfile = cfg.SeccompProfile
	}
	
	// Add cache mounts if enabled
	if decision.ShouldCache {
		sandboxCfg.CacheMounts = e.getCacheMounts()
	}
	
	// Add custom bind mounts
	for _, mount := range cfg.BindMounts {
		sandboxCfg.BindMounts = append(sandboxCfg.BindMounts, BindMount{
			HostPath:      mount.HostPath,
			ContainerPath: mount.ContainerPath,
			ReadOnly:      mount.ReadOnly,
		})
	}
	
	return sandboxCfg
}

// executeDocker runs command in Docker container
func (e *Executor) executeDocker(ctx context.Context, cmdArgs []string, cfg SandboxConfig) error {
	dockerArgs := e.buildDockerArgs(cfg, cmdArgs)
	
	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	return cmd.Run()
}

// executePodman runs command in Podman container
func (e *Executor) executePodman(ctx context.Context, cmdArgs []string, cfg SandboxConfig) error {
	podmanArgs := e.buildDockerArgs(cfg, cmdArgs) // Podman is CLI-compatible with Docker
	
	cmd := exec.CommandContext(ctx, "podman", podmanArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	return cmd.Run()
}

// executeProcess runs command as isolated process (Linux namespaces)
func (e *Executor) executeProcess(ctx context.Context, cmdArgs []string, cfg SandboxConfig) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("process sandbox mode only supported on Linux")
	}
	
	// Use unshare for namespace isolation
	unshareArgs := []string{
		"--map-root-user", // Map to root in new namespace
		"--pid",           // PID namespace
		"--mount",         // Mount namespace
	}
	
	if cfg.NetworkMode == "none" {
		unshareArgs = append(unshareArgs, "--net") // Network namespace
	}
	
	unshareArgs = append(unshareArgs, "--")
	unshareArgs = append(unshareArgs, cmdArgs...)
	
	cmd := exec.CommandContext(ctx, "unshare", unshareArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = e.buildEnv(cfg)
	
	return cmd.Run()
}

// buildDockerArgs constructs Docker/Podman CLI arguments
func (e *Executor) buildDockerArgs(cfg SandboxConfig, cmdArgs []string) []string {
	args := []string{"run", "--rm", "-i"}
	
	// Add TTY if stdin is a terminal
	if isTerminal(os.Stdin) {
		args = append(args, "-t")
	}
	
	// Working directory mount
	args = append(args, "-v", fmt.Sprintf("%s:%s", cfg.WorkDir, cfg.WorkDir))
	args = append(args, "-w", cfg.WorkDir)
	
	// Network mode
	switch cfg.NetworkMode {
	case "none":
		args = append(args, "--network", "none")
	case "restricted":
		// TODO: Implement restricted network (custom network with egress filtering)
		args = append(args, "--network", "bridge")
	case "full":
		args = append(args, "--network", "host")
	default:
		args = append(args, "--network", "bridge")
	}
	
	// Security options
	if cfg.ReadOnlyRoot {
		args = append(args, "--read-only")
		// Add tmpfs for /tmp
		args = append(args, "--tmpfs", "/tmp:rw,noexec,nosuid,size=100m")
	}
	
	if cfg.NoNewPrivileges {
		args = append(args, "--security-opt", "no-new-privileges")
	}
	
	// Capabilities
	for _, cap := range cfg.CapDrop {
		args = append(args, "--cap-drop", cap)
	}
	for _, cap := range cfg.CapAdd {
		args = append(args, "--cap-add", cap)
	}
	
	// Seccomp profile
	if cfg.SeccompProfile != "" {
		args = append(args, "--security-opt", fmt.Sprintf("seccomp=%s", cfg.SeccompProfile))
	}
	
	// Resource limits
	if cfg.MemoryLimit != "" {
		args = append(args, "--memory", cfg.MemoryLimit)
	}
	if cfg.CPULimit != "" {
		args = append(args, "--cpus", cfg.CPULimit)
	}
	if cfg.PidsLimit > 0 {
		args = append(args, "--pids-limit", fmt.Sprintf("%d", cfg.PidsLimit))
	}
	
	// Cache mounts
	for _, mount := range cfg.CacheMounts {
		args = append(args, "-v", mount)
	}
	
	// Custom bind mounts
	for _, mount := range cfg.BindMounts {
		mountStr := fmt.Sprintf("%s:%s", mount.HostPath, mount.ContainerPath)
		if mount.ReadOnly {
			mountStr += ":ro"
		}
		args = append(args, "-v", mountStr)
	}
	
	// Environment variables
	for _, envVar := range cfg.EnvWhitelist {
		if val, ok := os.LookupEnv(envVar); ok {
			args = append(args, "-e", fmt.Sprintf("%s=%s", envVar, val))
		}
	}
	for key, val := range cfg.EnvOverrides {
		args = append(args, "-e", fmt.Sprintf("%s=%s", key, val))
	}
	
	// Image
	args = append(args, cfg.Image)
	
	// Command
	args = append(args, cmdArgs...)
	
	return args
}

// buildEnv constructs environment variables for process sandbox
func (e *Executor) buildEnv(cfg SandboxConfig) []string {
	env := []string{}
	
	for _, envVar := range cfg.EnvWhitelist {
		if val, ok := os.LookupEnv(envVar); ok {
			env = append(env, fmt.Sprintf("%s=%s", envVar, val))
		}
	}
	
	for key, val := range cfg.EnvOverrides {
		env = append(env, fmt.Sprintf("%s=%s", key, val))
	}
	
	return env
}

// matchesAllowlist checks if command matches any allowlist pattern
func (e *Executor) matchesAllowlist(cmdString string) bool {
	for _, pattern := range e.config.Policies.Allowlist {
		if pattern != "" && strings.Contains(cmdString, pattern) {
			return true
		}
	}
	return false
}

// isNetworkedInstall detects if command is a networked install operation
func (e *Executor) isNetworkedInstall(cmdString string) bool {
	lower := strings.ToLower(cmdString)
	
	installCommands := []string{
		"npm install", "yarn install", "pnpm install",
		"pip install", "pip3 install",
		"cargo install", "cargo build",
		"go get", "go install",
		"gem install",
		"apt-get install", "apt install",
		"yum install", "dnf install",
		"brew install",
		"composer install",
		"maven install", "mvn install",
		"gradle build",
	}
	
	for _, cmd := range installCommands {
		if strings.Contains(lower, cmd) {
			return true
		}
	}
	
	return false
}

// shouldEnableCache determines if caching should be enabled for this command
func (e *Executor) shouldEnableCache(cmdArgs []string) bool {
	if !e.config.Sandbox.EnableCache {
		return false
	}
	
	cmdString := strings.ToLower(strings.Join(cmdArgs, " "))
	
	// Enable cache for package managers and build tools
	cacheableCommands := []string{
		"npm", "yarn", "pnpm",
		"pip", "pip3",
		"cargo",
		"go",
		"gem",
		"composer",
		"maven", "mvn",
		"gradle",
	}
	
	for _, cmd := range cacheableCommands {
		if strings.HasPrefix(cmdString, cmd+" ") || cmdString == cmd {
			return true
		}
	}
	
	return false
}

// generateCacheKey creates a unique key for caching
func (e *Executor) generateCacheKey(cmdArgs []string) string {
	// Simple key based on command and working directory
	workDir, _ := os.Getwd()
	return fmt.Sprintf("%s:%s", filepath.Base(workDir), cmdArgs[0])
}

// getCacheMounts returns cache mount specifications
func (e *Executor) getCacheMounts() []string {
	mounts := []string{}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return mounts
	}
	
	// Common cache directories
	cacheDirs := map[string]string{
		// Node.js
		filepath.Join(homeDir, ".npm"):     "/.npm",
		filepath.Join(homeDir, ".yarn"):    "/.yarn",
		filepath.Join(homeDir, ".pnpm"):    "/.pnpm",
		// Python
		filepath.Join(homeDir, ".cache", "pip"): "/.cache/pip",
		// Go
		filepath.Join(homeDir, "go", "pkg"): "/go/pkg",
		// Rust
		filepath.Join(homeDir, ".cargo"): "/.cargo",
		// Ruby
		filepath.Join(homeDir, ".gem"): "/.gem",
	}
	
	for hostPath, containerPath := range cacheDirs {
		if _, err := os.Stat(hostPath); err == nil {
			mounts = append(mounts, fmt.Sprintf("%s:%s", hostPath, containerPath))
		}
	}
	
	return mounts
}

// isTerminal checks if a file is a terminal
func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

