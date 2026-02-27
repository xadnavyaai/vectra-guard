package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/vectra-guard/vectra-guard/internal/logging"
	"github.com/vectra-guard/vectra-guard/internal/sandbox"
	"github.com/vectra-guard/vectra-guard/internal/session"
)

// maxSSEClients limits the number of concurrent SSE connections to prevent
// goroutine exhaustion from a local DoS.
const maxSSEClients = 32

// sseEventNameRe allows only safe characters in SSE event type names.
var sseEventNameRe = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// Event represents a server-sent event pushed to dashboard clients.
type Event struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}

// EventHub manages SSE clients and polls for file changes to broadcast events.
type EventHub struct {
	logger     *logging.Logger
	clients    map[chan Event]struct{}
	mu         sync.RWMutex
	register   chan chan Event
	unregister chan chan Event
	broadcast  chan Event
}

// NewEventHub creates a new event hub for SSE broadcasting.
func NewEventHub(logger *logging.Logger) *EventHub {
	return &EventHub{
		logger:     logger,
		clients:    make(map[chan Event]struct{}),
		register:   make(chan chan Event),
		unregister: make(chan chan Event),
		broadcast:  make(chan Event, 64),
	}
}

// Run starts the event hub loop. It manages client registration and polls
// the sessions directory and metrics file for changes every 2 seconds.
func (h *EventHub) Run(sessionM *session.Manager, metrics *sandbox.MetricsCollector) {
	// Track last known state for change detection
	var lastSessionCount int
	var lastMetricsTotal int64
	var lastSessionModTimes map[string]time.Time

	pollTicker := time.NewTicker(2 * time.Second)
	defer pollTicker.Stop()

	// Resolve sessions directory
	sessionDir := resolveSessionDir()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = struct{}{}
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			delete(h.clients, client)
			close(client)
			h.mu.Unlock()

		case evt := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client <- evt:
				default:
					// Client buffer full, skip
				}
			}
			h.mu.RUnlock()

		case <-pollTicker.C:
			// Poll for session changes
			h.pollSessions(sessionM, sessionDir, &lastSessionCount, &lastSessionModTimes)

			// Poll for metrics changes
			h.pollMetrics(metrics, &lastMetricsTotal)
		}
	}
}

func (h *EventHub) pollSessions(sessionM *session.Manager, sessionDir string, lastCount *int, lastModTimes *map[string]time.Time) {
	if sessionDir == "" {
		return
	}

	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return
	}

	currentModTimes := make(map[string]time.Time)
	currentCount := 0

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		currentCount++
		info, err := entry.Info()
		if err != nil {
			continue
		}
		currentModTimes[entry.Name()] = info.ModTime()
	}

	if *lastModTimes == nil {
		*lastModTimes = currentModTimes
		*lastCount = currentCount
		return
	}

	// Detect new sessions
	if currentCount > *lastCount {
		for name := range currentModTimes {
			if _, existed := (*lastModTimes)[name]; !existed {
				sessionID := name[:len(name)-5]
				sess, err := sessionM.Load(sessionID)
				if err == nil {
					h.broadcast <- Event{
						Type:      "session_started",
						Timestamp: time.Now(),
						Data: map[string]interface{}{
							"session_id": sess.ID,
							"agent":      sess.AgentName,
							"workspace":  sess.Workspace,
						},
					}
				}
			}
		}
	}

	// Detect modified sessions (new commands added)
	for name, modTime := range currentModTimes {
		if oldTime, existed := (*lastModTimes)[name]; existed && modTime.After(oldTime) {
			sessionID := name[:len(name)-5]
			sess, err := sessionM.Load(sessionID)
			if err != nil || len(sess.Commands) == 0 {
				continue
			}
			lastCmd := sess.Commands[len(sess.Commands)-1]
			evtType := "command_executed"
			if !lastCmd.Approved && (lastCmd.RiskLevel == "critical" || lastCmd.RiskLevel == "high") {
				evtType = "command_blocked"
			}

			h.broadcast <- Event{
				Type:      evtType,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"session_id": sess.ID,
					"agent":      sess.AgentName,
					"command":    lastCmd.Command,
					"args":       lastCmd.Args,
					"risk_level": lastCmd.RiskLevel,
					"approved":   lastCmd.Approved,
					"exit_code":  lastCmd.ExitCode,
				},
			}

			// Detect session ended
			if sess.EndTime != nil {
				h.broadcast <- Event{
					Type:      "session_ended",
					Timestamp: time.Now(),
					Data: map[string]interface{}{
						"session_id": sess.ID,
						"agent":      sess.AgentName,
						"risk_score": sess.RiskScore,
						"commands":   len(sess.Commands),
					},
				}
			}
		}
	}

	*lastModTimes = currentModTimes
	*lastCount = currentCount
}

func (h *EventHub) pollMetrics(metrics *sandbox.MetricsCollector, lastTotal *int64) {
	m := metrics.GetMetrics()
	if m.TotalExecutions > *lastTotal && *lastTotal > 0 {
		// New executions detected via metrics
		if len(m.ExecutionHistory) > 0 {
			lastRec := m.ExecutionHistory[len(m.ExecutionHistory)-1]
			h.broadcast <- Event{
				Type:      "metric_recorded",
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"command":    lastRec.Command,
					"mode":       lastRec.Mode,
					"risk_level": lastRec.RiskLevel,
					"duration":   lastRec.Duration.String(),
					"exit_code":  lastRec.ExitCode,
				},
			}
		}
	}
	*lastTotal = m.TotalExecutions
}

// ServeHTTP handles SSE connections from clients.
func (h *EventHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Enforce maximum concurrent SSE clients to prevent goroutine exhaustion.
	h.mu.RLock()
	clientCount := len(h.clients)
	h.mu.RUnlock()
	if clientCount >= maxSSEClients {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// No CORS header — dashboard is same-origin, no cross-origin access needed.

	client := make(chan Event, 16)
	h.register <- client

	// Send initial connection event
	writeSSE(w, Event{
		Type:      "connected",
		Timestamp: time.Now(),
		Data:      map[string]string{"status": "connected"},
	})
	flusher.Flush()

	// Clean up on disconnect
	ctx := r.Context()
	defer func() {
		h.unregister <- client
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-client:
			if !ok {
				return
			}
			writeSSE(w, evt)
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, evt Event) {
	// Sanitize event type: only allow safe alphanumeric + underscore names.
	// This prevents SSE frame injection via newlines or control characters.
	evtType := evt.Type
	if !sseEventNameRe.MatchString(evtType) {
		evtType = "unknown"
	}

	data, err := json.Marshal(evt)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", evtType, data)
}

func resolveSessionDir() string {
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return ""
		}
	}
	return filepath.Join(homeDir, ".vectra-guard", "sessions")
}
