package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TrustEntry represents a trusted command approval
type TrustEntry struct {
	CommandHash  string    `json:"command_hash"`
	Command      string    `json:"command"`
	ApprovedAt   time.Time `json:"approved_at"`
	ApprovedBy   string    `json:"approved_by"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"`
	UseCount     int       `json:"use_count"`
	LastUsed     time.Time `json:"last_used"`
	Tags         []string  `json:"tags,omitempty"`
	Note         string    `json:"note,omitempty"`
}

// TrustStore manages approved and remembered commands
type TrustStore struct {
	path    string
	entries map[string]*TrustEntry
	mu      sync.RWMutex
}

// NewTrustStore creates or loads a trust store
func NewTrustStore(path string) (*TrustStore, error) {
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		path = filepath.Join(homeDir, ".vectra-guard", "trust.json")
	}
	
	store := &TrustStore{
		path:    path,
		entries: make(map[string]*TrustEntry),
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create trust store directory: %w", err)
	}
	
	// Load existing entries
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load trust store: %w", err)
	}
	
	return store, nil
}

// IsTrusted checks if a command is in the trust store
func (ts *TrustStore) IsTrusted(command string) bool {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	hash := hashCommand(command)
	entry, exists := ts.entries[hash]
	
	if !exists {
		return false
	}
	
	// Check if entry has expired
	if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
		return false
	}
	
	return true
}

// Add adds a command to the trust store
func (ts *TrustStore) Add(command string, duration time.Duration, note string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	hash := hashCommand(command)
	
	entry := &TrustEntry{
		CommandHash: hash,
		Command:     command,
		ApprovedAt:  time.Now(),
		ApprovedBy:  getCurrentUser(),
		UseCount:    0,
		LastUsed:    time.Time{},
		Note:        note,
	}
	
	if duration != 0 {
		entry.ExpiresAt = time.Now().Add(duration)
	}
	
	ts.entries[hash] = entry
	
	return ts.save()
}

// RecordUse increments the use count for a trusted command
func (ts *TrustStore) RecordUse(command string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	hash := hashCommand(command)
	entry, exists := ts.entries[hash]
	
	if !exists {
		return fmt.Errorf("command not in trust store")
	}
	
	entry.UseCount++
	entry.LastUsed = time.Now()
	
	return ts.save()
}

// Remove removes a command from the trust store
func (ts *TrustStore) Remove(command string) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	hash := hashCommand(command)
	delete(ts.entries, hash)
	
	return ts.save()
}

// List returns all trust entries
func (ts *TrustStore) List() []*TrustEntry {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	
	entries := make([]*TrustEntry, 0, len(ts.entries))
	for _, entry := range ts.entries {
		// Skip expired entries
		if !entry.ExpiresAt.IsZero() && time.Now().After(entry.ExpiresAt) {
			continue
		}
		entries = append(entries, entry)
	}
	
	return entries
}

// CleanExpired removes expired entries
func (ts *TrustStore) CleanExpired() error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	
	now := time.Now()
	modified := false
	
	for hash, entry := range ts.entries {
		if !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt) {
			delete(ts.entries, hash)
			modified = true
		}
	}
	
	if modified {
		return ts.save()
	}
	
	return nil
}

// load reads the trust store from disk
func (ts *TrustStore) load() error {
	data, err := os.ReadFile(ts.path)
	if err != nil {
		return err
	}
	
	var entries []*TrustEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("unmarshal trust store: %w", err)
	}
	
	for _, entry := range entries {
		ts.entries[entry.CommandHash] = entry
	}
	
	return nil
}

// save writes the trust store to disk
func (ts *TrustStore) save() error {
	entries := make([]*TrustEntry, 0, len(ts.entries))
	for _, entry := range ts.entries {
		entries = append(entries, entry)
	}
	
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal trust store: %w", err)
	}
	
	// Write atomically with temp file
	tmpPath := ts.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("write trust store: %w", err)
	}
	
	if err := os.Rename(tmpPath, ts.path); err != nil {
		return fmt.Errorf("rename trust store: %w", err)
	}
	
	return nil
}

// hashCommand creates a hash of the command for indexing
func hashCommand(command string) string {
	h := sha256.New()
	h.Write([]byte(command))
	return hex.EncodeToString(h.Sum(nil))
}

// getCurrentUser returns the current username
func getCurrentUser() string {
	if user := os.Getenv("USER"); user != "" {
		return user
	}
	if user := os.Getenv("USERNAME"); user != "" {
		return user
	}
	return "unknown"
}

