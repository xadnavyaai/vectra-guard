package llmtrace

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/promptfw"
)

// Analyzer runs security checks on LLM traces.
type Analyzer struct {
	scorePrefix string

	// Running cost baseline
	costSum   float64
	costSumSq float64
	costCount int
}

// NewAnalyzer creates a new trace analyzer with the given score prefix.
func NewAnalyzer(scorePrefix string) *Analyzer {
	if scorePrefix == "" {
		scorePrefix = "vg_"
	}
	return &Analyzer{scorePrefix: scorePrefix}
}

// UpdateCostBaseline updates the running mean/stddev from a batch of traces.
func (a *Analyzer) UpdateCostBaseline(traces []Trace) {
	for _, t := range traces {
		if t.TotalCost > 0 {
			a.costSum += t.TotalCost
			a.costSumSq += t.TotalCost * t.TotalCost
			a.costCount++
		}
	}
}

func (a *Analyzer) costMean() float64 {
	if a.costCount == 0 {
		return 0
	}
	return a.costSum / float64(a.costCount)
}

func (a *Analyzer) costStdDev() float64 {
	if a.costCount < 2 {
		return 0
	}
	n := float64(a.costCount)
	mean := a.costMean()
	variance := (a.costSumSq / n) - (mean * mean)
	if variance < 0 {
		variance = 0
	}
	return math.Sqrt(variance)
}

// AnalyzeTrace runs all security checks on a single trace and returns findings.
func (a *Analyzer) AnalyzeTrace(trace *Trace) *TraceAnalysis {
	analysis := &TraceAnalysis{
		TraceID: trace.ID,
	}

	// 1. Injection check
	analysis.Findings = append(analysis.Findings, a.checkInjection(trace)...)

	// 2. Cost anomaly check
	if f := a.checkCostAnomaly(trace); f != nil {
		analysis.Findings = append(analysis.Findings, *f)
	}

	// 3. Tool abuse check
	analysis.Findings = append(analysis.Findings, a.checkToolAbuse(trace)...)

	// 4. Agent loop check
	if f := a.checkAgentLoop(trace); f != nil {
		analysis.Findings = append(analysis.Findings, *f)
	}

	// 5. Data exfiltration check
	analysis.Findings = append(analysis.Findings, a.checkDataExfil(trace)...)

	// Compute overall risk
	analysis.RiskScore = 0
	for _, f := range analysis.Findings {
		analysis.RiskScore += f.Score
	}

	switch {
	case analysis.RiskScore >= 0.7:
		analysis.OverallRisk = "high"
	case analysis.RiskScore >= 0.3:
		analysis.OverallRisk = "medium"
	default:
		analysis.OverallRisk = "low"
	}

	return analysis
}

// checkInjection uses promptfw.Analyze on trace inputs/outputs.
func (a *Analyzer) checkInjection(trace *Trace) []Finding {
	var findings []Finding
	scoreName := a.scorePrefix + "injection_risk"

	// Check trace-level input
	if text := extractText(trace.Input); text != "" {
		result := promptfw.Analyze(text)
		if result.RiskLevel != "low" {
			findings = append(findings, Finding{
				Type:      "injection",
				Severity:  result.RiskLevel,
				Detail:    fmt.Sprintf("Trace input: %s", strings.Join(result.Reasons, ", ")),
				ScoreName: scoreName,
				Score:     normalizeScore(result.Score),
			})
		}
	}

	// Check trace-level output
	if text := extractText(trace.Output); text != "" {
		result := promptfw.Analyze(text)
		if result.RiskLevel != "low" {
			findings = append(findings, Finding{
				Type:      "injection",
				Severity:  result.RiskLevel,
				Detail:    fmt.Sprintf("Trace output: %s", strings.Join(result.Reasons, ", ")),
				ScoreName: scoreName,
				Score:     normalizeScore(result.Score),
			})
		}
	}

	// Check each observation
	for _, obs := range trace.Observations {
		if text := extractText(obs.Input); text != "" {
			result := promptfw.Analyze(text)
			if result.RiskLevel != "low" {
				findings = append(findings, Finding{
					Type:      "injection",
					Severity:  result.RiskLevel,
					Detail:    fmt.Sprintf("Observation %s input: %s", obs.Name, strings.Join(result.Reasons, ", ")),
					ObsID:     obs.ID,
					ScoreName: scoreName,
					Score:     normalizeScore(result.Score),
				})
			}
		}
		if text := extractText(obs.Output); text != "" {
			result := promptfw.Analyze(text)
			if result.RiskLevel != "low" {
				findings = append(findings, Finding{
					Type:      "injection",
					Severity:  result.RiskLevel,
					Detail:    fmt.Sprintf("Observation %s output: %s", obs.Name, strings.Join(result.Reasons, ", ")),
					ObsID:     obs.ID,
					ScoreName: scoreName,
					Score:     normalizeScore(result.Score),
				})
			}
		}
	}

	return findings
}

// checkCostAnomaly flags traces with cost > mean + 2*stddev.
func (a *Analyzer) checkCostAnomaly(trace *Trace) *Finding {
	if a.costCount < 5 || trace.TotalCost <= 0 {
		return nil
	}
	mean := a.costMean()
	sd := a.costStdDev()
	threshold := mean + 2*sd
	if threshold <= 0 {
		return nil
	}
	if trace.TotalCost > threshold {
		deviation := (trace.TotalCost - mean) / sd
		score := math.Min(1.0, deviation/5.0)
		return &Finding{
			Type:      "cost_anomaly",
			Severity:  costSeverity(deviation),
			Detail:    fmt.Sprintf("Cost $%.4f exceeds baseline $%.4f (%.1f stddev)", trace.TotalCost, mean, deviation),
			ScoreName: a.scorePrefix + "cost_anomaly",
			Score:     score,
		}
	}
	return nil
}

// suspiciousToolPatterns matches tool names that indicate shell/file/network access.
var suspiciousToolPatterns = regexp.MustCompile(`(?i)(shell|exec|bash|terminal|cmd|command|run_code|file_write|file_delete|http_request|curl|wget|send_email|network)`)

// checkToolAbuse flags suspicious tool calls in generation outputs.
func (a *Analyzer) checkToolAbuse(trace *Trace) []Finding {
	var findings []Finding
	scoreName := a.scorePrefix + "tool_anomaly"

	for _, obs := range trace.Observations {
		if obs.Type != "GENERATION" {
			continue
		}
		calls := extractToolCalls(&obs)
		for _, call := range calls {
			if suspiciousToolPatterns.MatchString(call.Name) {
				findings = append(findings, Finding{
					Type:      "tool_abuse",
					Severity:  "medium",
					Detail:    fmt.Sprintf("Suspicious tool call: %s", call.Name),
					ObsID:     obs.ID,
					ScoreName: scoreName,
					Score:     0.5,
				})
			}
		}
	}
	return findings
}

// checkAgentLoop detects repetitive agent behavior.
func (a *Analyzer) checkAgentLoop(trace *Trace) *Finding {
	if len(trace.Observations) < 5 {
		return nil
	}

	// Count calls by name+type
	callCounts := make(map[string]int)
	for _, obs := range trace.Observations {
		key := obs.Type + ":" + obs.Name
		callCounts[key]++
	}

	// Flag if any single call repeated 5+ times
	for key, count := range callCounts {
		if count >= 5 {
			ratio := float64(count) / float64(len(trace.Observations))
			return &Finding{
				Type:      "agent_loop",
				Severity:  loopSeverity(count),
				Detail:    fmt.Sprintf("Repeated %d times: %s (%.0f%% of observations)", count, key, ratio*100),
				ScoreName: a.scorePrefix + "agent_loop",
				Score:     math.Min(1.0, float64(count)/20.0),
			}
		}
	}

	// Also flag very high observation count
	if len(trace.Observations) > 50 {
		return &Finding{
			Type:      "agent_loop",
			Severity:  "medium",
			Detail:    fmt.Sprintf("High observation count: %d", len(trace.Observations)),
			ScoreName: a.scorePrefix + "agent_loop",
			Score:     0.4,
		}
	}

	return nil
}

// exfilPattern reuses the EXFILTRATION_CHANNEL concept from promptfw.
var exfilPattern = regexp.MustCompile(`(?i)(!\[.*\]\(https?://.*\?.*=|fetch\s*\(.*data=|curl\s+.*-d\s|wget\s+.*--post-data)`)

// checkDataExfil scans outputs for data exfiltration patterns.
func (a *Analyzer) checkDataExfil(trace *Trace) []Finding {
	var findings []Finding
	scoreName := a.scorePrefix + "data_exfil"

	if text := extractText(trace.Output); text != "" {
		if exfilPattern.MatchString(text) {
			findings = append(findings, Finding{
				Type:      "data_exfil",
				Severity:  "high",
				Detail:    "Exfiltration pattern detected in trace output",
				ScoreName: scoreName,
				Score:     0.8,
			})
		}
	}

	for _, obs := range trace.Observations {
		if text := extractText(obs.Output); text != "" {
			if exfilPattern.MatchString(text) {
				findings = append(findings, Finding{
					Type:      "data_exfil",
					Severity:  "high",
					Detail:    fmt.Sprintf("Exfiltration pattern in observation %s output", obs.Name),
					ObsID:     obs.ID,
					ScoreName: scoreName,
					Score:     0.8,
				})
			}
		}
	}
	return findings
}

// extractText converts an interface{} (string, map, slice) into a searchable string.
func extractText(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case map[string]interface{}:
		data, err := json.Marshal(val)
		if err != nil {
			return ""
		}
		return string(data)
	case []interface{}:
		data, err := json.Marshal(val)
		if err != nil {
			return ""
		}
		return string(data)
	default:
		data, err := json.Marshal(val)
		if err != nil {
			return fmt.Sprintf("%v", val)
		}
		return string(data)
	}
}

// extractToolCalls parses tool_calls from a generation output JSON.
func extractToolCalls(obs *Observation) []ToolCall {
	if obs.Output == nil {
		return nil
	}

	text := extractText(obs.Output)
	if text == "" {
		return nil
	}

	// Try to parse as JSON with tool_calls array
	var wrapper struct {
		ToolCalls []struct {
			Function struct {
				Name      string `json:"name"`
				Arguments string `json:"arguments"`
			} `json:"function"`
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"tool_calls"`
	}
	if err := json.Unmarshal([]byte(text), &wrapper); err == nil && len(wrapper.ToolCalls) > 0 {
		var calls []ToolCall
		for _, tc := range wrapper.ToolCalls {
			name := tc.Function.Name
			args := tc.Function.Arguments
			if name == "" {
				name = tc.Name
				args = tc.Arguments
			}
			if name != "" {
				calls = append(calls, ToolCall{Name: name, Arguments: args})
			}
		}
		return calls
	}

	// Try direct output as map with function_call
	var fcWrapper struct {
		FunctionCall struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function_call"`
	}
	if err := json.Unmarshal([]byte(text), &fcWrapper); err == nil && fcWrapper.FunctionCall.Name != "" {
		return []ToolCall{{Name: fcWrapper.FunctionCall.Name, Arguments: fcWrapper.FunctionCall.Arguments}}
	}

	return nil
}

// normalizeScore maps promptfw's raw score (0-10+) to a 0.0-1.0 range.
func normalizeScore(raw float64) float64 {
	if raw <= 0 {
		return 0
	}
	return math.Min(1.0, raw/10.0)
}

func costSeverity(stddevs float64) string {
	switch {
	case stddevs >= 5:
		return "critical"
	case stddevs >= 3:
		return "high"
	default:
		return "medium"
	}
}

func loopSeverity(count int) string {
	switch {
	case count >= 15:
		return "high"
	case count >= 10:
		return "medium"
	default:
		return "low"
	}
}
