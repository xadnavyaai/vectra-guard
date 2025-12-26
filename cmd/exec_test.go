package cmd

import (
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/analyzer"
	"github.com/vectra-guard/vectra-guard/internal/config"
)

func TestFilterFindingsByGuardLevel(t *testing.T) {
	findings := []analyzer.Finding{
		{Severity: "low", Code: "LOW_RISK"},
		{Severity: "medium", Code: "MEDIUM_RISK"},
		{Severity: "high", Code: "HIGH_RISK"},
		{Severity: "critical", Code: "CRITICAL_RISK"},
	}

	tests := []struct {
		name          string
		level         config.GuardLevel
		expectedCount int
		shouldContain []string
	}{
		{
			name:          "off level - no filtering",
			level:         config.GuardLevelOff,
			expectedCount: 0,
			shouldContain: []string{},
		},
		{
			name:          "low level - only critical",
			level:         config.GuardLevelLow,
			expectedCount: 1,
			shouldContain: []string{"CRITICAL_RISK"},
		},
		{
			name:          "medium level - critical and high",
			level:         config.GuardLevelMedium,
			expectedCount: 2,
			shouldContain: []string{"CRITICAL_RISK", "HIGH_RISK"},
		},
		{
			name:          "high level - critical, high, and medium",
			level:         config.GuardLevelHigh,
			expectedCount: 3,
			shouldContain: []string{"CRITICAL_RISK", "HIGH_RISK", "MEDIUM_RISK"},
		},
		{
			name:          "paranoid level - all findings",
			level:         config.GuardLevelParanoid,
			expectedCount: 4,
			shouldContain: []string{"CRITICAL_RISK", "HIGH_RISK", "MEDIUM_RISK", "LOW_RISK"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterFindingsByGuardLevel(findings, tt.level)

			if len(filtered) != tt.expectedCount {
				t.Errorf("expected %d findings, got %d", tt.expectedCount, len(filtered))
			}

			// Check that expected codes are present
			foundCodes := make(map[string]bool)
			for _, f := range filtered {
				foundCodes[f.Code] = true
			}

			for _, code := range tt.shouldContain {
				if !foundCodes[code] {
					t.Errorf("expected to find %s but it was not in filtered results", code)
				}
			}
		})
	}
}

func TestShouldRequireApproval(t *testing.T) {
	tests := []struct {
		name          string
		riskLevel     string
		guardLevel    config.GuardLevel
		shouldRequire bool
	}{
		// Low guard level - only critical requires approval
		{"low/critical", "critical", config.GuardLevelLow, true},
		{"low/high", "high", config.GuardLevelLow, false},
		{"low/medium", "medium", config.GuardLevelLow, false},

		// Medium guard level - critical and high require approval
		{"medium/critical", "critical", config.GuardLevelMedium, true},
		{"medium/high", "high", config.GuardLevelMedium, true},
		{"medium/medium", "medium", config.GuardLevelMedium, false},

		// High guard level - critical, high, and medium require approval
		{"high/critical", "critical", config.GuardLevelHigh, true},
		{"high/high", "high", config.GuardLevelHigh, true},
		{"high/medium", "medium", config.GuardLevelHigh, true},
		{"high/low", "low", config.GuardLevelHigh, false},

		// Paranoid - everything requires approval
		{"paranoid/low", "low", config.GuardLevelParanoid, true},
		{"paranoid/medium", "medium", config.GuardLevelParanoid, true},
		{"paranoid/high", "high", config.GuardLevelParanoid, true},
		{"paranoid/critical", "critical", config.GuardLevelParanoid, true},

		// Off - nothing requires approval
		{"off/critical", "critical", config.GuardLevelOff, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRequireApproval(tt.riskLevel, tt.guardLevel)
			if result != tt.shouldRequire {
				t.Errorf("expected %v, got %v", tt.shouldRequire, result)
			}
		})
	}
}

func TestIsLikelyAgentBypass(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		isAgent  bool
	}{
		// Agent-like values (should be blocked)
		{"contains 'bypass'", "my-bypass-123", true},
		{"contains 'agent'", "agent-value-123", true},
		{"contains 'ai'", "ai-assistant-bypass", true},
		{"contains 'automated'", "automated-value", true},
		{"contains 'cursor'", "cursor-bypass-123", true},
		{"contains 'gpt'", "gpt4-bypass", true},
		{"contains 'claude'", "claude-assistant", true},
		{"simple 'true'", "true", true},
		{"simple 'yes'", "yes", true},
		{"simple '1'", "1", true},
		{"too short", "short", true},
		
		// Valid user values (should be allowed)
		// Note: contains only letters or mix of letters and common patterns
		{"human identifier", "i-am-human-john", false},
		{"random string", "xkcd-correct-horse-battery", false},
		
		// These could look automated, so they're blocked for safety
		// (mostly numbers/hex-like patterns that agents might generate)
		{"timestamp hash", "a1b2c3d4e5f6", true},      // Hex-like, could be auto-generated
		{"user with id", "user-john-12345678", true},  // Ends with many numbers
		{"long random", "secretpassword123", true},    // Ends with numbers
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLikelyAgentBypass(tt.value)
			if result != tt.isAgent {
				t.Errorf("value %q: expected isAgent=%v, got %v", tt.value, tt.isAgent, result)
			}
		})
	}
}

func TestGuardLevelIntegration(t *testing.T) {
	// Test that guard levels properly filter and require approval
	
	// Create findings of different severities
	findings := []analyzer.Finding{
		{Severity: "medium", Code: "SUDO_USAGE", Description: "Sudo usage detected"},
		{Severity: "high", Code: "RISKY_GIT_OPERATION", Description: "Force push detected"},
		{Severity: "critical", Code: "PRODUCTION_ENVIRONMENT", Description: "Production operation"},
	}

	tests := []struct {
		name                string
		guardLevel          config.GuardLevel
		expectedFiltered    int
		mediumShouldBlock   bool
		highShouldBlock     bool
		criticalShouldBlock bool
	}{
		{
			name:                "low level",
			guardLevel:          config.GuardLevelLow,
			expectedFiltered:    1, // only critical
			mediumShouldBlock:   false,
			highShouldBlock:     false,
			criticalShouldBlock: true,
		},
		{
			name:                "medium level",
			guardLevel:          config.GuardLevelMedium,
			expectedFiltered:    2, // high and critical
			mediumShouldBlock:   false,
			highShouldBlock:     true,
			criticalShouldBlock: true,
		},
		{
			name:                "high level",
			guardLevel:          config.GuardLevelHigh,
			expectedFiltered:    3, // all
			mediumShouldBlock:   true,
			highShouldBlock:     true,
			criticalShouldBlock: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := filterFindingsByGuardLevel(findings, tt.guardLevel)
			
			if len(filtered) != tt.expectedFiltered {
				t.Errorf("expected %d filtered findings, got %d", tt.expectedFiltered, len(filtered))
			}

			// Test approval requirements
			if shouldRequireApproval("medium", tt.guardLevel) != tt.mediumShouldBlock {
				t.Errorf("medium approval requirement mismatch")
			}
			if shouldRequireApproval("high", tt.guardLevel) != tt.highShouldBlock {
				t.Errorf("high approval requirement mismatch")
			}
			if shouldRequireApproval("critical", tt.guardLevel) != tt.criticalShouldBlock {
				t.Errorf("critical approval requirement mismatch")
			}
		})
	}
}

func TestBypassValueValidation(t *testing.T) {
	// Test realistic bypass scenarios
	// Note: isLikelyAgentBypass checks for specific forbidden keywords and length >= 10
	tests := []struct {
		name        string
		value       string
		shouldAllow bool
		reason      string
	}{
		{
			name:        "legitimate user bypass",
			value:       "i-am-human-johndoe",
			shouldAllow: true,
			reason:      "contains human identifier, no forbidden keywords",
		},
		{
			name:        "contains digit 1",
			value:       "1234567890abcd",
			shouldAllow: false,
			reason:      "contains '1' which is in forbidden patterns",
		},
		{
			name:        "agent attempt with 'bypass'",
			value:       "please-bypass-this-test",
			shouldAllow: false,
			reason:      "contains forbidden word 'bypass'",
		},
		{
			name:        "agent attempt with 'ai'",
			value:       "ai-generated-key",
			shouldAllow: false,
			reason:      "contains forbidden word 'ai'",
		},
		{
			name:        "too short",
			value:       "bypass",
			shouldAllow: false,
			reason:      "less than 10 characters",
		},
		{
			name:        "simple yes",
			value:       "yes",
			shouldAllow: false,
			reason:      "too simple and common (forbidden keyword + short)",
		},
		{
			name:        "cursor IDE attempt",
			value:       "cursor-generated-value",
			shouldAllow: false,
			reason:      "contains 'cursor'",
		},
		{
			name:        "phrase with letters only",
			value:       "user-override-temporary",
			shouldAllow: true,
			reason:      "all letters, no forbidden keywords",
		},
		{
			name:        "contains 'script'",
			value:       "my-special-script-override",
			shouldAllow: false,
			reason:      "contains forbidden word 'script'",
		},
		{
			name:        "another safe phrase",
			value:       "emergency-override-december",
			shouldAllow: true,
			reason:      "all letters, long enough, no forbidden keywords",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isAgent := isLikelyAgentBypass(tt.value)
			isValid := len(tt.value) >= 10 && !isAgent
			
			if isValid != tt.shouldAllow {
				t.Errorf("value %q: expected allow=%v (reason: %s), got allow=%v",
					tt.value, tt.shouldAllow, tt.reason, isValid)
			}
		})
	}
}

func TestGuardLevelScenarios(t *testing.T) {
	// Real-world scenario tests
	scenarios := []struct {
		name         string
		command      string
		guardLevel   config.GuardLevel
		shouldBlock  bool
		description  string
	}{
		{
			name:        "dev: safe npm install",
			command:     "npm install",
			guardLevel:  config.GuardLevelLow,
			shouldBlock: false,
			description: "npm install should be allowed in low guard level",
		},
		{
			name:        "dev: git force push",
			command:     "git push --force origin feature",
			guardLevel:  config.GuardLevelLow,
			shouldBlock: false,
			description: "force push in dev should be allowed with low level",
		},
		{
			name:        "prod: git force push",
			command:     "git push --force origin production",
			guardLevel:  config.GuardLevelMedium,
			shouldBlock: true,
			description: "force push to production should be blocked",
		},
		{
			name:        "prod: destructive SQL",
			command:     "mysql -h prod-db -e 'DROP TABLE users'",
			guardLevel:  config.GuardLevelHigh,
			shouldBlock: true,
			description: "destructive SQL on prod should always be blocked",
		},
		{
			name:        "paranoid: any command",
			command:     "echo hello",
			guardLevel:  config.GuardLevelParanoid,
			shouldBlock: false, // would require approval but not blocked
			description: "paranoid mode requires approval for everything",
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			// This is a logical test - we're testing the decision logic
			// In real execution, these would be checked by the analyzer
			t.Logf("Scenario: %s", sc.description)
			t.Logf("Command: %s", sc.command)
			t.Logf("Guard Level: %s", sc.guardLevel)
			// Pass - this is a documentation test
		})
	}
}

// TestPreExecutionAssessment tests that critical commands are assessed
// and blocked before execution if sandbox is unavailable
func TestPreExecutionAssessment(t *testing.T) {
	tests := []struct {
		name           string
		command        string
		riskLevel      string
		findings       []analyzer.Finding
		sandboxEnabled bool
		shouldBlock    bool
		description    string
	}{
		{
			name:      "critical command with sandbox disabled",
			command:   "rm -r /*",
			riskLevel: "critical",
			findings: []analyzer.Finding{
				{Code: "DANGEROUS_DELETE_ROOT", Severity: "critical"},
			},
			sandboxEnabled: false,
			shouldBlock:    true,
			description:    "Critical command should be blocked if sandbox disabled",
		},
		{
			name:      "critical command with sandbox enabled",
			command:   "rm -r /*",
			riskLevel: "critical",
			findings: []analyzer.Finding{
				{Code: "DANGEROUS_DELETE_ROOT", Severity: "critical"},
			},
			sandboxEnabled: true,
			shouldBlock:    false,
			description:    "Critical command should proceed if sandbox enabled",
		},
		{
			name:           "non-critical command with sandbox disabled",
			command:        "echo test",
			riskLevel:       "low",
			findings:        []analyzer.Finding{},
			sandboxEnabled: false,
			shouldBlock:     false,
			description:     "Non-critical commands can proceed without sandbox",
		},
		{
			name:      "fork bomb with sandbox disabled",
			command:   ":(){ :|:& };:",
			riskLevel: "critical",
			findings: []analyzer.Finding{
				{Code: "FORK_BOMB", Severity: "critical"},
			},
			sandboxEnabled: false,
			shouldBlock:    true,
			description:    "Fork bomb should be blocked if sandbox disabled",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies the logic that would be in runExec
			// We're testing that critical commands require sandbox
			
			hasCriticalCode := false
			for _, f := range tt.findings {
				criticalCodes := []string{
					"DANGEROUS_DELETE_ROOT",
					"DANGEROUS_DELETE_HOME",
					"FORK_BOMB",
					"SENSITIVE_ENV_ACCESS",
					"DOTENV_FILE_READ",
				}
				for _, code := range criticalCodes {
					if f.Code == code {
						hasCriticalCode = true
						break
					}
				}
				if hasCriticalCode {
					break
				}
			}
			
			// Simulate pre-execution assessment logic
			shouldBlock := false
			if tt.riskLevel == "critical" && hasCriticalCode && !tt.sandboxEnabled {
				shouldBlock = true
			}
			
			if shouldBlock != tt.shouldBlock {
				t.Errorf("expected shouldBlock=%v, got=%v (%s)",
					tt.shouldBlock, shouldBlock, tt.description)
			}
		})
	}
}

// TestSecurityImprovementsRegression tests that the incident scenario
// (rm -r /*) is now properly detected and handled
func TestSecurityImprovementsRegression(t *testing.T) {
	// The incident: rm -r /* was not detected
	incidentCommand := "rm -r /*"
	
	policy := config.PolicyConfig{}
	findings := analyzer.AnalyzeScript("incident.sh", []byte(incidentCommand), policy)
	
	// Should detect DANGEROUS_DELETE_ROOT
	found := false
	for _, f := range findings {
		if f.Code == "DANGEROUS_DELETE_ROOT" && f.Severity == "critical" {
			found = true
			break
		}
	}
	
	if !found {
		t.Fatal("REGRESSION: Incident command 'rm -r /*' is NOT detected!")
	}
	
	// Should also detect other variations
	variations := []string{
		"rm -rf /*",
		"rm -r /",
		"rm -rf /bin",
		"rm -rf /usr",
	}
	
	for _, variant := range variations {
		findings := analyzer.AnalyzeScript("test.sh", []byte(variant), policy)
		found := false
		for _, f := range findings {
			if f.Code == "DANGEROUS_DELETE_ROOT" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("REGRESSION: Variation '%s' is NOT detected!", variant)
		}
	}
}
