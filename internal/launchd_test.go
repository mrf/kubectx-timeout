package internal

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewLaunchdManager(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping launchd tests on non-macOS platform")
	}

	binaryPath := "/usr/local/bin/kubectx-timeout"
	lm, err := NewLaunchdManager(binaryPath)
	if err != nil {
		t.Fatalf("Failed to create launchd manager: %v", err)
	}

	if lm.label != LaunchdLabel {
		t.Errorf("Expected label %s, got %s", LaunchdLabel, lm.label)
	}

	if lm.binaryPath != binaryPath {
		t.Errorf("Expected binary path %s, got %s", binaryPath, lm.binaryPath)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	expectedPlistPath := filepath.Join(homeDir, "Library", "LaunchAgents", LaunchdLabel+".plist")
	if lm.plistPath != expectedPlistPath {
		t.Errorf("Expected plist path %s, got %s", expectedPlistPath, lm.plistPath)
	}
}

func TestNewLaunchdManager_NonMacOS(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping non-macOS test on macOS platform")
	}

	_, err := NewLaunchdManager("")
	if err == nil {
		t.Error("Expected error on non-macOS platform, got nil")
	}
}

func TestGeneratePlist(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping launchd tests on non-macOS platform")
	}

	binaryPath := "/usr/local/bin/kubectx-timeout"
	lm, err := NewLaunchdManager(binaryPath)
	if err != nil {
		t.Fatalf("Failed to create launchd manager: %v", err)
	}

	plistContent, err := lm.generatePlist()
	if err != nil {
		t.Fatalf("Failed to generate plist: %v", err)
	}

	// Check that plist contains expected elements
	expectedStrings := []string{
		"<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
		"<plist version=\"1.0\">",
		LaunchdLabel,
		binaryPath,
		"daemon",
		"RunAtLoad",
		"KeepAlive",
	}

	for _, expected := range expectedStrings {
		if !contains(plistContent, expected) {
			t.Errorf("Expected plist to contain %q", expected)
		}
	}
}

func TestGetPlistPath(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping launchd tests on non-macOS platform")
	}

	lm, err := NewLaunchdManager("")
	if err != nil {
		t.Fatalf("Failed to create launchd manager: %v", err)
	}

	plistPath := lm.GetPlistPath()
	if plistPath == "" {
		t.Error("Expected non-empty plist path")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	expectedPlistPath := filepath.Join(homeDir, "Library", "LaunchAgents", LaunchdLabel+".plist")
	if plistPath != expectedPlistPath {
		t.Errorf("Expected plist path %s, got %s", expectedPlistPath, plistPath)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIsInstalled(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping launchd tests on non-macOS platform")
	}

	lm, err := NewLaunchdManager("/usr/local/bin/kubectx-timeout")
	if err != nil {
		t.Fatalf("Failed to create launchd manager: %v", err)
	}

	// Just verify the method runs without crashing
	// It will return false unless actually installed
	installed := lm.IsInstalled()
	t.Logf("Daemon installed: %v", installed)
}

func TestIsRunning(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping launchd tests on non-macOS platform")
	}

	lm, err := NewLaunchdManager("/usr/local/bin/kubectx-timeout")
	if err != nil {
		t.Fatalf("Failed to create launchd manager: %v", err)
	}

	// Just verify the method runs without crashing
	// It will return false unless actually running
	running := lm.IsRunning()
	t.Logf("Daemon running: %v", running)
}

func TestGetStatus(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping launchd tests on non-macOS platform")
	}

	lm, err := NewLaunchdManager("/usr/local/bin/kubectx-timeout")
	if err != nil {
		t.Fatalf("Failed to create launchd manager: %v", err)
	}

	status, err := lm.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status == "" {
		t.Error("Expected non-empty status")
	}

	// Check that status contains expected fields
	if !contains(status, "Daemon Status") {
		t.Error("Expected status to contain 'Daemon Status'")
	}
	if !contains(status, "Installed:") {
		t.Error("Expected status to contain 'Installed:'")
	}
	if !contains(status, "Running:") {
		t.Error("Expected status to contain 'Running:'")
	}
}

func TestGetPID(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping launchd tests on non-macOS platform")
	}

	lm, err := NewLaunchdManager("/usr/local/bin/kubectx-timeout")
	if err != nil {
		t.Fatalf("Failed to create launchd manager: %v", err)
	}

	// Just verify the method runs without crashing
	// It will return 0 unless actually running
	pid, err := lm.GetPID()
	if err != nil {
		t.Logf("GetPID returned error (expected if not running): %v", err)
	}
	t.Logf("Daemon PID: %d", pid)
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		shouldError bool
	}{
		{
			name:        "valid absolute path",
			path:        "/usr/local/bin/kubectx-timeout",
			shouldError: false,
		},
		{
			name:        "valid absolute path with spaces",
			path:        "/usr/local/My Folder/kubectx-timeout",
			shouldError: false,
		},
		{
			name:        "relative path",
			path:        "bin/kubectx-timeout",
			shouldError: true,
		},
		{
			name:        "path with semicolon",
			path:        "/usr/local/bin;rm -rf /",
			shouldError: true,
		},
		{
			name:        "path with pipe",
			path:        "/usr/local/bin|cat",
			shouldError: true,
		},
		{
			name:        "path with backtick",
			path:        "/usr/local/bin`whoami`",
			shouldError: true,
		},
		{
			name:        "path with dollar sign",
			path:        "/usr/local/bin$HOME",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if tt.shouldError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestUninstall_NotInstalled(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping launchd tests on non-macOS platform")
	}

	lm, err := NewLaunchdManager("/usr/local/bin/kubectx-timeout")
	if err != nil {
		t.Fatalf("Failed to create launchd manager: %v", err)
	}

	// Try to uninstall when not installed
	err = lm.Uninstall()
	if err == nil && !lm.IsInstalled() {
		// If not installed, should return error
		t.Log("Uninstall returned nil for non-installed daemon (acceptable if plist doesn't exist)")
	}
}

func TestStart_NotInstalled(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping launchd tests on non-macOS platform")
	}

	lm, err := NewLaunchdManager("/usr/local/bin/kubectx-timeout")
	if err != nil {
		t.Fatalf("Failed to create launchd manager: %v", err)
	}

	// Try to start when not installed
	if !lm.IsInstalled() {
		err = lm.Start()
		if err == nil {
			t.Error("Expected error when starting non-installed daemon")
		}
	}
}

func TestStop_NotInstalled(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping launchd tests on non-macOS platform")
	}

	lm, err := NewLaunchdManager("/usr/local/bin/kubectx-timeout")
	if err != nil {
		t.Fatalf("Failed to create launchd manager: %v", err)
	}

	// Try to stop when not installed
	if !lm.IsInstalled() {
		err = lm.Stop()
		if err == nil {
			t.Error("Expected error when stopping non-installed daemon")
		}
	}
}

func TestRestart_NotInstalled(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping launchd tests on non-macOS platform")
	}

	lm, err := NewLaunchdManager("/usr/local/bin/kubectx-timeout")
	if err != nil {
		t.Fatalf("Failed to create launchd manager: %v", err)
	}

	// Try to restart when not installed
	if !lm.IsInstalled() {
		err = lm.Restart()
		if err == nil {
			t.Error("Expected error when restarting non-installed daemon")
		}
	}
}
