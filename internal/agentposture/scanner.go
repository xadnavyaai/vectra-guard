package agentposture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/seed"
)

// agentDef describes a known AI agent integration and how to detect it.
type agentDef struct {
	Name      string
	Paths     []string // Directories and files to look for
	RiskLevel string   // Default risk level when found
	Details   string
}

var knownAgents = []agentDef{
	{
		Name:      "claude",
		Paths:     []string{".claude/", "CLAUDE.md", ".claude.md"},
		RiskLevel: "low",
		Details:   "Claude Code agent integration",
	},
	{
		Name:      "cursor",
		Paths:     []string{".cursor/", ".cursorrules", ".cursorignore"},
		RiskLevel: "medium",
		Details:   "Cursor AI editor agent",
	},
	{
		Name:      "copilot",
		Paths:     []string{".github/copilot/", ".copilotignore"},
		RiskLevel: "low",
		Details:   "GitHub Copilot integration",
	},
	{
		Name:      "continue",
		Paths:     []string{".continue/", ".continuerc.json"},
		RiskLevel: "medium",
		Details:   "Continue.dev agent integration",
	},
	{
		Name:      "windsurf",
		Paths:     []string{".windsurf/", ".windsurfrules"},
		RiskLevel: "medium",
		Details:   "Windsurf AI editor agent",
	},
	{
		Name:      "aider",
		Paths:     []string{".aider.conf.yml", ".aider/"},
		RiskLevel: "medium",
		Details:   "Aider coding assistant",
	},
}

// packageAgentDef describes agents detected via package manifests.
type packageAgentDef struct {
	Name      string
	Packages  []string // Dependency names to match
	RiskLevel string
	Details   string
}

var packageAgents = []packageAgentDef{
	{
		Name:      "vercel-ai",
		Packages:  []string{"ai", "@ai-sdk/", "@vercel/ai"},
		RiskLevel: "high",
		Details:   "Vercel AI SDK (executes agent actions with OAuth scopes)",
	},
	{
		Name:      "openai-sdk",
		Packages:  []string{"openai"},
		RiskLevel: "medium",
		Details:   "OpenAI SDK (API key-based access)",
	},
	{
		Name:      "anthropic-sdk",
		Packages:  []string{"anthropic"},
		RiskLevel: "medium",
		Details:   "Anthropic SDK (API key-based access)",
	},
}

// ScanAgents walks the workspace looking for AI agent configuration files
// and package manifest dependencies.
func ScanAgents(targetPath string) []AgentFinding {
	var findings []AgentFinding

	// 1. Filesystem-based agent detection
	for _, agent := range knownAgents {
		var found []string
		for _, p := range agent.Paths {
			full := filepath.Join(targetPath, p)
			if info, err := os.Stat(full); err == nil {
				if info.IsDir() || info.Mode().IsRegular() {
					found = append(found, p)
				}
			}
		}
		if len(found) > 0 {
			seeded := checkSeeded(targetPath, agent.Name)
			risk := agent.RiskLevel
			detail := agent.Details
			if !seeded {
				detail += " (no VectraGuard seed)"
				if risk == "low" {
					risk = "medium"
				}
			}
			findings = append(findings, AgentFinding{
				Name:        agent.Name,
				ConfigPaths: found,
				RiskLevel:   risk,
				Details:     detail,
				Seeded:      seeded,
			})
		}
	}

	// 2. VS Code AI extension detection
	if vscodeFindings := scanVSCodeAI(targetPath); vscodeFindings != nil {
		findings = append(findings, *vscodeFindings)
	}

	// 3. Package manifest-based agent detection
	findings = append(findings, scanPackageManifests(targetPath)...)

	return findings
}

// checkSeeded returns true if VectraGuard markers are present for the given agent.
func checkSeeded(targetPath, agentName string) bool {
	// Check for VectraGuard markers in the agent's primary config file
	markerFiles := map[string][]string{
		"claude":   {"CLAUDE.md", ".claude.md"},
		"cursor":   {".cursor/rules/vectra-guard.md", ".cursorrules"},
		"copilot":  {".github/copilot-instructions.md"},
		"windsurf": {".windsurf/rules.md"},
	}

	paths, ok := markerFiles[agentName]
	if !ok {
		// Also check the generic AGENTS.md for any agent
		paths = []string{"AGENTS.md"}
	}

	for _, p := range paths {
		full := filepath.Join(targetPath, p)
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		content := string(data)
		if strings.Contains(content, seed.MarkerBegin) ||
			strings.Contains(content, "vectra-guard") ||
			strings.Contains(content, "vectraguard") {
			return true
		}
	}
	return false
}

// scanVSCodeAI checks .vscode/settings.json for AI extension keys.
func scanVSCodeAI(targetPath string) *AgentFinding {
	settingsPath := filepath.Join(targetPath, ".vscode", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return nil
	}

	content := strings.ToLower(string(data))
	aiKeys := []string{
		"github.copilot",
		"codeium",
		"tabnine",
		"continue",
		"cursor",
		"sourcegraph.cody",
	}

	var found []string
	for _, key := range aiKeys {
		if strings.Contains(content, key) {
			found = append(found, key)
		}
	}

	if len(found) == 0 {
		return nil
	}

	return &AgentFinding{
		Name:        "vscode-ai",
		ConfigPaths: []string{".vscode/settings.json"},
		RiskLevel:   "low",
		Details:     "VS Code AI extensions: " + strings.Join(found, ", "),
		Seeded:      false,
	}
}

// scanPackageManifests checks package.json and requirements.txt for AI SDK dependencies.
func scanPackageManifests(targetPath string) []AgentFinding {
	var findings []AgentFinding

	// Check package.json
	pkgPath := filepath.Join(targetPath, "package.json")
	if data, err := os.ReadFile(pkgPath); err == nil {
		findings = append(findings, scanNodeDeps(data)...)
	}

	// Check requirements.txt
	reqPath := filepath.Join(targetPath, "requirements.txt")
	if data, err := os.ReadFile(reqPath); err == nil {
		findings = append(findings, scanPythonDeps(data)...)
	}

	return findings
}

// scanNodeDeps parses package.json and checks dependencies + devDependencies.
func scanNodeDeps(data []byte) []AgentFinding {
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

	var findings []AgentFinding
	for _, agent := range packageAgents {
		for depName := range allDeps {
			for _, pattern := range agent.Packages {
				if depName == pattern || strings.HasPrefix(depName, pattern) {
					findings = append(findings, AgentFinding{
						Name:        agent.Name,
						ConfigPaths: []string{"package.json (" + depName + ")"},
						RiskLevel:   agent.RiskLevel,
						Details:     agent.Details,
						Seeded:      false,
					})
					goto nextAgent
				}
			}
		}
	nextAgent:
	}

	return findings
}

// scanPythonDeps checks requirements.txt lines for known AI SDK packages.
func scanPythonDeps(data []byte) []AgentFinding {
	lines := strings.Split(string(data), "\n")
	depNames := make(map[string]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip version specifiers: "openai>=1.0" → "openai"
		for _, sep := range []string{">=", "<=", "==", "!=", "~=", ">"} {
			if idx := strings.Index(line, sep); idx >= 0 {
				line = line[:idx]
			}
		}
		depNames[strings.TrimSpace(line)] = true
	}

	var findings []AgentFinding
	for _, agent := range packageAgents {
		for _, pattern := range agent.Packages {
			if depNames[pattern] {
				findings = append(findings, AgentFinding{
					Name:        agent.Name,
					ConfigPaths: []string{"requirements.txt (" + pattern + ")"},
					RiskLevel:   agent.RiskLevel,
					Details:     agent.Details,
					Seeded:      false,
				})
				break
			}
		}
	}

	return findings
}
