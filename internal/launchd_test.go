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
