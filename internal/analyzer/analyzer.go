package analyzer

import (
	"bufio"
	"path/filepath"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/config"
)

// Finding describes a potential risk detected in a script.
type Finding struct {
	Severity       string `json:"severity"`
	Code           string `json:"code"`
	Description    string `json:"description"`
	Line           int    `json:"line"`
	Recommendation string `json:"recommendation"`
}

// AnalyzeScript scans the script content and returns findings sorted by line number.
func AnalyzeScript(path string, content []byte, policy config.PolicyConfig) []Finding {
	var findings []Finding
	scanner := bufio.NewScanner(strings.NewReader(string(content)))
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if isAllowed(trimmed, policy.Allowlist) {
			continue
		}

		lower := strings.ToLower(trimmed)
		if containsAny(lower, policy.Denylist) {
			findings = append(findings, Finding{
				Severity:       "high",
				Code:           "POLICY_DENYLIST",
				Description:    "Command matches a denylisted pattern",
				Line:           lineNum,
				Recommendation: "Remove or justify this command, or update allowlist with review.",
			})
			continue
		}

		if strings.Contains(lower, "rm -rf /") {
			findings = append(findings, Finding{
				Severity:       "critical",
				Code:           "DANGEROUS_DELETE_ROOT",
				Description:    "Recursive delete from filesystem root detected",
				Line:           lineNum,
				Recommendation: "Limit delete scope to a safe path and avoid operating on '/'.",
			})
		}
		if strings.Contains(lower, "sudo ") {
			findings = append(findings, Finding{
				Severity:       "medium",
				Code:           "SUDO_USAGE",
				Description:    "Sudo usage without guard rails",
				Line:           lineNum,
				Recommendation: "Run with least privilege or document why elevated rights are required.",
			})
		}
		if strings.Contains(lower, "curl") && strings.Contains(lower, "|") && strings.Contains(lower, "sh") {
			findings = append(findings, Finding{
				Severity:       "high",
				Code:           "PIPE_TO_SHELL",
				Description:    "Piping remote content directly to shell",
				Line:           lineNum,
				Recommendation: "Download scripts to disk and review checksum before execution.",
			})
		}
		if strings.Contains(lower, ":(){ :|:& };:") {
			findings = append(findings, Finding{
				Severity:       "critical",
				Code:           "FORK_BOMB",
				Description:    "Potential fork bomb detected",
				Line:           lineNum,
				Recommendation: "Remove fork bomb pattern; it can render systems unusable.",
			})
		}
		if strings.Contains(lower, ">/etc/passwd") || strings.Contains(lower, ">/etc/shadow") {
			findings = append(findings, Finding{
				Severity:       "high",
				Code:           "SYSTEM_FILE_WRITE",
				Description:    "Attempt to overwrite sensitive system file",
				Line:           lineNum,
				Recommendation: "Avoid writing directly to system credential files.",
			})
		}

		// SQL/NoSQL database command detection
		dbCommands := []string{"mysql", "psql", "sqlite", "sqlcmd", "mongo", "mongosh", 
			"redis-cli", "cassandra", "cql", "dynamodb", "influx", "clickhouse"}
		if containsAnyWord(lower, dbCommands) {
			// Check for production/staging environment indicators
			envIndicators := []string{"prod", "production", "prd", "staging", "stg", "live"}
			envSeverity := "medium"
			envWarning := ""
			
			for _, env := range envIndicators {
				if strings.Contains(lower, env) {
					envSeverity = "high"
					envWarning = " in " + strings.ToUpper(env) + " ENVIRONMENT"
					break
				}
			}
			
			// Check for destructive operations
			destructiveOps := []string{"drop", "delete", "truncate", "alter", "update", "insert"}
			isDestructive := false
			for _, op := range destructiveOps {
				if strings.Contains(lower, op) {
					isDestructive = true
					if envSeverity == "high" {
						envSeverity = "critical"
					} else {
						envSeverity = "high"
					}
					break
				}
			}
			
			description := "Database command detected"
			if isDestructive {
				description = "Destructive database operation detected"
			}
			description += envWarning
			
			findings = append(findings, Finding{
				Severity:       envSeverity,
				Code:           "DATABASE_OPERATION",
				Description:    description,
				Line:           lineNum,
				Recommendation: "Review database operation carefully. Use transactions and backups. Require manual approval for production changes.",
			})
		}

		// Production/Staging environment warnings (general)
		envIndicators := []string{"prod", "production", "prd", "staging", "stg", "live"}
		for _, env := range envIndicators {
			// Look for environment in variable names, paths, or URLs
			if strings.Contains(lower, env) && (
				strings.Contains(lower, "export ") ||
				strings.Contains(lower, "env") ||
				strings.Contains(lower, "config") ||
				strings.Contains(lower, "url") ||
				strings.Contains(lower, "host") ||
				strings.Contains(lower, "endpoint")) {
				
				findings = append(findings, Finding{
					Severity:       "high",
					Code:           "PRODUCTION_ENVIRONMENT",
					Description:    "Production or staging environment detected: " + strings.ToUpper(env),
					Line:           lineNum,
					Recommendation: "Extra caution required. Require human approval before executing against production systems.",
				})
				break
			}
		}
	}

	// Incorporate file extension heuristics if script extension implies something unexpected.
	if ext := strings.ToLower(filepath.Ext(path)); ext != "" && ext != ".sh" {
		findings = append(findings, Finding{
			Severity:       "low",
			Code:           "NON_STANDARD_EXTENSION",
			Description:    "Script does not use .sh extension",
			Line:           0,
			Recommendation: "Use a .sh extension to make shell scripts explicit.",
		})
	}

	return findings
}

func isAllowed(line string, allow []string) bool {
	for _, pattern := range allow {
		if pattern != "" && strings.Contains(line, pattern) {
			return true
		}
	}
	return false
}

func containsAny(line string, patterns []string) bool {
	for _, pattern := range patterns {
		if pattern != "" && strings.Contains(line, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func containsAnyWord(line string, words []string) bool {
	// Split line into words to avoid false positives in substrings
	lineWords := strings.Fields(line)
	for _, word := range lineWords {
		// Remove common command prefixes and special chars
		cleanWord := strings.TrimFunc(word, func(r rune) bool {
			return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_')
		})
		
		for _, target := range words {
			if strings.HasPrefix(strings.ToLower(cleanWord), strings.ToLower(target)) {
				return true
			}
		}
	}
	return false
}
