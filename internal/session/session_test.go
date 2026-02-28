package session

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func setTestHome(t *testing.T) {
	t.Helper()
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tempHome)
	}
}

func TestSessionLifecycle(t *testing.T) {
	setTestHome(t)
	tmpDir := t.TempDir()
	logger := logging.NewLogger("text", io.Discard)

	mgr, err := NewManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Start a session
	session, err := mgr.Start("test-agent", tmpDir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if session.ID == "" {
		t.Error("Session ID should not be empty")
	}
	if session.AgentName != "test-agent" {
		t.Errorf("Expected agent name 'test-agent', got '%s'", session.AgentName)
	}

	// Add a command
	cmd := Command{
		Timestamp: time.Now(),
		Command:   "echo",
		Args:      []string{"hello"},
		ExitCode:  0,
		RiskLevel: "low",
		Approved:  true,
	}
	if err := mgr.AddCommand(session, cmd); err != nil {
		t.Fatalf("AddCommand failed: %v", err)
	}

	// Load the session
	loaded, err := mgr.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(loaded.Commands))
	}

	// End the session
	if err := mgr.End(session); err != nil {
		t.Fatalf("End failed: %v", err)
	}

	if session.EndTime == nil {
		t.Error("EndTime should be set after ending session")
	}
}

func TestRiskScoring(t *testing.T) {
	setTestHome(t)
	tmpDir := t.TempDir()
	logger := logging.NewLogger("text", io.Discard)

	mgr, err := NewManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session, err := mgr.Start("test-agent", tmpDir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Add a critical command
	criticalCmd := Command{
		Timestamp: time.Now(),
		Command:   "rm",
		Args:      []string{"-rf", "/"},
		RiskLevel: "critical",
	}
	if err := mgr.AddCommand(session, criticalCmd); err != nil {
		t.Fatalf("AddCommand failed: %v", err)
	}

	if session.RiskScore != 100 {
		t.Errorf("Expected risk score 100, got %d", session.RiskScore)
	}
	if session.Violations != 1 {
		t.Errorf("Expected 1 violation, got %d", session.Violations)
	}

	// Add a high-risk command
	highCmd := Command{
		Timestamp: time.Now(),
		Command:   "sudo",
		Args:      []string{"cat", "/etc/shadow"},
		RiskLevel: "high",
	}
	if err := mgr.AddCommand(session, highCmd); err != nil {
		t.Fatalf("AddCommand failed: %v", err)
	}

	if session.RiskScore != 150 {
		t.Errorf("Expected risk score 150, got %d", session.RiskScore)
	}
}

func TestFileOperations(t *testing.T) {
	setTestHome(t)
	tmpDir := t.TempDir()
	logger := logging.NewLogger("text", io.Discard)

	mgr, err := NewManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session, err := mgr.Start("test-agent", tmpDir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Add allowed file operation
	allowedOp := FileOperation{
		Timestamp: time.Now(),
		Operation: "create",
		Path:      "/tmp/test.txt",
		RiskLevel: "low",
		Allowed:   true,
	}
	if err := mgr.AddFileOperation(session, allowedOp); err != nil {
		t.Fatalf("AddFileOperation failed: %v", err)
	}

	// Add blocked file operation
	blockedOp := FileOperation{
		Timestamp: time.Now(),
		Operation: "modify",
		Path:      "/etc/passwd",
		RiskLevel: "critical",
		Allowed:   false,
		Reason:    "Protected system file",
	}
	if err := mgr.AddFileOperation(session, blockedOp); err != nil {
		t.Fatalf("AddFileOperation failed: %v", err)
	}

	if len(session.FileOps) != 2 {
		t.Errorf("Expected 2 file operations, got %d", len(session.FileOps))
	}
	if session.Violations != 1 {
		t.Errorf("Expected 1 violation from blocked operation, got %d", session.Violations)
	}
}

func TestListSessions(t *testing.T) {
	setTestHome(t)
	tmpDir := t.TempDir()
	logger := logging.NewLogger("text", io.Discard)

	mgr, err := NewManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create multiple sessions
	session1, err := mgr.Start("agent1", tmpDir)
	if err != nil {
		t.Fatalf("Start session1 failed: %v", err)
	}
	session2, err := mgr.Start("agent2", tmpDir)
	if err != nil {
		t.Fatalf("Start session2 failed: %v", err)
	}

	sessions, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected exactly 2 sessions, got %d", len(sessions))
	}

	// Verify our session IDs are in the list
	ids := make(map[string]bool)
	for _, s := range sessions {
		ids[s.ID] = true
	}
	if !ids[session1.ID] {
		t.Error("session1 ID not found in list")
	}
	if !ids[session2.ID] {
		t.Error("session2 ID not found in list")
	}
}

func TestSessionPersistence(t *testing.T) {
	setTestHome(t)
	tmpDir := t.TempDir()
	logger := logging.NewLogger("text", io.Discard)

	mgr, err := NewManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	session, err := mgr.Start("test-agent", tmpDir)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// Note: Session files are stored in home directory, not tmpDir
	// The sessionDir is set in NewManager to ~/.vectra-guard/sessions
	// We just verify that we can load the session back

	// Create new manager and load session
	mgr2, err := NewManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	loaded, err := mgr2.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.ID != session.ID {
		t.Errorf("Loaded session ID mismatch: expected %s, got %s", session.ID, loaded.ID)
	}
}

func TestWorkspaceSessionIndex(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	logger := logging.NewLogger("text", io.Discard)

	workspace1 := t.TempDir()
	mgr, err := NewManager(workspace1, logger)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	sess1, err := mgr.Start("agent1", workspace1)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	SetCurrentSessionForWorkspace(workspace1, sess1.ID)
	_ = os.Unsetenv("VECTRAGUARD_SESSION_ID")

	if got := GetCurrentSessionForWorkspace(workspace1); got != sess1.ID {
		t.Fatalf("expected session %s, got %s", sess1.ID, got)
	}

	workspace2 := t.TempDir()
	sess2, err := mgr.Start("agent2", workspace2)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	SetCurrentSessionForWorkspace(workspace2, sess2.ID)
	os.Setenv("VECTRAGUARD_SESSION_ID", sess1.ID)

	if got := GetCurrentSessionForWorkspace(workspace2); got != sess2.ID {
		t.Fatalf("expected workspace2 session %s, got %s", sess2.ID, got)
	}
}

func TestWorkspaceSessionIndexHonorsEnvMismatch(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	logger := logging.NewLogger("text", io.Discard)

	workspace1 := t.TempDir()
	mgr, err := NewManager(workspace1, logger)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	sess1, err := mgr.Start("agent1", workspace1)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	workspace2 := t.TempDir()
	sess2, err := mgr.Start("agent2", workspace2)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	SetCurrentSessionForWorkspace(workspace2, sess2.ID)
	t.Setenv("VECTRAGUARD_SESSION_ID", sess1.ID)

	if got := GetCurrentSessionForWorkspace(workspace2); got != sess2.ID {
		t.Fatalf("expected workspace2 session %s, got %s", sess2.ID, got)
	}
}

func TestWorkspaceSessionIndexFallbackGlobalWhenNoWorkdir(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	path := globalSessionFilePath()
	if path == "" {
		t.Fatal("expected global session path")
	}
	if err := os.WriteFile(path, []byte("session-fallback\n"), 0o644); err != nil {
		t.Fatalf("write fallback file: %v", err)
	}
	_ = os.Unsetenv("VECTRAGUARD_SESSION_ID")

	if got := GetCurrentSessionForWorkspace(""); got != "session-fallback" {
		t.Fatalf("expected fallback session, got %s", got)
	}
}

func TestNormalizeWorkspacePath(t *testing.T) {
	base := t.TempDir()
	nested := base + string(os.PathSeparator) + ".." + string(os.PathSeparator) + filepath.Base(base)
	if got := normalizeWorkspacePath(nested); got != normalizeWorkspacePath(base) {
		t.Fatalf("expected normalized paths to match, got %s vs %s", got, base)
	}
}
