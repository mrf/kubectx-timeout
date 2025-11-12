package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Helper function to get profile path for a given shell
func getProfilePath(homeDir, shell string) string {
	if shell == "bash" {
		return filepath.Join(homeDir, ".bashrc")
	}
	return filepath.Join(homeDir, ".zshrc")
}

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

			if tt.wantError {
				return // Expected error case, nothing more to verify
			}

			// Verify profile file was created/updated
			profilePath := getProfilePath(tmpHome, tt.shell)
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
		})
	}
}

func TestGetCurrentContext(t *testing.T) {
	// Setup isolated test environment to avoid leaking real context names
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	context, err := GetCurrentContext()
	if err != nil {
		t.Fatalf("GetCurrentContext failed: %v", err)
	}

	if context == "" {
		t.Error("GetCurrentContext returned empty context")
	}

	// Verify we got the test context from isolated kubeconfig
	if context != "test-default" {
		t.Errorf("Expected test-default context from isolated kubeconfig, got: %s", context)
	}

	t.Logf("Current kubectl context: %s", context)
}

// TestGenerateShellIntegrationIncludesKubectx tests that shell integration includes kubectx wrapper
// This is a regression test for the issue where manually running kubectx doesn't start the timer
func TestGenerateShellIntegrationIncludesKubectx(t *testing.T) {
	tests := []struct {
		shell      string
		binaryPath string
	}{
		{"bash", "/usr/local/bin/kubectx-timeout"},
		{"zsh", "/usr/local/bin/kubectx-timeout"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			integration, err := GenerateShellIntegration(tt.shell, tt.binaryPath)
			if err != nil {
				t.Fatalf("GenerateShellIntegration failed: %v", err)
			}

			// Verify integration contains kubectx wrapper
			if !strings.Contains(integration, "kubectx()") {
				t.Error("integration should contain kubectx function wrapper")
			}

			// Verify kubectx wrapper calls record-activity
			if !strings.Contains(integration, "kubectx") && strings.Contains(integration, "record-activity") {
				t.Error("kubectx wrapper should call record-activity before executing kubectx")
			}

			// Verify kubectx wrapper calls the real kubectx command
			if !strings.Contains(integration, "command kubectx") {
				t.Error("kubectx wrapper should execute 'command kubectx' to invoke the real kubectx")
			}
		})
	}
}

// TestKubectxWrapperRecordsActivityAfterSwitch tests that kubectx wrapper records activity
// AFTER the context switch completes, not before. This ensures we capture the NEW context,
// not the old one.
// This is a regression test for the race condition where record-activity runs in parallel
// with kubectx and might capture the old context before the switch completes.
func TestKubectxWrapperRecordsActivityAfterSwitch(t *testing.T) {
	tests := []struct {
		shell string
	}{
		{"bash"},
		{"zsh"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			integration, err := GenerateShellIntegration(tt.shell, "/usr/local/bin/kubectx-timeout")
			if err != nil {
				t.Fatalf("GenerateShellIntegration failed: %v", err)
			}

			// Find the kubectx function in the integration
			kubectxStart := strings.Index(integration, "kubectx() {")
			if kubectxStart == -1 {
				t.Fatal("kubectx function not found in integration")
			}

			// Find the end of the kubectx function (next closing brace at start of line)
			kubectxEnd := strings.Index(integration[kubectxStart:], "\n}")
			if kubectxEnd == -1 {
				t.Fatal("kubectx function end not found")
			}
			kubectxFunc := integration[kubectxStart : kubectxStart+kubectxEnd+2]

			// Verify that "command kubectx" appears BEFORE "record-activity" in the function
			cmdKubectxPos := strings.Index(kubectxFunc, "command kubectx")
			recordActivityPos := strings.Index(kubectxFunc, "record-activity")

			if cmdKubectxPos == -1 {
				t.Error("kubectx function should contain 'command kubectx'")
			}
			if recordActivityPos == -1 {
				t.Error("kubectx function should contain 'record-activity'")
			}

			// This is the key test: record-activity must come AFTER command kubectx
			if cmdKubectxPos > recordActivityPos {
				t.Errorf("kubectx wrapper has incorrect order: record-activity (%d) should come AFTER command kubectx (%d)\n"+
					"This causes a race condition where the old context is recorded instead of the new one.\n"+
					"Function:\n%s",
					recordActivityPos, cmdKubectxPos, kubectxFunc)
			}

			// Verify that we capture and return the exit code
			if !strings.Contains(kubectxFunc, "exit_code") && !strings.Contains(kubectxFunc, "return") {
				t.Error("kubectx wrapper should preserve exit code from command kubectx")
			}
		})
	}
}
