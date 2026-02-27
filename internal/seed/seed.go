package seed

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
)

const (
	MarkerBegin = "<!-- vectraguard:begin -->"
	MarkerEnd   = "<!-- vectraguard:end -->"
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
			DestPath: ".openclaw/AGENTS.md",
			SrcPath:  "templates/.openclaw/AGENTS.md",
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
		// Skip openclaw — handled separately via WriteOpenClawInstructions
		if target.Key == "openclaw" {
			continue
		}

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

// WorkspaceContext holds detected information about the target workspace.
type WorkspaceContext struct {
	AbsPath      string
	IsGitRepo    bool
	RepoName     string
	Branch       string
	ProjectTypes []string
}

// AgentFileStatus describes the state of a single agent instruction file.
type AgentFileStatus struct {
	Key      string
	DestPath string
	Exists   bool
	Size     int64
	ModTime  time.Time
}

// DetectWorkspaceContext resolves the absolute path and detects git/project info.
func DetectWorkspaceContext(workdir string) WorkspaceContext {
	abs, err := filepath.Abs(workdir)
	if err != nil {
		abs = workdir
	}

	wc := WorkspaceContext{
		AbsPath:   abs,
		IsGitRepo: config.IsGitRepo(abs),
	}

	if wc.IsGitRepo {
		remote := config.GetGitRemoteURL(abs)
		wc.RepoName = config.ParseRepoName(remote)
		wc.Branch = config.GetCurrentGitBranch(abs)
	}

	wc.ProjectTypes = detectProjectTypes(abs)
	return wc
}

// detectProjectTypes checks for common project manifest files.
func detectProjectTypes(workdir string) []string {
	indicators := []struct {
		file  string
		label string
	}{
		{"go.mod", "Go"},
		{"package.json", "Node.js"},
		{"requirements.txt", "Python"},
		{"Pipfile", "Python"},
		{"pyproject.toml", "Python"},
		{"Cargo.toml", "Rust"},
		{"pom.xml", "Java"},
		{"build.gradle", "Java"},
		{"Gemfile", "Ruby"},
		{"Makefile", "Make"},
		{"Dockerfile", "Docker"},
		{"docker-compose.yml", "Docker"},
		{"docker-compose.yaml", "Docker"},
	}

	seen := map[string]bool{}
	var types []string
	for _, ind := range indicators {
		if _, err := os.Stat(filepath.Join(workdir, ind.file)); err == nil {
			if !seen[ind.label] {
				seen[ind.label] = true
				types = append(types, ind.label)
			}
		}
	}
	return types
}

// GetTemplateContent reads a template file from the embedded FS.
func GetTemplateContent(srcPath string) ([]byte, error) {
	return fs.ReadFile(templates, srcPath)
}

// MergeMarkedSection merges new VectraGuard content into existing file content
// using marker-based detection. Returns the merged content and a status string.
//
// Status values:
//   - "written"  — no existing content, created with markers
//   - "merged"   — existing content without markers, appended marked section
//   - "updated"  — existing content with markers, replaced marked section
func MergeMarkedSection(existing, newContent []byte) ([]byte, string) {
	markedNew := wrapWithMarkers(newContent)

	if len(existing) == 0 {
		return markedNew, "written"
	}

	beginIdx := bytes.Index(existing, []byte(MarkerBegin))
	endIdx := bytes.Index(existing, []byte(MarkerEnd))

	// Both markers found and in correct order → replace
	if beginIdx >= 0 && endIdx >= 0 && endIdx > beginIdx {
		var buf bytes.Buffer
		buf.Write(existing[:beginIdx])
		buf.Write(markedNew)
		buf.Write(existing[endIdx+len(MarkerEnd):])
		return buf.Bytes(), "updated"
	}

	// No valid marker pair → append
	var buf bytes.Buffer
	buf.Write(existing)
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		buf.WriteByte('\n')
	}
	buf.WriteByte('\n')
	buf.Write(markedNew)
	return buf.Bytes(), "merged"
}

// wrapWithMarkers wraps content with VectraGuard begin/end markers.
func wrapWithMarkers(content []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString(MarkerBegin)
	buf.WriteByte('\n')
	buf.Write(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		buf.WriteByte('\n')
	}
	buf.WriteString(MarkerEnd)
	return buf.Bytes()
}

// WriteOpenClawInstructions reads the existing file at destPath (if any),
// merges the VectraGuard template content using markers, and writes the result.
func WriteOpenClawInstructions(destPath string, templateContent []byte) (Result, error) {
	existing, err := os.ReadFile(destPath)
	if err != nil && !os.IsNotExist(err) {
		return Result{}, fmt.Errorf("read existing %s: %w", destPath, err)
	}

	merged, status := MergeMarkedSection(existing, templateContent)

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return Result{}, fmt.Errorf("create directory for %s: %w", destPath, err)
	}
	if err := os.WriteFile(destPath, merged, 0o644); err != nil {
		return Result{}, fmt.Errorf("write %s: %w", destPath, err)
	}

	return Result{Path: destPath, Status: status}, nil
}

// ScanAgentFiles checks all available targets for existence/size/age.
func ScanAgentFiles(workdir string) []AgentFileStatus {
	available := AvailableTargets()
	// Use sorted order for deterministic output
	keys := []string{"agents", "claude", "codex", "copilot", "cursor", "openclaw", "vscode", "windsurf"}

	var statuses []AgentFileStatus
	for _, key := range keys {
		t, ok := available[key]
		if !ok {
			continue
		}
		path := filepath.Join(workdir, t.DestPath)
		st := AgentFileStatus{
			Key:      key,
			DestPath: t.DestPath,
		}
		if info, err := os.Stat(path); err == nil {
			st.Exists = true
			st.Size = info.Size()
			st.ModTime = info.ModTime()
		}
		statuses = append(statuses, st)
	}
	return statuses
}
