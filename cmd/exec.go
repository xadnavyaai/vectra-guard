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
	"github.com/vectra-guard/vectra-guard/internal/session"
)

func runExec(ctx context.Context, cmdArgs []string, interactive bool, sessionID string) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	if len(cmdArgs) == 0 {
		return fmt.Errorf("no command specified")
	}

	cmdName := cmdArgs[0]
	args := cmdArgs[1:]

	// Build command string for analysis
	cmdString := strings.Join(cmdArgs, " ")

	// Analyze command for risks
	findings := analyzer.AnalyzeScript("inline-command", []byte(cmdString), cfg.Policies)
	
	riskLevel := "low"
	var findingCodes []string
	
	if len(findings) > 0 {
		// Determine highest risk level
		for _, f := range findings {
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
		for _, f := range findings {
			logger.Warn("command risk detected", map[string]any{
				"command":        cmdString,
				"code":           f.Code,
				"severity":       f.Severity,
				"description":    f.Description,
				"recommendation": f.Recommendation,
			})
		}

		// Handle interactive approval
		if interactive {
			if !promptForApproval(riskLevel, cmdString, findings) {
				logger.Info("command execution denied by user", map[string]any{
					"command": cmdString,
				})
				return &exitError{message: "execution denied", code: 3}
			}
		} else if riskLevel == "critical" {
			logger.Error("critical command blocked", map[string]any{
				"command": cmdString,
			})
			return &exitError{message: "critical command blocked (use --interactive to approve)", code: 3}
		}
	}

	// Execute command
	start := time.Now()
	cmd := exec.Command(cmdName, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			logger.Error("command execution failed", map[string]any{
				"command": cmdString,
				"error":   err.Error(),
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

func promptForApproval(riskLevel, cmdString string, findings []analyzer.Finding) bool {
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

	fmt.Fprintf(os.Stderr, "Do you want to proceed? [y/N]: ")
	
	var response string
	fmt.Scanln(&response)
	
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

