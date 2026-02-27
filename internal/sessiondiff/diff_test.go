package sessiondiff

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/session"
)

func TestComputeBasicDiff(t *testing.T) {
	now := time.Now()
	ws := filepath.Join(string(filepath.Separator), "workspace")
	ops := []session.FileOperation{
		{Timestamp: now, Operation: "create", Path: filepath.Join(ws, "a.txt")},
		{Timestamp: now, Operation: "modify", Path: filepath.Join(ws, "a.txt")},
		{Timestamp: now, Operation: "create", Path: filepath.Join(ws, "b.txt")},
		{Timestamp: now, Operation: "delete", Path: filepath.Join(ws, "b.txt")},
		{Timestamp: now, Operation: "modify", Path: filepath.Join(ws, "c.txt")},
	}

	sum := Compute(ops, "")

	if len(sum.Added) != 1 || sum.Added[0] != filepath.Join(ws, "a.txt") {
		t.Fatalf("expected a.txt as added, got %#v", sum.Added)
	}
	if len(sum.Modified) != 1 || sum.Modified[0] != filepath.Join(ws, "c.txt") {
		t.Fatalf("expected c.txt as modified, got %#v", sum.Modified)
	}
	if len(sum.Deleted) != 1 || sum.Deleted[0] != filepath.Join(ws, "b.txt") {
		t.Fatalf("expected b.txt as deleted, got %#v", sum.Deleted)
	}
}

func TestComputeWithRootFilter(t *testing.T) {
	now := time.Now()
	ws := filepath.Join(string(filepath.Separator), "workspace")
	other := filepath.Join(string(filepath.Separator), "other")
	ops := []session.FileOperation{
		{Timestamp: now, Operation: "create", Path: filepath.Join(ws, "a.txt")},
		{Timestamp: now, Operation: "create", Path: filepath.Join(other, "b.txt")},
	}

	sum := Compute(ops, ws)

	if len(sum.Added) != 1 || sum.Added[0] != filepath.Join(ws, "a.txt") {
		t.Fatalf("expected only %s, got %#v", filepath.Join(ws, "a.txt"), sum.Added)
	}
}
