package agentposture

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// oauthPattern describes a regex-free pattern match for OAuth/API scope issues.
type oauthPattern struct {
	// Match functions: at least one must return true for the pattern to fire.
	MatchLine   func(line string) bool
	Type        string
	Severity    string
	Detail      string
	Remediation string
}

var oauthPatterns = []oauthPattern{
	{
		MatchLine: func(line string) bool {
			lower := strings.ToLower(line)
			return (strings.Contains(lower, "_scope=") || strings.Contains(lower, "_scope =")) &&
				(strings.Contains(lower, "all") || strings.Contains(lower, "*") || strings.Contains(lower, "admin"))
		},
		Type:        "broad_scope",
		Severity:    "critical",
		Detail:      "OAuth/API scope contains broad access (all/*/admin)",
		Remediation: "Restrict scope to minimum required permissions",
	},
	{
		MatchLine: func(line string) bool {
			lower := strings.ToLower(line)
			return (strings.Contains(lower, "allow_all") || strings.Contains(lower, "\"permissions\":\"all\"") ||
				strings.Contains(lower, "'permissions':'all'")) &&
				!strings.HasPrefix(strings.TrimSpace(line), "#") &&
				!strings.HasPrefix(strings.TrimSpace(line), "//")
		},
		Type:        "broad_permissions",
		Severity:    "critical",
		Detail:      "Configuration grants all permissions to an integration",
		Remediation: "Set explicit, minimal permission list instead of 'all'",
	},
	{
		MatchLine: func(line string) bool {
			lower := strings.ToLower(line)
			return (strings.Contains(lower, "_client_secret=") || strings.Contains(lower, "_oauth_secret=") ||
				strings.Contains(lower, "_client_secret =") || strings.Contains(lower, "_oauth_secret ="))
		},
		Type:        "committed_secret",
		Severity:    "high",
		Detail:      "OAuth client secret appears in committed file",
		Remediation: "Move secrets to a secure vault; rotate the exposed secret immediately",
	},
	{
		MatchLine: func(line string) bool {
			lower := strings.ToLower(line)
			if !strings.Contains(lower, "_token=") && !strings.Contains(lower, "_token =") &&
				!strings.Contains(lower, "_api_key=") && !strings.Contains(lower, "_api_key =") &&
				!strings.Contains(lower, "_apikey=") && !strings.Contains(lower, "_apikey =") {
				return false
			}
			// Skip comments
			trimmed := strings.TrimSpace(line)
			return !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "//")
		},
		Type:        "unscoped_token",
		Severity:    "high",
		Detail:      "API token or key found without scope restriction",
		Remediation: "Add explicit scope limits and ensure token is in .gitignore",
	},
	{
		MatchLine: func(line string) bool {
			lower := strings.ToLower(line)
			return strings.Contains(lower, "vercel_token") || strings.Contains(lower, "vercel_api_token")
		},
		Type:        "vercel_token",
		Severity:    "high",
		Detail:      "Vercel token found (broad platform access if unscoped)",
		Remediation: "Use scoped Vercel tokens with minimal permissions; rotate regularly",
	},
}

// ScanOAuth walks the workspace looking for OAuth/API scope issues in
// environment files, config files, and Vercel configs.
func ScanOAuth(targetPath string) []OAuthFinding {
	var findings []OAuthFinding

	// Scan .env* files
	envFiles := findEnvFiles(targetPath)
	for _, f := range envFiles {
		findings = append(findings, scanFileForOAuth(targetPath, f)...)
	}

	// Scan vercel.json
	vercelPath := filepath.Join(targetPath, "vercel.json")
	if _, err := os.Stat(vercelPath); err == nil {
		findings = append(findings, scanFileForOAuth(targetPath, vercelPath)...)
	}

	// Check if .env files are gitignored
	findings = append(findings, checkGitignoreProtection(targetPath, envFiles)...)

	return findings
}

// findEnvFiles discovers .env, .env.local, .env.production, etc.
func findEnvFiles(targetPath string) []string {
	var files []string
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return files
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == ".env" || strings.HasPrefix(name, ".env.") {
			files = append(files, filepath.Join(targetPath, name))
		}
	}
	return files
}

// scanFileForOAuth reads a file line-by-line and checks each line against OAuth patterns.
func scanFileForOAuth(basePath, filePath string) []OAuthFinding {
	f, err := os.Open(filePath)
	if err != nil {
		return nil
	}
	defer f.Close()

	rel, _ := filepath.Rel(basePath, filePath)
	if rel == "" {
		rel = filePath
	}

	var findings []OAuthFinding
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		for _, pat := range oauthPatterns {
			if pat.MatchLine(line) {
				findings = append(findings, OAuthFinding{
					File:        rel,
					Line:        lineNum,
					Type:        pat.Type,
					Severity:    pat.Severity,
					Detail:      pat.Detail,
					Remediation: pat.Remediation,
				})
			}
		}
	}
	return findings
}

// checkGitignoreProtection verifies that .env files are protected by .gitignore.
func checkGitignoreProtection(targetPath string, envFiles []string) []OAuthFinding {
	if len(envFiles) == 0 {
		return nil
	}

	gitignorePath := filepath.Join(targetPath, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		// No .gitignore and env files exist → finding
		var findings []OAuthFinding
		for _, ef := range envFiles {
			rel, _ := filepath.Rel(targetPath, ef)
			if rel == "" {
				rel = ef
			}
			findings = append(findings, OAuthFinding{
				File:        rel,
				Line:        0,
				Type:        "missing_gitignore",
				Severity:    "high",
				Detail:      "No .gitignore found; env file may be committed to version control",
				Remediation: "Create .gitignore and add .env* pattern",
			})
		}
		return findings
	}

	// Check if .env is covered
	content := string(data)
	lines := strings.Split(content, "\n")
	envProtected := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == ".env" || trimmed == ".env*" || trimmed == ".env.*" ||
			trimmed == "*.env" || trimmed == ".env.local" {
			envProtected = true
			break
		}
	}

	if envProtected {
		return nil
	}

	var findings []OAuthFinding
	for _, ef := range envFiles {
		rel, _ := filepath.Rel(targetPath, ef)
		if rel == "" {
			rel = ef
		}
		findings = append(findings, OAuthFinding{
			File:        rel,
			Line:        0,
			Type:        "unprotected_env",
			Severity:    "high",
			Detail:      "Env file not covered by .gitignore",
			Remediation: "Add .env* to .gitignore to prevent committing secrets",
		})
	}
	return findings
}
