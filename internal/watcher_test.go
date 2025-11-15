package internal

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewKubeconfigWatcher(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create state manager
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Create logger
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	// Create context
	ctx := context.Background()

	// Test creating watcher with default kubeconfig path
	watcher, err := NewKubeconfigWatcher(sm, logger, ctx)
	if err != nil {
		t.Fatalf("Failed to create kubeconfig watcher: %v", err)
	}

	if watcher == nil {
		t.Fatal("Watcher should not be nil")
	}

	// Verify kubeconfig path was set correctly
	if watcher.kubeconfigPath == "" {
		t.Error("Kubeconfig path should not be empty")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}
	expectedPath := filepath.Join(home, ".kube", "config")

	if watcher.kubeconfigPath != expectedPath {
		t.Errorf("Expected kubeconfig path %s, got %s", expectedPath, watcher.kubeconfigPath)
	}
}

func TestNewKubeconfigWatcher_WithEnvVar(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	testKubeconfigPath := filepath.Join(tmpDir, "test-kubeconfig")

	// Set KUBECONFIG environment variable
	originalKubeconfig := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", testKubeconfigPath)
	defer os.Setenv("KUBECONFIG", originalKubeconfig)

	// Create state manager
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Create logger
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	// Create context
	ctx := context.Background()

	// Test creating watcher with KUBECONFIG env var
	watcher, err := NewKubeconfigWatcher(sm, logger, ctx)
	if err != nil {
		t.Fatalf("Failed to create kubeconfig watcher: %v", err)
	}

	if watcher.kubeconfigPath != testKubeconfigPath {
		t.Errorf("Expected kubeconfig path %s from env var, got %s", testKubeconfigPath, watcher.kubeconfigPath)
	}
}

func TestKubeconfigWatcher_IsFswatchAvailable(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create state manager
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Create logger
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	// Create context
	ctx := context.Background()

	watcher, err := NewKubeconfigWatcher(sm, logger, ctx)
	if err != nil {
		t.Fatalf("Failed to create kubeconfig watcher: %v", err)
	}

	// Test fswatch availability
	// On macOS with fswatch installed, this should return true
	// On other platforms or without fswatch, it should return false
	available := watcher.isFswatchAvailable()

	t.Logf("fswatch available: %v", available)
	// We don't assert the value since it depends on the environment
	// Just verify the method doesn't panic
}

func TestKubeconfigWatcher_HandleConfigChange(t *testing.T) {
	// Setup test kubeconfig
	tmpDir := t.TempDir()
	cleanup := setupTestKubeconfig(t, tmpDir)
	defer cleanup()

	statePath := filepath.Join(tmpDir, "state.json")

	// Create state manager
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	// Create logger
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)

	// Create context
	ctx := context.Background()

	watcher, err := NewKubeconfigWatcher(sm, logger, ctx)
	if err != nil {
		t.Fatalf("Failed to create kubeconfig watcher: %v", err)
	}

	// Record initial activity
	if err := sm.RecordActivity("staging"); err != nil {
		t.Fatalf("Failed to record initial activity: %v", err)
	}

	// Wait a moment
	time.Sleep(100 * time.Millisecond)

	// Get last activity timestamp
	lastActivity, _, err := sm.GetLastActivity()
	if err != nil {
		t.Fatalf("Failed to get last activity: %v", err)
	}

	// Simulate config change
	if err := watcher.handleConfigChange(); err != nil {
		t.Fatalf("Failed to handle config change: %v", err)
	}

	// Verify activity was recorded
	newActivity, context, err := sm.GetLastActivity()
	if err != nil {
		t.Fatalf("Failed to get new activity: %v", err)
	}

	if newActivity.Before(lastActivity) || newActivity.Equal(lastActivity) {
		t.Error("Activity timestamp should have been updated")
	}

	if context == "" {
		t.Error("Context should be recorded")
	}

	t.Logf("Context after change: %s", context)
}

func TestScanNullTerminated(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		atEOF   bool
		wantAdv int
		wantTok []byte
		wantErr error
	}{
		{
			name:    "simple null-terminated string",
			data:    []byte("hello\x00world"),
			atEOF:   false,
			wantAdv: 6,
			wantTok: []byte("hello"),
			wantErr: nil,
		},
		{
			name:    "empty string at EOF",
			data:    []byte{},
			atEOF:   true,
			wantAdv: 0,
			wantTok: nil,
			wantErr: nil,
		},
		{
			name:    "incomplete data, not at EOF",
			data:    []byte("incomplete"),
			atEOF:   false,
			wantAdv: 0,
			wantTok: nil,
			wantErr: nil,
		},
		{
			name:    "incomplete data at EOF",
			data:    []byte("incomplete"),
			atEOF:   true,
			wantAdv: 10,
			wantTok: []byte("incomplete"),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adv, tok, err := scanNullTerminated(tt.data, tt.atEOF)

			if adv != tt.wantAdv {
				t.Errorf("advance = %d, want %d", adv, tt.wantAdv)
			}

			if string(tok) != string(tt.wantTok) {
				t.Errorf("token = %q, want %q", tok, tt.wantTok)
			}

			if err != tt.wantErr {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
