package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/analyzer"
	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/sandbox"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

func runExec(ctx context.Context, cmdArgs []string, interactive bool, sessionID string) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
	}

	// Check for user bypass
	bypassEnvVar := cfg.GuardLevel.BypassEnvVar
	if bypassEnvVar == "" {
		bypassEnvVar = "VECTRAGUARD_BYPASS"
	}
	
	if cfg.GuardLevel.AllowUserBypass && os.Getenv(bypassEnvVar) != "" {
		bypassValue := os.Getenv(bypassEnvVar)
		// Require a specific non-trivial value that agents are unlikely to guess
		// Users can set: export VECTRAGUARD_BYPASS="$(date +%s | sha256sum | head -c 16)"
		// Or a simpler approach: export VECTRAGUARD_BYPASS="i-am-human-$(whoami)"
		if len(bypassValue) >= 10 && !isLikelyAgentBypass(bypassValue) {
			logger.Info("command executed with user bypass", map[string]any{
				"command": strings.Join(cmdArgs, " "),
				"bypass":  "user authenticated",
			})
			// Execute without protection
			return executeCommandDirectly(cmdArgs)
		}
	}
	
	// Check guard level
	if cfg.GuardLevel.Level == config.GuardLevelOff {
		logger.Info("guard level is OFF - executing without protection", map[string]any{
			"command": strings.Join(cmdArgs, " "),
		})
		return executeCommandDirectly(cmdArgs)
	}

	cmdName := cmdArgs[0]
	args := cmdArgs[1:]

	// Build command string for analysis
	cmdString := strings.Join(cmdArgs, " ")

	// Analyze command for risks
	findings := analyzer.AnalyzeScript("inline-command", []byte(cmdString), cfg.Policies)
	
	riskLevel := "low"
	var findingCodes []string
	
	// Filter findings based on guard level
	filteredFindings := filterFindingsByGuardLevel(findings, cfg.GuardLevel.Level)
	
	if len(filteredFindings) > 0 {
		// Determine highest risk level
		for _, f := range filteredFindings {
			findingCodes = append(findingCodes, f.Code)
			switch f.Severity {
			case "critical":
				riskLevel = "critical"
			case "high":
				if riskLevel != "critical" {
					riskLevel = "high"
				}
			case "medium":
				if riskLevel != "critical" && riskLevel != "high" {
					riskLevel = "medium"
				}
			}
		}

		// Log findings
		for _, f := range filteredFindings {
			logger.Warn("command risk detected", map[string]any{
				"command":        cmdString,
				"code":           f.Code,
				"severity":       f.Severity,
				"description":    f.Description,
				"recommendation": f.Recommendation,
			})
		}

		// Determine if approval is required based on guard level
		requiresApproval := shouldRequireApproval(riskLevel, cfg.GuardLevel.Level)
		
		// Handle interactive approval or blocking
		if requiresApproval {
			if interactive {
				approval := promptForApproval(riskLevel, cmdString, filteredFindings)
				if !approval.approved {
					logger.Info("command execution denied by user", map[string]any{
						"command": cmdString,
					})
					return &exitError{message: "execution denied", code: 3}
				}
				
				// Handle "remember" functionality
				if approval.remember && cfg.Sandbox.Enabled {
					trustStore, err := sandbox.NewTrustStore(cfg.Sandbox.TrustStorePath)
					if err == nil {
						duration := time.Duration(0) // Never expire by default
						if approval.duration > 0 {
							duration = approval.duration
						}
						if err := trustStore.Add(cmdString, duration, "User approved"); err == nil {
							fmt.Fprintln(os.Stderr, "✅ Approved and remembered")
							logger.Info("command added to trust store", map[string]any{
								"command": cmdString,
							})
						}
					}
				}
			} else {
				logger.Error("risky command blocked", map[string]any{
					"command":    cmdString,
					"risk_level": riskLevel,
					"guard_level": cfg.GuardLevel.Level,
				})
				return &exitError{
					message: fmt.Sprintf("%s risk command blocked by guard level %s (use --interactive to approve, or set bypass)", 
						riskLevel, cfg.GuardLevel.Level),
					code: 3,
				}
			}
		}
	}

	// PRE-EXECUTION PERMISSION ASSESSMENT
	// For critical commands, enforce mandatory sandboxing BEFORE any execution
	if riskLevel == "critical" {
		criticalCodes := []string{
			"DANGEROUS_DELETE_ROOT",
			"DANGEROUS_DELETE_HOME",
			"FORK_BOMB",
			"SENSITIVE_ENV_ACCESS",
			"DOTENV_FILE_READ",
		}
		
		hasCriticalCode := false
		for _, f := range filteredFindings {
			for _, code := range criticalCodes {
				if f.Code == code {
					hasCriticalCode = true
					break
				}
			}
			if hasCriticalCode {
				break
			}
		}
		
		if hasCriticalCode {
			// CRITICAL: These commands MUST be sandboxed - no bypass allowed
			logger.Error("CRITICAL command detected - mandatory sandbox required", map[string]any{
				"command":    cmdString,
				"risk_level": riskLevel,
				"findings":   findingCodes,
			})
			
			// Even if sandbox is disabled, we MUST enforce it for critical commands
			// This is a safety override that cannot be bypassed
			if !cfg.Sandbox.Enabled {
				return &exitError{
					message: fmt.Sprintf("CRITICAL: Command '%s' requires sandboxing but sandbox is disabled. Enable sandbox in config to proceed.", cmdString),
					code:    3,
				}
			}
		}
	}
	
	// Create sandbox executor
	executor, err := sandbox.NewExecutor(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize sandbox executor", map[string]any{
			"error": err.Error(),
		})
		
		// For critical commands, we cannot fallback to direct execution
		if riskLevel == "critical" {
			return &exitError{
				message: fmt.Sprintf("CRITICAL: Cannot execute critical command without sandbox. Sandbox initialization failed: %v", err),
				code:    3,
			}
		}
		
		// Fallback to direct execution only for non-critical commands
		return executeCommandDirectly(cmdArgs)
	}
	
	// Decide execution mode (host vs sandbox)
	// Pass findings so sandbox can make informed decisions
	decision := executor.DecideExecutionMode(ctx, cmdArgs, riskLevel, filteredFindings)
	
	// Show user-friendly notice
	displayExecutionNotice(decision, riskLevel)
	
	// Execute command in chosen mode
	start := time.Now()
	err = executor.Execute(ctx, cmdArgs, decision)
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			logger.Error("command execution failed", map[string]any{
				"command": cmdString,
				"error":   err.Error(),
				"mode":    decision.Mode,
			})
			return fmt.Errorf("execute command: %w", err)
		}
	}

	// Track in session if available
	if sessionID != "" || session.GetCurrentSession() != "" {
		if sessionID == "" {
			sessionID = session.GetCurrentSession()
		}
		
		workspace, _ := os.Getwd()
		mgr, err := session.NewManager(workspace, logger)
		if err == nil {
			sess, err := mgr.Load(sessionID)
			if err == nil {
				cmdRecord := session.Command{
					Timestamp: start,
					Command:   cmdName,
					Args:      args,
					ExitCode:  exitCode,
					Duration:  duration,
					RiskLevel: riskLevel,
					Approved:  interactive || riskLevel == "low",
					Findings:  findingCodes,
				}
				_ = mgr.AddCommand(sess, cmdRecord)
			}
		}
	}

	logger.Info("command executed", map[string]any{
		"command":   cmdString,
		"exit_code": exitCode,
		"duration":  duration.String(),
		"risk":      riskLevel,
	})

	if exitCode != 0 {
		return &exitError{message: fmt.Sprintf("command exited with code %d", exitCode), code: exitCode}
	}

	return nil
}

type approvalResult struct {
	approved bool
	remember bool
	duration time.Duration
}

func promptForApproval(riskLevel, cmdString string, findings []analyzer.Finding) approvalResult {
	result := approvalResult{approved: false, remember: false, duration: 0}
	
	fmt.Fprintf(os.Stderr, "\n⚠️  Command requires approval\n")
	fmt.Fprintf(os.Stderr, "Command: %s\n", cmdString)
	fmt.Fprintf(os.Stderr, "Risk Level: %s\n\n", strings.ToUpper(riskLevel))
	
	if len(findings) > 0 {
		fmt.Fprintf(os.Stderr, "Security concerns:\n")
		for i, f := range findings {
			fmt.Fprintf(os.Stderr, "%d. [%s] %s\n", i+1, f.Code, f.Description)
			fmt.Fprintf(os.Stderr, "   Recommendation: %s\n", f.Recommendation)
		}
		fmt.Fprintln(os.Stderr)
	}

	fmt.Fprintf(os.Stderr, "Options:\n")
	fmt.Fprintf(os.Stderr, "  y  - Yes, run once\n")
	fmt.Fprintf(os.Stderr, "  r  - Yes, and remember (trust permanently)\n")
	fmt.Fprintf(os.Stderr, "  n  - No, cancel\n")
	fmt.Fprintf(os.Stderr, "\nChoose [y/r/N]: ")
	
	var response string
	fmt.Scanln(&response)
	
	response = strings.ToLower(strings.TrimSpace(response))
	
	switch response {
	case "y", "yes":
		result.approved = true
	case "r", "remember":
		result.approved = true
		result.remember = true
	default:
		result.approved = false
	}
	
	return result
}

// filterFindingsByGuardLevel filters findings based on the configured guard level
func filterFindingsByGuardLevel(findings []analyzer.Finding, level config.GuardLevel) []analyzer.Finding {
	if level == config.GuardLevelOff {
		return nil
	}
	
	if level == config.GuardLevelParanoid {
		return findings // Return all findings
	}
	
	var filtered []analyzer.Finding
	for _, f := range findings {
		switch level {
		case config.GuardLevelLow:
			// Only critical
			if f.Severity == "critical" {
				filtered = append(filtered, f)
			}
		case config.GuardLevelMedium:
			// Critical and high
			if f.Severity == "critical" || f.Severity == "high" {
				filtered = append(filtered, f)
			}
		case config.GuardLevelHigh:
			// Critical, high, and medium
			if f.Severity == "critical" || f.Severity == "high" || f.Severity == "medium" {
				filtered = append(filtered, f)
			}
		}
	}
	
	return filtered
}

// shouldRequireApproval determines if a command should require approval
func shouldRequireApproval(riskLevel string, guardLevel config.GuardLevel) bool {
	if guardLevel == config.GuardLevelParanoid {
		return true // Everything requires approval
	}
	
	switch guardLevel {
	case config.GuardLevelLow:
		return riskLevel == "critical"
	case config.GuardLevelMedium:
		return riskLevel == "critical" || riskLevel == "high"
	case config.GuardLevelHigh:
		return riskLevel == "critical" || riskLevel == "high" || riskLevel == "medium"
	default:
		return false
	}
}

// isLikelyAgentBypass checks if the bypass value looks like it was set by an AI agent
func isLikelyAgentBypass(value string) bool {
	// Simple heuristics to detect agent-generated bypass values
	agentPatterns := []string{
		"bypass", "agent", "ai", "automated", "script",
		"cursor", "copilot", "gpt", "claude",
		"true", "yes", "1", "enabled",
	}
	
	lowerValue := strings.ToLower(value)
	for _, pattern := range agentPatterns {
		if strings.Contains(lowerValue, pattern) {
			return true
		}
	}
	
	// If it's too simple (less than 10 chars), likely not a proper bypass
	if len(value) < 10 {
		return true
	}
	
	return false
}

// displayExecutionNotice shows a user-friendly notice about execution mode
func displayExecutionNotice(decision sandbox.ExecutionDecision, riskLevel string) {
	if decision.Mode == sandbox.ExecutionModeHost {
		// Only show notice for interesting cases
		if riskLevel != "low" && decision.Reason != "" {
			fmt.Fprintf(os.Stderr, "🏠 Running on host: %s\n", decision.Reason)
		}
		return
	}
	
	// Sandbox execution - always inform user
	notice := "📦 Running in sandbox"
	
	if decision.ShouldCache {
		notice += " (cached)"
	}
	
	notice += "."
	
	// Explain the "why"
	if decision.Reason != "" {
		reasonParts := strings.Split(decision.Reason, "+")
		if len(reasonParts) > 1 {
			// Multiple reasons: format nicely
			notice += fmt.Sprintf("\n   Why: %s", strings.TrimSpace(reasonParts[0]))
			for _, part := range reasonParts[1:] {
				notice += fmt.Sprintf(" + %s", strings.TrimSpace(part))
			}
		} else {
			notice += fmt.Sprintf("\n   Why: %s", decision.Reason)
		}
	}
	
	fmt.Fprintln(os.Stderr, notice)
}

// executeCommandDirectly executes a command without protection
func executeCommandDirectly(cmdArgs []string) error {
	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
	}
	
	cmdName := cmdArgs[0]
	args := cmdArgs[1:]
	
	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &exitError{
				message: fmt.Sprintf("command exited with code %d", exitErr.ExitCode()),
				code:    exitErr.ExitCode(),
			}
		}
		return fmt.Errorf("execute command: %w", err)
	}
	
	return nil
}

