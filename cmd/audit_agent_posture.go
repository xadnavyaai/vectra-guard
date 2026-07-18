package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/vectra-guard/vectra-guard/internal/agentposture"
	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/llmtrace"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

func runAgentPostureAudit(ctx context.Context, targetPath string, outputFormat string, failOnFindings bool) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	if targetPath == "" {
		targetPath = "."
	}

	logger.Info("starting agent posture audit", map[string]any{
		"path":   targetPath,
		"output": outputFormat,
	})

	// 1. Discover AI agent integrations
	agents := agentposture.ScanAgents(targetPath)

	// 2. Scan for OAuth/API scope issues
	oauthFindings := agentposture.ScanOAuth(targetPath)

	// 3. Load sessions and analyze behavioral baseline
	workspace, err := os.Getwd()
	if err != nil {
		workspace = targetPath
	}
	mgr, mgrErr := session.NewManager(workspace, logger)
	var behavioralReport agentposture.BehavioralReport
	if mgrErr == nil {
		behavioralReport = agentposture.AnalyzeBaseline(mgr)
	} else {
		logger.Info("session manager unavailable, skipping behavioral analysis", map[string]any{
			"error": mgrErr.Error(),
		})
	}

	// 4. Run security hygiene checks
	hygieneChecks := agentposture.CheckHygiene(cfg, mgr, targetPath)

	// 4.5 LLM trace analysis (if configured)
	var llmtraceFindings []agentposture.LLMTraceFinding
	if cfg.LLMTrace.Enabled && cfg.LLMTrace.PublicKey != "" && cfg.LLMTrace.SecretKey != "" {
		lfClient, lfErr := llmtrace.NewClient(cfg.LLMTrace.Host, cfg.LLMTrace.PublicKey, cfg.LLMTrace.SecretKey)
		if lfErr != nil {
			logger.Warn("llmtrace client creation failed", map[string]any{"error": lfErr.Error()})
		} else {
			store, storeErr := llmtrace.LoadStateStore(llmtrace.DefaultStatePath())
			if storeErr != nil {
				logger.Warn("llmtrace state load failed", map[string]any{"error": storeErr.Error()})
			} else {
				opts := llmtrace.SyncOptions{
					DryRun:      true,
					WriteScores: false,
					BatchSize:   cfg.LLMTrace.BatchSize,
					ScorePrefix: cfg.LLMTrace.ScorePrefix,
				}
				report, syncErr := llmtrace.RunSync(ctx, lfClient, store, opts)
				if syncErr != nil {
					logger.Warn("llmtrace sync for posture failed", map[string]any{"error": syncErr.Error()})
				} else if report.FindingsTotal > 0 {
					logger.Info("llmtrace findings included in posture", map[string]any{
						"findings": report.FindingsTotal,
					})
				}
				// Pull recent traces and convert findings
				if syncErr == nil {
					recentTraces, fetchErr := lfClient.FetchTraces(ctx, 1, cfg.LLMTrace.BatchSize, nil)
					if fetchErr == nil {
						analyzer := llmtrace.NewAnalyzer(cfg.LLMTrace.ScorePrefix)
						analyzer.UpdateCostBaseline(recentTraces.Data)
						for i := range recentTraces.Data {
							t := &recentTraces.Data[i]
							if len(t.Observations) == 0 {
								obs, _ := lfClient.FetchObservations(ctx, t.ID)
								t.Observations = obs
							}
							analysis := analyzer.AnalyzeTrace(t)
							for _, f := range analysis.Findings {
								llmtraceFindings = append(llmtraceFindings, agentposture.LLMTraceFinding{
									TraceID:   t.ID,
									Type:      f.Type,
									Severity:  f.Severity,
									Score:     f.Score,
									Detail:    f.Detail,
									Timestamp: t.Timestamp.Format("2006-01-02T15:04:05Z"),
								})
							}
						}
					}
				}
			}
		}
	}

	// 5. Build report
	report := &agentposture.PostureReport{
		Path:             targetPath,
		Agents:           agents,
		OAuthFindings:    oauthFindings,
		BehavioralReport: behavioralReport,
		HygieneChecks:    hygieneChecks,
		LLMTraceFindings: llmtraceFindings,
	}

	// 6. Compute score and generate recommendations
	report.Score = agentposture.ComputeScore(report)
	report.Recommendations = agentposture.GenerateRecommendations(report)

	// 7. Emit output
	switch outputFormat {
	case "json":
		if err := agentposture.EmitJSON(os.Stdout, report); err != nil {
			return fmt.Errorf("emit JSON: %w", err)
		}
	default:
		agentposture.EmitText(os.Stdout, report)
	}

	logger.Info("agent posture audit complete", map[string]any{
		"score":           report.Score,
		"agents":          len(report.Agents),
		"oauth_findings":  len(report.OAuthFindings),
		"anomalies":       len(report.BehavioralReport.Anomalies),
		"hygiene_passing": countPassing(report.HygieneChecks),
		"hygiene_total":   len(report.HygieneChecks),
		"recommendations": len(report.Recommendations),
	})

	// 8. Fail if requested and there are findings
	if failOnFindings && hasPostureFindings(report) {
		return &exitError{message: "agent posture audit found issues", code: 2}
	}

	return nil
}

func hasPostureFindings(report *agentposture.PostureReport) bool {
	if len(report.OAuthFindings) > 0 {
		return true
	}
	if len(report.BehavioralReport.Anomalies) > 0 {
		return true
	}
	if len(report.LLMTraceFindings) > 0 {
		return true
	}
	for _, c := range report.HygieneChecks {
		if c.Status == "fail" {
			return true
		}
	}
	return false
}

func countPassing(checks []agentposture.HygieneCheck) int {
	n := 0
	for _, c := range checks {
		if c.Status == "pass" {
			n++
		}
	}
	return n
}
