package agentposture

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

// CheckHygiene runs security configuration checks against the loaded config
// and the workspace state.
func CheckHygiene(cfg config.Config, mgr *session.Manager, targetPath string) []HygieneCheck {
	var checks []HygieneCheck

	// Sandbox enabled
	checks = append(checks, checkSandboxEnabled(cfg))

	// Sandbox mode
	checks = append(checks, checkSandboxMode(cfg))

	// Env protection
	checks = append(checks, checkEnvProtection(cfg))

	// Env dotenv blocking
	checks = append(checks, checkDotenvBlocking(cfg))

	// CVE scanning
	checks = append(checks, checkCVEScanning(cfg))

	// Soft delete
	checks = append(checks, checkSoftDelete(cfg))

	// Guard level
	checks = append(checks, checkGuardLevel(cfg))

	// Git ops monitoring
	checks = append(checks, checkGitOpsMonitoring(cfg))

	// Force git blocking
	checks = append(checks, checkForceGitBlocking(cfg))

	// Sessions active
	checks = append(checks, checkSessionsActive(mgr))

	// Agent seeded
	checks = append(checks, checkAgentSeeded(targetPath))

	// Behavioral profiles
	checks = append(checks, checkBehavioralProfiles())

	// LLM trace integration
	checks = append(checks, checkLLMTraceConfigured(cfg))

	return checks
}

func checkSandboxEnabled(cfg config.Config) HygieneCheck {
	if cfg.Sandbox.Enabled {
		return HygieneCheck{
			Name:   "Sandbox enabled",
			Status: "pass",
			Detail: "Sandbox is enabled",
		}
	}
	return HygieneCheck{
		Name:   "Sandbox enabled",
		Status: "fail",
		Detail: "Sandbox is disabled; commands run without isolation",
	}
}

func checkSandboxMode(cfg config.Config) HygieneCheck {
	mode := string(cfg.Sandbox.Mode)
	switch cfg.Sandbox.Mode {
	case config.SandboxModeAlways, config.SandboxModeRisky:
		return HygieneCheck{
			Name:   "Sandbox mode",
			Status: "pass",
			Detail: "Sandbox mode: " + mode,
		}
	case config.SandboxModeAuto:
		return HygieneCheck{
			Name:   "Sandbox mode",
			Status: "warn",
			Detail: "Sandbox mode: auto (recommend always or risky)",
		}
	default:
		return HygieneCheck{
			Name:   "Sandbox mode",
			Status: "fail",
			Detail: "Sandbox mode: " + mode + " (commands not sandboxed)",
		}
	}
}

func checkEnvProtection(cfg config.Config) HygieneCheck {
	if cfg.EnvProtection.Enabled {
		return HygieneCheck{
			Name:   "Env protection",
			Status: "pass",
			Detail: "Environment variable protection enabled",
		}
	}
	return HygieneCheck{
		Name:   "Env protection",
		Status: "fail",
		Detail: "Environment variable protection disabled",
	}
}

func checkDotenvBlocking(cfg config.Config) HygieneCheck {
	if cfg.EnvProtection.BlockDotenvRead {
		return HygieneCheck{
			Name:   "Dotenv blocking",
			Status: "pass",
			Detail: "Dotenv file read blocking enabled",
		}
	}
	return HygieneCheck{
		Name:   "Dotenv blocking",
		Status: "fail",
		Detail: "Dotenv file reads are not blocked",
	}
}

func checkCVEScanning(cfg config.Config) HygieneCheck {
	if cfg.CVE.Enabled {
		return HygieneCheck{
			Name:   "CVE scanning",
			Status: "pass",
			Detail: "CVE scanning enabled",
		}
	}
	return HygieneCheck{
		Name:   "CVE scanning",
		Status: "fail",
		Detail: "CVE scanning disabled; dependencies not checked for vulnerabilities",
	}
}

func checkSoftDelete(cfg config.Config) HygieneCheck {
	if cfg.SoftDelete.Enabled {
		return HygieneCheck{
			Name:   "Soft delete",
			Status: "pass",
			Detail: "Soft delete (backup on rm) enabled",
		}
	}
	return HygieneCheck{
		Name:   "Soft delete",
		Status: "fail",
		Detail: "Soft delete disabled; deleted files cannot be recovered",
	}
}

func checkGuardLevel(cfg config.Config) HygieneCheck {
	level := string(cfg.GuardLevel.Level)
	switch cfg.GuardLevel.Level {
	case config.GuardLevelHigh, config.GuardLevelParanoid:
		return HygieneCheck{
			Name:   "Guard level",
			Status: "pass",
			Detail: "Guard level: " + level,
		}
	case config.GuardLevelMedium, config.GuardLevelAuto:
		return HygieneCheck{
			Name:   "Guard level",
			Status: "warn",
			Detail: "Guard level: " + level + " (recommend high for production)",
		}
	default:
		return HygieneCheck{
			Name:   "Guard level",
			Status: "fail",
			Detail: "Guard level: " + level + " (insufficient protection)",
		}
	}
}

func checkGitOpsMonitoring(cfg config.Config) HygieneCheck {
	if cfg.Policies.MonitorGitOps {
		return HygieneCheck{
			Name:   "Git ops monitoring",
			Status: "pass",
			Detail: "Git operations are monitored",
		}
	}
	return HygieneCheck{
		Name:   "Git ops monitoring",
		Status: "fail",
		Detail: "Git operations not monitored",
	}
}

func checkForceGitBlocking(cfg config.Config) HygieneCheck {
	if cfg.Policies.BlockForceGit {
		return HygieneCheck{
			Name:   "Force git blocking",
			Status: "pass",
			Detail: "Force push/reset operations are blocked",
		}
	}
	return HygieneCheck{
		Name:   "Force git blocking",
		Status: "fail",
		Detail: "Force push/reset operations are allowed",
	}
}

func checkSessionsActive(mgr *session.Manager) HygieneCheck {
	if mgr == nil {
		return HygieneCheck{
			Name:   "Sessions active",
			Status: "fail",
			Detail: "Session manager unavailable",
		}
	}
	sessions, err := mgr.List()
	if err != nil || len(sessions) == 0 {
		return HygieneCheck{
			Name:   "Sessions active",
			Status: "fail",
			Detail: "No sessions found; agent activity is not being tracked",
		}
	}
	return HygieneCheck{
		Name:   "Sessions active",
		Status: "pass",
		Detail: fmt.Sprintf("%d session(s) recorded", len(sessions)),
	}
}

func checkAgentSeeded(targetPath string) HygieneCheck {
	// Check for VectraGuard markers in common agent instruction files
	markerFiles := []string{
		"CLAUDE.md", ".claude.md", "AGENTS.md",
		".cursor/rules/vectra-guard.md",
		".github/copilot-instructions.md",
		".windsurf/rules.md",
	}

	for _, f := range markerFiles {
		full := filepath.Join(targetPath, f)
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		content := string(data)
		if strings.Contains(content, "vectra-guard") || strings.Contains(content, "vectraguard") {
			return HygieneCheck{
				Name:   "Agent seeded",
				Status: "pass",
				Detail: "VectraGuard markers found in " + f,
			}
		}
	}

	return HygieneCheck{
		Name:   "Agent seeded",
		Status: "fail",
		Detail: "No VectraGuard seed markers found; run `vg seed agents`",
	}
}

func checkBehavioralProfiles() HygieneCheck {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return HygieneCheck{
				Name:   "Behavioral profiles",
				Status: "fail",
				Detail: "Cannot determine home directory",
			}
		}
	}

	profileDir := filepath.Join(homeDir, ".vectra-guard", "profiles")
	entries, err := os.ReadDir(profileDir)
	if err != nil || len(entries) == 0 {
		return HygieneCheck{
			Name:   "Behavioral profiles",
			Status: "fail",
			Detail: "No behavioral profiles found; run `vg behavioral profile --session <id>`",
		}
	}

	return HygieneCheck{
		Name:   "Behavioral profiles",
		Status: "pass",
		Detail: fmt.Sprintf("%d behavioral profile(s) stored", len(entries)),
	}
}

func checkLLMTraceConfigured(cfg config.Config) HygieneCheck {
	if !cfg.LLMTrace.Enabled {
		return HygieneCheck{
			Name:   "LLM trace integration",
			Status: "warn",
			Detail: "LLM trace monitoring not configured; enable in llmtrace config",
		}
	}
	if cfg.LLMTrace.Host == "" || cfg.LLMTrace.PublicKey == "" {
		return HygieneCheck{
			Name:   "LLM trace integration",
			Status: "fail",
			Detail: "LLM trace enabled but credentials incomplete (host/public_key required)",
		}
	}
	return HygieneCheck{
		Name:   "LLM trace integration",
		Status: "pass",
		Detail: "LLM trace monitoring configured",
	}
}
