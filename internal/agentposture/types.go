// Package agentposture audits the security posture of AI agent integrations
// in a workspace: agent discovery, OAuth/API scope analysis, behavioral
// baseline anomaly detection, and security hygiene checks.
package agentposture

// PostureReport is the top-level output of an agent security posture audit.
type PostureReport struct {
	Path             string            `json:"path"`
	Score            int               `json:"score"` // 0-100 overall posture score
	Agents           []AgentFinding    `json:"agents"`
	OAuthFindings    []OAuthFinding    `json:"oauth_findings"`
	BehavioralReport BehavioralReport  `json:"behavioral_report"`
	HygieneChecks    []HygieneCheck    `json:"hygiene_checks"`
	LLMTraceFindings []LLMTraceFinding `json:"llmtrace_findings,omitempty"`
	Recommendations  []string          `json:"recommendations"`
}

// LLMTraceFinding represents a security finding from LLM trace analysis.
type LLMTraceFinding struct {
	TraceID   string  `json:"trace_id"`
	Type      string  `json:"type"`
	Severity  string  `json:"severity"`
	Score     float64 `json:"score"`
	Detail    string  `json:"detail"`
	Timestamp string  `json:"timestamp"`
}

// AgentFinding represents a discovered AI agent integration.
type AgentFinding struct {
	Name        string   `json:"name"`         // "cursor", "claude", "copilot", etc.
	ConfigPaths []string `json:"config_paths"` // Files discovered
	RiskLevel   string   `json:"risk_level"`   // low, medium, high
	Details     string   `json:"details"`
	Seeded      bool     `json:"seeded"` // Has VectraGuard integration markers
}

// OAuthFinding represents an OAuth/API scope issue.
type OAuthFinding struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Type        string `json:"type"`     // "broad_scope", "unscoped_token", "admin_scope", etc.
	Severity    string `json:"severity"` // low, medium, high, critical
	Detail      string `json:"detail"`
	Remediation string `json:"remediation"`
}

// BehavioralReport summarises session behavioral analysis.
type BehavioralReport struct {
	TotalSessions  int             `json:"total_sessions"`
	AgentBaselines []AgentBaseline `json:"agent_baselines"`
	Anomalies      []Anomaly       `json:"anomalies"`
}

// AgentBaseline holds computed baseline statistics for one agent.
type AgentBaseline struct {
	AgentName     string   `json:"agent_name"`
	SessionCount  int      `json:"session_count"`
	AvgCommands   float64  `json:"avg_commands"`
	AvgRiskScore  float64  `json:"avg_risk_score"`
	StdDevRisk    float64  `json:"std_dev_risk"`
	TopCommands   []string `json:"top_commands"`
	ViolationRate float64  `json:"violation_rate"`
}

// Anomaly represents a behavioral anomaly detected in a recent session.
type Anomaly struct {
	SessionID string `json:"session_id"`
	AgentName string `json:"agent_name"`
	Type      string `json:"type"` // "risk_spike", "new_commands", "violation_surge"
	Detail    string `json:"detail"`
	Severity  string `json:"severity"` // low, medium, high, critical
}

// HygieneCheck represents a single security configuration check.
type HygieneCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "pass", "warn", "fail"
	Detail string `json:"detail"`
}
