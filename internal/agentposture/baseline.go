package agentposture

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/session"
)

// AnalyzeBaseline loads sessions via the provided Manager, groups by agent,
// computes baseline stats, and flags anomalies in recent sessions.
func AnalyzeBaseline(mgr *session.Manager) BehavioralReport {
	sessions, err := mgr.List()
	if err != nil || len(sessions) == 0 {
		return BehavioralReport{}
	}

	report := BehavioralReport{
		TotalSessions: len(sessions),
	}

	// Group sessions by agent name
	byAgent := make(map[string][]*session.Session)
	for _, s := range sessions {
		agent := normalizeAgentName(s.AgentName)
		byAgent[agent] = append(byAgent[agent], s)
	}

	// Compute baselines per agent
	for agent, agentSessions := range byAgent {
		baseline := computeBaseline(agent, agentSessions)
		report.AgentBaselines = append(report.AgentBaselines, baseline)

		// Detect anomalies in recent sessions (last 7 days)
		cutoff := time.Now().AddDate(0, 0, -7)
		anomalies := detectAnomalies(baseline, agentSessions, cutoff)
		report.Anomalies = append(report.Anomalies, anomalies...)
	}

	// Sort baselines by agent name for deterministic output
	sort.Slice(report.AgentBaselines, func(i, j int) bool {
		return report.AgentBaselines[i].AgentName < report.AgentBaselines[j].AgentName
	})

	return report
}

func normalizeAgentName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return "unknown"
	}
	return name
}

func computeBaseline(agent string, sessions []*session.Session) AgentBaseline {
	n := len(sessions)
	if n == 0 {
		return AgentBaseline{AgentName: agent}
	}

	var (
		totalCmds       float64
		totalRisk       float64
		totalViolations float64
		riskScores      []float64
		cmdFreq         = make(map[string]int)
	)

	for _, s := range sessions {
		totalCmds += float64(len(s.Commands))
		totalRisk += float64(s.RiskScore)
		totalViolations += float64(s.Violations)
		riskScores = append(riskScores, float64(s.RiskScore))

		for _, cmd := range s.Commands {
			verb := extractVerb(cmd.Command)
			if verb != "" {
				cmdFreq[verb]++
			}
		}
	}

	avgCmds := totalCmds / float64(n)
	avgRisk := totalRisk / float64(n)
	stdRisk := stddev(riskScores, avgRisk)

	violationRate := 0.0
	if totalCmds > 0 {
		violationRate = totalViolations / totalCmds
	}

	// Top commands by frequency
	topCmds := topN(cmdFreq, 5)

	return AgentBaseline{
		AgentName:     agent,
		SessionCount:  n,
		AvgCommands:   math.Round(avgCmds*100) / 100,
		AvgRiskScore:  math.Round(avgRisk*100) / 100,
		StdDevRisk:    math.Round(stdRisk*100) / 100,
		TopCommands:   topCmds,
		ViolationRate: math.Round(violationRate*10000) / 10000,
	}
}

// extractVerb returns the first word of a command string (the binary name).
func extractVerb(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	// Strip path prefix
	if idx := strings.LastIndexAny(command, "/\\"); idx >= 0 {
		command = command[idx+1:]
	}
	// Take first word
	if idx := strings.IndexAny(command, " \t"); idx >= 0 {
		command = command[:idx]
	}
	return strings.ToLower(command)
}

func detectAnomalies(baseline AgentBaseline, sessions []*session.Session, cutoff time.Time) []Anomaly {
	var anomalies []Anomaly

	// Build set of known command verbs from all sessions
	knownVerbs := make(map[string]bool)
	for _, s := range sessions {
		for _, cmd := range s.Commands {
			verb := extractVerb(cmd.Command)
			if verb != "" {
				knownVerbs[verb] = true
			}
		}
	}

	// Only check recent sessions
	for _, s := range sessions {
		if s.StartTime.Before(cutoff) {
			continue
		}

		// Risk spike: risk > mean + 2*stddev
		if baseline.StdDevRisk > 0 {
			threshold := baseline.AvgRiskScore + 2*baseline.StdDevRisk
			if float64(s.RiskScore) > threshold {
				anomalies = append(anomalies, Anomaly{
					SessionID: truncateID(s.ID),
					AgentName: baseline.AgentName,
					Type:      "risk_spike",
					Detail: fmt.Sprintf("risk %d (mean %.0f, σ %.0f)",
						s.RiskScore, baseline.AvgRiskScore, baseline.StdDevRisk),
					Severity: classifyRiskSpikeSeverity(float64(s.RiskScore), baseline.AvgRiskScore, baseline.StdDevRisk),
				})
			}
		}

		// New commands: commands never seen in historical sessions
		var newCmds []string
		recentVerbs := make(map[string]bool)
		for _, cmd := range s.Commands {
			verb := extractVerb(cmd.Command)
			if verb != "" && !recentVerbs[verb] {
				recentVerbs[verb] = true
				if !knownVerbs[verb] {
					newCmds = append(newCmds, verb)
				}
			}
		}
		if len(newCmds) > 0 {
			anomalies = append(anomalies, Anomaly{
				SessionID: truncateID(s.ID),
				AgentName: baseline.AgentName,
				Type:      "new_commands",
				Detail:    fmt.Sprintf("unseen commands: %s", strings.Join(newCmds, ", ")),
				Severity:  "medium",
			})
		}

		// Violation surge: violation rate > 2x historical average
		if baseline.ViolationRate > 0 && len(s.Commands) > 0 {
			sessionViolRate := float64(s.Violations) / float64(len(s.Commands))
			if sessionViolRate > 2*baseline.ViolationRate {
				anomalies = append(anomalies, Anomaly{
					SessionID: truncateID(s.ID),
					AgentName: baseline.AgentName,
					Type:      "violation_surge",
					Detail: fmt.Sprintf("violation rate %.2f%% (baseline %.2f%%)",
						sessionViolRate*100, baseline.ViolationRate*100),
					Severity: "high",
				})
			}
		}
	}

	return anomalies
}

func classifyRiskSpikeSeverity(risk, mean, stddev float64) string {
	deviations := (risk - mean) / stddev
	switch {
	case deviations > 4:
		return "critical"
	case deviations > 3:
		return "high"
	default:
		return "medium"
	}
}

func truncateID(id string) string {
	if len(id) > 16 {
		return id[:16] + "..."
	}
	return id
}

// stddev computes the population standard deviation.
func stddev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		d := v - mean
		sum += d * d
	}
	return math.Sqrt(sum / float64(len(values)))
}

// topN returns the top n keys from a frequency map, sorted by count descending.
func topN(freq map[string]int, n int) []string {
	type kv struct {
		key   string
		count int
	}
	var pairs []kv
	for k, v := range freq {
		pairs = append(pairs, kv{k, v})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].count != pairs[j].count {
			return pairs[i].count > pairs[j].count
		}
		return pairs[i].key < pairs[j].key
	})
	var result []string
	for i, p := range pairs {
		if i >= n {
			break
		}
		result = append(result, p.key)
	}
	return result
}
