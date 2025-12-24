package sandbox

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetricsCollector(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vectra-guard-metrics-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	metricsPath := filepath.Join(tmpDir, "metrics.json")
	
	t.Run("NewMetricsCollector", func(t *testing.T) {
		collector, err := NewMetricsCollector(metricsPath, true)
		if err != nil {
			t.Fatalf("NewMetricsCollector() error = %v", err)
		}
		
		if !collector.enabled {
			t.Error("Collector should be enabled")
		}
		
		if collector.path != metricsPath {
			t.Errorf("path = %v, want %v", collector.path, metricsPath)
		}
	})
	
	t.Run("RecordHostExecution", func(t *testing.T) {
		collector, _ := NewMetricsCollector(metricsPath, true)
		
		record := ExecutionRecord{
			Timestamp: time.Now(),
			Command:   "echo test",
			Mode:      ExecutionModeHost,
			Duration:  100 * time.Millisecond,
			RiskLevel: "low",
			ExitCode:  0,
			Reason:    "low risk",
		}
		
		err := collector.Record(record)
		if err != nil {
			t.Fatalf("Record() error = %v", err)
		}
		
		metrics := collector.GetMetrics()
		
		if metrics.TotalExecutions != 1 {
			t.Errorf("TotalExecutions = %d, want 1", metrics.TotalExecutions)
		}
		
		if metrics.HostExecutions != 1 {
			t.Errorf("HostExecutions = %d, want 1", metrics.HostExecutions)
		}
		
		if metrics.SandboxExecutions != 0 {
			t.Errorf("SandboxExecutions = %d, want 0", metrics.SandboxExecutions)
		}
	})
	
	t.Run("RecordSandboxExecution", func(t *testing.T) {
		collector, _ := NewMetricsCollector(metricsPath, true)
		
		record := ExecutionRecord{
			Timestamp: time.Now(),
			Command:   "npm install",
			Mode:      ExecutionModeSandbox,
			Runtime:   "docker",
			Duration:  2 * time.Second,
			RiskLevel: "medium",
			Cached:    true,
			ExitCode:  0,
			Reason:    "medium risk + networked install",
		}
		
		err := collector.Record(record)
		if err != nil {
			t.Fatalf("Record() error = %v", err)
		}
		
		metrics := collector.GetMetrics()
		
		if metrics.SandboxExecutions != 1 {
			t.Errorf("SandboxExecutions = %d, want 1", metrics.SandboxExecutions)
		}
		
		if metrics.CachedExecutions != 1 {
			t.Errorf("CachedExecutions = %d, want 1", metrics.CachedExecutions)
		}
		
		if metrics.ByRuntime["docker"] != 1 {
			t.Errorf("ByRuntime[docker] = %d, want 1", metrics.ByRuntime["docker"])
		}
	})
	
	t.Run("RecordByRiskLevel", func(t *testing.T) {
		riskPath := filepath.Join(tmpDir, "metrics-risk.json")
		collector, _ := NewMetricsCollector(riskPath, true)
		
		riskLevels := []string{"low", "medium", "high", "critical"}
		
		for _, level := range riskLevels {
			record := ExecutionRecord{
				Timestamp: time.Now(),
				Command:   "test command",
				Mode:      ExecutionModeHost,
				Duration:  100 * time.Millisecond,
				RiskLevel: level,
				ExitCode:  0,
			}
			
			err := collector.Record(record)
			if err != nil {
				t.Fatalf("Record() error = %v", err)
			}
		}
		
		metrics := collector.GetMetrics()
		
		for _, level := range riskLevels {
			if metrics.ByRiskLevel[level] != 1 {
				t.Errorf("ByRiskLevel[%s] = %d, want 1", level, metrics.ByRiskLevel[level])
			}
		}
	})
	
	t.Run("AverageDuration", func(t *testing.T) {
		durationPath := filepath.Join(tmpDir, "metrics-duration.json")
		collector, _ := NewMetricsCollector(durationPath, true)
		
		durations := []time.Duration{
			100 * time.Millisecond,
			200 * time.Millisecond,
			300 * time.Millisecond,
		}
		
		for _, duration := range durations {
			record := ExecutionRecord{
				Timestamp: time.Now(),
				Command:   "test",
				Mode:      ExecutionModeHost,
				Duration:  duration,
				RiskLevel: "low",
				ExitCode:  0,
			}
			
			err := collector.Record(record)
			if err != nil {
				t.Fatalf("Record() error = %v", err)
			}
		}
		
		metrics := collector.GetMetrics()
		
		// Average should be 200ms
		expected := 200 * time.Millisecond
		if metrics.AverageDuration != expected {
			t.Errorf("AverageDuration = %v, want %v", metrics.AverageDuration, expected)
		}
	})
	
	t.Run("ExecutionHistory", func(t *testing.T) {
		historyPath := filepath.Join(tmpDir, "metrics-history.json")
		collector, _ := NewMetricsCollector(historyPath, true)
		
		// Record 150 executions
		for i := 0; i < 150; i++ {
			record := ExecutionRecord{
				Timestamp: time.Now(),
				Command:   "test",
				Mode:      ExecutionModeHost,
				Duration:  100 * time.Millisecond,
				RiskLevel: "low",
				ExitCode:  0,
			}
			
			err := collector.Record(record)
			if err != nil {
				t.Fatalf("Record() error = %v", err)
			}
		}
		
		metrics := collector.GetMetrics()
		
		// Should only keep last 100
		if len(metrics.ExecutionHistory) != 100 {
			t.Errorf("ExecutionHistory length = %d, want 100", len(metrics.ExecutionHistory))
		}
		
		if metrics.TotalExecutions != 150 {
			t.Errorf("TotalExecutions = %d, want 150", metrics.TotalExecutions)
		}
	})
	
	t.Run("GetSummary", func(t *testing.T) {
		collector, _ := NewMetricsCollector(metricsPath, true)
		
		// Record some executions
		for i := 0; i < 5; i++ {
			mode := ExecutionModeHost
			if i%2 == 0 {
				mode = ExecutionModeSandbox
			}
			
			record := ExecutionRecord{
				Timestamp: time.Now(),
				Command:   "test",
				Mode:      mode,
				Runtime:   "docker",
				Duration:  100 * time.Millisecond,
				RiskLevel: "medium",
				ExitCode:  0,
			}
			
			err := collector.Record(record)
			if err != nil {
				t.Fatalf("Record() error = %v", err)
			}
		}
		
		summary := collector.GetSummary()
		
		if summary == "" {
			t.Error("GetSummary() returned empty string")
		}
		
		// Summary should contain key metrics
		if !contains(summary, "Total Executions") {
			t.Error("Summary should contain 'Total Executions'")
		}
		
		if !contains(summary, "Average Duration") {
			t.Error("Summary should contain 'Average Duration'")
		}
	})
	
	t.Run("Reset", func(t *testing.T) {
		collector, _ := NewMetricsCollector(metricsPath, true)
		
		// Record some data
		record := ExecutionRecord{
			Timestamp: time.Now(),
			Command:   "test",
			Mode:      ExecutionModeHost,
			Duration:  100 * time.Millisecond,
			RiskLevel: "low",
			ExitCode:  0,
		}
		
		err := collector.Record(record)
		if err != nil {
			t.Fatalf("Record() error = %v", err)
		}
		
		// Reset
		err = collector.Reset()
		if err != nil {
			t.Fatalf("Reset() error = %v", err)
		}
		
		// Metrics should be cleared
		metrics := collector.GetMetrics()
		
		if metrics.TotalExecutions != 0 {
			t.Errorf("TotalExecutions = %d, want 0 after reset", metrics.TotalExecutions)
		}
	})
	
	t.Run("Disabled", func(t *testing.T) {
		collector, _ := NewMetricsCollector(metricsPath, false)
		
		record := ExecutionRecord{
			Timestamp: time.Now(),
			Command:   "test",
			Mode:      ExecutionModeHost,
			Duration:  100 * time.Millisecond,
			RiskLevel: "low",
			ExitCode:  0,
		}
		
		// Should not error when disabled
		err := collector.Record(record)
		if err != nil {
			t.Fatalf("Record() error = %v when disabled", err)
		}
		
		// Should return empty metrics
		metrics := collector.GetMetrics()
		if metrics.TotalExecutions != 0 {
			t.Error("Disabled collector should return empty metrics")
		}
	})
	
	t.Run("Persistence", func(t *testing.T) {
		// Create first collector and record data
		collector1, _ := NewMetricsCollector(metricsPath, true)
		
		record := ExecutionRecord{
			Timestamp: time.Now(),
			Command:   "persistent test",
			Mode:      ExecutionModeHost,
			Duration:  100 * time.Millisecond,
			RiskLevel: "low",
			ExitCode:  0,
		}
		
		err := collector1.Record(record)
		if err != nil {
			t.Fatalf("Record() error = %v", err)
		}
		
		// Create new collector with same path
		collector2, _ := NewMetricsCollector(metricsPath, true)
		
		// Should have loaded previous data
		metrics := collector2.GetMetrics()
		if metrics.TotalExecutions == 0 {
			t.Error("Metrics should persist across collector instances")
		}
	})
}

func TestPercentage(t *testing.T) {
	tests := []struct {
		name     string
		part     int64
		total    int64
		expected float64
	}{
		{"50 percent", 50, 100, 50.0},
		{"25 percent", 25, 100, 25.0},
		{"0 percent", 0, 100, 0.0},
		{"100 percent", 100, 100, 100.0},
		{"zero total", 50, 0, 0.0},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := percentage(tt.part, tt.total)
			if result != tt.expected {
				t.Errorf("percentage(%d, %d) = %f, want %f", tt.part, tt.total, result, tt.expected)
			}
		})
	}
}

func BenchmarkMetricsRecord(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "vectra-guard-metrics-bench")
	defer os.RemoveAll(tmpDir)
	
	metricsPath := filepath.Join(tmpDir, "metrics.json")
	collector, _ := NewMetricsCollector(metricsPath, true)
	
	record := ExecutionRecord{
		Timestamp: time.Now(),
		Command:   "test",
		Mode:      ExecutionModeHost,
		Duration:  100 * time.Millisecond,
		RiskLevel: "low",
		ExitCode:  0,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.Record(record)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInner(s, substr)))
}

func containsInner(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

