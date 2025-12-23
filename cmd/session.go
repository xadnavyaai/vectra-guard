package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

func runSessionStart(ctx context.Context, agentName, workspace string) error {
	logger := logging.FromContext(ctx)
	
	if workspace == "" {
		var err error
		workspace, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get workspace: %w", err)
		}
	}

	mgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return fmt.Errorf("create session manager: %w", err)
	}

	sess, err := mgr.Start(agentName, workspace)
	if err != nil {
		return fmt.Errorf("start session: %w", err)
	}

	// Set current session in environment
	session.SetCurrentSession(sess.ID)

	logger.Info("session started", map[string]any{
		"session_id": sess.ID,
		"agent":      agentName,
		"workspace":  workspace,
	})

	// Print for easy capture in scripts
	fmt.Println(sess.ID)
	return nil
}

func runSessionEnd(ctx context.Context, sessionID string) error {
	logger := logging.FromContext(ctx)
	
	workspace, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}

	mgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return fmt.Errorf("create session manager: %w", err)
	}

	sess, err := mgr.Load(sessionID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	if err := mgr.End(sess); err != nil {
		return fmt.Errorf("end session: %w", err)
	}

	logger.Info("session ended", map[string]any{
		"session_id": sessionID,
		"commands":   len(sess.Commands),
		"violations": sess.Violations,
		"risk_score": sess.RiskScore,
	})

	return nil
}

func runSessionList(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	
	workspace, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}

	mgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return fmt.Errorf("create session manager: %w", err)
	}

	sessions, err := mgr.List()
	if err != nil {
		return fmt.Errorf("list sessions: %w", err)
	}

	if len(sessions) == 0 {
		logger.Info("no sessions found", nil)
		return nil
	}

	// Format output as table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SESSION ID\tAGENT\tSTART TIME\tDURATION\tCOMMANDS\tVIOLATIONS\tRISK")
	fmt.Fprintln(w, "----------\t-----\t----------\t--------\t--------\t----------\t----")

	for _, sess := range sessions {
		duration := "active"
		if sess.EndTime != nil {
			duration = sess.EndTime.Sub(sess.StartTime).Round(time.Second).String()
		}
		
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%d\n",
			sess.ID,
			sess.AgentName,
			sess.StartTime.Format("2006-01-02 15:04:05"),
			duration,
			len(sess.Commands),
			sess.Violations,
			sess.RiskScore,
		)
	}
	w.Flush()

	return nil
}

func runSessionShow(ctx context.Context, sessionID string) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)
	
	workspace, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get workspace: %w", err)
	}

	mgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return fmt.Errorf("create session manager: %w", err)
	}

	sess, err := mgr.Load(sessionID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	// Output in configured format
	if cfg.Logging.Format == "json" {
		logger.Info("session", map[string]any{
			"session_id":      sess.ID,
			"agent":           sess.AgentName,
			"workspace":       sess.Workspace,
			"start_time":      sess.StartTime,
			"end_time":        sess.EndTime,
			"commands":        sess.Commands,
			"file_operations": sess.FileOps,
			"risk_score":      sess.RiskScore,
			"violations":      sess.Violations,
		})
	} else {
		// Human-readable format
		fmt.Printf("Session: %s\n", sess.ID)
		fmt.Printf("Agent: %s\n", sess.AgentName)
		fmt.Printf("Workspace: %s\n", sess.Workspace)
		fmt.Printf("Start Time: %s\n", sess.StartTime.Format(time.RFC3339))
		if sess.EndTime != nil {
			fmt.Printf("End Time: %s\n", sess.EndTime.Format(time.RFC3339))
			fmt.Printf("Duration: %s\n", sess.EndTime.Sub(sess.StartTime).Round(time.Second))
		} else {
			fmt.Println("Status: Active")
		}
		fmt.Printf("Risk Score: %d\n", sess.RiskScore)
		fmt.Printf("Violations: %d\n\n", sess.Violations)

		if len(sess.Commands) > 0 {
			fmt.Printf("Commands (%d):\n", len(sess.Commands))
			for i, cmd := range sess.Commands {
				fmt.Printf("  %d. [%s] %s %v (exit: %d, risk: %s)\n",
					i+1,
					cmd.Timestamp.Format("15:04:05"),
					cmd.Command,
					cmd.Args,
					cmd.ExitCode,
					cmd.RiskLevel,
				)
				if len(cmd.Findings) > 0 {
					fmt.Printf("     Findings: %v\n", cmd.Findings)
				}
			}
			fmt.Println()
		}

		if len(sess.FileOps) > 0 {
			fmt.Printf("File Operations (%d):\n", len(sess.FileOps))
			for i, op := range sess.FileOps {
				status := "✓"
				if !op.Allowed {
					status = "✗"
				}
				fmt.Printf("  %d. [%s] %s %s %s (risk: %s)\n",
					i+1,
					status,
					op.Timestamp.Format("15:04:05"),
					op.Operation,
					op.Path,
					op.RiskLevel,
				)
				if op.Reason != "" {
					fmt.Printf("     Reason: %s\n", op.Reason)
				}
			}
		}
	}

	return nil
}

