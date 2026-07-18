package behavioral

import (
	"fmt"
	"sort"

	"github.com/vectra-guard/vectra-guard/internal/session"
)

// LoadTimedActions reads a session from disk using the existing
// session.Manager.Load() API and returns a timestamp-ordered stream of
// categorized TimedActions. The adapter does NOT re-parse session
// JSON by hand — all reads go through the session package.
func LoadTimedActions(mgr *session.Manager, sessionID string) ([]TimedAction, error) {
	if mgr == nil {
		return nil, fmt.Errorf("session manager is nil")
	}
	sess, err := mgr.Load(sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session %s: %w", sessionID, err)
	}
	return actionsFromSession(sess), nil
}

// actionsFromSession merges a session's Commands and FileOps into a
// single timestamp-ordered stream of categorized TimedActions. Split
// out of LoadTimedActions so tests can exercise it with an in-memory
// fixture (no disk).
func actionsFromSession(sess *session.Session) []TimedAction {
	if sess == nil {
		return nil
	}
	actions := make([]TimedAction, 0, len(sess.Commands)+len(sess.FileOps))
	for _, cmd := range sess.Commands {
		actions = append(actions, TimedAction{
			Timestamp: cmd.Timestamp,
			Category:  CategorizeCommand(cmd),
			Source:    "command",
			Label:     cmd.Command,
		})
	}
	for _, op := range sess.FileOps {
		actions = append(actions, TimedAction{
			Timestamp: op.Timestamp,
			Category:  CategorizeFileOp(op),
			Source:    "file_op",
			Label:     op.Operation + ":" + op.Path,
		})
	}
	sort.SliceStable(actions, func(i, j int) bool {
		return actions[i].Timestamp.Before(actions[j].Timestamp)
	})
	return actions
}
