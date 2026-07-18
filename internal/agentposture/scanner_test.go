package agentposture

import (
	"os"
	"path/filepath"
	"testing"
)

// findingOfType reports whether findings contain a given OAuth finding type.
func findingOfType(findings []OAuthFinding, typ string) bool {
	for _, f := range findings {
		if f.Type == typ {
			return true
		}
	}
	return false
}

// TestScanOAuth_BroadScopeAndSecrets verifies OAuth scope/secret issues are detected in env files.
func TestScanOAuth_BroadScopeAndSecrets(t *testing.T) {
	dir := t.TempDir()
	env := "GITHUB_SCOPE=all\n" +
		"SLACK_CLIENT_SECRET=shhh-do-not-share\n" +
		"OPENAI_API_KEY=sk-test-1234567890\n"
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte(env), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	// No .gitignore on purpose -> should also flag missing_gitignore.

	findings := ScanOAuth(dir)
	for _, typ := range []string{"broad_scope", "committed_secret", "unscoped_token", "missing_gitignore"} {
		if !findingOfType(findings, typ) {
			t.Errorf("expected an OAuth finding of type %q, got %+v", typ, findings)
		}
	}
}

// TestScanOAuth_CleanWorkspace verifies a protected, non-sensitive env file yields no findings.
func TestScanOAuth_CleanWorkspace(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("APP_NAME=demo\nPORT=3000\n"), 0o600); err != nil {
		t.Fatalf("write .env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".env*\n"), 0o600); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	findings := ScanOAuth(dir)
	if len(findings) != 0 {
		t.Fatalf("expected no OAuth findings for clean protected workspace, got %+v", findings)
	}
}

// TestComputeScore_Perfect verifies an empty/healthy report scores well and clamps to 100.
func TestComputeScore_Perfect(t *testing.T) {
	report := &PostureReport{
		Agents:           nil,                                                // no agents -> 20
		OAuthFindings:    nil,                                                // no findings -> 30
		BehavioralReport: BehavioralReport{TotalSessions: 3, Anomalies: nil}, // -> 25
		HygieneChecks: []HygieneCheck{
			{Name: "a", Status: "pass"},
			{Name: "b", Status: "pass"},
		}, // all pass -> 25
	}
	if got := ComputeScore(report); got != 100 {
		t.Fatalf("expected perfect score 100, got %d", got)
	}
}

// TestComputeScore_Deductions verifies severe findings deduct and the score never goes negative.
func TestComputeScore_Deductions(t *testing.T) {
	report := &PostureReport{
		Agents: []AgentFinding{{Name: "cursor", Seeded: false}}, // 0/1 seeded -> 0
		OAuthFindings: []OAuthFinding{
			{Severity: "critical"}, {Severity: "critical"}, {Severity: "high"}, // 15+15+10 -> clamped 0
		},
		BehavioralReport: BehavioralReport{
			TotalSessions: 2,
			Anomalies:     []Anomaly{{Severity: "critical"}, {Severity: "high"}}, // 12+8 -> 5
		},
		HygieneChecks: []HygieneCheck{
			{Status: "pass"}, {Status: "fail"}, {Status: "fail"}, {Status: "fail"}, // 1/4 -> 6
		},
	}
	got := ComputeScore(report)
	if got < 0 || got > 100 {
		t.Fatalf("score out of range: %d", got)
	}
	// Agent(0) + OAuth(0) + Behavioral(5) + Hygiene(6) = 11
	if got != 11 {
		t.Fatalf("expected score 11 from documented breakdown, got %d", got)
	}
}
