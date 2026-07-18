package llmtrace

import "time"

// --- LLM Observability API types (Langfuse-compatible) ---

// TracesResponse is the paginated response from GET /api/public/traces.
type TracesResponse struct {
	Data []Trace        `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// PaginationMeta contains pagination info from the trace API.
type PaginationMeta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}

// Trace represents a single LLM trace.
type Trace struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	SessionID        string                 `json:"sessionId"`
	UserID           string                 `json:"userId"`
	Release          string                 `json:"release"`
	Version          string                 `json:"version"`
	Input            interface{}            `json:"input"`
	Output           interface{}            `json:"output"`
	Metadata         map[string]interface{} `json:"metadata"`
	Tags             []string               `json:"tags"`
	Timestamp        time.Time              `json:"timestamp"`
	Observations     []Observation          `json:"observations"`
	TotalCost        float64                `json:"totalCost"`
	LatencyMs        int                    `json:"latency"`
	PromptTokens     int                    `json:"promptTokens"`
	CompletionTokens int                    `json:"completionTokens"`
	TotalTokens      int                    `json:"totalTokens"`
}

// Observation represents a single observation (span/generation/event) within a trace.
type Observation struct {
	ID               string                 `json:"id"`
	TraceID          string                 `json:"traceId"`
	Type             string                 `json:"type"` // "GENERATION", "SPAN", "EVENT"
	Name             string                 `json:"name"`
	Model            string                 `json:"model"`
	Level            string                 `json:"level"`
	StatusMessage    string                 `json:"statusMessage"`
	ParentObsID      string                 `json:"parentObservationId"`
	Input            interface{}            `json:"input"`
	Output           interface{}            `json:"output"`
	ModelParameters  map[string]interface{} `json:"modelParameters"`
	Metadata         map[string]interface{} `json:"metadata"`
	StartTime        time.Time              `json:"startTime"`
	EndTime          *time.Time             `json:"endTime"`
	PromptTokens     int                    `json:"promptTokens"`
	CompletionTokens int                    `json:"completionTokens"`
	TotalTokens      int                    `json:"totalTokens"`
	TotalCost        float64                `json:"totalCost"`
}

// ObservationsResponse is the paginated response from GET /api/public/observations.
type ObservationsResponse struct {
	Data []Observation  `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

// ScoreRequest is the payload for POST /api/public/scores.
type ScoreRequest struct {
	TraceID       string  `json:"traceId"`
	Name          string  `json:"name"`
	Value         float64 `json:"value"`
	DataType      string  `json:"dataType,omitempty"`
	Comment       string  `json:"comment,omitempty"`
	ObservationID string  `json:"observationId,omitempty"`
}

// --- VG analysis types ---

// TraceAnalysis holds the security analysis results for a single trace.
type TraceAnalysis struct {
	TraceID     string    `json:"trace_id"`
	Findings    []Finding `json:"findings"`
	OverallRisk string    `json:"overall_risk"` // low, medium, high
	RiskScore   float64   `json:"risk_score"`
}

// Finding represents one security finding from trace analysis.
type Finding struct {
	Type      string  `json:"type"`     // injection, cost_anomaly, tool_abuse, agent_loop, data_exfil
	Severity  string  `json:"severity"` // low, medium, high, critical
	Detail    string  `json:"detail"`
	ObsID     string  `json:"obs_id,omitempty"`
	ScoreName string  `json:"score_name"` // score name to write back to provider
	Score     float64 `json:"score"`      // 0.0-1.0 risk value
}

// SyncState tracks the last sync position.
type SyncState struct {
	LastSyncTime time.Time `json:"last_sync_time"`
	TracesTotal  int       `json:"traces_total"`
	ScoresTotal  int       `json:"scores_total"`
}

// SyncReport summarizes a sync run.
type SyncReport struct {
	TracesAnalyzed     int            `json:"traces_analyzed"`
	FindingsTotal      int            `json:"findings_total"`
	ScoresWritten      int            `json:"scores_written"`
	FindingsBySeverity map[string]int `json:"findings_by_severity"`
	DurationMs         int64          `json:"duration_ms"`
	Errors             []string       `json:"errors,omitempty"`
}

// SyncOptions configures a sync run.
type SyncOptions struct {
	From        *time.Time
	DryRun      bool
	WriteScores bool
	BatchSize   int
	ScorePrefix string
}

// ToolCall represents a parsed tool/function call from a generation output.
type ToolCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
