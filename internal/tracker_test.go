package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewActivityTracker(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	configPath := filepath.Join(tmpDir, "config.yaml")

	tracker, err := NewActivityTracker(statePath, configPath)
	if err != nil {
		t.Fatalf("NewActivityTracker failed: %v", err)
	}

	if tracker == nil {
		t.Fatal("NewActivityTracker returned nil")
	}

	if tracker.stateManager == nil {
		t.Error("ActivityTracker has nil stateManager")
	}
}

func TestActivityTrackerRecordActivity(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	configPath := filepath.Join(tmpDir, "config.yaml")

	tracker, err := NewActivityTracker(statePath, configPath)
	if err != nil {
		t.Fatalf("NewActivityTracker failed: %v", err)
	}

	// Record activity
	before := time.Now()
	if err := tracker.RecordActivity(); err != nil {
		t.Fatalf("RecordActivity failed: %v", err)
	}
	after := time.Now()

	// Verify activity was recorded
	info, err := tracker.GetLastActivity()
	if err != nil {
		t.Fatalf("GetLastActivity failed: %v", err)
	}

	if info.LastActivity.Before(before) || info.LastActivity.After(after) {
		t.Errorf("LastActivity %v is outside expected range [%v, %v]", info.LastActivity, before, after)
	}

	// Context might be "unknown" if kubectl is not available or not configured
	// That's OK for this test
	if info.CurrentContext == "" {
		t.Error("CurrentContext should not be empty")
	}
}

func TestActivityTrackerGetLastActivity(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")
	configPath := filepath.Join(tmpDir, "config.yaml")

	tracker, err := NewActivityTracker(statePath, configPath)
	if err != nil {
		t.Fatalf("NewActivityTracker failed: %v", err)
	}

	// Get activity before any recorded (should work, return empty)
	info, err := tracker.GetLastActivity()
	if err != nil {
		t.Fatalf("GetLastActivity failed: %v", err)
	}

	if !info.LastActivity.IsZero() {
		t.Error("Expected zero LastActivity for new tracker")
	}

	// Record activity
	if err := tracker.RecordActivity(); err != nil {
		t.Fatalf("RecordActivity failed: %v", err)
	}

	// Get activity again
	info, err = tracker.GetLastActivity()
	if err != nil {
		t.Fatalf("GetLastActivity failed: %v", err)
	}

	if info.LastActivity.IsZero() {
		t.Error("Expected non-zero LastActivity after recording")
	}
}

func TestGenerateShellIntegration(t *testing.T) {
	tests := []struct {
		shell      string
		binaryPath string
		wantError  bool
	}{
		{"bash", "/usr/local/bin/kubectx-timeout", false},
		{"zsh", "/usr/local/bin/kubectx-timeout", false},
		{"bash", "", false}, // Should use default path
		{"fish", "/usr/local/bin/kubectx-timeout", true}, // Unsupported
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			integration, err := GenerateShellIntegration(tt.shell, tt.binaryPath)

			if (err != nil) != tt.wantError {
				t.Errorf("GenerateShellIntegration() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				// Verify integration contains expected elements
				if !strings.Contains(integration, "kubectl()") {
					t.Error("integration should contain kubectl function")
				}

				if !strings.Contains(integration, "record-activity") {
					t.Error("integration should contain record-activity command")
				}

				if !strings.Contains(integration, tt.shell) {
					t.Errorf("integration should mention shell '%s'", tt.shell)
				}

				// If binary path specified, it should be in the integration
				if tt.binaryPath != "" && !strings.Contains(integration, tt.binaryPath) {
					t.Errorf("integration should contain binary path '%s'", tt.binaryPath)
				}
			}
		})
	}
}

func TestInstallShellIntegration(t *testing.T) {
	// Create a temporary home directory
	tmpHome := t.TempDir()

	// Override HOME for this test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	tests := []struct {
		name      string
		shell     string
		wantError bool
	}{
		{"bash", "bash", false},
		{"zsh", "zsh", false},
		{"unsupported", "fish", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InstallShellIntegration(tt.shell)

			if (err != nil) != tt.wantError {
				t.Errorf("InstallShellIntegration() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				// Verify profile file was created/updated
				var profilePath string
				if tt.shell == "bash" {
					profilePath = filepath.Join(tmpHome, ".bashrc")
				} else if tt.shell == "zsh" {
					profilePath = filepath.Join(tmpHome, ".zshrc")
				}

				content, err := os.ReadFile(profilePath)
				if err != nil {
					t.Fatalf("Failed to read profile: %v", err)
				}

				if !strings.Contains(string(content), "kubectx-timeout shell integration") {
					t.Error("Profile should contain integration marker")
				}

				// Try installing again - should fail (already installed)
				err = InstallShellIntegration(tt.shell)
				if err == nil {
					t.Error("Installing twice should return error")
				}
			}
		})
	}
}

func TestGetCurrentContext(t *testing.T) {
	// This test will only pass if kubectl is installed and configured
	// We'll make it optional based on kubectl availability

	context, err := GetCurrentContext()

	if err != nil {
		// kubectl might not be available or configured in test environment
		// That's OK, just log it
		t.Logf("GetCurrentContext failed (expected if kubectl not configured): %v", err)
		return
	}

	if context == "" {
		t.Error("GetCurrentContext returned empty context")
	}

	t.Logf("Current kubectl context: %s", context)
}
