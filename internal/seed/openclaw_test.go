package seed

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectOpenClaw_EnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "custom-openclaw")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OPENCLAW_STATE_DIR", stateDir)
	// Clear legacy env to avoid interference
	t.Setenv("CLAWDBOT_STATE_DIR", "")

	det := DetectOpenClaw()

	if !det.Installed {
		t.Fatal("expected Installed to be true")
	}
	if det.StateDir != stateDir {
		t.Errorf("expected StateDir %q, got %q", stateDir, det.StateDir)
	}
	if det.Source != "env:OPENCLAW_STATE_DIR" {
		t.Errorf("expected Source 'env:OPENCLAW_STATE_DIR', got %q", det.Source)
	}
}

func TestDetectOpenClaw_LegacyEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "clawdbot-state")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OPENCLAW_STATE_DIR", "")
	t.Setenv("CLAWDBOT_STATE_DIR", stateDir)

	det := DetectOpenClaw()

	if !det.Installed {
		t.Fatal("expected Installed to be true")
	}
	if det.Source != "env:CLAWDBOT_STATE_DIR" {
		t.Errorf("expected Source 'env:CLAWDBOT_STATE_DIR', got %q", det.Source)
	}
}

func TestDetectOpenClaw_NotInstalled(t *testing.T) {
	// Point env vars to non-existent paths
	t.Setenv("OPENCLAW_STATE_DIR", "/nonexistent/path/openclaw-test")
	t.Setenv("CLAWDBOT_STATE_DIR", "/nonexistent/path/clawdbot-test")

	// Note: This test may still find ~/.openclaw if it exists on the test machine.
	// That's acceptable — we're primarily testing the env var path.
	det := DetectOpenClaw()

	// If the default ~/.openclaw doesn't exist, it should report not installed
	if det.Installed && det.Source != "default" && !isLegacySource(det.Source) {
		t.Errorf("unexpected detection: Source=%q StateDir=%q", det.Source, det.StateDir)
	}
}

func isLegacySource(s string) bool {
	return s == "legacy:clawdbot" || s == "legacy:moldbot" || s == "legacy:moltbot" || s == "default"
}

func TestDetectOpenClaw_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "openclaw-with-config")
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create openclaw.json with agent entries
	cfg := openClawConfig{
		Agents: []struct {
			ID        string `json:"id"`
			Workspace string `json:"workspace"`
		}{
			{ID: "agent-1", Workspace: "/tmp/workspace-1"},
			{ID: "agent-2", Workspace: "/tmp/workspace-2"},
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "openclaw.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OPENCLAW_STATE_DIR", stateDir)
	t.Setenv("CLAWDBOT_STATE_DIR", "")

	det := DetectOpenClaw()

	if !det.Installed {
		t.Fatal("expected Installed to be true")
	}
	if !det.ConfigExists {
		t.Error("expected ConfigExists to be true")
	}
	if det.ConfigFile != filepath.Join(stateDir, "openclaw.json") {
		t.Errorf("expected ConfigFile %q, got %q",
			filepath.Join(stateDir, "openclaw.json"), det.ConfigFile)
	}
	if len(det.AgentDirs) < 2 {
		t.Errorf("expected at least 2 agent dirs, got %d", len(det.AgentDirs))
	}
}

func TestDetectOpenClaw_WithAgentSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	stateDir := filepath.Join(tmpDir, "openclaw-agents")
	agentDir := filepath.Join(stateDir, "agents", "my-agent", "agent")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create minimal openclaw.json (no agents array — dirs come from scanning)
	if err := os.WriteFile(filepath.Join(stateDir, "openclaw.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("OPENCLAW_STATE_DIR", stateDir)
	t.Setenv("CLAWDBOT_STATE_DIR", "")

	det := DetectOpenClaw()

	if !det.Installed {
		t.Fatal("expected Installed to be true")
	}

	found := false
	for _, d := range det.AgentDirs {
		if d == agentDir {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected agent dir %q in AgentDirs %v", agentDir, det.AgentDirs)
	}
}

func TestOpenClawSeedPaths_Installed(t *testing.T) {
	stateDir := filepath.Join(string(filepath.Separator), "home", "user", ".openclaw")
	ws1 := filepath.Join(string(filepath.Separator), "tmp", "workspace-1")
	ws2 := filepath.Join(string(filepath.Separator), "tmp", "workspace-2")

	det := OpenClawDetection{
		StateDir:  stateDir,
		Installed: true,
		AgentDirs: []string{ws1, ws2},
	}

	paths := OpenClawSeedPaths(det)

	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d: %v", len(paths), paths)
	}
	if paths[0] != filepath.Join(stateDir, "AGENTS.md") {
		t.Errorf("expected primary path, got %q", paths[0])
	}
	if paths[1] != filepath.Join(ws1, "AGENTS.md") {
		t.Errorf("expected workspace-1 path, got %q", paths[1])
	}
	if paths[2] != filepath.Join(ws2, "AGENTS.md") {
		t.Errorf("expected workspace-2 path, got %q", paths[2])
	}
}

func TestOpenClawSeedPaths_NotInstalled(t *testing.T) {
	det := OpenClawDetection{
		Installed: false,
	}

	paths := OpenClawSeedPaths(det)

	if paths != nil {
		t.Errorf("expected nil paths for not-installed, got %v", paths)
	}
}

func TestOpenClawSeedPaths_Dedup(t *testing.T) {
	det := OpenClawDetection{
		StateDir:  "/home/user/.openclaw",
		Installed: true,
		// Duplicate: same as primary path's dir
		AgentDirs: []string{"/home/user/.openclaw"},
	}

	paths := OpenClawSeedPaths(det)

	// Should deduplicate: /home/user/.openclaw/AGENTS.md appears only once
	if len(paths) != 1 {
		t.Errorf("expected 1 deduplicated path, got %d: %v", len(paths), paths)
	}
}
