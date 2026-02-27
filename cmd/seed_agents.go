package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/seed"
)

func runSeedAgents(ctx context.Context, target string, force bool, targets []string, listOnly bool, autoYes bool) error {
	if target == "" {
		target = "."
	}

	if listOnly {
		available := seed.AvailableTargets()
		keys := make([]string, 0, len(available))
		for key := range available {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		fmt.Println("Available targets:")
		for _, key := range keys {
			t := available[key]
			fmt.Printf("  %-12s -> %s\n", key, t.DestPath)
		}
		return nil
	}

	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("target not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("target is not a directory: %s", target)
	}

	// Detect workspace context
	wc := seed.DetectWorkspaceContext(target)

	// Scan pre-state
	preScan := seed.ScanAgentFiles(target)

	// Check if openclaw is in selected targets
	hasOpenClaw := false
	for _, t := range targets {
		if t == "openclaw" {
			hasOpenClaw = true
			break
		}
	}
	// Also check resolved targets (if no targets specified, default is "agents" only)
	if len(targets) == 0 {
		hasOpenClaw = false
	}

	// Handle openclaw separately with smart detection
	var openClawResults []seed.Result
	if hasOpenClaw {
		ocResults, err := handleOpenClawSeed(target, autoYes)
		if err != nil {
			return err
		}
		openClawResults = ocResults
	}

	// Perform seeding for remaining targets (openclaw is skipped inside WriteAgentInstructions)
	results, err := seed.WriteAgentInstructions(target, force, targets)
	if err != nil {
		return err
	}

	// Combine results: standard targets + openclaw
	results = append(results, openClawResults...)

	// Scan post-state
	postScan := seed.ScanAgentFiles(target)

	// Build set of newly written paths for marking NEW
	writtenSet := map[string]bool{}
	for _, r := range results {
		if r.Status == "written" {
			writtenSet[r.Path] = true
		}
	}

	// Determine which files were new (not in pre-scan)
	preExisted := map[string]bool{}
	for _, s := range preScan {
		if s.Exists {
			preExisted[s.Key] = true
		}
	}

	// Load config for protection status
	cfg := config.FromContext(ctx)
	configPaths := config.ConfigPathsFromContext(ctx)

	// Render
	renderSeedOutput(wc, results, postScan, preExisted, writtenSet, cfg, configPaths)
	return nil
}

// handleOpenClawSeed performs the smart OpenClaw detection, confirmation, and merge flow.
func handleOpenClawSeed(targetDir string, autoYes bool) ([]seed.Result, error) {
	detection := seed.DetectOpenClaw()

	// Load template content
	templateContent, err := seed.GetTemplateContent("templates/.openclaw/AGENTS.md")
	if err != nil {
		return nil, fmt.Errorf("read openclaw template: %w", err)
	}

	var destPath string

	if detection.Installed {
		// OpenClaw is installed — use detected state dir
		defaultDest := filepath.Join(detection.StateDir, "AGENTS.md")

		if autoYes || !isInteractive() {
			// Auto-confirm
			fmt.Fprintf(os.Stderr, "\n  OpenClaw detected at: %s (auto-confirmed)\n\n", detection.StateDir)
			destPath = defaultDest
		} else {
			// Interactive confirmation
			destPath = promptOpenClawPath(detection, defaultDest)
			if destPath == "" {
				// User cancelled
				return []seed.Result{{Path: defaultDest, Status: "cancelled"}}, nil
			}
		}
	} else {
		// OpenClaw not installed — fall back to cwd-relative
		fallbackDest := filepath.Join(targetDir, ".openclaw", "AGENTS.md")

		if autoYes || !isInteractive() {
			destPath = fallbackDest
		} else {
			fmt.Fprintf(os.Stderr, "\n  OpenClaw state directory not found.\n")
			fmt.Fprintf(os.Stderr, "  Will create: %s\n", fallbackDest)
			fmt.Fprintf(os.Stderr, "  Continue? [Y/n]: ")

			var response string
			fmt.Scanln(&response)
			response = strings.ToLower(strings.TrimSpace(response))

			if response == "n" || response == "no" {
				return []seed.Result{{Path: fallbackDest, Status: "cancelled"}}, nil
			}
			destPath = fallbackDest
		}
	}

	result, err := seed.WriteOpenClawInstructions(destPath, templateContent)
	if err != nil {
		return nil, err
	}

	return []seed.Result{result}, nil
}

// promptOpenClawPath shows the detected OpenClaw path and lets the user confirm,
// cancel, or type a custom path.
func promptOpenClawPath(detection seed.OpenClawDetection, defaultDest string) string {
	sourceLabel := detection.Source
	switch {
	case strings.HasPrefix(sourceLabel, "env:"):
		sourceLabel = "environment variable " + sourceLabel[4:]
	case sourceLabel == "default":
		sourceLabel = "default"
	case strings.HasPrefix(sourceLabel, "legacy:"):
		sourceLabel = "legacy (" + sourceLabel[7:] + ")"
	}

	fmt.Fprintf(os.Stderr, "\n  OpenClaw detected (%s)\n", sourceLabel)
	fmt.Fprintf(os.Stderr, "  State directory: %s\n\n", detection.StateDir)

	candidates := seed.OpenClawSeedPaths(detection)
	if len(candidates) > 1 {
		fmt.Fprintf(os.Stderr, "  Candidate paths:\n")
		for i, p := range candidates {
			fmt.Fprintf(os.Stderr, "    %d. %s\n", i+1, p)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	fmt.Fprintf(os.Stderr, "  Destination: %s\n", defaultDest)
	fmt.Fprintf(os.Stderr, "  Confirm? [Y/n/custom path]: ")

	var response string
	fmt.Scanln(&response)
	response = strings.TrimSpace(response)

	if response == "" || strings.ToLower(response) == "y" || strings.ToLower(response) == "yes" {
		return defaultDest
	}
	if strings.ToLower(response) == "n" || strings.ToLower(response) == "no" {
		return ""
	}

	// Custom path — if it's a directory, append AGENTS.md
	customPath := response
	if info, err := os.Stat(customPath); err == nil && info.IsDir() {
		customPath = filepath.Join(customPath, "AGENTS.md")
	} else if !strings.HasSuffix(strings.ToLower(customPath), ".md") {
		customPath = filepath.Join(customPath, "AGENTS.md")
	}

	return customPath
}

func renderSeedOutput(
	wc seed.WorkspaceContext,
	results []seed.Result,
	postScan []seed.AgentFileStatus,
	preExisted map[string]bool,
	writtenSet map[string]bool,
	cfg config.Config,
	configPaths []string,
) {
	line := strings.Repeat("=", 60)
	dash := strings.Repeat("-", 50)

	fmt.Println(line)
	fmt.Println("  VECTRA GUARD — Seed Agent Instructions")
	fmt.Println(line)
	fmt.Println()

	// Workspace info
	if wc.IsGitRepo && wc.RepoName != "" {
		fmt.Printf("  Repository:  %s\n", wc.RepoName)
	}
	if wc.IsGitRepo && wc.Branch != "" {
		fmt.Printf("  Branch:      %s\n", wc.Branch)
	}
	fmt.Printf("  Workspace:   %s\n", wc.AbsPath)
	if len(wc.ProjectTypes) > 0 {
		fmt.Printf("  Project:     %s\n", strings.Join(wc.ProjectTypes, ", "))
	}
	if !wc.IsGitRepo {
		fmt.Printf("  Git:         (not a git repository)\n")
	}
	fmt.Println()

	// Seed results
	if len(results) > 0 {
		fmt.Printf("  Seed Results:\n")
		fmt.Printf("  %s\n", dash)
		for _, r := range results {
			switch r.Status {
			case "written":
				sizeStr := ""
				if info, err := os.Stat(r.Path); err == nil {
					sizeStr = seedFormatSize(info.Size())
				}
				fmt.Printf("    [+] %-40s %s\n", r.Path, sizeStr)
			case "merged":
				sizeStr := ""
				if info, err := os.Stat(r.Path); err == nil {
					sizeStr = seedFormatSize(info.Size())
				}
				fmt.Printf("    [~] %-40s %s (merged)\n", r.Path, sizeStr)
			case "updated":
				sizeStr := ""
				if info, err := os.Stat(r.Path); err == nil {
					sizeStr = seedFormatSize(info.Size())
				}
				fmt.Printf("    [~] %-40s %s (updated)\n", r.Path, sizeStr)
			case "skipped":
				fmt.Printf("    [=] %-40s (exists, skipped)\n", r.Path)
			case "cancelled":
				fmt.Printf("    [-] %-40s (cancelled)\n", r.Path)
			default:
				fmt.Printf("    [?] %-40s %s\n", r.Path, r.Status)
			}
		}
		fmt.Println()
	}

	// Agent coverage
	fmt.Printf("  Agent Coverage:\n")
	fmt.Printf("  %s\n", dash)
	covered := 0
	total := len(postScan)
	for _, s := range postScan {
		marker := " "
		detail := "(not seeded)"
		suffix := ""

		if s.Exists {
			covered++
			marker = "*"
			age := formatAge(s.ModTime)
			sizeStr := seedFormatSize(s.Size)
			detail = fmt.Sprintf("(%s, %s)", age, sizeStr)

			// Check if this was newly written in this run
			fullPath := ""
			for path := range writtenSet {
				if strings.HasSuffix(path, s.DestPath) {
					fullPath = path
					break
				}
			}
			if fullPath != "" && !preExisted[s.Key] {
				suffix = "  NEW"
			}
		}

		destDisplay := s.DestPath
		if len(destDisplay) > 30 {
			destDisplay = destDisplay[:27] + "..."
		}

		fmt.Printf("    [%s] %-10s %-30s %s%s\n", marker, s.Key, destDisplay, detail, suffix)
	}
	fmt.Println()
	fmt.Printf("  Coverage: %d/%d agents configured\n", covered, total)
	fmt.Println()

	// Protection status
	fmt.Printf("  VectraGuard Protection:\n")
	fmt.Printf("  %s\n", dash)
	fmt.Printf("    Guard Level:    %s\n", guardLevelDisplay(cfg.GuardLevel.Level))
	fmt.Printf("    Sandbox:        %s\n", sandboxDisplay(cfg.Sandbox))
	fmt.Printf("    CVE Scanner:    %s\n", enabledDisplay(cfg.CVE.Enabled))
	fmt.Printf("    Soft Delete:    %s\n", enabledDisplay(cfg.SoftDelete.Enabled))
	fmt.Printf("    Env Protection: %s\n", envProtectionDisplay(cfg.EnvProtection))
	fmt.Printf("    Git Ops Guard:  %s\n", gitOpsDisplay(cfg.Policies))

	if len(configPaths) > 0 {
		fmt.Println()
		fmt.Printf("    Config: %s\n", configPaths[len(configPaths)-1])
	}

	fmt.Println()
	fmt.Printf("  %s\n", dash)

	// Tips
	fmt.Println("  Tip: `vg seed agents -targets agents,cursor,copilot` to add more")
	fmt.Println("  Tip: `vg serve` to open the security dashboard")
	fmt.Println(line)
}

func seedFormatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

func formatAge(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func guardLevelDisplay(level config.GuardLevel) string {
	switch level {
	case config.GuardLevelAuto:
		return "auto (context-aware)"
	default:
		return string(level)
	}
}

func sandboxDisplay(sb config.SandboxConfig) string {
	if !sb.Enabled {
		return "DISABLED"
	}
	return fmt.Sprintf("ENABLED (mode: %s)", sb.Mode)
}

func enabledDisplay(on bool) string {
	if on {
		return "ENABLED"
	}
	return "DISABLED"
}

func envProtectionDisplay(ep config.EnvProtectionConfig) string {
	if !ep.Enabled {
		return "DISABLED"
	}
	return fmt.Sprintf("ENABLED (masking: %s)", ep.MaskingMode)
}

func gitOpsDisplay(p config.PolicyConfig) string {
	if !p.MonitorGitOps {
		return "DISABLED"
	}
	return "ENABLED"
}

func parseSeedTargets(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' '
	})
	var out []string
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, strings.ToLower(trimmed))
		}
	}
	return out
}
