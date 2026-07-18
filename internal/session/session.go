package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/logging"
)

// Session represents an agent's tracked activity session.
type Session struct {
	ID         string                 `json:"id"`
	AgentName  string                 `json:"agent_name"`
	Workspace  string                 `json:"workspace"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Commands   []Command              `json:"commands"`
	FileOps    []FileOperation        `json:"file_operations"`
	RiskScore  int                    `json:"risk_score"`
	Violations int                    `json:"violations"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// Command represents a single command execution in a session.
type Command struct {
	Timestamp  time.Time              `json:"timestamp"`
	Command    string                 `json:"command"`
	Args       []string               `json:"args"`
	ExitCode   int                    `json:"exit_code"`
	Output     string                 `json:"output,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Duration   time.Duration          `json:"duration"`
	RiskLevel  string                 `json:"risk_level"`
	Approved   bool                   `json:"approved"`
	ApprovedBy string                 `json:"approved_by,omitempty"`
	Findings   []string               `json:"findings,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// FileOperation represents a file system operation.
type FileOperation struct {
	Timestamp time.Time `json:"timestamp"`
	Operation string    `json:"operation"` // create, modify, delete, read
	Path      string    `json:"path"`
	Size      int64     `json:"size,omitempty"`
	RiskLevel string    `json:"risk_level"`
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason,omitempty"`
}

// Manager handles session lifecycle and persistence.
type Manager struct {
	sessionDir string
	logger     *logging.Logger
}

// NewManager creates a new session manager.
func NewManager(workspace string, logger *logging.Logger) (*Manager, error) {
	// Always use home directory for global session storage (respect HOME when set)
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
	}
	sessionDir := filepath.Join(homeDir, ".vectra-guard", "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return nil, fmt.Errorf("create session directory: %w", err)
	}
	return &Manager{
		sessionDir: sessionDir,
		logger:     logger,
	}, nil
}

// Start creates and saves a new session.
func (m *Manager) Start(agentName, workspace string) (*Session, error) {
	session := &Session{
		ID:        generateSessionID(),
		AgentName: agentName,
		Workspace: workspace,
		StartTime: time.Now(),
		Commands:  []Command{},
		FileOps:   []FileOperation{},
		Metadata:  make(map[string]interface{}),
	}

	if err := m.save(session); err != nil {
		return nil, err
	}

	m.logger.Info("session started", map[string]any{
		"session_id": session.ID,
		"agent":      agentName,
		"workspace":  workspace,
	})

	return session, nil
}

// Load retrieves an existing session.
func (m *Manager) Load(sessionID string) (*Session, error) {
	path := filepath.Join(m.sessionDir, sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read session: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}

	return &session, nil
}

// End marks a session as complete and saves it.
func (m *Manager) End(session *Session) error {
	now := time.Now()
	session.EndTime = &now

	if err := m.save(session); err != nil {
		return err
	}

	duration := now.Sub(session.StartTime)
	m.logger.Info("session ended", map[string]any{
		"session_id": session.ID,
		"duration":   duration.String(),
		"commands":   len(session.Commands),
		"violations": session.Violations,
		"risk_score": session.RiskScore,
	})

	return nil
}

// AddCommand appends a command to the session and updates risk score.
func (m *Manager) AddCommand(session *Session, cmd Command) error {
	session.Commands = append(session.Commands, cmd)

	// Update risk score based on command risk level
	switch cmd.RiskLevel {
	case "critical":
		session.RiskScore += 100
		session.Violations++
	case "high":
		session.RiskScore += 50
		session.Violations++
	case "medium":
		session.RiskScore += 10
	}

	return m.save(session)
}

// AddFileOperation appends a file operation to the session.
func (m *Manager) AddFileOperation(session *Session, op FileOperation) error {
	session.FileOps = append(session.FileOps, op)

	if !op.Allowed {
		session.Violations++
		session.RiskScore += 25
	}

	return m.save(session)
}

// List returns all sessions in the workspace.
func (m *Manager) List() ([]*Session, error) {
	entries, err := os.ReadDir(m.sessionDir)
	if err != nil {
		return nil, fmt.Errorf("read session directory: %w", err)
	}

	var sessions []*Session
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		sessionID := entry.Name()[:len(entry.Name())-5]
		session, err := m.Load(sessionID)
		if err != nil {
			m.logger.Warn("failed to load session", map[string]any{
				"session_id": sessionID,
				"error":      err.Error(),
			})
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// save persists the session to disk.
func (m *Manager) save(session *Session) error {
	path := filepath.Join(m.sessionDir, session.ID+".json")
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write session: %w", err)
	}

	return nil
}

// generateSessionID creates a unique session identifier.
func generateSessionID() string {
	// A timestamp alone is not collision-safe: on platforms with a
	// low-resolution wall clock (notably Windows) two sessions started in
	// quick succession can share the same UnixNano value, producing
	// duplicate IDs that clobber each other's session file. Append random
	// bytes so IDs stay unique regardless of clock granularity.
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand should never fail; fall back to the raw timestamp.
		return fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("session-%d-%s", time.Now().UnixNano(), hex.EncodeToString(b[:]))
}

// GetCurrentSession returns the active session ID from environment.
func GetCurrentSession() string {
	return os.Getenv("VECTRAGUARD_SESSION_ID")
}

// SetCurrentSession sets the active session in the environment.
func SetCurrentSession(sessionID string) {
	os.Setenv("VECTRAGUARD_SESSION_ID", sessionID)
}

// GetCurrentSessionForWorkspace resolves the session ID by checking:
// env var (validated against workspace), then global per-workspace index,
// and finally a global "last session" file for fallback cases.
func GetCurrentSessionForWorkspace(workdir string) string {
	normalized := normalizeWorkspacePath(workdir)
	if sessionID := GetCurrentSession(); sessionID != "" {
		if normalized == "" || sessionMatchesWorkspace(sessionID, normalized) {
			return sessionID
		}
	}

	if normalized != "" {
		index := readSessionIndex()
		if sessionID := index[normalized]; sessionID != "" {
			return sessionID
		}
	}

	if normalized == "" {
		if path := globalSessionFilePath(); path != "" {
			return readSessionIDFromFile(path)
		}
	}

	return ""
}

// SetCurrentSessionForWorkspace stores the session in env and syncs to global index.
// File writes are best-effort; failures do not block execution.
func SetCurrentSessionForWorkspace(workdir, sessionID string) {
	if sessionID == "" {
		return
	}
	SetCurrentSession(sessionID)

	normalized := normalizeWorkspacePath(workdir)
	if normalized != "" {
		index := readSessionIndex()
		index[normalized] = sessionID
		_ = writeSessionIndex(index)
	}

	if path := globalSessionFilePath(); path != "" {
		_ = writeSessionIDToFile(path, sessionID)
	}
}

func normalizeWorkspacePath(workdir string) string {
	if workdir == "" {
		return ""
	}
	abs, err := filepath.Abs(workdir)
	if err == nil {
		workdir = abs
	}
	if real, err := filepath.EvalSymlinks(workdir); err == nil {
		workdir = real
	}
	return filepath.Clean(workdir)
}

func sessionMatchesWorkspace(sessionID, workdir string) bool {
	sessionPath := filepath.Join(sessionDirPath(), sessionID+".json")
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return false
	}
	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return false
	}
	return normalizeWorkspacePath(sess.Workspace) == workdir
}

func sessionDirPath() string {
	if homeEnv := os.Getenv("HOME"); homeEnv != "" {
		return filepath.Join(homeEnv, ".vectra-guard", "sessions")
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".vectra-guard", "sessions")
	}
	return ""
}

func sessionIndexPath() string {
	if homeEnv := os.Getenv("HOME"); homeEnv != "" {
		return filepath.Join(homeEnv, ".vectra-guard", "session-index.json")
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".vectra-guard", "session-index.json")
	}
	return ""
}

func globalSessionFilePath() string {
	if homeEnv := os.Getenv("HOME"); homeEnv != "" {
		return filepath.Join(homeEnv, ".vectra-guard-session")
	}
	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".vectra-guard-session")
	}
	return ""
}

func readSessionIndex() map[string]string {
	path := sessionIndexPath()
	if path == "" {
		return map[string]string{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return map[string]string{}
	}
	var index map[string]string
	if err := json.Unmarshal(data, &index); err != nil {
		return map[string]string{}
	}
	if index == nil {
		return map[string]string{}
	}
	return index
}

func writeSessionIndex(index map[string]string) error {
	path := sessionIndexPath()
	if path == "" {
		return nil
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func readSessionIDFromFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return ""
	}
	last := strings.TrimSpace(lines[len(lines)-1])
	return last
}

func writeSessionIDToFile(path, sessionID string) error {
	if path == "" || sessionID == "" {
		return nil
	}
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(sessionID+"\n"), 0o644)
}
