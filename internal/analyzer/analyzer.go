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
		
		// Smart Python command parsing: Extract shell commands from Python code
		if isPythonCommand(trimmed) {
			pythonCode := extractPythonCodeFromCommand(trimmed)
			if pythonCode != "" {
				extractedCommands := extractPythonCommands(pythonCode)
				for _, extractedCmd := range extractedCommands {
					// Recursively analyze extracted commands
					extractedFindings := analyzeExtractedCommand(extractedCmd, lineNum, policy)
					findings = append(findings, extractedFindings...)
				}
			}
		}
		
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
			homeDeleteFound := false

			// IMPORTANT: Check home directory deletion patterns FIRST (before root patterns)
			// Pattern: rm -rf ~/* or rm -rf $HOME/* (could delete user's entire home)
			homeDeletePatterns := []string{
				"rm -rf ~/*",
				"rm -r ~/*",
				"rm -rf ~/ *",
				"rm -r ~/ *",
				"rm -rf $HOME/*",
				"rm -r $HOME/*",
				"rm -rf $HOME/ *",
				"rm -r $HOME/ *",
			}

			// Also check for expanded home paths like /home/username/* or /home/*
			if strings.Contains(lower, "rm -rf /home/") && strings.Contains(lower, "/*") {
				homeDeletePatterns = append(homeDeletePatterns, "rm -rf /home/")
			}
			if strings.Contains(lower, "rm -rf /home/*") || strings.Contains(lower, "rm -r /home/*") {
				homeDeletePatterns = append(homeDeletePatterns, "rm -rf /home/*", "rm -r /home/*")
			}

			for _, pattern := range homeDeletePatterns {
				if strings.Contains(lower, pattern) {
					findings = append(findings, Finding{
						Severity:       "critical",
						Code:           "DANGEROUS_DELETE_HOME",
						Description:    "Recursive delete targeting home directory detected",
						Line:           lineNum,
						Recommendation: "BLOCKED: This could delete all user data. Never delete from home directory with wildcards.",
					})
					homeDeleteFound = true
					break // Only report once
				}
			}

			// Check for root deletion patterns (catches rm -rf /, rm -r /*, rm -rf /*, etc.)
			// Skip if home delete was found to prevent double detection
			if !homeDeleteFound {
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
					"rm -rf /usr/local",
					"rm -rf /home",  // Entire /home directory (not /home/* which is caught above)
					"rm -rf /srv",
					"rm -rf /var/log",
					"rm -rf /etc/ssh",
					"rm -rf /system",
					"rm -rf /library",
					"rm -rf /applications",
					"rm -rf /tmp/*",  // Temp directory wildcard deletion
					"rm -r /tmp/*",   // Temp directory wildcard deletion (without force)
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
			}
		}

		// Destructive find operations (find / -delete)
		if strings.Contains(lower, "find /") && strings.Contains(lower, "-delete") {
			findings = append(findings, Finding{
				Severity:       "critical",
				Code:           "DANGEROUS_DELETE_ROOT",
				Description:    "Destructive find command targeting root directory with -delete",
				Line:           lineNum,
				Recommendation: "BLOCKED: This command would destroy the system. Never use find / with -delete.",
			})
		}
		if strings.Contains(lower, "find /") && (strings.Contains(lower, "-type f -delete") || strings.Contains(lower, "-type d -delete")) {
			findings = append(findings, Finding{
				Severity:       "critical",
				Code:           "DANGEROUS_DELETE_ROOT",
				Description:    "Destructive find command targeting root directory with type-specific delete",
				Line:           lineNum,
				Recommendation: "BLOCKED: This command would destroy the system. Never use find / with -delete.",
			})
		}

		// Destructive disk operations
		diskWipeKeywords := []string{
			"wipefs", "sfdisk", "fdisk", "parted", "sgdisk", "blkdiscard",
			"pvremove", "vgremove", "lvremove",
		}
		// Check for dd if=/dev/zero (disk wiping)
		if strings.Contains(lower, "dd if=/dev/zero") || strings.Contains(lower, "dd if=/dev/urandom") {
			findings = append(findings, Finding{
				Severity:       "critical",
				Code:           "DISK_WIPE",
				Description:    "Destructive disk wipe operation detected (dd if=/dev/zero)",
				Line:           lineNum,
				Recommendation: "BLOCK this command. It can destroy filesystems or volumes irreversibly.",
			})
		}
		// Check for mkfs (filesystem creation - destructive if targeting mounted drives)
		if strings.Contains(lower, "mkfs") && (strings.Contains(lower, "/dev/") || strings.Contains(lower, "/dev/sd") || strings.Contains(lower, "/dev/hd")) {
			findings = append(findings, Finding{
				Severity:       "critical",
				Code:           "DISK_WIPE",
				Description:    "Destructive filesystem creation detected (mkfs)",
				Line:           lineNum,
				Recommendation: "BLOCK this command. It can destroy filesystems or volumes irreversibly.",
			})
		}
		if containsAnyWord(lower, diskWipeKeywords) ||
			(strings.Contains(lower, "cryptsetup") && strings.Contains(lower, "luksformat")) {
			findings = append(findings, Finding{
				Severity:       "critical",
				Code:           "DISK_WIPE",
				Description:    "Destructive disk or volume operation detected",
				Line:           lineNum,
				Recommendation: "BLOCK this command. It can destroy filesystems or volumes irreversibly.",
			})
		}

		// Destructive permission changes on system paths
		if strings.Contains(lower, "chmod -r") || strings.Contains(lower, "chmod -R") {
			if containsSystemPath(strings.Fields(lower)) {
				findings = append(findings, Finding{
					Severity:       "high",
					Code:           "DANGEROUS_PERMISSIONS",
					Description:    "Recursive chmod on system path detected",
					Line:           lineNum,
					Recommendation: "Avoid recursive permission changes on system paths. Scope to specific files.",
				})
			}
		}
		if strings.Contains(lower, "chown -r") || strings.Contains(lower, "chown -R") {
			if containsSystemPath(strings.Fields(lower)) {
				findings = append(findings, Finding{
					Severity:       "high",
					Code:           "DANGEROUS_PERMISSIONS",
					Description:    "Recursive chown on system path detected",
					Line:           lineNum,
					Recommendation: "Avoid recursive ownership changes on system paths. Scope to specific files.",
				})
			}
		}

		// Destructive container cleanup operations
		containerDestructivePatterns := []string{
			"docker system prune", "docker rm -f", "docker rmi -f",
			"docker image prune -a", "docker volume rm", "docker volume prune",
			"docker network prune",
		}
		for _, pattern := range containerDestructivePatterns {
			if strings.Contains(lower, pattern) {
				findings = append(findings, Finding{
					Severity:       "high",
					Code:           "DESTRUCTIVE_CONTAINER_OP",
					Description:    "Destructive container operation detected",
					Line:           lineNum,
					Recommendation: "Review container cleanup commands; they can delete images, volumes, or networks.",
				})
				break
			}
		}

		// Destructive Kubernetes operations
		if strings.Contains(lower, "kubectl delete") &&
			(strings.Contains(lower, "--all") || strings.Contains(lower, "namespace") ||
				strings.Contains(lower, "--all-namespaces")) {
			findings = append(findings, Finding{
				Severity:       "high",
				Code:           "DESTRUCTIVE_K8S_OP",
				Description:    "Destructive Kubernetes delete operation detected",
				Line:           lineNum,
				Recommendation: "Avoid bulk delete operations; require approval and verify target namespace.",
			})
		}

		// Destructive cloud storage operations
		if (strings.Contains(lower, "aws s3 rm") && strings.Contains(lower, "--recursive")) ||
			(strings.Contains(lower, "gsutil rm") && strings.Contains(lower, "-r")) ||
			strings.Contains(lower, "az storage blob delete-batch") ||
			strings.Contains(lower, "rclone purge") {
			findings = append(findings, Finding{
				Severity:       "high",
				Code:           "DESTRUCTIVE_CLOUD_STORAGE",
				Description:    "Destructive cloud storage operation detected",
				Line:           lineNum,
				Recommendation: "Use dry runs or retention policies before deleting cloud storage.",
			})
		}

		// Destructive infrastructure operations
		if strings.Contains(lower, "terraform destroy") ||
			strings.Contains(lower, "pulumi destroy") ||
			strings.Contains(lower, "helm uninstall") ||
			strings.Contains(lower, "helm delete") {
			findings = append(findings, Finding{
				Severity:       "high",
				Code:           "DESTRUCTIVE_INFRA",
				Description:    "Destructive infrastructure operation detected",
				Line:           lineNum,
				Recommendation: "Require approval before running infrastructure destroy or uninstall commands.",
			})
		}

		// Destructive package removal operations
		packageRemovalPatterns := []string{
			"apt-get remove", "apt remove", "apt-get purge", "apt purge",
			"yum remove", "dnf remove", "pacman -r", "apk del",
		}
		for _, pattern := range packageRemovalPatterns {
			if strings.Contains(lower, pattern) {
				findings = append(findings, Finding{
					Severity:       "high",
					Code:           "DESTRUCTIVE_PACKAGE_REMOVAL",
					Description:    "Package removal operation detected",
					Line:           lineNum,
					Recommendation: "Ensure package removal is intended and scoped. Use dry runs where possible.",
				})
				break
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
		if (strings.Contains(lower, "curl ") || strings.Contains(lower, "wget ")) &&
			(strings.Contains(lower, "http://") || strings.Contains(lower, "https://")) {
			isScriptDownload := strings.Contains(lower, ".sh") ||
				strings.Contains(lower, ".bash") ||
				strings.Contains(lower, ".ps1") ||
				strings.Contains(lower, "install.sh") ||
				strings.Contains(lower, "script.sh")
			outputsToStdout := strings.Contains(lower, "-o-") ||
				strings.Contains(lower, "-o -") ||
				strings.Contains(lower, "--output -") ||
				strings.Contains(lower, "-o /dev/stdout") ||
				strings.Contains(lower, "-o /proc/self/fd/1") ||
				strings.Contains(lower, "-o /dev/fd/1") ||
				strings.Contains(lower, "-O-")
			if isScriptDownload || outputsToStdout {
				findings = append(findings, Finding{
					Severity:       "high",
					Code:           "NETWORK_SCRIPT_DOWNLOAD",
					Description:    "Remote script download detected",
					Line:           lineNum,
					Recommendation: "Download to disk, verify checksum, and review before execution.",
				})
			}
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
		// Reverse shell detection - multiple patterns
		isReverseShell := false
		
		// Pattern 1: socket.socket + dup2 + shell
		if strings.Contains(lower, "socket.socket") && strings.Contains(lower, "dup2") &&
			(strings.Contains(lower, "/bin/sh") || strings.Contains(lower, "/bin/bash")) {
			isReverseShell = true
		}
		
		// Pattern 2: nc/netcat with -e flag and shell
		if (strings.Contains(lower, "nc ") || strings.Contains(lower, "netcat ")) && 
			strings.Contains(lower, " -e ") &&
			(strings.Contains(lower, "/bin/sh") || strings.Contains(lower, "/bin/bash")) {
			isReverseShell = true
		}
		
		// Pattern 3: bash/dev/tcp reverse shell
		if strings.Contains(lower, "/dev/tcp/") && strings.Contains(lower, "bash -i") {
			isReverseShell = true
		}
		
		// Pattern 4: /bin/sh -i or /bin/bash -i (interactive shell, often used in reverse shells)
		// Especially when combined with subprocess or socket operations
		if (strings.Contains(lower, "/bin/sh") || strings.Contains(lower, "/bin/bash")) &&
			strings.Contains(lower, "-i") &&
			(strings.Contains(lower, "subprocess") || strings.Contains(lower, "socket") || 
			 strings.Contains(lower, "dup2") || strings.Contains(lower, "connect")) {
			isReverseShell = true
		}
		
		if isReverseShell {
			findings = append(findings, Finding{
				Severity:       "critical",
				Code:           "REVERSE_SHELL",
				Description:    "Reverse shell pattern detected",
				Line:           lineNum,
				Recommendation: "BLOCK this command. Reverse shells allow remote code execution.",
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
				"git push --force":    {"high", "Force push detected - can overwrite remote history", "Use --force-with-lease instead or coordinate with team before force pushing."},
				"git push -f":         {"high", "Force push detected - can overwrite remote history", "Use --force-with-lease instead or coordinate with team before force pushing."},
				"git reset --hard":    {"medium", "Hard reset detected - will discard local changes", "Ensure you have backups or stash important changes first."},
				"git clean -fd":       {"medium", "Git clean with force - will delete untracked files", "Review untracked files before cleaning. Consider using -n flag first for dry run."},
				"git branch -D":       {"medium", "Force branch deletion detected", "Ensure branch is merged or no longer needed before force deleting."},
				"git branch -d":       {"low", "Branch deletion detected", "Verify branch is fully merged before deletion."},
				"git rebase":          {"low", "Git rebase detected - will rewrite commit history", "Only rebase local commits. Never rebase published commits."},
				"git filter-branch":   {"high", "Git filter-branch - rewrites entire repository history", "Extremely dangerous. Coordinate with entire team and backup repository first."},
				"git filter-repo":     {"high", "Git filter-repo - rewrites entire repository history", "Extremely dangerous. Coordinate with entire team and backup repository first."},
				"git reflog expire":   {"high", "Reflog expiration - will permanently delete commit references", "Only use if you know what you're doing. Lost commits cannot be recovered."},
				"git gc --aggressive": {"medium", "Aggressive garbage collection - may make recovery difficult", "Ensure no important dangling commits exist before running."},
				"git update-ref -d":   {"high", "Direct ref manipulation detected", "Advanced operation. Ensure you understand git internals before proceeding."},
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
				"dropdatabase", "db.dropdatabase", "db.dropdatabase()",
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

// analyzeExtractedCommand analyzes a command extracted from Python code
func analyzeExtractedCommand(cmd string, lineNum int, policy config.PolicyConfig) []Finding {
	// Create a single-line "script" for analysis
	content := []byte(cmd)
	// Use a special path to indicate this is an extracted command
	extractedFindings := AnalyzeScript("extracted-from-python", content, policy)
	
	// Update line numbers to point to the original Python command
	for i := range extractedFindings {
		extractedFindings[i].Line = lineNum
		extractedFindings[i].Description = fmt.Sprintf("Extracted from Python code: %s", extractedFindings[i].Description)
	}
	
	return extractedFindings
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

func containsSystemPath(fields []string) bool {
	systemPrefixes := []string{
		"/", "/etc", "/usr", "/bin", "/sbin", "/lib", "/lib64", "/var", "/opt",
		"/boot", "/root", "/home", "/usr/local", "/var/log", "/srv",
		"/system", "/library", "/applications",
	}
	for _, field := range fields {
		for _, prefix := range systemPrefixes {
			if prefix == "/" {
				if field == "/" {
					return true
				}
				continue
			}
			if field == prefix || strings.HasPrefix(field, prefix+"/") {
				return true
			}
		}
	}
	return false
}
