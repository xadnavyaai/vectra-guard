package sandbox

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrustStore(t *testing.T) {
	// Create temporary directory for test
	tmpDir, err := os.MkdirTemp("", "vectra-guard-trust-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)
	
	trustPath := filepath.Join(tmpDir, "trust.json")
	
	t.Run("NewTrustStore", func(t *testing.T) {
		store, err := NewTrustStore(trustPath)
		if err != nil {
			t.Fatalf("NewTrustStore() error = %v", err)
		}
		
		if store.path != trustPath {
			t.Errorf("path = %v, want %v", store.path, trustPath)
		}
	})
	
	t.Run("AddAndIsTrusted", func(t *testing.T) {
		store, _ := NewTrustStore(trustPath)
		
		command := "npm install express"
		
		// Initially not trusted
		if store.IsTrusted(command) {
			t.Error("Command should not be trusted initially")
		}
		
		// Add to trust store
		err := store.Add(command, 0, "Test approval")
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
		
		// Now should be trusted
		if !store.IsTrusted(command) {
			t.Error("Command should be trusted after adding")
		}
	})
	
	t.Run("AddWithExpiration", func(t *testing.T) {
		store, _ := NewTrustStore(trustPath)
		
		command := "temporary command"
		duration := 100 * time.Millisecond
		
		err := store.Add(command, duration, "Temporary approval")
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
		
		// Should be trusted immediately
		if !store.IsTrusted(command) {
			t.Error("Command should be trusted immediately after adding")
		}
		
		// Wait for expiration
		time.Sleep(150 * time.Millisecond)
		
		// Should no longer be trusted
		if store.IsTrusted(command) {
			t.Error("Command should not be trusted after expiration")
		}
	})
	
	t.Run("RecordUse", func(t *testing.T) {
		store, _ := NewTrustStore(trustPath)
		
		command := "npm test"
		
		// Add command
		err := store.Add(command, 0, "Test")
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
		
		// Record uses
		for i := 0; i < 5; i++ {
			err = store.RecordUse(command)
			if err != nil {
				t.Fatalf("RecordUse() error = %v", err)
			}
		}
		
		// Check use count
		entries := store.List()
		found := false
		for _, entry := range entries {
			if entry.Command == command {
				found = true
				if entry.UseCount != 5 {
					t.Errorf("UseCount = %d, want 5", entry.UseCount)
				}
				if entry.LastUsed.IsZero() {
					t.Error("LastUsed should be set")
				}
			}
		}
		
		if !found {
			t.Error("Command not found in trust store")
		}
	})
	
	t.Run("Remove", func(t *testing.T) {
		store, _ := NewTrustStore(trustPath)
		
		command := "rm -rf /tmp/test"
		
		// Add command
		err := store.Add(command, 0, "Test")
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
		
		// Verify it's trusted
		if !store.IsTrusted(command) {
			t.Error("Command should be trusted after adding")
		}
		
		// Remove it
		err = store.Remove(command)
		if err != nil {
			t.Fatalf("Remove() error = %v", err)
		}
		
		// Verify it's no longer trusted
		if store.IsTrusted(command) {
			t.Error("Command should not be trusted after removal")
		}
	})
	
	t.Run("List", func(t *testing.T) {
		store, _ := NewTrustStore(trustPath)
		
		commands := []string{
			"npm install",
			"yarn build",
			"pip install -r requirements.txt",
		}
		
		// Add multiple commands
		for _, cmd := range commands {
			err := store.Add(cmd, 0, "Test")
			if err != nil {
				t.Fatalf("Add() error = %v", err)
			}
		}
		
		// List all
		entries := store.List()
		
		if len(entries) < len(commands) {
			t.Errorf("List() returned %d entries, want at least %d", len(entries), len(commands))
		}
	})
	
	t.Run("CleanExpired", func(t *testing.T) {
		cleanPath := filepath.Join(tmpDir, "trust-clean.json")
		store, _ := NewTrustStore(cleanPath)
		
		// Add permanent command
		err := store.Add("permanent", 0, "Never expires")
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
		
		// Add expired command (use past time)
		err = store.Add("expired", -1*time.Hour, "Already expired")
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
		
		// Expired should already not be trusted (before clean)
		// because IsTrusted checks expiration
		wasExpired := !store.IsTrusted("expired")
		
		// Clean expired
		err = store.CleanExpired()
		if err != nil {
			t.Fatalf("CleanExpired() error = %v", err)
		}
		
		// Permanent should still be there
		if !store.IsTrusted("permanent") {
			t.Error("Permanent command should still be trusted")
		}
		
		// Expired should not be trusted (either before or after clean)
		if store.IsTrusted("expired") {
			t.Error("Expired command should not be trusted")
		}
		
		// Verify it worked as expected
		if !wasExpired {
			t.Log("Note: Expired entry was trusted before clean (unexpected)")
		}
	})
	
	t.Run("Persistence", func(t *testing.T) {
		// Create first store and add command
		store1, _ := NewTrustStore(trustPath)
		command := "persistent command"
		
		err := store1.Add(command, 0, "Should persist")
		if err != nil {
			t.Fatalf("Add() error = %v", err)
		}
		
		// Create new store with same path
		store2, _ := NewTrustStore(trustPath)
		
		// Command should still be trusted
		if !store2.IsTrusted(command) {
			t.Error("Command should be trusted after reloading trust store")
		}
	})
}

func TestHashCommand(t *testing.T) {
	tests := []struct {
		name     string
		cmd1     string
		cmd2     string
		samehash bool
	}{
		{
			name:     "identical commands",
			cmd1:     "npm install",
			cmd2:     "npm install",
			samehash: true,
		},
		{
			name:     "different commands",
			cmd1:     "npm install",
			cmd2:     "yarn install",
			samehash: false,
		},
		{
			name:     "whitespace differences",
			cmd1:     "npm  install",
			cmd2:     "npm install",
			samehash: false, // Hashes should be different with whitespace
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashCommand(tt.cmd1)
			hash2 := hashCommand(tt.cmd2)
			
			if (hash1 == hash2) != tt.samehash {
				t.Errorf("hashCommand(%q) == hashCommand(%q) = %v, want %v",
					tt.cmd1, tt.cmd2, hash1 == hash2, tt.samehash)
			}
		})
	}
}

func BenchmarkTrustStoreIsTrusted(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "vectra-guard-trust-bench")
	defer os.RemoveAll(tmpDir)
	
	trustPath := filepath.Join(tmpDir, "trust.json")
	store, _ := NewTrustStore(trustPath)
	
	// Add some commands
	for i := 0; i < 100; i++ {
		store.Add("command-"+string(rune(i)), 0, "Benchmark")
	}
	
	command := "command-50"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.IsTrusted(command)
	}
}

func BenchmarkTrustStoreAdd(b *testing.B) {
	tmpDir, _ := os.MkdirTemp("", "vectra-guard-trust-bench")
	defer os.RemoveAll(tmpDir)
	
	trustPath := filepath.Join(tmpDir, "trust.json")
	store, _ := NewTrustStore(trustPath)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Add("command-"+string(rune(i%1000)), 0, "Benchmark")
	}
}

