package cve

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFreshness_CacheMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	store, err := LoadStore(path)
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	report := Freshness(store, []string{"osv"})
	if !report.CacheMissing {
		t.Fatalf("expected CacheMissing=true for empty store, got %+v", report)
	}
	if !report.IsStale(time.Hour) {
		t.Fatalf("missing cache should be stale for any positive maxAge")
	}
}

func TestFreshness_FreshCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	store, err := LoadStore(path)
	if err != nil {
		t.Fatalf("load store: %v", err)
	}

	now := time.Now().UTC()
	store.Cache.UpdatedAt = now.Add(-30 * time.Minute)
	store.Set(PackageVuln{
		Package:     PackageRef{Ecosystem: "npm", Name: "foo", Version: "1.0.0"},
		RetrievedAt: now.Add(-45 * time.Minute),
	})
	store.Set(PackageVuln{
		Package:     PackageRef{Ecosystem: "npm", Name: "bar", Version: "2.0.0"},
		RetrievedAt: now.Add(-15 * time.Minute),
	})

	report := Freshness(store, []string{"osv"})
	if report.CacheMissing {
		t.Fatalf("expected CacheMissing=false, got %+v", report)
	}
	if report.EntryCount != 2 {
		t.Errorf("EntryCount=%d, want 2", report.EntryCount)
	}
	// maxAge = 1h — cache is 30m old, should NOT be stale.
	if report.IsStale(time.Hour) {
		t.Errorf("expected not stale at 1h threshold, age=%v", report.Age)
	}
	// maxAge = 10m — cache is 30m old, should be stale.
	if !report.IsStale(10 * time.Minute) {
		t.Errorf("expected stale at 10m threshold, age=%v", report.Age)
	}
	// Oldest / newest from entries.
	if report.OldestEntry.IsZero() || report.NewestEntry.IsZero() {
		t.Errorf("expected non-zero oldest/newest, got %+v", report)
	}
	if !report.OldestEntry.Before(report.NewestEntry) {
		t.Errorf("expected oldest < newest, got oldest=%v newest=%v", report.OldestEntry, report.NewestEntry)
	}
}

func TestFreshnessReport_IsStaleDisabled(t *testing.T) {
	r := FreshnessReport{LastSync: time.Now().Add(-365 * 24 * time.Hour), Age: 365 * 24 * time.Hour}
	if r.IsStale(0) {
		t.Errorf("expected IsStale(0) to be false even for very old cache")
	}
	if r.IsStale(-time.Hour) {
		t.Errorf("expected IsStale(negative) to be false")
	}
}

// --- New: exhaustive coverage ---

func TestFreshness_StaleCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	store, err := LoadStore(path)
	if err != nil {
		t.Fatalf("load store: %v", err)
	}

	now := time.Now().UTC()
	// Cache is 20 days old.
	store.Cache.UpdatedAt = now.Add(-20 * 24 * time.Hour)
	store.Set(PackageVuln{
		Package:     PackageRef{Ecosystem: "npm", Name: "foo", Version: "1.0.0"},
		RetrievedAt: now.Add(-20 * 24 * time.Hour),
	})

	report := Freshness(store, []string{"osv"})
	if report.CacheMissing {
		t.Error("expected CacheMissing=false")
	}
	if !report.IsStale(7 * 24 * time.Hour) {
		t.Errorf("expected stale at 7d threshold, age=%v", report.Age)
	}
	if report.IsStale(30 * 24 * time.Hour) {
		t.Errorf("expected not stale at 30d threshold, age=%v", report.Age)
	}
}

func TestFreshness_SourcesPropagated(t *testing.T) {
	dir := t.TempDir()
	store, err := LoadStore(filepath.Join(dir, "cache.json"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	store.Cache.UpdatedAt = time.Now().UTC()
	sources := []string{"osv", "ghsa", "nvd"}
	report := Freshness(store, sources)
	if len(report.Sources) != 3 {
		t.Errorf("sources len = %d, want 3", len(report.Sources))
	}
	for i, want := range sources {
		if report.Sources[i] != want {
			t.Errorf("sources[%d] = %q, want %q", i, report.Sources[i], want)
		}
	}
}

func TestFreshness_CachePathPropagated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "custom-cache.json")
	store, err := LoadStore(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	report := Freshness(store, nil)
	if report.CachePath != path {
		t.Errorf("CachePath = %q, want %q", report.CachePath, path)
	}
}

func TestFreshness_OldestNewestSkipsZeroRetrievedAt(t *testing.T) {
	dir := t.TempDir()
	store, err := LoadStore(filepath.Join(dir, "cache.json"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	now := time.Now().UTC()
	store.Cache.UpdatedAt = now
	// One entry with zero RetrievedAt should be skipped; other two
	// set the bounds.
	store.Set(PackageVuln{
		Package:     PackageRef{Ecosystem: "npm", Name: "zero", Version: "1.0.0"},
		RetrievedAt: time.Time{},
	})
	store.Set(PackageVuln{
		Package:     PackageRef{Ecosystem: "npm", Name: "old", Version: "1.0.0"},
		RetrievedAt: now.Add(-2 * time.Hour),
	})
	store.Set(PackageVuln{
		Package:     PackageRef{Ecosystem: "npm", Name: "new", Version: "1.0.0"},
		RetrievedAt: now.Add(-30 * time.Minute),
	})
	report := Freshness(store, nil)
	if report.OldestEntry.IsZero() {
		t.Error("OldestEntry should not be zero (real entries present)")
	}
	if report.NewestEntry.IsZero() {
		t.Error("NewestEntry should not be zero")
	}
	if !report.OldestEntry.Before(report.NewestEntry) {
		t.Errorf("oldest %v not before newest %v", report.OldestEntry, report.NewestEntry)
	}
	// The zero-time entry must not shift either bound.
	if report.OldestEntry.IsZero() {
		t.Error("zero-time entry leaked into OldestEntry")
	}
}

func TestFreshness_EntryCountAccurate(t *testing.T) {
	dir := t.TempDir()
	store, err := LoadStore(filepath.Join(dir, "cache.json"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	store.Cache.UpdatedAt = time.Now().UTC()
	for i := 0; i < 7; i++ {
		store.Set(PackageVuln{
			Package: PackageRef{
				Ecosystem: "npm",
				Name:      "pkg" + string(rune('a'+i)),
				Version:   "1.0.0",
			},
			RetrievedAt: time.Now().UTC(),
		})
	}
	report := Freshness(store, nil)
	if report.EntryCount != 7 {
		t.Errorf("EntryCount = %d, want 7", report.EntryCount)
	}
}

func TestFreshnessReport_StringMissing(t *testing.T) {
	r := FreshnessReport{
		CachePath:    "/tmp/cache.json",
		CacheMissing: true,
	}
	s := r.String()
	if !strings.Contains(s, "not found") {
		t.Errorf("missing String() should mention 'not found': %q", s)
	}
	if !strings.Contains(s, "/tmp/cache.json") {
		t.Errorf("missing String() should contain path: %q", s)
	}
}

func TestFreshnessReport_StringFresh(t *testing.T) {
	now := time.Now().UTC()
	r := FreshnessReport{
		CachePath:   "/tmp/cache.json",
		LastSync:    now.Add(-10 * time.Minute),
		Age:         10 * time.Minute,
		EntryCount:  42,
		OldestEntry: now.Add(-time.Hour),
		NewestEntry: now.Add(-1 * time.Minute),
		Sources:     []string{"osv"},
	}
	s := r.String()
	for _, want := range []string{"path:", "last sync:", "entries:", "oldest:", "newest:", "sources:", "/tmp/cache.json", "42"} {
		if !strings.Contains(s, want) {
			t.Errorf("String() missing %q in %q", want, s)
		}
	}
}

func TestFreshnessReport_StringHandlesZeroEntries(t *testing.T) {
	now := time.Now().UTC()
	r := FreshnessReport{
		CachePath:  "/tmp/cache.json",
		LastSync:   now,
		Age:        0,
		EntryCount: 0,
		Sources:    []string{"osv"},
	}
	s := r.String()
	// OldestEntry and NewestEntry are zero — must print "—".
	if !strings.Contains(s, "—") {
		t.Errorf("expected dash placeholder for zero timestamps: %q", s)
	}
}

func TestFreshness_NeverSyncedButHasEntries(t *testing.T) {
	// Edge case: UpdatedAt zero but entries exist (corrupt cache).
	// The function should NOT flag as missing because entries are
	// present.
	dir := t.TempDir()
	store, err := LoadStore(filepath.Join(dir, "cache.json"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	store.Set(PackageVuln{
		Package:     PackageRef{Ecosystem: "npm", Name: "x", Version: "1.0.0"},
		RetrievedAt: time.Now().UTC(),
	})
	report := Freshness(store, nil)
	if report.CacheMissing {
		t.Error("cache with entries but zero UpdatedAt should not be CacheMissing")
	}
	if report.EntryCount != 1 {
		t.Errorf("EntryCount = %d, want 1", report.EntryCount)
	}
	// LastSync is zero → IsStale(positive) should return true.
	if !report.IsStale(time.Hour) {
		t.Error("zero LastSync should be treated as stale")
	}
}

func TestFreshness_RoundTripThroughDisk(t *testing.T) {
	// Hermetic test: write a cache via Save(), reload it, call
	// Freshness(). Verifies integration with the disk layer without
	// touching real user state.
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "cache.json")

	store, err := LoadStore(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	store.Set(PackageVuln{
		Package:     PackageRef{Ecosystem: "npm", Name: "disk-test", Version: "1.0.0"},
		RetrievedAt: time.Now().UTC().Add(-30 * time.Minute),
	})
	if err := store.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Reload from disk.
	reloaded, err := LoadStore(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	report := Freshness(reloaded, []string{"osv"})
	if report.CacheMissing {
		t.Error("expected not missing after Save()/Load() round-trip")
	}
	if report.EntryCount != 1 {
		t.Errorf("EntryCount = %d, want 1", report.EntryCount)
	}
	if report.Age <= 0 {
		t.Errorf("Age should be positive after round-trip, got %v", report.Age)
	}
}
