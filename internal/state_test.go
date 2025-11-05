package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewStateManager(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "subdir", "state.json")
	
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}
	
	if sm == nil {
		t.Fatal("NewStateManager returned nil")
	}
	
	// Verify directory was created
	dir := filepath.Dir(statePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Errorf("NewStateManager did not create directory: %s", dir)
	}
}

func TestStateManagerLoadEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}
	
	// Load from non-existent file should return empty state
	state, err := sm.Load()
	if err != nil {
		t.Fatalf("Load failed on missing file: %v", err)
	}
	
	if state == nil {
		t.Fatal("Load returned nil for missing file")
	}
	
	if state.Version != stateVersion {
		t.Errorf("expected version %d, got %d", stateVersion, state.Version)
	}
	
	if !state.LastActivity.IsZero() {
		t.Error("expected empty LastActivity")
	}
}

func TestStateManagerSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}
	
	// Create state
	now := time.Now()
	state := &State{
		LastActivity:   now,
		CurrentContext: "test-context",
		Version:        stateVersion,
	}
	
	// Save state
	if err := sm.Save(state); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	
	// Verify file exists
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		t.Error("Save did not create state file")
	}
	
	// Load state
	loaded, err := sm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	// Verify loaded state matches saved state
	if loaded.CurrentContext != state.CurrentContext {
		t.Errorf("expected context %s, got %s", state.CurrentContext, loaded.CurrentContext)
	}
	
	// Compare timestamps (allow small difference due to JSON serialization)
	timeDiff := loaded.LastActivity.Sub(now)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	if timeDiff > time.Second {
		t.Errorf("timestamp mismatch: expected %v, got %v", now, loaded.LastActivity)
	}
}

func TestStateManagerRecordActivity(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}
	
	// Record activity
	before := time.Now()
	if err := sm.RecordActivity("production"); err != nil {
		t.Fatalf("RecordActivity failed: %v", err)
	}
	after := time.Now()
	
	// Load and verify
	state, err := sm.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	
	if state.CurrentContext != "production" {
		t.Errorf("expected context 'production', got '%s'", state.CurrentContext)
	}
	
	if state.LastActivity.Before(before) || state.LastActivity.After(after) {
		t.Errorf("LastActivity %v is outside expected range [%v, %v]", state.LastActivity, before, after)
	}
}

func TestStateManagerGetLastActivity(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}
	
	// Record activity
	if err := sm.RecordActivity("dev"); err != nil {
		t.Fatalf("RecordActivity failed: %v", err)
	}
	
	// Get last activity
	lastActivity, context, err := sm.GetLastActivity()
	if err != nil {
		t.Fatalf("GetLastActivity failed: %v", err)
	}
	
	if context != "dev" {
		t.Errorf("expected context 'dev', got '%s'", context)
	}
	
	if lastActivity.IsZero() {
		t.Error("LastActivity should not be zero")
	}
	
	// Should be very recent
	timeSince := time.Since(lastActivity)
	if timeSince > 5*time.Second {
		t.Errorf("LastActivity is too old: %v", timeSince)
	}
}

func TestStateManagerTimeSinceLastActivity(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}
	
	// Test with no activity (should return large duration)
	duration, err := sm.TimeSinceLastActivity()
	if err != nil {
		t.Fatalf("TimeSinceLastActivity failed: %v", err)
	}
	
	if duration < 24*time.Hour {
		t.Errorf("expected duration >= 24h for no activity, got %v", duration)
	}
	
	// Record activity
	if err := sm.RecordActivity("staging"); err != nil {
		t.Fatalf("RecordActivity failed: %v", err)
	}
	
	// Check duration again
	duration, err = sm.TimeSinceLastActivity()
	if err != nil {
		t.Fatalf("TimeSinceLastActivity failed: %v", err)
	}
	
	if duration > 5*time.Second {
		t.Errorf("expected duration < 5s after recording activity, got %v", duration)
	}
}

func TestStateManagerConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}
	
	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()
			
			// Record activity
			if err := sm.RecordActivity("concurrent-test"); err != nil {
				t.Errorf("RecordActivity failed: %v", err)
			}
			
			// Read activity
			if _, _, err := sm.GetLastActivity(); err != nil {
				t.Errorf("GetLastActivity failed: %v", err)
			}
		}(i)
	}
	
	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
	
	// Verify state is still valid
	state, err := sm.Load()
	if err != nil {
		t.Fatalf("Load failed after concurrent access: %v", err)
	}
	
	if state.CurrentContext != "concurrent-test" {
		t.Errorf("expected context 'concurrent-test', got '%s'", state.CurrentContext)
	}
}

func TestStateFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a subdirectory so NewStateManager creates it with proper permissions
	stateDir := filepath.Join(tmpDir, ".kubectx-timeout")
	statePath := filepath.Join(stateDir, "state.json")

	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}

	// Record activity to create file
	if err := sm.RecordActivity("test"); err != nil {
		t.Fatalf("RecordActivity failed: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("Failed to stat state file: %v", err)
	}

	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("expected file permissions 0600, got %o", mode)
	}

	// Check directory permissions (the directory created by NewStateManager)
	dirInfo, err := os.Stat(stateDir)
	if err != nil {
		t.Fatalf("Failed to stat state directory: %v", err)
	}

	dirMode := dirInfo.Mode().Perm()
	if dirMode != 0700 {
		t.Errorf("expected directory permissions 0700, got %o", dirMode)
	}
}
