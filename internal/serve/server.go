package serve

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/config"
	"github.com/vectra-guard/vectra-guard/internal/cve"
	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/sandbox"
	"github.com/vectra-guard/vectra-guard/internal/seed"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

// validSessionID matches the session-{UnixNano} format used by the session manager.
// Only alphanumeric characters and hyphens are allowed to prevent path traversal.
var validSessionID = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-]*$`)

// Server wraps the HTTP server for the local dashboard.
type Server struct {
	logger    *logging.Logger
	sessionM  *session.Manager
	metrics   *sandbox.MetricsCollector
	trust     *sandbox.TrustStore
	cveStore  *cve.Store
	cfg       config.Config
	mux       *http.ServeMux
	eventHub  *EventHub
	startTime time.Time
	workspace string
}

// New creates a new dashboard server bound to the given workspace.
func New(workspace string, logger *logging.Logger, cfg config.Config, trust *sandbox.TrustStore, cveStore *cve.Store) (*Server, error) {
	mgr, err := session.NewManager(workspace, logger)
	if err != nil {
		return nil, fmt.Errorf("create session manager: %w", err)
	}
	metrics, err := sandbox.NewMetricsCollector("", cfg.Sandbox.EnableMetrics)
	if err != nil {
		return nil, fmt.Errorf("create metrics collector: %w", err)
	}

	hub := NewEventHub(logger)

	s := &Server{
		logger:    logger,
		sessionM:  mgr,
		metrics:   metrics,
		trust:     trust,
		cveStore:  cveStore,
		cfg:       cfg,
		mux:       http.NewServeMux(),
		eventHub:  hub,
		startTime: time.Now(),
		workspace: workspace,
	}
	s.routes()
	return s, nil
}

// routes registers HTTP handlers.
func (s *Server) routes() {
	// Dashboard
	s.mux.HandleFunc("/", s.handleDashboard)

	// Existing APIs
	s.mux.HandleFunc("/api/sessions", s.handleSessions)
	s.mux.HandleFunc("/api/metrics", s.handleMetrics)

	// New APIs
	s.mux.HandleFunc("/api/sessions/", s.handleSessionByID)
	s.mux.HandleFunc("/api/trust", s.handleTrust)
	s.mux.HandleFunc("/api/cve", s.handleCVE)
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/agents", s.handleAgents)
	s.mux.HandleFunc("/api/events", s.handleEvents)
}

// ListenAndServe starts the HTTP server bound to 127.0.0.1:port.
func (s *Server) ListenAndServe(port int) error {
	// Start the event hub for SSE
	go s.eventHub.Run(s.sessionM, s.metrics)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	s.logger.Info("serve dashboard started", map[string]any{
		"addr": addr,
	})
	srv := &http.Server{
		Handler:           securityHeaders(s.mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
		// WriteTimeout intentionally omitted: SSE requires long-lived responses
	}
	return srv.Serve(ln)
}

// securityHeaders wraps a handler to inject security response headers on every request.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; "+
				"script-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline'; "+
				"connect-src 'self'; "+
				"img-src 'self' data:; "+
				"font-src 'self'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	ServeDashboard(w, r)
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sessions, err := s.sessionM.List()
	if err != nil {
		s.logger.Error("list sessions failed", map[string]any{"error": err.Error()})
		http.Error(w, "failed to list sessions", http.StatusInternalServerError)
		return
	}
	writeJSON(w, sessions)
}

func (s *Server) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract session ID from /api/sessions/{id}
	id := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if id == "" {
		http.Error(w, "missing session id", http.StatusBadRequest)
		return
	}

	// Validate session ID format to prevent path traversal.
	// Session IDs are "session-{UnixNano}" — only alphanumeric + hyphens.
	if !validSessionID.MatchString(id) || len(id) > 64 {
		http.Error(w, "invalid session id", http.StatusBadRequest)
		return
	}

	sess, err := s.sessionM.Load(id)
	if err != nil {
		s.logger.Error("load session failed", map[string]any{
			"session_id": id,
			"error":      err.Error(),
		})
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	writeJSON(w, sess)
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.metrics.GetMetrics())
}

func (s *Server) handleTrust(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.trust == nil {
		writeJSON(w, []interface{}{})
		return
	}
	entries := s.trust.List()
	if entries == nil {
		entries = []*sandbox.TrustEntry{}
	}
	writeJSON(w, entries)
}

func (s *Server) handleCVE(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.cveStore == nil {
		writeJSON(w, cveSummary{})
		return
	}

	summary := buildCVESummary(s.cveStore)
	writeJSON(w, summary)
}

type cveSummary struct {
	TotalPackages int                `json:"total_packages"`
	TotalVulns    int                `json:"total_vulns"`
	BySeverity    map[string]int     `json:"by_severity"`
	Entries       []cvePackageEntry  `json:"entries"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

type cvePackageEntry struct {
	Ecosystem       string              `json:"ecosystem"`
	Name            string              `json:"name"`
	Version         string              `json:"version"`
	Vulnerabilities []cveVulnEntry      `json:"vulnerabilities"`
}

type cveVulnEntry struct {
	ID       string  `json:"id"`
	Severity string  `json:"severity"`
	Summary  string  `json:"summary"`
	CVSS     float64 `json:"cvss"`
}

func buildCVESummary(store *cve.Store) cveSummary {
	s := cveSummary{
		BySeverity: make(map[string]int),
		Entries:    []cvePackageEntry{},
		UpdatedAt:  store.Cache.UpdatedAt,
	}

	for _, pv := range store.Cache.Entries {
		s.TotalPackages++
		entry := cvePackageEntry{
			Ecosystem:       pv.Package.Ecosystem,
			Name:            pv.Package.Name,
			Version:         pv.Package.Version,
			Vulnerabilities: []cveVulnEntry{},
		}
		for _, v := range pv.Vulnerabilities {
			s.TotalVulns++
			sev := v.Severity
			if sev == "" {
				sev = "unknown"
			}
			s.BySeverity[sev]++
			entry.Vulnerabilities = append(entry.Vulnerabilities, cveVulnEntry{
				ID:       v.ID,
				Severity: sev,
				Summary:  v.Summary,
				CVSS:     v.CVSS,
			})
		}
		s.Entries = append(s.Entries, entry)
	}

	return s
}

type statusResponse struct {
	Version    string            `json:"version"`
	GuardLevel string            `json:"guard_level"`
	Sandbox    statusSandbox     `json:"sandbox"`
	Features   map[string]bool   `json:"features"`
	DataDir    string            `json:"data_dir"`
	Uptime     string            `json:"uptime"`
	StartedAt  time.Time         `json:"started_at"`
}

type statusSandbox struct {
	Enabled       bool   `json:"enabled"`
	Mode          string `json:"mode"`
	SecurityLevel string `json:"security_level"`
	Runtime       string `json:"runtime"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	homeDir, _ := os.UserHomeDir()
	dataDir := filepath.Join(homeDir, ".vectra-guard")

	resp := statusResponse{
		Version:    getVersion(),
		GuardLevel: string(s.cfg.GuardLevel.Level),
		Sandbox: statusSandbox{
			Enabled:       s.cfg.Sandbox.Enabled,
			Mode:          string(s.cfg.Sandbox.Mode),
			SecurityLevel: string(s.cfg.Sandbox.SecurityLevel),
			Runtime:       s.cfg.Sandbox.Runtime,
		},
		Features: map[string]bool{
			"sandbox":     s.cfg.Sandbox.Enabled,
			"metrics":     s.cfg.Sandbox.EnableMetrics,
			"cve":         s.cfg.CVE.Enabled,
			"soft_delete": s.cfg.SoftDelete.Enabled,
		},
		DataDir:   dataDir,
		Uptime:    time.Since(s.startTime).Round(time.Second).String(),
		StartedAt: s.startTime,
	}

	writeJSON(w, resp)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	s.eventHub.ServeHTTP(w, r)
}

// Agent coverage types
type agentCoverageResponse struct {
	Agents   []agentEntry        `json:"agents"`
	Covered  int                 `json:"covered"`
	Total    int                 `json:"total"`
	OpenClaw *openClawInfo       `json:"openclaw,omitempty"`
}

type agentEntry struct {
	Key      string `json:"key"`
	DestPath string `json:"dest_path"`
	Exists   bool   `json:"exists"`
	Size     int64  `json:"size"`
	ModTime  string `json:"mod_time,omitempty"`
	Age      string `json:"age,omitempty"`
}

type openClawInfo struct {
	Installed    bool     `json:"installed"`
	StateDir     string   `json:"state_dir"`
	Source       string   `json:"source"`
	ConfigExists bool    `json:"config_exists"`
	AgentDirs    []string `json:"agent_dirs,omitempty"`
	SeedPaths    []string `json:"seed_paths,omitempty"`
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	statuses := seed.ScanAgentFiles(s.workspace)
	resp := agentCoverageResponse{
		Agents: make([]agentEntry, 0, len(statuses)),
		Total:  len(statuses),
	}

	for _, st := range statuses {
		entry := agentEntry{
			Key:      st.Key,
			DestPath: st.DestPath,
			Exists:   st.Exists,
			Size:     st.Size,
		}
		if st.Exists {
			resp.Covered++
			entry.ModTime = st.ModTime.Format(time.RFC3339)
			entry.Age = agentAge(st.ModTime)
		}
		resp.Agents = append(resp.Agents, entry)
	}

	// OpenClaw detection
	det := seed.DetectOpenClaw()
	if det.Installed {
		oc := &openClawInfo{
			Installed:    true,
			StateDir:     det.StateDir,
			Source:       det.Source,
			ConfigExists: det.ConfigExists,
			AgentDirs:    det.AgentDirs,
			SeedPaths:    seed.OpenClawSeedPaths(det),
		}
		resp.OpenClaw = oc

		// Check if OpenClaw AGENTS.md exists at the detected path
		ocPath := filepath.Join(det.StateDir, "AGENTS.md")
		if info, err := os.Stat(ocPath); err == nil {
			// Update the openclaw entry in the agents list with real path info
			for i := range resp.Agents {
				if resp.Agents[i].Key == "openclaw" {
					if !resp.Agents[i].Exists {
						resp.Agents[i].Exists = true
						resp.Agents[i].Size = info.Size()
						resp.Agents[i].ModTime = info.ModTime().Format(time.RFC3339)
						resp.Agents[i].Age = agentAge(info.ModTime())
						resp.Agents[i].DestPath = ocPath
						resp.Covered++
					}
					break
				}
			}
		}
	}

	writeJSON(w, resp)
}

func agentAge(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	_ = enc.Encode(v)
}

// versionFunc is set by cmd/serve.go to provide the binary version.
var versionFunc func() string

// SetVersionFunc sets the function used to retrieve the binary version.
func SetVersionFunc(fn func() string) {
	versionFunc = fn
}

func getVersion() string {
	if versionFunc != nil {
		return versionFunc()
	}
	return "dev"
}

// Helper for basic environment workspace detection (mainly used by CLI).
func DefaultWorkspace() (string, error) {
	if wd, err := os.Getwd(); err == nil {
		return wd, nil
	}
	return "", fmt.Errorf("unable to determine workspace")
}
