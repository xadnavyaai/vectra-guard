package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/cve"
)

func runCVESync(ctx context.Context, target string, force bool) error {
	cfg := config.FromContext(ctx)
	if !cfg.CVE.Enabled {
		return fmt.Errorf("cve awareness disabled (set cve.enabled=true)")
	}

	cachePath, err := resolveCVECachePath(cfg.CVE)
	if err != nil {
		return err
	}
	store, err := cve.LoadStore(cachePath)
	if err != nil {
		return err
	}

	packages, warnings, err := cve.DiscoverPackages(target)
	if err != nil {
		return err
	}
	for _, warn := range warnings {
		fmt.Fprintln(os.Stderr, "⚠", warn)
	}
	if len(packages) == 0 {
		fmt.Println("No supported manifests/lockfiles found.")
		return nil
	}

	maxAge := time.Duration(cfg.CVE.UpdateIntervalHours) * time.Hour
	fetched := 0
	skipped := 0
	errors := 0

	for _, ref := range packages {
		if !force {
			if entry, ok := store.Get(ref); ok && store.IsFresh(entry, maxAge) {
				skipped++
				continue
			}
		}
		vulns, err := cve.FetchOSVVulns(ctx, ref)
		if err != nil {
			errors++
			fmt.Fprintf(os.Stderr, "⚠ osv lookup failed for %s@%s (%s): %v\n", ref.Name, ref.Version, ref.Ecosystem, err)
			continue
		}
		store.Set(cve.PackageVuln{
			Package:         ref,
			Vulnerabilities: vulns,
			RetrievedAt:     time.Now().UTC(),
		})
		fetched++
	}

	if err := store.Save(); err != nil {
		return err
	}

	fmt.Printf("CVE sync complete: %d fetched, %d skipped, %d errors\n", fetched, skipped, errors)
	return nil
}

func runCVEScan(ctx context.Context, target string, refresh bool, maxAgeDays int) error {
	cfg := config.FromContext(ctx)
	if !cfg.CVE.Enabled {
		return fmt.Errorf("cve awareness disabled (set cve.enabled=true)")
	}

	cachePath, err := resolveCVECachePath(cfg.CVE)
	if err != nil {
		return err
	}
	store, err := cve.LoadStore(cachePath)
	if err != nil {
		return err
	}

	// Post-Mythos defaults: if --max-age-days is set, hard-fail before we
	// trust the cache. Cache age beyond N days is a live exposure.
	if maxAgeDays > 0 {
		report := cve.Freshness(store, cfg.CVE.Sources)
		threshold := time.Duration(maxAgeDays) * 24 * time.Hour
		if report.IsStale(threshold) {
			if report.CacheMissing {
				return &exitError{
					message: fmt.Sprintf("cve cache missing at %s — run `vg cve sync`", cachePath),
					code:    2,
				}
			}
			return &exitError{
				message: fmt.Sprintf("cve cache is stale (age %s > %s) — run `vg cve sync`",
					report.Age.Round(time.Second), threshold),
				code: 2,
			}
		}
	}

	packages, warnings, err := cve.DiscoverPackages(target)
	if err != nil {
		return err
	}
	for _, warn := range warnings {
		fmt.Fprintln(os.Stderr, "⚠", warn)
	}
	if len(packages) == 0 {
		fmt.Println("No supported manifests/lockfiles found.")
		return nil
	}

	maxAge := time.Duration(cfg.CVE.UpdateIntervalHours) * time.Hour
	results := make([]cve.PackageVuln, 0, len(packages))

	for _, ref := range packages {
		if !refresh {
			if entry, ok := store.Get(ref); ok && store.IsFresh(entry, maxAge) {
				results = append(results, entry)
				continue
			}
		}
		vulns, err := cve.FetchOSVVulns(ctx, ref)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ osv lookup failed for %s@%s (%s): %v\n", ref.Name, ref.Version, ref.Ecosystem, err)
			continue
		}
		entry := cve.PackageVuln{
			Package:         ref,
			Vulnerabilities: vulns,
			RetrievedAt:     time.Now().UTC(),
		}
		store.Set(entry)
		results = append(results, entry)
	}

	if err := store.Save(); err != nil {
		return err
	}

	printCVEReport(results)
	return nil
}

func runCVEExplain(ctx context.Context, pkgArg string, ecosystem string, refresh bool) error {
	cfg := config.FromContext(ctx)
	if !cfg.CVE.Enabled {
		return fmt.Errorf("cve awareness disabled (set cve.enabled=true)")
	}

	cachePath, err := resolveCVECachePath(cfg.CVE)
	if err != nil {
		return err
	}
	store, err := cve.LoadStore(cachePath)
	if err != nil {
		return err
	}

	name, version := parsePackageArg(pkgArg)
	if name == "" {
		return fmt.Errorf("invalid package reference")
	}
	if ecosystem == "" {
		ecosystem = "npm"
	}

	var entries []cve.PackageVuln
	if version != "" {
		ref := cve.PackageRef{Ecosystem: ecosystem, Name: name, Version: version}
		if !refresh {
			if cached, ok := store.Get(ref); ok {
				entries = append(entries, cached)
			}
		}
		if len(entries) == 0 {
			vulns, err := cve.FetchOSVVulns(ctx, ref)
			if err != nil {
				return fmt.Errorf("osv lookup failed for %s@%s (%s): %w", name, version, ecosystem, err)
			}
			entry := cve.PackageVuln{
				Package:         ref,
				Vulnerabilities: vulns,
				RetrievedAt:     time.Now().UTC(),
			}
			store.Set(entry)
			entries = append(entries, entry)
			if err := store.Save(); err != nil {
				return err
			}
		}
	} else {
		for _, entry := range store.Cache.Entries {
			if strings.EqualFold(entry.Package.Ecosystem, ecosystem) && entry.Package.Name == name {
				entries = append(entries, entry)
			}
		}
		if len(entries) == 0 && refresh {
			return fmt.Errorf("no cached versions found for %s (%s); provide name@version", name, ecosystem)
		}
	}

	if len(entries) == 0 {
		return fmt.Errorf("no cached CVE data for %s (run `vg cve sync` or `vg cve explain %s@version`)", name, name)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Package.Version < entries[j].Package.Version
	})
	printCVEReport(entries)
	return nil
}

func runCVEFreshness(ctx context.Context, outputJSON bool) error {
	cfg := config.FromContext(ctx)
	if !cfg.CVE.Enabled {
		return fmt.Errorf("cve awareness disabled (set cve.enabled=true)")
	}

	cachePath, err := resolveCVECachePath(cfg.CVE)
	if err != nil {
		return err
	}
	store, err := cve.LoadStore(cachePath)
	if err != nil {
		return err
	}

	report := cve.Freshness(store, cfg.CVE.Sources)

	if outputJSON {
		data, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	}

	fmt.Print(report.String())
	return nil
}

func resolveCVECachePath(cfg config.CVEConfig) (string, error) {
	dir := strings.TrimSpace(cfg.CacheDir)
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		dir = filepath.Join(home, ".vectra-guard", "cve")
	}
	if strings.HasPrefix(dir, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			dir = filepath.Join(home, strings.TrimPrefix(dir, "~"))
		}
	}
	if !filepath.IsAbs(dir) {
		abs, err := filepath.Abs(dir)
		if err == nil {
			dir = abs
		}
	}
	return filepath.Join(dir, "cache.json"), nil
}

func parsePackageArg(arg string) (string, string) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return "", ""
	}
	at := strings.LastIndex(arg, "@")
	if at > 0 {
		return arg[:at], arg[at+1:]
	}
	return arg, ""
}

func printCVEReport(entries []cve.PackageVuln) {
	if len(entries) == 0 {
		fmt.Println("✅ No CVE data available.")
		return
	}

	totalVulns := 0
	for _, entry := range entries {
		totalVulns += len(entry.Vulnerabilities)
	}

	fmt.Printf("🔎 CVE report (%d packages, %d advisories)\n", len(entries), totalVulns)

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Package.Name == entries[j].Package.Name {
			return entries[i].Package.Version < entries[j].Package.Version
		}
		return entries[i].Package.Name < entries[j].Package.Name
	})

	for _, entry := range entries {
		if len(entry.Vulnerabilities) == 0 {
			continue
		}
		fmt.Printf("\n⚠ %s@%s (%s)\n", entry.Package.Name, entry.Package.Version, entry.Package.Ecosystem)
		sort.Slice(entry.Vulnerabilities, func(i, j int) bool {
			return entry.Vulnerabilities[i].CVSS > entry.Vulnerabilities[j].CVSS
		})
		for _, vuln := range entry.Vulnerabilities {
			displayID := bestVulnID(vuln)
			if vuln.CVSS > 0 {
				fmt.Printf("- %s (CVSS %.1f, %s): %s\n", displayID, vuln.CVSS, vuln.Severity, shortSummary(vuln.Summary))
			} else {
				fmt.Printf("- %s (%s): %s\n", displayID, vuln.Severity, shortSummary(vuln.Summary))
			}
		}
	}

	if totalVulns == 0 {
		fmt.Println("✅ No known vulnerabilities found.")
	}
}

func bestVulnID(v cve.Vulnerability) string {
	for _, alias := range v.Aliases {
		if strings.HasPrefix(alias, "CVE-") {
			return alias
		}
	}
	if v.ID != "" {
		return v.ID
	}
	return "UNKNOWN"
}

func shortSummary(summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return "No summary available"
	}
	if len(summary) > 140 {
		return summary[:137] + "..."
	}
	return summary
}
