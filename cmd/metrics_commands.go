package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/sandbox"
)

func runMetricsShow(ctx context.Context, jsonFormat bool) error {
	cfg := config.FromContext(ctx)
	
	if !cfg.Sandbox.EnableMetrics {
		fmt.Println("⚠️  Metrics collection is disabled in configuration")
		return nil
	}
	
	collector, err := sandbox.NewMetricsCollector("", cfg.Sandbox.EnableMetrics)
	if err != nil {
		return fmt.Errorf("open metrics: %w", err)
	}
	
	if jsonFormat {
		metrics := collector.GetMetrics()
		data, err := json.MarshalIndent(metrics, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal metrics: %w", err)
		}
		fmt.Println(string(data))
	} else {
		summary := collector.GetSummary()
		fmt.Println(summary)
	}
	
	return nil
}

func runMetricsReset(ctx context.Context) error {
	cfg := config.FromContext(ctx)
	
	collector, err := sandbox.NewMetricsCollector("", cfg.Sandbox.EnableMetrics)
	if err != nil {
		return fmt.Errorf("open metrics: %w", err)
	}
	
	if err := collector.Reset(); err != nil {
		return fmt.Errorf("reset metrics: %w", err)
	}
	
	fmt.Println("✅ Metrics have been reset")
	return nil
}

