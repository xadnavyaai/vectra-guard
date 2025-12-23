package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
	Timestamp   time.Time              `json:"timestamp"`
	Command     string                 `json:"command"`
	Args        []string               `json:"args"`
	ExitCode    int                    `json:"exit_code"`
	Output      string                 `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
	RiskLevel   string                 `json:"risk_level"`
	Approved    bool                   `json:"approved"`
	ApprovedBy  string                 `json:"approved_by,omitempty"`
	Findings    []string               `json:"findings,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
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
	sessionDir := filepath.Join(workspace, ".vectra-guard", "sessions")
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
		"session_id":  session.ID,
		"agent":       agentName,
		"workspace":   workspace,
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
	return fmt.Sprintf("session-%d", time.Now().UnixNano())
}

// GetCurrentSession returns the active session ID from environment.
func GetCurrentSession() string {
	return os.Getenv("VECTRAGUARD_SESSION_ID")
}

// SetCurrentSession sets the active session in the environment.
func SetCurrentSession(sessionID string) {
	os.Setenv("VECTRAGUARD_SESSION_ID", sessionID)
}

