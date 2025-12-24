package sandbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ExecutionMetrics tracks sandbox usage and performance
type ExecutionMetrics struct {
	TotalExecutions      int64                    `json:"total_executions"`
	HostExecutions       int64                    `json:"host_executions"`
	SandboxExecutions    int64                    `json:"sandbox_executions"`
	CachedExecutions     int64                    `json:"cached_executions"`
	AverageDuration      time.Duration            `json:"average_duration"`
	ByRiskLevel          map[string]int64         `json:"by_risk_level"`
	ByRuntime            map[string]int64         `json:"by_runtime"`
	ExecutionHistory     []ExecutionRecord        `json:"execution_history"`
	LastUpdated          time.Time                `json:"last_updated"`
}

// ExecutionRecord represents a single execution event
type ExecutionRecord struct {
	Timestamp      time.Time     `json:"timestamp"`
	Command        string        `json:"command"`
	Mode           ExecutionMode `json:"mode"`
	Runtime        string        `json:"runtime,omitempty"`
	Duration       time.Duration `json:"duration"`
	RiskLevel      string        `json:"risk_level"`
	Cached         bool          `json:"cached"`
	ExitCode       int           `json:"exit_code"`
	Reason         string        `json:"reason"`
}

// MetricsCollector collects and persists execution metrics
type MetricsCollector struct {
	path    string
	metrics *ExecutionMetrics
	mu      sync.RWMutex
	enabled bool
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(path string, enabled bool) (*MetricsCollector, error) {
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		path = filepath.Join(homeDir, ".vectra-guard", "metrics.json")
	}
	
	collector := &MetricsCollector{
		path:    path,
		enabled: enabled,
		metrics: &ExecutionMetrics{
			ByRiskLevel: make(map[string]int64),
			ByRuntime:   make(map[string]int64),
			ExecutionHistory: []ExecutionRecord{},
		},
	}
	
	if !enabled {
		return collector, nil
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create metrics directory: %w", err)
	}
	
	// Load existing metrics
	if err := collector.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load metrics: %w", err)
	}
	
	return collector, nil
}

// Record records an execution event
func (mc *MetricsCollector) Record(record ExecutionRecord) error {
	if !mc.enabled {
		return nil
	}
	
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	// Update totals
	mc.metrics.TotalExecutions++
	
	if record.Mode == ExecutionModeHost {
		mc.metrics.HostExecutions++
	} else {
		mc.metrics.SandboxExecutions++
	}
	
	if record.Cached {
		mc.metrics.CachedExecutions++
	}
	
	// Update by risk level
	mc.metrics.ByRiskLevel[record.RiskLevel]++
	
	// Update by runtime
	if record.Runtime != "" {
		mc.metrics.ByRuntime[record.Runtime]++
	}
	
	// Update average duration
	if mc.metrics.TotalExecutions == 1 {
		mc.metrics.AverageDuration = record.Duration
	} else {
		// Running average
		mc.metrics.AverageDuration = time.Duration(
			(int64(mc.metrics.AverageDuration)*(mc.metrics.TotalExecutions-1) + int64(record.Duration)) / mc.metrics.TotalExecutions,
		)
	}
	
	// Add to history (keep last 100 records)
	mc.metrics.ExecutionHistory = append(mc.metrics.ExecutionHistory, record)
	if len(mc.metrics.ExecutionHistory) > 100 {
		mc.metrics.ExecutionHistory = mc.metrics.ExecutionHistory[1:]
	}
	
	mc.metrics.LastUpdated = time.Now()
	
	return mc.save()
}

// GetMetrics returns current metrics
func (mc *MetricsCollector) GetMetrics() ExecutionMetrics {
	if !mc.enabled {
		return ExecutionMetrics{}
	}
	
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	// Return a copy
	return *mc.metrics
}

// GetSummary returns a human-readable summary
func (mc *MetricsCollector) GetSummary() string {
	if !mc.enabled {
		return "Metrics collection is disabled"
	}
	
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	m := mc.metrics
	
	summary := fmt.Sprintf(`Vectra Guard Sandbox Metrics
===============================
Total Executions:    %d
  - Host:            %d (%.1f%%)
  - Sandbox:         %d (%.1f%%)
  - Cached:          %d (%.1f%%)

Average Duration:    %s

By Risk Level:
`,
		m.TotalExecutions,
		m.HostExecutions, percentage(m.HostExecutions, m.TotalExecutions),
		m.SandboxExecutions, percentage(m.SandboxExecutions, m.TotalExecutions),
		m.CachedExecutions, percentage(m.CachedExecutions, m.TotalExecutions),
		m.AverageDuration.Round(time.Millisecond),
	)
	
	for level, count := range m.ByRiskLevel {
		summary += fmt.Sprintf("  - %s: %d (%.1f%%)\n", level, count, percentage(count, m.TotalExecutions))
	}
	
	if len(m.ByRuntime) > 0 {
		summary += "\nBy Runtime:\n"
		for runtime, count := range m.ByRuntime {
			summary += fmt.Sprintf("  - %s: %d\n", runtime, count)
		}
	}
	
	summary += fmt.Sprintf("\nLast Updated: %s\n", m.LastUpdated.Format(time.RFC3339))
	
	return summary
}

// Reset clears all metrics
func (mc *MetricsCollector) Reset() error {
	if !mc.enabled {
		return nil
	}
	
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.metrics = &ExecutionMetrics{
		ByRiskLevel:      make(map[string]int64),
		ByRuntime:        make(map[string]int64),
		ExecutionHistory: []ExecutionRecord{},
		LastUpdated:      time.Now(),
	}
	
	return mc.save()
}

// load reads metrics from disk
func (mc *MetricsCollector) load() error {
	data, err := os.ReadFile(mc.path)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, mc.metrics)
}

// save writes metrics to disk
func (mc *MetricsCollector) save() error {
	data, err := json.MarshalIndent(mc.metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metrics: %w", err)
	}
	
	// Write atomically with temp file
	tmpPath := mc.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("write metrics: %w", err)
	}
	
	if err := os.Rename(tmpPath, mc.path); err != nil {
		return fmt.Errorf("rename metrics: %w", err)
	}
	
	return nil
}

// percentage calculates percentage with proper handling of zero
func percentage(part, total int64) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(part) / float64(total) * 100
}

