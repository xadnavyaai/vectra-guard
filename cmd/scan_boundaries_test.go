package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/logging"
)

// newSilentCtx returns a context wired with default config + a discard logger
// so tests don't spam os.Stdout.
func newSilentCtx() context.Context {
	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", io.Discard))
	return ctx
}

// newCapturingCtx returns (ctx, *bytes.Buffer) for tests that want to assert
// on log output.
func newCapturingCtx() (context.Context, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	ctx := context.Background()
	ctx = config.WithConfig(ctx, config.DefaultConfig())
	ctx = logging.WithLogger(ctx, logging.NewLogger("text", buf))
	return ctx, buf
}

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	full := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
	return full
}

func TestRunScanBoundaries_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	ctx := newSilentCtx()
	if err := runScanBoundaries(ctx, dir, ""); err != nil {
		t.Fatalf("expected nil error on empty dir, got %v", err)
	}
}

func TestRunScanBoundaries_FindsRustUnsafe(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/lib.rs", `fn do_it() {
    unsafe {
        println!("danger");
    }
}
`)
	ctx := newSilentCtx()
	err := runScanBoundaries(ctx, dir, "")
	if err == nil {
		t.Fatalf("expected exit error when findings present")
	}
	exitErr, ok := err.(*exitError)
	if !ok {
		t.Fatalf("expected *exitError, got %T: %v", err, err)
	}
	if exitErr.code != 2 {
		t.Errorf("expected exit code 2, got %d", exitErr.code)
	}
}

func TestRunScanBoundaries_LanguageFilter(t *testing.T) {
	dir := t.TempDir()
	// Rust finding present, but filter to "python" only — expect 0 findings.
	writeFile(t, dir, "src/lib.rs", `fn main() { unsafe { } }`)
	ctx := newSilentCtx()
	if err := runScanBoundaries(ctx, dir, "python"); err != nil {
		t.Fatalf("expected nil error with python-only filter, got %v", err)
	}
}

func TestRunScanBoundaries_MultipleLangFilterCSV(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.rs", `fn x() { unsafe { } }`)
	writeFile(t, dir, "b.py", `import ctypes`)
	ctx := newSilentCtx()
	// "rust,python" — both present, should find both → exit 2.
	err := runScanBoundaries(ctx, dir, "rust, python")
	if err == nil {
		t.Fatalf("expected exit error with rust+python filter")
	}
}

func TestRunScanBoundaries_NonExistentPath(t *testing.T) {
	ctx := newSilentCtx()
	// A directory guaranteed not to exist inside a temp dir.
	dir := t.TempDir()
	bogus := filepath.Join(dir, "does-not-exist")
	err := runScanBoundaries(ctx, bogus, "")
	if err == nil {
		t.Fatalf("expected error for nonexistent path")
	}
	if _, ok := err.(*exitError); ok {
		t.Errorf("nonexistent path should return raw error, not *exitError")
	}
}

func TestRunScanBoundaries_DoesNotModifyFilesystem(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "src/lib.rs", `fn x() { unsafe { } }`)

	beforeInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat before: %v", err)
	}
	beforeContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read before: %v", err)
	}

	ctx := newSilentCtx()
	_ = runScanBoundaries(ctx, dir, "")

	afterInfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat after: %v", err)
	}
	afterContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if beforeInfo.Size() != afterInfo.Size() {
		t.Errorf("size changed: %d → %d", beforeInfo.Size(), afterInfo.Size())
	}
	if !bytes.Equal(beforeContent, afterContent) {
		t.Errorf("file content changed during scan")
	}
}

func TestRunScanBoundaries_EmitsFindingLog(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "src/lib.rs", `fn y() { unsafe { } }`)
	ctx, buf := newCapturingCtx()
	_ = runScanBoundaries(ctx, dir, "")
	out := buf.String()
	if len(out) == 0 {
		t.Errorf("expected log output, got empty")
	}
}
