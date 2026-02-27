package seed

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed templates/**
var templates embed.FS

type Result struct {
	Path   string
	Status string
}

type Target struct {
	Key      string
	DestPath string
	SrcPath  string
}

func AvailableTargets() map[string]Target {
	return map[string]Target{
		"agents": {
			Key:      "agents",
			DestPath: "AGENTS.md",
			SrcPath:  "templates/AGENTS.md",
		},
		"claude": {
			Key:      "claude",
			DestPath: "CLAUDE.md",
			SrcPath:  "templates/CLAUDE.md",
		},
		"codex": {
			Key:      "codex",
			DestPath: "CODEX.md",
			SrcPath:  "templates/CODEX.md",
		},
		"copilot": {
			Key:      "copilot",
			DestPath: ".github/copilot-instructions.md",
			SrcPath:  "templates/.github/copilot-instructions.md",
		},
		"cursor": {
			Key:      "cursor",
			DestPath: ".cursor/rules/vectra-guard.md",
			SrcPath:  "templates/.cursor/rules/vectra-guard.md",
		},
		"windsurf": {
			Key:      "windsurf",
			DestPath: ".windsurf/rules.md",
			SrcPath:  "templates/.windsurf/rules.md",
		},
		"vscode": {
			Key:      "vscode",
			DestPath: ".vscode/vectra-guard.instructions.md",
			SrcPath:  "templates/.vscode/vectra-guard.instructions.md",
		},
		"openclaw": {
			Key:      "openclaw",
			DestPath: ".openclaw/plugins/vectraguard.yaml",
			SrcPath:  "templates/.openclaw/plugins/vectraguard.yaml",
		},
	}
}

func ResolveTargets(selected []string) ([]Target, error) {
	available := AvailableTargets()
	if len(selected) == 0 {
		return []Target{available["agents"]}, nil
	}

	var targets []Target
	for _, key := range selected {
		t, ok := available[key]
		if !ok {
			return nil, fmt.Errorf("unknown seed target: %s", key)
		}
		targets = append(targets, t)
	}
	return targets, nil
}

func WriteAgentInstructions(targetDir string, force bool, selected []string) ([]Result, error) {
	if targetDir == "" {
		targetDir = "."
	}

	targets, err := ResolveTargets(selected)
	if err != nil {
		return nil, err
	}

	var results []Result
	for _, target := range targets {
		dst := filepath.Join(targetDir, target.DestPath)
		if !force {
			if _, err := os.Stat(dst); err == nil {
				results = append(results, Result{Path: dst, Status: "skipped"})
				continue
			}
		}

		payload, err := fs.ReadFile(templates, target.SrcPath)
		if err != nil {
			return results, fmt.Errorf("read template %s: %w", target.SrcPath, err)
		}

		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return results, fmt.Errorf("create directory for %s: %w", dst, err)
		}
		if err := os.WriteFile(dst, payload, 0o644); err != nil {
			return results, fmt.Errorf("write %s: %w", dst, err)
		}
		results = append(results, Result{Path: dst, Status: "written"})
	}

	return results, nil
}
