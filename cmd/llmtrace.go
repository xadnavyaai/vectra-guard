package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/llmtrace"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func runLLMTraceConnect(ctx context.Context, host, publicKey, secretKey string) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	// Use flags or fall back to config
	if host == "" {
		host = cfg.LLMTrace.Host
	}
	if publicKey == "" {
		publicKey = cfg.LLMTrace.PublicKey
	}
	if secretKey == "" {
		secretKey = cfg.LLMTrace.SecretKey
	}

	client, err := llmtrace.NewClient(host, publicKey, secretKey)
	if err != nil {
		return fmt.Errorf("llmtrace connect: %w", err)
	}

	logger.Info("testing LLM trace provider connectivity", map[string]any{"host": host})

	if err := client.HealthCheck(ctx); err != nil {
		return fmt.Errorf("llmtrace connection failed: %w", err)
	}

	fmt.Printf("✓ Connected to LLM trace provider at %s\n", host)
	return nil
}

func runLLMTraceSync(ctx context.Context, fromStr string, dryRun bool) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	client, err := llmtrace.NewClient(cfg.LLMTrace.Host, cfg.LLMTrace.PublicKey, cfg.LLMTrace.SecretKey)
	if err != nil {
		return fmt.Errorf("llmtrace sync: %w", err)
	}

	store, err := llmtrace.LoadStateStore(llmtrace.DefaultStatePath())
	if err != nil {
		return fmt.Errorf("load llmtrace state: %w", err)
	}

	opts := llmtrace.SyncOptions{
		DryRun:      dryRun,
		WriteScores: !dryRun,
		BatchSize:   cfg.LLMTrace.BatchSize,
		ScorePrefix: cfg.LLMTrace.ScorePrefix,
	}

	if fromStr != "" {
		t, parseErr := time.Parse(time.RFC3339, fromStr)
		if parseErr != nil {
			return fmt.Errorf("invalid --from timestamp (use RFC3339 format): %w", parseErr)
		}
		opts.From = &t
	}

	mode := "live"
	if dryRun {
		mode = "dry-run"
	}
	logger.Info("starting llmtrace sync", map[string]any{"mode": mode})

	report, err := llmtrace.RunSync(ctx, client, store, opts)
	if err != nil {
		return fmt.Errorf("llmtrace sync: %w", err)
	}

	fmt.Printf("LLM trace sync complete (%s)\n", mode)
	fmt.Printf("  Traces analyzed: %d\n", report.TracesAnalyzed)
	fmt.Printf("  Findings:        %d\n", report.FindingsTotal)
	if !dryRun {
		fmt.Printf("  Scores written:  %d\n", report.ScoresWritten)
	}
	fmt.Printf("  Duration:        %dms\n", report.DurationMs)

	if len(report.FindingsBySeverity) > 0 {
		fmt.Println("  By severity:")
		for sev, count := range report.FindingsBySeverity {
			fmt.Printf("    %-10s %d\n", sev, count)
		}
	}

	if len(report.Errors) > 0 {
		fmt.Printf("  Errors: %d\n", len(report.Errors))
		for _, e := range report.Errors {
			fmt.Printf("    - %s\n", e)
		}
	}

	return nil
}

func runLLMTraceScan(ctx context.Context, traceID string, writeScores bool) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	client, err := llmtrace.NewClient(cfg.LLMTrace.Host, cfg.LLMTrace.PublicKey, cfg.LLMTrace.SecretKey)
	if err != nil {
		return fmt.Errorf("llmtrace scan: %w", err)
	}

	logger.Info("scanning LLM trace", map[string]any{"trace_id": traceID})

	trace, err := client.FetchTrace(ctx, traceID)
	if err != nil {
		return fmt.Errorf("fetch trace: %w", err)
	}

	// Fetch observations
	obs, err := client.FetchObservations(ctx, traceID)
	if err != nil {
		logger.Warn("could not fetch observations", map[string]any{"error": err.Error()})
	} else {
		trace.Observations = obs
	}

	analyzer := llmtrace.NewAnalyzer(cfg.LLMTrace.ScorePrefix)
	analysis := analyzer.AnalyzeTrace(trace)

	fmt.Printf("Trace: %s\n", trace.ID)
	fmt.Printf("Name:  %s\n", trace.Name)
	fmt.Printf("Risk:  %s (%.2f)\n", analysis.OverallRisk, analysis.RiskScore)
	fmt.Printf("Observations: %d\n\n", len(trace.Observations))

	if len(analysis.Findings) == 0 {
		fmt.Println("No security findings.")
	} else {
		fmt.Printf("FINDINGS (%d)\n", len(analysis.Findings))
		for i, f := range analysis.Findings {
			icon := "·"
			switch f.Severity {
			case "critical":
				icon = "✗"
			case "high":
				icon = "⚠"
			case "medium":
				icon = "~"
			}
			fmt.Printf("  %d. %s [%s] %s (score: %.2f)\n", i+1, icon, f.Severity, f.Detail, f.Score)
		}
	}

	if writeScores && len(analysis.Findings) > 0 {
		fmt.Println("\nWriting scores to LLM trace provider...")
		written := 0
		for _, f := range analysis.Findings {
			scoreReq := llmtrace.ScoreRequest{
				TraceID:       trace.ID,
				Name:          f.ScoreName,
				Value:         f.Score,
				DataType:      "NUMERIC",
				Comment:       fmt.Sprintf("[VectraGuard] %s: %s", f.Type, f.Detail),
				ObservationID: f.ObsID,
			}
			if postErr := client.PostScore(ctx, scoreReq); postErr != nil {
				fmt.Printf("  Error writing %s: %s\n", f.ScoreName, postErr)
			} else {
				written++
			}
		}
		fmt.Printf("  %d score(s) written\n", written)
	}

	return nil
}

func runLLMTraceWatch(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	cfg := config.FromContext(ctx)

	client, err := llmtrace.NewClient(cfg.LLMTrace.Host, cfg.LLMTrace.PublicKey, cfg.LLMTrace.SecretKey)
	if err != nil {
		return fmt.Errorf("llmtrace watch: %w", err)
	}

	store, err := llmtrace.LoadStateStore(llmtrace.DefaultStatePath())
	if err != nil {
		return fmt.Errorf("load llmtrace state: %w", err)
	}

	interval := time.Duration(cfg.LLMTrace.PollInterval) * time.Second
	if interval < 10*time.Second {
		interval = 60 * time.Second
	}

	// Set up graceful shutdown
	watchCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	fmt.Printf("Watching LLM traces (poll every %s, Ctrl-C to stop)\n", interval)
	logger.Info("llmtrace watch started", map[string]any{"interval": interval.String()})

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run once immediately
	runWatchCycle(watchCtx, client, store, cfg, logger)

	for {
		select {
		case <-watchCtx.Done():
			fmt.Println("\nStopping llmtrace watch...")
			return nil
		case <-ticker.C:
			runWatchCycle(watchCtx, client, store, cfg, logger)
		}
	}
}

func runWatchCycle(ctx context.Context, client *llmtrace.Client, store *llmtrace.StateStore, cfg config.Config, logger *logging.Logger) {
	opts := llmtrace.SyncOptions{
		WriteScores: true,
		BatchSize:   cfg.LLMTrace.BatchSize,
		ScorePrefix: cfg.LLMTrace.ScorePrefix,
	}

	report, err := llmtrace.RunSync(ctx, client, store, opts)
	if err != nil {
		logger.Error("watch cycle failed", map[string]any{"error": err.Error()})
		return
	}

	if report.TracesAnalyzed > 0 || report.FindingsTotal > 0 {
		fmt.Printf("[%s] analyzed=%d findings=%d scores=%d\n",
			time.Now().Format("15:04:05"),
			report.TracesAnalyzed, report.FindingsTotal, report.ScoresWritten)
	}
}
