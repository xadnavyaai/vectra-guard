package seed

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// OpenClawDetection holds the result of detecting an OpenClaw installation.
type OpenClawDetection struct {
	StateDir     string // e.g. ~/.openclaw
	Source       string // "env:OPENCLAW_STATE_DIR", "default", "legacy:clawdbot", etc.
	ConfigFile   string // path to openclaw.json
	ConfigExists bool
	AgentDirs    []string // per-agent workspace dirs found
	Installed    bool
}

// openClawConfig is the minimal JSON struct for parsing openclaw.json.
type openClawConfig struct {
	Agents []struct {
		ID        string `json:"id"`
		Workspace string `json:"workspace"`
	} `json:"agents"`
}

// DetectOpenClaw checks for an OpenClaw installation.
// Detection order: env vars, default path, legacy fallbacks.
func DetectOpenClaw() OpenClawDetection {
	home, err := os.UserHomeDir()
	if err != nil {
		return OpenClawDetection{}
	}

	type candidate struct {
		dir    string
		source string
	}

	var candidates []candidate

	// 1. OPENCLAW_STATE_DIR env var
	if v := os.Getenv("OPENCLAW_STATE_DIR"); v != "" {
		candidates = append(candidates, candidate{v, "env:OPENCLAW_STATE_DIR"})
	}

	// 2. CLAWDBOT_STATE_DIR env var (legacy)
	if v := os.Getenv("CLAWDBOT_STATE_DIR"); v != "" {
		candidates = append(candidates, candidate{v, "env:CLAWDBOT_STATE_DIR"})
	}

	// 3. Default path
	candidates = append(candidates, candidate{filepath.Join(home, ".openclaw"), "default"})

	// 4. Legacy fallbacks
	candidates = append(candidates,
		candidate{filepath.Join(home, ".clawdbot"), "legacy:clawdbot"},
		candidate{filepath.Join(home, ".moldbot"), "legacy:moldbot"},
		candidate{filepath.Join(home, ".moltbot"), "legacy:moltbot"},
	)

	for _, c := range candidates {
		info, err := os.Stat(c.dir)
		if err != nil || !info.IsDir() {
			continue
		}

		det := OpenClawDetection{
			StateDir:  c.dir,
			Source:    c.source,
			Installed: true,
		}

		// Check for openclaw.json config
		configPath := filepath.Join(c.dir, "openclaw.json")
		if _, err := os.Stat(configPath); err == nil {
			det.ConfigFile = configPath
			det.ConfigExists = true
			det.AgentDirs = parseOpenClawAgentDirs(configPath, c.dir)
		}

		return det
	}

	return OpenClawDetection{}
}

// parseOpenClawAgentDirs reads openclaw.json and scans for per-agent workspace dirs.
func parseOpenClawAgentDirs(configPath, stateDir string) []string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var cfg openClawConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	var dirs []string
	seen := map[string]bool{}

	// Collect workspace dirs from config
	for _, agent := range cfg.Agents {
		if agent.Workspace != "" && !seen[agent.Workspace] {
			seen[agent.Workspace] = true
			dirs = append(dirs, agent.Workspace)
		}
	}

	// Scan {stateDir}/agents/*/agent/ for workspace dirs
	agentsDir := filepath.Join(stateDir, "agents")
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return dirs
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		agentPath := filepath.Join(agentsDir, entry.Name(), "agent")
		if info, err := os.Stat(agentPath); err == nil && info.IsDir() {
			if !seen[agentPath] {
				seen[agentPath] = true
				dirs = append(dirs, agentPath)
			}
		}
	}

	return dirs
}

// OpenClawSeedPaths returns deduplicated candidate AGENTS.md paths for seeding.
// The primary path is {stateDir}/AGENTS.md. Additional paths come from agent workspace dirs.
func OpenClawSeedPaths(det OpenClawDetection) []string {
	if !det.Installed {
		return nil
	}

	seen := map[string]bool{}
	var paths []string

	// Primary: state dir AGENTS.md
	primary := filepath.Join(det.StateDir, "AGENTS.md")
	seen[primary] = true
	paths = append(paths, primary)

	// Per-agent workspace dirs
	for _, dir := range det.AgentDirs {
		p := filepath.Join(dir, "AGENTS.md")
		if !seen[p] {
			seen[p] = true
			paths = append(paths, p)
		}
	}

	return paths
}
