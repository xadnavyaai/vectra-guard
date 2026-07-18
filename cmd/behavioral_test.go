package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

// setTestHome points HOME (and USERPROFILE on Windows) at a temp dir so
// session.Manager never touches the real user home. Cleanup is automatic
// via t.Setenv / t.TempDir.
func setTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}
	return home
}

func newBehavioralCtx() (context.Context, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", buf))
	return ctx, buf
}

// captureStdout redirects os.Stdout around fn and returns the captured bytes.
func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan struct {
		err error
	}, 1)
	go func() {
		done <- struct{ err error }{fn()}
	}()

	// Wait for fn, then close writer so Read returns.
	result := <-done
	_ = w.Close()
	data, _ := io.ReadAll(r)
	_ = r.Close()
	return string(data), result.err
}

func TestRunBehavioralProfile_RequiresSessionID(t *testing.T) {
	setTestHome(t)
	ctx, _ := newBehavioralCtx()
	err := runBehavioralProfile(ctx, "", "text", "")
	if err == nil {
		t.Fatal("expected error for empty session id")
	}
	if !strings.Contains(err.Error(), "--session") {
		t.Errorf("expected error to mention --session flag, got %v", err)
	}
}

func TestRunBehavioralProfile_UnknownSession(t *testing.T) {
	setTestHome(t)
	ctx, _ := newBehavioralCtx()
	err := runBehavioralProfile(ctx, "does-not-exist", "text", "")
	if err == nil {
		t.Fatal("expected error for unknown session")
	}
}

func TestRunBehavioralProfile_JSONOutput(t *testing.T) {
	setTestHome(t)
	ctx, _ := newBehavioralCtx()

	// Create a real session via the session manager (same workspace the
	// command uses — cwd). Using a sub-workspace avoids polluting cwd.
	workspaceDir := t.TempDir()
	mgr, err := session.NewManager(workspaceDir, logging.NewLogger("text", io.Discard))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	sess, err := mgr.Start("test-agent", workspaceDir)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	now := time.Now().UTC()
	_ = mgr.AddCommand(sess, session.Command{
		Timestamp: now,
		Command:   "git",
		Args:      []string{"status"},
		RiskLevel: "low",
	})
	_ = mgr.AddFileOperation(sess, session.FileOperation{
		Timestamp: now.Add(time.Second),
		Operation: "read",
		Path:      "README.md",
		RiskLevel: "low",
	})

	// Run the CLI handler from the workspace dir (so cwd matches).
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(workspaceDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	out, err := captureStdout(t, func() error {
		return runBehavioralProfile(ctx, sess.ID, "json", "")
	})
	if err != nil {
		t.Fatalf("runBehavioralProfile: %v", err)
	}

	// Output should be valid JSON with a session_id field.
	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nout=%s", err, out)
	}
	if parsed["session_id"] != sess.ID {
		t.Errorf("session_id = %v, want %s", parsed["session_id"], sess.ID)
	}
	if _, ok := parsed["nodes"]; !ok {
		t.Error("expected nodes field in JSON output")
	}
}

func TestRunBehavioralProfile_TextOutput(t *testing.T) {
	setTestHome(t)
	ctx, _ := newBehavioralCtx()

	workspaceDir := t.TempDir()
	mgr, err := session.NewManager(workspaceDir, logging.NewLogger("text", io.Discard))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	sess, err := mgr.Start("test-agent", workspaceDir)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	_ = mgr.AddCommand(sess, session.Command{
		Timestamp: time.Now().UTC(),
		Command:   "curl",
		Args:      []string{"https://example.com"},
	})

	origWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(workspaceDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	out, err := captureStdout(t, func() error {
		return runBehavioralProfile(ctx, sess.ID, "text", "")
	})
	if err != nil {
		t.Fatalf("runBehavioralProfile: %v", err)
	}
	if !strings.Contains(out, sess.ID) {
		t.Errorf("expected session ID %s in text output: %s", sess.ID, out)
	}
	if !strings.Contains(out, "CATEGORY") {
		t.Errorf("expected CATEGORY header in text output: %s", out)
	}
}

func TestRunBehavioralProfile_EmptySessionText(t *testing.T) {
	setTestHome(t)
	ctx, _ := newBehavioralCtx()

	workspaceDir := t.TempDir()
	mgr, err := session.NewManager(workspaceDir, logging.NewLogger("text", io.Discard))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	sess, err := mgr.Start("empty-agent", workspaceDir)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	origWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(workspaceDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	out, err := captureStdout(t, func() error {
		return runBehavioralProfile(ctx, sess.ID, "text", "")
	})
	if err != nil {
		t.Fatalf("runBehavioralProfile: %v", err)
	}
	if !strings.Contains(out, "No categorized actions") {
		t.Errorf("expected 'No categorized actions' message, got: %s", out)
	}
}

func TestRunBehavioralProfile_DeterministicJSON(t *testing.T) {
	setTestHome(t)
	ctx, _ := newBehavioralCtx()

	workspaceDir := t.TempDir()
	mgr, err := session.NewManager(workspaceDir, logging.NewLogger("text", io.Discard))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	sess, err := mgr.Start("det-agent", workspaceDir)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	now := time.Now().UTC()
	_ = mgr.AddCommand(sess, session.Command{Timestamp: now, Command: "git", Args: []string{"status"}})
	_ = mgr.AddFileOperation(sess, session.FileOperation{Timestamp: now.Add(1 * time.Second), Operation: "read", Path: "a.txt"})
	_ = mgr.AddCommand(sess, session.Command{Timestamp: now.Add(2 * time.Second), Command: "curl", Args: []string{"https://x"}})

	origWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(workspaceDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	out1, err1 := captureStdout(t, func() error { return runBehavioralProfile(ctx, sess.ID, "json", "") })
	if err1 != nil {
		t.Fatalf("run1: %v", err1)
	}
	out2, err2 := captureStdout(t, func() error { return runBehavioralProfile(ctx, sess.ID, "json", "") })
	if err2 != nil {
		t.Fatalf("run2: %v", err2)
	}
	if out1 != out2 {
		t.Errorf("expected deterministic JSON across runs\nrun1=%s\nrun2=%s", out1, out2)
	}
}

// TestRunBehavioralProfile_DiffJSON exercises the --diff code path end
// to end: two real sessions, different shapes, diff rendered as JSON.
func TestRunBehavioralProfile_DiffJSON(t *testing.T) {
	setTestHome(t)
	ctx, _ := newBehavioralCtx()

	workspaceDir := t.TempDir()
	mgr, err := session.NewManager(workspaceDir, logging.NewLogger("text", io.Discard))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	// Base session: git status + read README.md
	baseSess, err := mgr.Start("base-agent", workspaceDir)
	if err != nil {
		t.Fatalf("start base: %v", err)
	}
	now := time.Now().UTC()
	_ = mgr.AddCommand(baseSess, session.Command{Timestamp: now, Command: "git", Args: []string{"status"}})
	_ = mgr.AddFileOperation(baseSess, session.FileOperation{Timestamp: now.Add(1 * time.Second), Operation: "read", Path: "README.md"})

	// Target session: git status + curl (different shape — network_call appears)
	targetSess, err := mgr.Start("target-agent", workspaceDir)
	if err != nil {
		t.Fatalf("start target: %v", err)
	}
	_ = mgr.AddCommand(targetSess, session.Command{Timestamp: now, Command: "git", Args: []string{"status"}})
	_ = mgr.AddCommand(targetSess, session.Command{Timestamp: now.Add(1 * time.Second), Command: "curl", Args: []string{"https://example.com"}})

	origWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(workspaceDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	out, err := captureStdout(t, func() error {
		return runBehavioralProfile(ctx, baseSess.ID, "json", targetSess.ID)
	})
	if err != nil {
		t.Fatalf("runBehavioralProfile diff: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("diff output is not valid JSON: %v\nout=%s", err, out)
	}
	if parsed["base_session_id"] != baseSess.ID {
		t.Errorf("base_session_id = %v, want %s", parsed["base_session_id"], baseSess.ID)
	}
	if parsed["target_session_id"] != targetSess.ID {
		t.Errorf("target_session_id = %v, want %s", parsed["target_session_id"], targetSess.ID)
	}
	// target has network_call, base does not → should be in added_nodes.
	addedNodes, ok := parsed["added_nodes"].([]any)
	if !ok || len(addedNodes) == 0 {
		t.Errorf("expected non-empty added_nodes in diff, got: %v", parsed["added_nodes"])
	}
}

// TestRunBehavioralProfile_DiffText checks that the text-mode diff
// renders a header even when the diff is empty (sanity).
func TestRunBehavioralProfile_DiffText_Empty(t *testing.T) {
	setTestHome(t)
	ctx, _ := newBehavioralCtx()

	workspaceDir := t.TempDir()
	mgr, err := session.NewManager(workspaceDir, logging.NewLogger("text", io.Discard))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	// Two identical sessions → diff should be empty.
	sess1, _ := mgr.Start("a", workspaceDir)
	sess2, _ := mgr.Start("b", workspaceDir)
	now := time.Now().UTC()
	for _, s := range []*session.Session{sess1, sess2} {
		_ = mgr.AddCommand(s, session.Command{Timestamp: now, Command: "git", Args: []string{"status"}})
	}

	origWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(workspaceDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	out, err := captureStdout(t, func() error {
		return runBehavioralProfile(ctx, sess1.ID, "text", sess2.ID)
	})
	if err != nil {
		t.Fatalf("runBehavioralProfile diff text: %v", err)
	}
	if !strings.Contains(out, "Behavioral diff:") {
		t.Errorf("expected 'Behavioral diff:' header in text output, got: %s", out)
	}
	if !strings.Contains(out, "no structural differences") {
		t.Errorf("expected empty-diff message, got: %s", out)
	}
}

// TestRunBehavioralProfile_DiffRejectsDot asserts --output dot with
// --diff returns an explicit error (dot is for single graphs only).
func TestRunBehavioralProfile_DiffRejectsDot(t *testing.T) {
	setTestHome(t)
	ctx, _ := newBehavioralCtx()

	workspaceDir := t.TempDir()
	mgr, err := session.NewManager(workspaceDir, logging.NewLogger("text", io.Discard))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	sess1, _ := mgr.Start("a", workspaceDir)
	sess2, _ := mgr.Start("b", workspaceDir)

	origWd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if err := os.Chdir(workspaceDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	err = runBehavioralProfile(ctx, sess1.ID, "dot", sess2.ID)
	if err == nil {
		t.Fatal("expected error when combining --output dot with --diff")
	}
	if !strings.Contains(err.Error(), "dot") {
		t.Errorf("expected error to mention 'dot', got: %v", err)
	}
}
