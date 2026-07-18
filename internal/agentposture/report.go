package agentposture

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ComputeScore calculates the overall posture score (0-100).
//
// Breakdown:
//   - Agent inventory:    20 points (all agents seeded with VG → full score)
//   - OAuth scope:        30 points (no high/critical findings → full score)
//   - Behavioral health:  25 points (no anomalies → full score)
//   - Security hygiene:   25 points (proportional to pass rate)
func ComputeScore(report *PostureReport) int {
	score := 0

	// Agent inventory: 20 points
	score += scoreAgentInventory(report.Agents)

	// OAuth scope: 30 points
	score += scoreOAuth(report.OAuthFindings)

	// Behavioral health: 25 points (includes LLM trace anomaly deductions)
	behavioralScore := scoreBehavioral(report.BehavioralReport)
	// Deduct from behavioral budget for LLM trace findings
	for _, f := range report.LLMTraceFindings {
		switch f.Severity {
		case "critical":
			behavioralScore -= 6
		case "high":
			behavioralScore -= 4
		case "medium":
			behavioralScore -= 2
		}
	}
	if behavioralScore < 0 {
		behavioralScore = 0
	}
	score += behavioralScore

	// Security hygiene: 25 points
	score += scoreHygiene(report.HygieneChecks)

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func scoreAgentInventory(agents []AgentFinding) int {
	if len(agents) == 0 {
		return 20 // No agents found = nothing to worry about
	}
	seeded := 0
	for _, a := range agents {
		if a.Seeded {
			seeded++
		}
	}
	return 20 * seeded / len(agents)
}

func scoreOAuth(findings []OAuthFinding) int {
	if len(findings) == 0 {
		return 30
	}
	deductions := 0
	for _, f := range findings {
		switch f.Severity {
		case "critical":
			deductions += 15
		case "high":
			deductions += 10
		case "medium":
			deductions += 5
		case "low":
			deductions += 2
		}
	}
	score := 30 - deductions
	if score < 0 {
		return 0
	}
	return score
}

func scoreBehavioral(br BehavioralReport) int {
	if br.TotalSessions == 0 {
		return 15 // Partial credit: no sessions to evaluate
	}
	if len(br.Anomalies) == 0 {
		return 25
	}
	deductions := 0
	for _, a := range br.Anomalies {
		switch a.Severity {
		case "critical":
			deductions += 12
		case "high":
			deductions += 8
		case "medium":
			deductions += 4
		case "low":
			deductions += 2
		}
	}
	score := 25 - deductions
	if score < 0 {
		return 0
	}
	return score
}

func scoreHygiene(checks []HygieneCheck) int {
	if len(checks) == 0 {
		return 0
	}
	passing := 0
	for _, c := range checks {
		if c.Status == "pass" {
			passing++
		}
	}
	return 25 * passing / len(checks)
}

// GenerateRecommendations creates actionable recommendations based on findings.
func GenerateRecommendations(report *PostureReport) []string {
	var recs []string

	// OAuth recommendations
	for _, f := range report.OAuthFindings {
		if f.Severity == "critical" || f.Severity == "high" {
			recs = append(recs, f.Remediation)
		}
	}

	// Unseeded agent recommendations
	for _, a := range report.Agents {
		if !a.Seeded {
			recs = append(recs, fmt.Sprintf("Seed VectraGuard into %s: `vg seed agents --targets %s`", a.Name, a.Name))
		}
	}

	// Behavioral recommendations
	if report.BehavioralReport.TotalSessions == 0 {
		recs = append(recs, "Start tracking agent sessions: `vg session start --agent <name>`")
	}
	for _, a := range report.BehavioralReport.Anomalies {
		if a.Severity == "critical" || a.Severity == "high" {
			recs = append(recs, fmt.Sprintf("Investigate %s anomaly in %s: %s", a.Type, a.AgentName, a.Detail))
		}
	}

	// Hygiene recommendations
	for _, c := range report.HygieneChecks {
		if c.Status == "fail" {
			recs = append(recs, c.Detail)
		}
	}

	// LLM trace recommendations
	for _, f := range report.LLMTraceFindings {
		if f.Severity == "critical" || f.Severity == "high" {
			recs = append(recs, fmt.Sprintf("Investigate LLM trace %s: %s %s", f.TraceID, f.Type, f.Detail))
		}
	}

	// Run behavioral profile recommendation
	hasBehavioralProfile := false
	for _, c := range report.HygieneChecks {
		if c.Name == "Behavioral profiles" && c.Status == "pass" {
			hasBehavioralProfile = true
			break
		}
	}
	if !hasBehavioralProfile && report.BehavioralReport.TotalSessions > 0 {
		recs = append(recs, "Run `vg behavioral profile --session <id>` after agent sessions")
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, r := range recs {
		if !seen[r] {
			seen[r] = true
			unique = append(unique, r)
		}
	}
	return unique
}

// EmitText writes the posture report in human-readable text format.
func EmitText(w io.Writer, report *PostureReport) {
	fmt.Fprintln(w, "Agent Security Posture Report")
	fmt.Fprintln(w, strings.Repeat("=", 40))
	fmt.Fprintf(w, "Path: %s\n", report.Path)
	fmt.Fprintf(w, "Score: %d/100\n\n", report.Score)

	// Agent inventory
	fmt.Fprintf(w, "AGENT INVENTORY (%d found)\n", len(report.Agents))
	for _, a := range report.Agents {
		icon := iconForRisk(a.RiskLevel)
		paths := strings.Join(a.ConfigPaths, ", ")
		fmt.Fprintf(w, "  %s %-14s %-8s %s\n", icon, a.Name, a.RiskLevel, paths)
	}
	if len(report.Agents) == 0 {
		fmt.Fprintln(w, "  No AI agent integrations detected")
	}
	fmt.Fprintln(w)

	// OAuth findings
	fmt.Fprintf(w, "OAUTH & API SCOPE ANALYSIS (%d findings)\n", len(report.OAuthFindings))
	for _, f := range report.OAuthFindings {
		icon := iconForSeverity(f.Severity)
		loc := f.File
		if f.Line > 0 {
			loc = fmt.Sprintf("%s:%d", f.File, f.Line)
		}
		fmt.Fprintf(w, "  %s %-10s %-20s %s\n", icon, strings.ToUpper(f.Severity), loc, f.Detail)
	}
	if len(report.OAuthFindings) == 0 {
		fmt.Fprintln(w, "  No OAuth/API scope issues detected")
	}
	fmt.Fprintln(w)

	// Behavioral health
	br := report.BehavioralReport
	fmt.Fprintf(w, "BEHAVIORAL HEALTH (%d sessions, %d agents)\n",
		br.TotalSessions, len(br.AgentBaselines))
	for _, b := range br.AgentBaselines {
		fmt.Fprintf(w, "  %s: %d sessions, avg risk %.0f",
			b.AgentName, b.SessionCount, b.AvgRiskScore)
		// Count anomalies for this agent
		agentAnomalies := 0
		for _, a := range br.Anomalies {
			if a.AgentName == b.AgentName {
				agentAnomalies++
			}
		}
		if agentAnomalies == 0 {
			fmt.Fprintln(w, ", no anomalies")
		} else {
			fmt.Fprintln(w)
		}
	}
	for _, a := range br.Anomalies {
		icon := iconForSeverity(a.Severity)
		fmt.Fprintf(w, "    %s %s: %s %s\n", icon, a.Type, a.SessionID, a.Detail)
	}
	if br.TotalSessions == 0 {
		fmt.Fprintln(w, "  No session data available")
	}
	fmt.Fprintln(w)

	// Hygiene checks
	passing := 0
	for _, c := range report.HygieneChecks {
		if c.Status == "pass" {
			passing++
		}
	}
	fmt.Fprintf(w, "SECURITY HYGIENE (%d/%d passing)\n", passing, len(report.HygieneChecks))
	for _, c := range report.HygieneChecks {
		icon := iconForStatus(c.Status)
		fmt.Fprintf(w, "  %s %s\n", icon, c.Detail)
	}
	fmt.Fprintln(w)

	// LLM trace findings
	if len(report.LLMTraceFindings) > 0 {
		fmt.Fprintf(w, "LLM TRACE ANALYSIS (%d findings)\n", len(report.LLMTraceFindings))
		for _, f := range report.LLMTraceFindings {
			icon := iconForSeverity(f.Severity)
			fmt.Fprintf(w, "  %s [%s] trace=%s type=%s: %s\n", icon, strings.ToUpper(f.Severity), f.TraceID, f.Type, f.Detail)
		}
		fmt.Fprintln(w)
	}

	// Recommendations
	if len(report.Recommendations) > 0 {
		fmt.Fprintln(w, "RECOMMENDATIONS")
		for i, r := range report.Recommendations {
			fmt.Fprintf(w, "  %d. %s\n", i+1, r)
		}
	}
}

// EmitJSON writes the posture report as indented JSON.
func EmitJSON(w io.Writer, report *PostureReport) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

func iconForRisk(risk string) string {
	switch risk {
	case "low":
		return "✓"
	case "medium":
		return "⚠"
	case "high":
		return "✗"
	default:
		return "?"
	}
}

func iconForSeverity(severity string) string {
	switch severity {
	case "critical":
		return "✗"
	case "high":
		return "⚠"
	case "medium":
		return "~"
	case "low":
		return "·"
	default:
		return "?"
	}
}

func iconForStatus(status string) string {
	switch status {
	case "pass":
		return "✓"
	case "warn":
		return "⚠"
	case "fail":
		return "✗"
	default:
		return "?"
	}
}
