package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/secrets"
	"github.com/vectra-guard/vectra-guard/internal/secscan"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

type auditSummary struct {
	Tool   string
	Path   string
	Total  int
	Counts map[string]int
	Mode   string
}

type sessionAuditSummary struct {
	SessionID       string
	Workspace       string
	Agent           string
	SessionCount    int
	Total           int
	RiskCounts      map[string]int
	SourceCounts    map[string]int
	ExecutionCounts map[string]int
	Bypassed        int
	Blocked         int
}

// repoAuditSummary is a higher-level, repo-wide summary that aggregates
// static security scan findings, secret detections, and package audits.
type repoAuditSummary struct {
	Path string

	// Code-level static analysis (scan-security)
	CodeFindings      []secscan.Finding
	CodeFindingsTotal int
	CodeBySeverity    map[string]int
	CodeByLanguage    map[string]int

	// Secret scanning (scan-secrets)
	SecretsTotal int

	// Package / dependency audits (npm, python, cve)
	PackageAudits []auditSummary
}

// repoAuditOptions holds options for repo audit only (output format, allowlist, ignore paths).
type repoAuditOptions struct {
	OutputFormat  string // text, markdown, json
	AllowlistPath string
	IgnoreGlobs   string // comma-separated
}

// codeRemediation maps secscan finding codes to short remediation guidance.
var codeRemediation = map[string]string{
	"PY_ENV_ACCESS":       "Use env vars; avoid logging or exposing in responses.",
	"PY_SUBPROCESS":       "Prefer subprocess with shell=False and validate arguments.",
	"PY_EVAL":             "Avoid eval(); use ast.literal_eval or structured parsing for untrusted input.",
	"PY_EXEC":             "Avoid exec(); validate and sandbox any dynamic code.",
	"PY_REMOTE_HTTP":      "Validate URLs and responses; guard against SSRF.",
	"GO_EXEC_COMMAND":     "Validate and sanitize command arguments; avoid shell invocation.",
	"GO_DANGEROUS_SHELL":  "Remove or restrict dangerous shell patterns.",
	"GO_NET_HTTP":         "Authenticate and sanitize remote calls.",
	"GO_ENV_READ":         "Avoid leaking credentials via env; use secret managers where possible.",
	"GO_SYSTEM_WRITE":     "Avoid writing to system dirs from app code; use config management.",
	"C_SHELL_EXEC":        "Validate inputs to system/popen/exec*; avoid shell injection.",
	"C_GETS":              "Replace gets() with fgets() or safe alternatives.",
	"C_UNSAFE_STRING":     "Use bounded APIs (strncpy, strlcpy) or safe string libraries.",
	"C_MEMCPY":            "Validate bounds before memcpy to prevent buffer overflows.",
	"C_RAW_SOCKET":        "Review socket use for abuse or exfiltration.",
	"PY_EXTERNAL_HTTP":    "Ensure non-localhost URLs are not used with untrusted input (SSRF risk).",
	"GO_EXTERNAL_HTTP":    "Ensure non-localhost URLs are not used with untrusted input (SSRF risk).",
	"BIND_ALL_INTERFACES": "Binding to 0.0.0.0 exposes the service on all interfaces; ensure auth and TLS.",
}

func runAudit(ctx context.Context, tool string, targetPath string, failOnFindings bool, autoInstall bool, sessionID string, allSessions bool, repoOpts *repoAuditOptions) error {
	logger := logging.FromContext(ctx)
	if targetPath == "" {
		targetPath = "."
	}

	switch strings.ToLower(tool) {
	case "repo":
		summary, err := runRepoAudit(ctx, targetPath, autoInstall, repoOpts)
		if err != nil {
			return err
		}
		outputFormat := "text"
		if repoOpts != nil && repoOpts.OutputFormat != "" {
			outputFormat = strings.ToLower(strings.TrimSpace(repoOpts.OutputFormat))
		}
		switch outputFormat {
		case "markdown":
			emitRepoAuditMarkdown(summary)
		case "json":
			emitRepoAuditJSON(summary)
		default:
			logRepoAuditSummary(logger, summary)
			printRepoAuditFindingsText(summary, 50)
		}
		if failOnFindings && (summary.CodeFindingsTotal > 0 || summary.SecretsTotal > 0 || hasPackageFindings(summary.PackageAudits)) {
			return &exitError{message: "repo audit found issues", code: 2}
		}
	case "session":
		summary, err := runSessionAudit(ctx, sessionID, allSessions)
		if err != nil {
			return err
		}
		logSessionAuditSummary(logger, summary)
	case "npm":
		if autoInstall {
			if err := ensureNpm(ctx); err != nil {
				return err
			}
		}
		summary, err := runNpmAudit(targetPath)
		if err != nil {
			return err
		}
		logAuditSummary(logger, summary)
		if failOnFindings && summary.Total > 0 {
			return &exitError{message: "npm audit found vulnerabilities", code: 2}
		}
	case "python", "pip":
		if autoInstall {
			if err := ensurePipAudit(ctx); err != nil {
				return err
			}
		}
		summary, err := runPythonAudit(targetPath)
		if err != nil {
			return err
		}
		logAuditSummary(logger, summary)
		if failOnFindings && summary.Total > 0 {
			return &exitError{message: "python audit found vulnerabilities", code: 2}
		}
	case "agent-posture":
		outputFormat := "text"
		if repoOpts != nil && repoOpts.OutputFormat != "" {
			outputFormat = strings.ToLower(strings.TrimSpace(repoOpts.OutputFormat))
		}
		return runAgentPostureAudit(ctx, targetPath, outputFormat, failOnFindings)
	default:
		return fmt.Errorf("unsupported audit tool: %s", tool)
	}

	return nil
}

// runRepoAudit orchestrates a basic, repo-wide security audit by:
//   - running scan-security (static security patterns)
//   - running scan-secrets (secret detection)
//   - running package audits (npm, python) in project mode
func runRepoAudit(ctx context.Context, targetPath string, autoInstall bool, opts *repoAuditOptions) (repoAuditSummary, error) {
	logger := logging.FromContext(ctx)
	summary := repoAuditSummary{
		Path:           targetPath,
		CodeFindings:   nil,
		CodeBySeverity: map[string]int{},
		CodeByLanguage: map[string]int{},
	}

	// 1) Static security scan (code-level findings)
	secFindings, err := secscan.ScanPath(targetPath, secscan.Options{})
	if err != nil {
		return repoAuditSummary{}, fmt.Errorf("scan-security: %w", err)
	}
	summary.CodeFindings = secFindings
	for _, f := range secFindings {
		summary.CodeFindingsTotal++
		sev := strings.TrimSpace(strings.ToLower(f.Severity))
		if sev == "" {
			sev = "unknown"
		}
		summary.CodeBySeverity[sev] = summary.CodeBySeverity[sev] + 1

		lang := strings.TrimSpace(strings.ToLower(f.Language))
		if lang == "" {
			lang = "unknown"
		}
		summary.CodeByLanguage[lang] = summary.CodeByLanguage[lang] + 1
	}

	// 2) Secret scan with optional allowlist and path ignore
	secretsOpts := secrets.Options{}
	if opts != nil {
		if opts.AllowlistPath != "" {
			allowlist, errLoad := loadAllowlist(opts.AllowlistPath)
			if errLoad != nil {
				logger.Warn("load allowlist failed, continuing without", map[string]any{
					"path": opts.AllowlistPath, "error": errLoad.Error(),
				})
			} else {
				secretsOpts.Allowlist = allowlist
			}
		}
		for _, p := range strings.Split(opts.IgnoreGlobs, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				secretsOpts.IgnorePaths = append(secretsOpts.IgnorePaths, trimmed)
			}
		}
	}
	secretCount, err := countSecrets(ctx, targetPath, secretsOpts)
	if err != nil {
		logger.Warn("secret scan failed during repo audit", map[string]any{
			"path":  targetPath,
			"error": err.Error(),
		})
	} else {
		summary.SecretsTotal = secretCount
	}

	// 3) Package audits (npm, python) – best-effort; ignore unsupported cases.
	if autoInstall {
		if err := ensureNpm(ctx); err != nil {
			logger.Info("npm auto-install failed for repo audit (continuing without npm audit)", map[string]any{
				"error": err.Error(),
			})
		}
		if err := ensurePipAudit(ctx); err != nil {
			logger.Info("pip-audit auto-install failed for repo audit (continuing without python audit)", map[string]any{
				"error": err.Error(),
			})
		}
	}

	if s, err := runNpmAudit(targetPath); err == nil {
		summary.PackageAudits = append(summary.PackageAudits, s)
	} else {
		logger.Info("npm audit skipped during repo audit", map[string]any{
			"error": err.Error(),
		})
	}

	if s, err := runPythonAudit(targetPath); err == nil {
		summary.PackageAudits = append(summary.PackageAudits, s)
	} else {
		logger.Info("python audit skipped during repo audit", map[string]any{
			"error": err.Error(),
		})
	}

	return summary, nil
}

// countSecrets returns the number of secret findings for the given path.
func countSecrets(ctx context.Context, targetPath string, opts secrets.Options) (int, error) {
	logger := logging.FromContext(ctx)

	findings, err := secrets.ScanPath(targetPath, opts)
	if err != nil {
		logger.Warn("scan-secrets failed", map[string]any{
			"path":  targetPath,
			"error": err.Error(),
		})
		return 0, err
	}
	return len(findings), nil
}

func hasPackageFindings(audits []auditSummary) bool {
	for _, a := range audits {
		if a.Total > 0 {
			return true
		}
	}
	return false
}

func runSessionAudit(ctx context.Context, sessionID string, allSessions bool) (sessionAuditSummary, error) {
	logger := logging.FromContext(ctx)
	workspace, err := os.Getwd()
	if err != nil {
		return sessionAuditSummary{}, fmt.Errorf("get workspace: %w", err)
	}

	mgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return sessionAuditSummary{}, fmt.Errorf("create session manager: %w", err)
	}

	if allSessions {
		sessions, err := mgr.List()
		if err != nil {
			return sessionAuditSummary{}, fmt.Errorf("list sessions: %w", err)
		}
		if len(sessions) == 0 {
			return sessionAuditSummary{}, fmt.Errorf("no sessions found")
		}
		return buildSessionAuditSummaryFromSessions(sessions), nil
	}

	if sessionID == "" {
		sessionID = session.GetCurrentSessionForWorkspace(workspace)
	}
	if sessionID == "" {
		return sessionAuditSummary{}, fmt.Errorf("no active session found")
	}

	sess, err := mgr.Load(sessionID)
	if err != nil {
		return sessionAuditSummary{}, fmt.Errorf("load session: %w", err)
	}

	return buildSessionAuditSummary(sess), nil
}

func buildSessionAuditSummary(sess *session.Session) sessionAuditSummary {
	summary := sessionAuditSummary{
		SessionID:       sess.ID,
		Workspace:       sess.Workspace,
		Agent:           sess.AgentName,
		SessionCount:    1,
		Total:           len(sess.Commands),
		RiskCounts:      map[string]int{},
		SourceCounts:    map[string]int{},
		ExecutionCounts: map[string]int{},
	}

	for _, cmd := range sess.Commands {
		risk := strings.TrimSpace(cmd.RiskLevel)
		if risk == "" {
			risk = "unknown"
		}
		summary.RiskCounts[risk] = summary.RiskCounts[risk] + 1

		source := "exec"
		execution := ""
		if cmd.Metadata != nil {
			if val, ok := cmd.Metadata["source"].(string); ok && val != "" {
				source = val
			}
			if val, ok := cmd.Metadata["execution"].(string); ok && val != "" {
				execution = val
			}
			if val, ok := cmd.Metadata["bypass"].(bool); ok && val {
				summary.Bypassed++
			}
			if val, ok := cmd.Metadata["blocked"].(bool); ok && val {
				summary.Blocked++
			}
		}

		summary.SourceCounts[source] = summary.SourceCounts[source] + 1
		if execution != "" {
			summary.ExecutionCounts[execution] = summary.ExecutionCounts[execution] + 1
		}
	}

	return summary
}

func buildSessionAuditSummaryFromSessions(sessions []*session.Session) sessionAuditSummary {
	summary := sessionAuditSummary{
		SessionID:       "all",
		SessionCount:    len(sessions),
		Total:           0,
		RiskCounts:      map[string]int{},
		SourceCounts:    map[string]int{},
		ExecutionCounts: map[string]int{},
	}

	for _, sess := range sessions {
		if summary.Workspace == "" {
			summary.Workspace = sess.Workspace
		}
		if summary.Agent == "" {
			summary.Agent = sess.AgentName
		}
		summary.Total += len(sess.Commands)
		single := buildSessionAuditSummary(sess)
		mergeCounts(summary.RiskCounts, single.RiskCounts)
		mergeCounts(summary.SourceCounts, single.SourceCounts)
		mergeCounts(summary.ExecutionCounts, single.ExecutionCounts)
		summary.Bypassed += single.Bypassed
		summary.Blocked += single.Blocked
	}

	return summary
}

func mergeCounts(dst, src map[string]int) {
	for key, val := range src {
		dst[key] = dst[key] + val
	}
}

func runNpmAudit(targetPath string) (auditSummary, error) {
	if !commandAvailable("npm") {
		return auditSummary{}, fmt.Errorf("npm not found in PATH")
	}
	if !fileExists(filepath.Join(targetPath, "package.json")) {
		return auditSummary{}, fmt.Errorf("package.json not found in %s", targetPath)
	}

	cmd := exec.Command("npm", "audit", "--json")
	cmd.Dir = targetPath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()

	if len(output) == 0 && err != nil {
		return auditSummary{}, fmt.Errorf("npm audit failed: %s", strings.TrimSpace(stderr.String()))
	}
	counts, total, parseErr := parseNpmAuditOutput(output)
	if parseErr != nil {
		return auditSummary{}, parseErr
	}

	return auditSummary{
		Tool:   "npm",
		Path:   targetPath,
		Total:  total,
		Counts: counts,
		Mode:   "project",
	}, nil
}

type pipAuditEntry struct {
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Vulns   []pipAuditVuln `json:"vulns"`
}

type pipAuditVuln struct {
	ID          string   `json:"id"`
	Severity    string   `json:"severity"`
	FixVersions []string `json:"fix_versions"`
}

func runPythonAudit(targetPath string) (auditSummary, error) {
	mode := "environment"
	args := []string{"-f", "json"}
	reqPath := filepath.Join(targetPath, "requirements.txt")
	if fileExists(reqPath) {
		mode = "requirements.txt"
		args = append([]string{"-r", reqPath}, args...)
	}

	cmd, err := pythonAuditCommand(args)
	if err != nil {
		return auditSummary{}, err
	}
	cmd.Dir = targetPath
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if len(output) == 0 && err != nil {
		return auditSummary{}, fmt.Errorf("pip-audit failed: %s", strings.TrimSpace(stderr.String()))
	}

	var entries []pipAuditEntry
	entries, err = parsePipAuditOutput(output)
	if err != nil {
		return auditSummary{}, err
	}

	counts := map[string]int{}
	total := 0
	for _, entry := range entries {
		for _, vuln := range entry.Vulns {
			sev := strings.ToLower(strings.TrimSpace(vuln.Severity))
			if sev == "" {
				sev = "unknown"
			}
			counts[sev] = counts[sev] + 1
			total++
		}
	}

	return auditSummary{
		Tool:   "python",
		Path:   targetPath,
		Total:  total,
		Counts: counts,
		Mode:   mode,
	}, nil
}

func pythonAuditCommand(args []string) (*exec.Cmd, error) {
	if commandAvailable("pip-audit") {
		return exec.Command("pip-audit", args...), nil
	}
	if commandAvailable("python3") {
		return exec.Command("python3", append([]string{"-m", "pip_audit"}, args...)...), nil
	}
	if commandAvailable("python") {
		return exec.Command("python", append([]string{"-m", "pip_audit"}, args...)...), nil
	}
	return nil, fmt.Errorf("pip-audit not found (install with: python -m pip install pip-audit)")
}

func parsePipAuditOutput(output []byte) ([]pipAuditEntry, error) {
	start := bytes.IndexAny(output, "{[")
	if start < 0 {
		return nil, fmt.Errorf("parse pip-audit output: no JSON found")
	}

	trimmed := output[start:]
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	if trimmed[0] == '{' {
		wrapped := struct {
			Dependencies []pipAuditEntry `json:"dependencies"`
		}{}
		if err := decoder.Decode(&wrapped); err != nil {
			return nil, fmt.Errorf("parse pip-audit output: %w", err)
		}
		return wrapped.Dependencies, nil
	}

	var entries []pipAuditEntry
	if err := decoder.Decode(&entries); err != nil {
		return nil, fmt.Errorf("parse pip-audit output: %w", err)
	}
	return entries, nil
}

func logAuditSummary(logger *logging.Logger, summary auditSummary) {
	fields := map[string]any{
		"tool":  summary.Tool,
		"path":  summary.Path,
		"mode":  summary.Mode,
		"total": summary.Total,
	}
	if len(summary.Counts) > 0 {
		fields["counts"] = summary.Counts
	}

	if summary.Total > 0 {
		logger.Warn("package audit findings", fields)
	} else {
		logger.Info("package audit clean", fields)
	}
}

func logSessionAuditSummary(logger *logging.Logger, summary sessionAuditSummary) {
	fields := map[string]any{
		"session_id": summary.SessionID,
		"workspace":  summary.Workspace,
		"agent":      summary.Agent,
		"sessions":   summary.SessionCount,
		"total":      summary.Total,
	}
	if len(summary.RiskCounts) > 0 {
		fields["risk_counts"] = summary.RiskCounts
	}
	if len(summary.SourceCounts) > 0 {
		fields["source_counts"] = summary.SourceCounts
	}
	if len(summary.ExecutionCounts) > 0 {
		fields["execution_counts"] = summary.ExecutionCounts
	}
	if summary.Bypassed > 0 {
		fields["bypassed"] = summary.Bypassed
	}
	if summary.Blocked > 0 {
		fields["blocked"] = summary.Blocked
	}

	logger.Info("session audit summary", fields)
}

// logRepoAuditSummary prints a concise, high-level summary of the repo audit.
func logRepoAuditSummary(logger *logging.Logger, summary repoAuditSummary) {
	fields := map[string]any{
		"path":                 summary.Path,
		"code_findings_total":  summary.CodeFindingsTotal,
		"secrets_total":        summary.SecretsTotal,
		"code_by_severity":     summary.CodeBySeverity,
		"code_by_language":     summary.CodeByLanguage,
		"package_audits_total": len(summary.PackageAudits),
	}

	// Optionally embed per-tool package audit counts for quick inspection.
	if len(summary.PackageAudits) > 0 {
		pkgs := make(map[string]map[string]any)
		for _, a := range summary.PackageAudits {
			pkgs[a.Tool] = map[string]any{
				"total":  a.Total,
				"counts": a.Counts,
				"mode":   a.Mode,
			}
		}
		fields["package_audits"] = pkgs
	}

	logger.Info("repo audit summary", fields)
}

func getRemediation(code string) string {
	if s, ok := codeRemediation[code]; ok {
		return s
	}
	return ""
}

const (
	repoAuditTextCap = 50
	repoAuditJSONCap = 200
)

func printRepoAuditFindingsText(summary repoAuditSummary, cap int) {
	if cap <= 0 {
		cap = repoAuditTextCap
	}
	for i, f := range summary.CodeFindings {
		if i >= cap {
			fmt.Fprintf(os.Stdout, "... and %d more code findings\n", len(summary.CodeFindings)-cap)
			return
		}
		rem := getRemediation(f.Code)
		if rem != "" {
			fmt.Fprintf(os.Stdout, "%s:%d  %s  %s  → %s\n", f.File, f.Line, f.Code, f.Description, rem)
		} else {
			fmt.Fprintf(os.Stdout, "%s:%d  %s  %s\n", f.File, f.Line, f.Code, f.Description)
		}
	}
}

func emitRepoAuditMarkdown(summary repoAuditSummary) {
	fmt.Fprintln(os.Stdout, "# Repo Audit Report")
	fmt.Fprintf(os.Stdout, "\n**Path:** %s\n\n", summary.Path)
	fmt.Fprintln(os.Stdout, "## Code findings")
	fmt.Fprintln(os.Stdout, "| File | Line | Severity | Code | Description | Remediation |")
	fmt.Fprintln(os.Stdout, "|------|------|----------|------|-------------|-------------|")
	cap := repoAuditTextCap
	for i, f := range summary.CodeFindings {
		if i >= cap {
			fmt.Fprintf(os.Stdout, "| ... | ... | ... | ... | *… and %d more* | ... |\n", len(summary.CodeFindings)-cap)
			break
		}
		rem := getRemediation(f.Code)
		desc := strings.ReplaceAll(f.Description, "|", "\\|")
		remEsc := strings.ReplaceAll(rem, "|", "\\|")
		fmt.Fprintf(os.Stdout, "| %s | %d | %s | %s | %s | %s |\n",
			f.File, f.Line, f.Severity, f.Code, desc, remEsc)
	}
	fmt.Fprintf(os.Stdout, "\n## Secrets\n\n**Total:** %d\n\n", summary.SecretsTotal)
	fmt.Fprintln(os.Stdout, "## Dependencies")
	for _, a := range summary.PackageAudits {
		fmt.Fprintf(os.Stdout, "\n### %s\n\n**Total:** %d  \n**Mode:** %s\n\n", a.Tool, a.Total, a.Mode)
		if len(a.Counts) > 0 {
			fmt.Fprintln(os.Stdout, "| Severity | Count |")
			fmt.Fprintln(os.Stdout, "|----------|-------|")
			for sev, n := range a.Counts {
				fmt.Fprintf(os.Stdout, "| %s | %d |\n", sev, n)
			}
		}
	}
}

type repoAuditJSONOut struct {
	Path           string             `json:"path"`
	CodeFindings   []codeFindingJSON  `json:"code_findings"`
	CodeBySeverity map[string]int     `json:"code_by_severity"`
	SecretsTotal   int                `json:"secrets_total"`
	PackageAudits  []auditSummaryJSON `json:"package_audits"`
}

type codeFindingJSON struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Severity    string `json:"severity"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Remediation string `json:"remediation"`
}

type auditSummaryJSON struct {
	Tool   string         `json:"tool"`
	Path   string         `json:"path"`
	Total  int            `json:"total"`
	Counts map[string]int `json:"counts"`
	Mode   string         `json:"mode"`
}

func emitRepoAuditJSON(summary repoAuditSummary) {
	findings := summary.CodeFindings
	if len(findings) > repoAuditJSONCap {
		findings = findings[:repoAuditJSONCap]
	}
	codeList := make([]codeFindingJSON, 0, len(findings))
	for _, f := range findings {
		codeList = append(codeList, codeFindingJSON{
			File:        f.File,
			Line:        f.Line,
			Severity:    f.Severity,
			Code:        f.Code,
			Description: f.Description,
			Remediation: getRemediation(f.Code),
		})
	}
	pkgList := make([]auditSummaryJSON, 0, len(summary.PackageAudits))
	for _, a := range summary.PackageAudits {
		pkgList = append(pkgList, auditSummaryJSON{
			Tool:   a.Tool,
			Path:   a.Path,
			Total:  a.Total,
			Counts: a.Counts,
			Mode:   a.Mode,
		})
	}
	out := repoAuditJSONOut{
		Path:           summary.Path,
		CodeFindings:   codeList,
		CodeBySeverity: summary.CodeBySeverity,
		SecretsTotal:   summary.SecretsTotal,
		PackageAudits:  pkgList,
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}

func ensureNpm(ctx context.Context) error {
	if commandAvailable("npm") {
		return nil
	}
	logger := logging.FromContext(ctx)
	logger.Info("npm not found; attempting install", nil)

	switch runtime.GOOS {
	case "darwin":
		if !commandAvailable("brew") {
			return fmt.Errorf("npm not found and Homebrew missing (install Node.js first)")
		}
		return runInstallCommand("brew", "install", "node")
	case "linux":
		if commandAvailable("apt-get") {
			if err := runInstallCommand("sudo", "apt-get", "update", "-y"); err != nil {
				return err
			}
			return runInstallCommand("sudo", "apt-get", "install", "-y", "nodejs", "npm")
		}
		if commandAvailable("dnf") {
			return runInstallCommand("sudo", "dnf", "install", "-y", "nodejs", "npm")
		}
		if commandAvailable("yum") {
			return runInstallCommand("sudo", "yum", "install", "-y", "nodejs", "npm")
		}
	}
	return fmt.Errorf("npm not found (install Node.js/npm first)")
}

func ensurePipAudit(ctx context.Context) error {
	if commandAvailable("pip-audit") {
		return nil
	}
	if commandAvailable("python3") || commandAvailable("python") {
		logger := logging.FromContext(ctx)
		logger.Info("pip-audit not found; attempting install", nil)
		if commandAvailable("python3") {
			return runInstallCommand("python3", "-m", "pip", "install", "--user", "pip-audit")
		}
		return runInstallCommand("python", "-m", "pip", "install", "--user", "pip-audit")
	}
	return fmt.Errorf("pip-audit not found (install with: python -m pip install pip-audit)")
}

func runInstallCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s failed: %s", name, strings.TrimSpace(string(output)))
	}
	return nil
}

func parseNpmAuditOutput(output []byte) (map[string]int, int, error) {
	report := struct {
		Metadata struct {
			Vulnerabilities map[string]int `json:"vulnerabilities"`
		} `json:"metadata"`
		Error interface{} `json:"error"`
	}{}
	if parseErr := json.Unmarshal(output, &report); parseErr != nil {
		return nil, 0, fmt.Errorf("parse npm audit output: %w", parseErr)
	}

	counts := map[string]int{}
	total := 0
	for sev, count := range report.Metadata.Vulnerabilities {
		counts[sev] = count
		total += count
	}
	return counts, total, nil
}
