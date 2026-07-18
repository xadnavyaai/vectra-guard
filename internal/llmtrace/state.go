package llmtrace

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StateStore manages sync state persistence.
type StateStore struct {
	Path  string
	State SyncState
}

// DefaultStatePath returns the default path for llmtrace sync state.
func DefaultStatePath() string {
	home := os.Getenv("HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			home = "."
		}
	}
	return filepath.Join(home, ".vectra-guard", "llmtrace", "sync-state.json")
}

// LoadStateStore loads sync state from disk or creates an empty one.
func LoadStateStore(path string) (*StateStore, error) {
	state := SyncState{}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &StateStore{Path: path, State: state}, nil
		}
		return nil, fmt.Errorf("read llmtrace state: %w", err)
	}
	if len(data) == 0 {
		return &StateStore{Path: path, State: state}, nil
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse llmtrace state: %w", err)
	}
	return &StateStore{Path: path, State: state}, nil
}

// Save persists the sync state to disk.
func (s *StateStore) Save() error {
	s.State.LastSyncTime = time.Now().UTC()
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o755); err != nil {
		return fmt.Errorf("create llmtrace state dir: %w", err)
	}
	data, err := json.MarshalIndent(s.State, "", "  ")
	if err != nil {
		return fmt.Errorf("encode llmtrace state: %w", err)
	}
	if err := os.WriteFile(s.Path, data, 0o644); err != nil {
		return fmt.Errorf("write llmtrace state: %w", err)
	}
	return nil
}
