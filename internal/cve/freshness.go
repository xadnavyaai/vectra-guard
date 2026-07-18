package cve

import (
	"fmt"
	"time"
)

// FreshnessReport summarizes the age of the local CVE cache. It is
// intentionally cheap: it reads only the already-loaded Store and
// computes staleness against the current time.
type FreshnessReport struct {
	CachePath    string        `json:"cache_path"`
	LastSync     time.Time     `json:"last_sync"`
	Age          time.Duration `json:"age"`
	EntryCount   int           `json:"entry_count"`
	OldestEntry  time.Time     `json:"oldest_entry"`
	NewestEntry  time.Time     `json:"newest_entry"`
	Sources      []string      `json:"sources"`
	CacheMissing bool          `json:"cache_missing"`
}

// Freshness builds a FreshnessReport from a Store. If the cache has
// never been synced, LastSync is zero and CacheMissing is true.
func Freshness(store *Store, sources []string) FreshnessReport {
	now := time.Now().UTC()
	report := FreshnessReport{
		CachePath: store.Path,
		LastSync:  store.Cache.UpdatedAt,
		Sources:   sources,
	}
	if store.Cache.UpdatedAt.IsZero() && len(store.Cache.Entries) == 0 {
		report.CacheMissing = true
		return report
	}
	if !store.Cache.UpdatedAt.IsZero() {
		report.Age = now.Sub(store.Cache.UpdatedAt)
	}

	report.EntryCount = len(store.Cache.Entries)
	for _, entry := range store.Cache.Entries {
		if entry.RetrievedAt.IsZero() {
			continue
		}
		if report.OldestEntry.IsZero() || entry.RetrievedAt.Before(report.OldestEntry) {
			report.OldestEntry = entry.RetrievedAt
		}
		if report.NewestEntry.IsZero() || entry.RetrievedAt.After(report.NewestEntry) {
			report.NewestEntry = entry.RetrievedAt
		}
	}
	return report
}

// IsStale returns true if the cache's last sync is older than maxAge.
// A zero or negative maxAge disables the staleness check.
func (r FreshnessReport) IsStale(maxAge time.Duration) bool {
	if maxAge <= 0 {
		return false
	}
	if r.CacheMissing || r.LastSync.IsZero() {
		return true
	}
	return r.Age > maxAge
}

// String returns a human-readable summary.
func (r FreshnessReport) String() string {
	if r.CacheMissing {
		return fmt.Sprintf("CVE cache not found at %s — run `vg cve sync` first.", r.CachePath)
	}
	return fmt.Sprintf(
		"CVE cache:\n  path:        %s\n  last sync:   %s (%s ago)\n  entries:     %d\n  oldest:      %s\n  newest:      %s\n  sources:     %v\n",
		r.CachePath,
		r.LastSync.Format(time.RFC3339),
		r.Age.Round(time.Second),
		r.EntryCount,
		zeroOrTime(r.OldestEntry),
		zeroOrTime(r.NewestEntry),
		r.Sources,
	)
}

func zeroOrTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format(time.RFC3339)
}
