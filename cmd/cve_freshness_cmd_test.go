package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/cve"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

// newCVECtx builds a context with CVE enabled and a custom CacheDir pointed
// at a temp dir, plus a discard logger. Returns (ctx, cacheDir).
func newCVECtx(t *testing.T) (context.Context, string) {
	t.Helper()
	cacheDir := t.TempDir()
	cfg := config.DefaultConfig()
	cfg.CVE.Enabled = true
	cfg.CVE.CacheDir = cacheDir
	cfg.CVE.Sources = []string{"osv"}
	ctx := context.Background()
	ctx = config.WithConfig(ctx, cfg)
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))
	return ctx, cacheDir
}

func TestRunCVEFreshness_Disabled(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.CVE.Enabled = false
	ctx := context.Background()
	ctx = config.WithConfig(ctx, cfg)
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	err := runCVEFreshness(ctx, false)
	if err == nil {
		t.Fatal("expected error when CVE disabled")
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Errorf("expected 'disabled' in error, got %v", err)
	}
}

func TestRunCVEFreshness_EmptyCache(t *testing.T) {
	ctx, _ := newCVECtx(t)
	out, err := captureStdout(t, func() error {
		return runCVEFreshness(ctx, false)
	})
	if err != nil {
		t.Fatalf("runCVEFreshness: %v", err)
	}
	if !strings.Contains(out, "not found") {
		t.Errorf("expected 'not found' in text output: %s", out)
	}
}

func TestRunCVEFreshness_JSONOutput(t *testing.T) {
	ctx, _ := newCVECtx(t)
	out, err := captureStdout(t, func() error {
		return runCVEFreshness(ctx, true)
	})
	if err != nil {
		t.Fatalf("runCVEFreshness JSON: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Errorf("expected JSON object, got: %s", out)
	}
	if !strings.Contains(out, "cache_path") && !strings.Contains(out, "CachePath") {
		t.Errorf("expected cache path field in JSON: %s", out)
	}
}

func TestRunCVEFreshness_WithPopulatedCache(t *testing.T) {
	ctx, cacheDir := newCVECtx(t)

	// Write a populated cache via the real store so the json layout
	// matches what runCVEFreshness will read back.
	cachePath := filepath.Join(cacheDir, "cache.json")
	store, err := cve.LoadStore(cachePath)
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	now := time.Now().UTC()
	store.Cache.UpdatedAt = now.Add(-15 * time.Minute)
	store.Set(cve.PackageVuln{
		Package:     cve.PackageRef{Ecosystem: "npm", Name: "test-pkg", Version: "1.0.0"},
		RetrievedAt: now.Add(-15 * time.Minute),
	})
	if err := store.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	out, err := captureStdout(t, func() error {
		return runCVEFreshness(ctx, false)
	})
	if err != nil {
		t.Fatalf("runCVEFreshness: %v", err)
	}
	if !strings.Contains(out, "entries") {
		t.Errorf("expected 'entries' in text output: %s", out)
	}
}

// --- runCVEScan --max-age-days tests ---

func TestRunCVEScan_MaxAgeDaysCacheMissing(t *testing.T) {
	ctx, _ := newCVECtx(t)
	// No cache present, --max-age-days=1 → should hard-fail with exit 2.
	workspace := t.TempDir()
	err := runCVEScan(ctx, workspace, false, 1)
	if err == nil {
		t.Fatal("expected exit error for missing cache with max-age-days")
	}
	exitErr, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected *exitError, got %T: %v", err, err)
	}
	if exitErr.code != 2 {
		t.Errorf("expected exit code 2, got %d", exitErr.code)
	}
	if !strings.Contains(exitErr.message, "missing") {
		t.Errorf("expected 'missing' in message, got %q", exitErr.message)
	}
}

func TestRunCVEScan_MaxAgeDaysStaleCache(t *testing.T) {
	ctx, cacheDir := newCVECtx(t)
	cachePath := filepath.Join(cacheDir, "cache.json")
	// Write cache.json directly with a 10-day-old updated_at; calling
	// Save() would stomp UpdatedAt to now. The schema mirrors Cache +
	// PackageVuln as defined in internal/cve/types.go and store.go.
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	stalePayload := `{
  "updated_at": "` + time.Now().UTC().Add(-10*24*time.Hour).Format(time.RFC3339) + `",
  "entries": {
    "npm:p@1.0.0": {
      "package": {"ecosystem": "npm", "name": "p", "version": "1.0.0"},
      "vulnerabilities": [],
      "retrieved_at": "` + time.Now().UTC().Add(-10*24*time.Hour).Format(time.RFC3339) + `"
    }
  }
}`
	if err := os.WriteFile(cachePath, []byte(stalePayload), 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}

	workspace := t.TempDir()
	err := runCVEScan(ctx, workspace, false, 7)
	if err == nil {
		t.Fatal("expected exit error for stale cache")
	}
	exitErr, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected *exitError, got %T", err)
	}
	if exitErr.code != 2 {
		t.Errorf("expected exit code 2, got %d", exitErr.code)
	}
	if !strings.Contains(exitErr.message, "stale") {
		t.Errorf("expected 'stale' in message, got %q", exitErr.message)
	}
}

func TestRunCVEScan_MaxAgeDaysFreshCachePasses(t *testing.T) {
	ctx, cacheDir := newCVECtx(t)
	cachePath := filepath.Join(cacheDir, "cache.json")
	store, err := cve.LoadStore(cachePath)
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	store.Cache.UpdatedAt = time.Now().UTC().Add(-1 * time.Hour)
	store.Set(cve.PackageVuln{
		Package:     cve.PackageRef{Ecosystem: "npm", Name: "p", Version: "1.0.0"},
		RetrievedAt: time.Now().UTC().Add(-1 * time.Hour),
	})
	if err := store.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	workspace := t.TempDir()
	// Fresh cache + no manifests in workspace should short-circuit
	// successfully (no error) after the freshness gate passes.
	// Capture stdout to avoid "No supported manifests" spam.
	_, err = captureStdout(t, func() error {
		return runCVEScan(ctx, workspace, false, 7)
	})
	if err != nil {
		t.Fatalf("expected no error for fresh cache with empty workspace, got %v", err)
	}
}

func TestRunCVEScan_MaxAgeDaysZeroDisabled(t *testing.T) {
	ctx, _ := newCVECtx(t)
	// max-age-days=0 means "don't check" — empty cache should pass
	// the freshness gate entirely.
	workspace := t.TempDir()
	_, err := captureStdout(t, func() error {
		return runCVEScan(ctx, workspace, false, 0)
	})
	if err != nil {
		t.Fatalf("expected no error with max-age-days=0, got %v", err)
	}
}

// --- runPromptFirewallBenchmark ---

func TestRunPromptFirewallBenchmark(t *testing.T) {
	// Capture stdout because the benchmark prints to fmt.Printf.
	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	done := make(chan error, 1)
	go func() {
		done <- runPromptFirewallBenchmark(context.Background())
	}()
	if err := <-done; err != nil {
		_ = w.Close()
		t.Fatalf("benchmark error: %v", err)
	}
	_ = w.Close()
	_, _ = io.Copy(&buf, r)

	out := buf.String()
	for _, want := range []string{
		"promptfw benchmark",
		"precision",
		"recall",
		"f1",
		"malicious prompts",
		"benign prompts",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("benchmark output missing %q: %s", want, out)
		}
	}
}

func TestRunPromptFirewallBenchmarkViaHandler(t *testing.T) {
	// Exercise the public entrypoint runPromptFirewall(ctx, "", true).
	var buf bytes.Buffer
	origStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = origStdout })

	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))

	done := make(chan error, 1)
	go func() {
		done <- runPromptFirewall(ctx, "", true)
	}()
	if err := <-done; err != nil {
		_ = w.Close()
		t.Fatalf("runPromptFirewall(benchmark): %v", err)
	}
	_ = w.Close()
	_, _ = io.Copy(&buf, r)

	if !strings.Contains(buf.String(), "precision") {
		t.Errorf("expected benchmark output, got: %s", buf.String())
	}
}

// --- truncate helper test ---

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("short: got %q, want %q", got, "hello")
	}
	if got := truncate("hello world", 5); got != "hello..." {
		t.Errorf("long: got %q, want %q", got, "hello...")
	}
	if got := truncate("", 5); got != "" {
		t.Errorf("empty: got %q, want empty", got)
	}
}
