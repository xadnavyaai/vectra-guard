package analyzer

import (
	"bufio"
	"fmt"
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

		// Enhanced destructive file operations detection
		// Check for ANY rm command targeting root or system directories
		if strings.Contains(lower, "rm ") {
			// Check for root deletion patterns (catches rm -rf /, rm -r /*, rm -rf /*, etc.)
			rootDeletePatterns := []string{
				"rm -rf /",      // Original pattern
				"rm -r /",       // Without force flag
				"rm -rf / ",     // With trailing space
				"rm -r / ",      // Without force, with space
				"rm -rf /*",     // With wildcard
				"rm -r /*",      // Without force, with wildcard
				"rm -rf /* ",    // With wildcard and space
				"rm -r /* ",     // Without force, wildcard, space
				"rm -rf / *",    // Space between / and *
				"rm -r / *",     // Without force, space between
				"rm -rf /bin",   // System directories
				"rm -rf /usr",   // System directories
				"rm -rf /etc",   // System directories
				"rm -rf /var",   // System directories
				"rm -rf /opt",   // System directories
				"rm -rf /sbin",  // System directories
				"rm -rf /lib",   // System directories
				"rm -rf /lib64", // System directories
				"rm -rf /sys",   // System directories
				"rm -rf /proc",  // System directories
				"rm -rf /dev",   // System directories
				"rm -rf /boot",  // System directories
				"rm -rf /root",  // Root home directory
			}
			
			for _, pattern := range rootDeletePatterns {
				if strings.Contains(lower, pattern) {
					findings = append(findings, Finding{
						Severity:       "critical",
						Code:           "DANGEROUS_DELETE_ROOT",
						Description:    fmt.Sprintf("Recursive delete targeting root or system directory detected: %s", pattern),
						Line:           lineNum,
						Recommendation: "BLOCKED: This command would destroy the system. Never delete from root or system directories.",
					})
					break // Only report once
				}
			}
			
			// Check for home directory deletion patterns
			// Pattern: rm -rf ~/* or rm -rf $HOME/* (could delete user's entire home)
			homeDeletePatterns := []string{
				"rm -rf ~/*",
				"rm -r ~/*",
				"rm -rf ~/ *",
				"rm -r ~/ *",
				"rm -rf $home/*",
				"rm -r $home/*",
				"rm -rf $home/ *",
				"rm -r $home/ *",
				"rm -rf $home/*",
				"rm -r $home/*",
				"rm -rf $home/ *",
				"rm -r $home/ *",
			}
			
			for _, pattern := range homeDeletePatterns {
				if strings.Contains(lower, pattern) {
					findings = append(findings, Finding{
						Severity:       "high",
						Code:           "DANGEROUS_DELETE_HOME",
						Description:    "Recursive delete targeting home directory detected",
						Line:           lineNum,
						Recommendation: "This could delete all user data. Specify exact paths instead of wildcards.",
					})
					break // Only report once
				}
			}
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

		// Git operations monitoring
		if policy.MonitorGitOps {
			gitRiskyOps := map[string]struct {
				severity string
				desc     string
				rec      string
			}{
				"git push --force":     {"high", "Force push detected - can overwrite remote history", "Use --force-with-lease instead or coordinate with team before force pushing."},
				"git push -f":          {"high", "Force push detected - can overwrite remote history", "Use --force-with-lease instead or coordinate with team before force pushing."},
				"git reset --hard":     {"medium", "Hard reset detected - will discard local changes", "Ensure you have backups or stash important changes first."},
				"git clean -fd":        {"medium", "Git clean with force - will delete untracked files", "Review untracked files before cleaning. Consider using -n flag first for dry run."},
				"git branch -D":        {"medium", "Force branch deletion detected", "Ensure branch is merged or no longer needed before force deleting."},
				"git branch -d":        {"low", "Branch deletion detected", "Verify branch is fully merged before deletion."},
				"git rebase":           {"low", "Git rebase detected - will rewrite commit history", "Only rebase local commits. Never rebase published commits."},
				"git filter-branch":    {"high", "Git filter-branch - rewrites entire repository history", "Extremely dangerous. Coordinate with entire team and backup repository first."},
				"git filter-repo":      {"high", "Git filter-repo - rewrites entire repository history", "Extremely dangerous. Coordinate with entire team and backup repository first."},
				"git reflog expire":    {"high", "Reflog expiration - will permanently delete commit references", "Only use if you know what you're doing. Lost commits cannot be recovered."},
				"git gc --aggressive":  {"medium", "Aggressive garbage collection - may make recovery difficult", "Ensure no important dangling commits exist before running."},
				"git update-ref -d":    {"high", "Direct ref manipulation detected", "Advanced operation. Ensure you understand git internals before proceeding."},
			}
			
			for pattern, info := range gitRiskyOps {
				if strings.Contains(lower, pattern) {
					severity := info.severity
					
					// Elevate severity if production environment detected
					if policy.DetectProdEnv {
						for _, env := range policy.ProdEnvPatterns {
							if strings.Contains(lower, env) {
								if severity == "medium" {
									severity = "high"
								} else if severity == "high" {
									severity = "critical"
								}
								info.desc += " in " + strings.ToUpper(env) + " environment"
								break
							}
						}
					}
					
					// Block force operations if configured
					if policy.BlockForceGit && (strings.Contains(pattern, "--force") || strings.Contains(pattern, "-f")) {
						severity = "critical"
					}
					
					findings = append(findings, Finding{
						Severity:       severity,
						Code:           "RISKY_GIT_OPERATION",
						Description:    info.desc,
						Line:           lineNum,
						Recommendation: info.rec,
					})
					break
				}
			}
		}

		// SQL/NoSQL database command detection (refined to only flag destructive operations)
		dbCommands := []string{"mysql", "psql", "sqlite", "sqlcmd", "mongo", "mongosh", 
			"redis-cli", "cassandra", "cql", "dynamodb", "influx", "clickhouse"}
		if containsAnyWord(lower, dbCommands) {
			// Check for destructive SQL operations
			destructiveSQLOps := []string{
				"drop database", "drop table", "drop schema", "drop index",
				"truncate table", "truncate",
				"delete from", "delete ",
				"update ", "alter table", "alter database",
				"grant all", "revoke",
			}
			
			isDestructive := false
			destructiveOp := ""
			for _, op := range destructiveSQLOps {
				if strings.Contains(lower, op) {
					isDestructive = true
					destructiveOp = op
					break
				}
			}
			
			// Only flag if destructive OR if OnlyDestructiveSQL is false
			if isDestructive || !policy.OnlyDestructiveSQL {
				envSeverity := "medium"
				envWarning := ""
				
				// Check for production/staging environment indicators
				if policy.DetectProdEnv {
					for _, env := range policy.ProdEnvPatterns {
						if strings.Contains(lower, env) {
							envSeverity = "high"
							envWarning = " in " + strings.ToUpper(env) + " ENVIRONMENT"
							break
						}
					}
				}
				
				if isDestructive {
					if envSeverity == "high" {
						envSeverity = "critical"
					} else {
						envSeverity = "high"
					}
				}
				
				description := "Database command detected"
				if isDestructive {
					description = "Destructive database operation detected: " + destructiveOp
				}
				description += envWarning
				
				recommendation := "Review database operation carefully. Use transactions and backups."
				if envWarning != "" {
					recommendation += " REQUIRE MANUAL APPROVAL for production changes."
				}
				
				findings = append(findings, Finding{
					Severity:       envSeverity,
					Code:           "DATABASE_OPERATION",
					Description:    description,
					Line:           lineNum,
					Recommendation: recommendation,
				})
			}
		}

		// Production/Staging environment warnings (general)
		if policy.DetectProdEnv {
			envPatterns := policy.ProdEnvPatterns
			if len(envPatterns) == 0 {
				envPatterns = []string{"prod", "production", "prd", "staging", "stg", "live"}
			}
			
			for _, env := range envPatterns {
				// Look for environment in variable names, paths, URLs, or commands
				if strings.Contains(lower, env) {
					// Check if it's in a meaningful context
					inContext := false
					contextIndicators := []string{
						"export ", "env", "config", "url", "host", "endpoint",
						"deploy", "kubectl", "docker", "aws", "gcloud", "azure",
						"ssh", "scp", "rsync", "curl", "wget", "ansible", "terraform",
						"database", "db", "server", "cluster", "namespace",
					}
					
					for _, indicator := range contextIndicators {
						if strings.Contains(lower, indicator) {
							inContext = true
							break
						}
					}
					
					// Also check if env pattern appears in a path-like string
					if strings.Contains(lower, "/"+env+"/") || 
					   strings.Contains(lower, "-"+env+"-") ||
					   strings.Contains(lower, "_"+env+"_") ||
					   strings.Contains(lower, "."+env+".") ||
					   strings.Contains(lower, "@"+env) {
						inContext = true
					}
					
					if inContext {
						findings = append(findings, Finding{
							Severity:       "high",
							Code:           "PRODUCTION_ENVIRONMENT",
							Description:    "Production or staging environment detected: " + strings.ToUpper(env),
							Line:           lineNum,
							Recommendation: "Extra caution required. REQUIRE HUMAN APPROVAL before executing against production systems.",
						})
						break
					}
				}
			}
		}

		// Environment variable access detection
		envAccessPatterns := []string{
			"printenv", "env", "export -p", "set |", "declare -p",
			"cat .env", "cat ~/.env", "source .env",
		}
		if containsAnyWord(lower, envAccessPatterns) || strings.Contains(lower, "cat .env") {
			findings = append(findings, Finding{
				Severity:       "high",
				Code:           "ENV_ACCESS",
				Description:    "Environment variable access detected",
				Line:           lineNum,
				Recommendation: "Agent attempting to read environment variables. Consider masking sensitive values or blocking access.",
			})
		}

		// Sensitive environment variable patterns
		sensitiveEnvPatterns := []string{
			"$password", "$secret", "$key", "$token", "$api_key",
			"$aws_secret", "$aws_access_key", "$github_token", "$ssh_key",
			"$db_password", "$database_url", "$private_key", "$auth_token",
		}
		for _, pattern := range sensitiveEnvPatterns {
			if strings.Contains(lower, pattern) || 
			   strings.Contains(lower, strings.ToUpper(pattern)) ||
			   strings.Contains(lower, strings.ReplaceAll(pattern, "_", "")) {
				findings = append(findings, Finding{
					Severity:       "critical",
					Code:           "SENSITIVE_ENV_ACCESS",
					Description:    "Attempt to access sensitive environment variable: " + pattern,
					Line:           lineNum,
					Recommendation: "BLOCK or MASK this operation. Agent should not access credentials directly. Use secure secret management.",
				})
				break
			}
		}

		// .env file operations
		if (strings.Contains(lower, ".env") || strings.Contains(lower, "dotenv")) &&
		   (strings.Contains(lower, "cat") || strings.Contains(lower, "less") || 
		    strings.Contains(lower, "head") || strings.Contains(lower, "tail") ||
		    strings.Contains(lower, "grep") || strings.Contains(lower, "awk")) {
			findings = append(findings, Finding{
				Severity:       "critical",
				Code:           "DOTENV_FILE_READ",
				Description:    "Attempt to read .env file containing credentials",
				Line:           lineNum,
				Recommendation: "BLOCK this operation. Provide sanitized config instead of exposing raw .env files.",
			})
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
