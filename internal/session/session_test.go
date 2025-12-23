package session

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/logging"
)

func TestSessionLifecycle(t *testing.T) {
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
	tmpDir := t.TempDir()
	logger := logging.NewLogger("text", io.Discard)
	
	mgr, err := NewManager(tmpDir, logger)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Create multiple sessions
	session1, _ := mgr.Start("agent1", tmpDir)
	session2, _ := mgr.Start("agent2", tmpDir)

	sessions, err := mgr.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(sessions) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(sessions))
	}

	// Verify session IDs match
	ids := make(map[string]bool)
	for _, s := range sessions {
		ids[s.ID] = true
	}
	if !ids[session1.ID] || !ids[session2.ID] {
		t.Error("Session IDs don't match created sessions")
	}
}

func TestSessionPersistence(t *testing.T) {
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

	// Verify session file exists
	sessionPath := filepath.Join(tmpDir, ".vectra-guard", "sessions", session.ID+".json")
	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		t.Errorf("Session file not created at %s", sessionPath)
	}

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

