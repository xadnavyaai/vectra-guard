package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vectra-guard/vectra-guard/internal/config"
)

func TestParsePackageArg(t *testing.T) {
	name, version := parsePackageArg("express@4.18.2")
	if name != "express" || version != "4.18.2" {
		t.Fatalf("unexpected parse: %s %s", name, version)
	}
	name, version = parsePackageArg("left-pad")
	if name != "left-pad" || version != "" {
		t.Fatalf("unexpected parse: %s %s", name, version)
	}
}

func TestResolveCVECachePath(t *testing.T) {
	tmp := t.TempDir()
	prevHome := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", prevHome) })
	_ = os.Setenv("HOME", tmp)
	if runtime.GOOS == "windows" {
		prevProfile := os.Getenv("USERPROFILE")
		t.Cleanup(func() { _ = os.Setenv("USERPROFILE", prevProfile) })
		_ = os.Setenv("USERPROFILE", tmp)
	}

	cfg := config.CVEConfig{
		Enabled:             true,
		UpdateIntervalHours: 24,
	}
	path, err := resolveCVECachePath(cfg)
	if err != nil {
		t.Fatalf("resolve cache path: %v", err)
	}
	expected := filepath.Join(tmp, ".vectra-guard", "cve", "cache.json")
	if path != expected {
		t.Fatalf("expected %s, got %s", expected, path)
	}
}

func TestShortSummary(t *testing.T) {
	if shortSummary("") == "" {
		t.Fatalf("expected fallback summary")
	}
	long := "this is a very long summary that should be truncated for display purposes without losing context for the caller"
	if len(shortSummary(long)) > 140 {
		t.Fatalf("expected truncated summary")
	}
}
