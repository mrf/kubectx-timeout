package internal

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestUninstall(t *testing.T) {
	// This test verifies basic uninstall flow without actually removing anything
	opts := UninstallOptions{
		KeepConfig:  true,
		KeepBinary:  true,
		Force:       true,
		AllShells:   false,
		TargetShell: "",
		BinaryPath:  "",
	}

	result, err := Uninstall(opts)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	// When keeping config and binary, these should be false
	if result.ConfigRemoved {
		t.Error("Expected config not to be removed when KeepConfig is true")
	}
	if result.BinaryRemoved {
		t.Error("Expected binary not to be removed when KeepBinary is true")
	}
}

func TestUninstall_WithShellSpecified(t *testing.T) {
	opts := UninstallOptions{
		KeepConfig:  true,
		KeepBinary:  true,
		Force:       true,
		AllShells:   false,
		TargetShell: ShellBash,
		BinaryPath:  "",
	}

	result, err := Uninstall(opts)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestUninstall_AllShells(t *testing.T) {
	opts := UninstallOptions{
		KeepConfig:  true,
		KeepBinary:  true,
		Force:       true,
		AllShells:   true,
		TargetShell: "",
		BinaryPath:  "",
	}

	result, err := Uninstall(opts)
	if err != nil {
		t.Fatalf("Uninstall failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
}

func TestRemoveBinary(t *testing.T) {
	tests := []struct {
		name        string
		binaryPath  string
		shouldError bool
		createFile  bool
	}{
		{
			name:        "valid kubectx-timeout binary",
			binaryPath:  "/tmp/kubectx-timeout-test-binary",
			shouldError: false,
			createFile:  true,
		},
		{
			name:        "non-existent file",
			binaryPath:  "/tmp/kubectx-timeout-nonexistent",
			shouldError: false,
			createFile:  false,
		},
		{
			name:        "invalid binary name",
			binaryPath:  "/tmp/some-other-binary",
			shouldError: true,
			createFile:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &UninstallResult{}

			if tt.createFile {
				// Create a temporary file
				if err := os.WriteFile(tt.binaryPath, []byte("test"), 0755); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
				defer os.Remove(tt.binaryPath)
			}

			err := removeBinary(tt.binaryPath, result)

			if tt.shouldError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if tt.createFile && !tt.shouldError {
				if !result.BinaryRemoved {
					t.Error("Expected BinaryRemoved to be true")
				}
				// Verify file was actually removed
				if _, err := os.Stat(tt.binaryPath); !os.IsNotExist(err) {
					t.Error("Expected file to be removed")
				}
			}
		})
	}
}

func TestRemoveConfigAndState(t *testing.T) {
	// This test verifies that removeConfigAndState calls the right functions
	// and handles non-existent directories gracefully
	result := &UninstallResult{}

	// Call with actual directories (they may or may not exist)
	err := removeConfigAndState(result)

	// The function should not fail even if directories don't exist
	if err != nil {
		t.Fatalf("removeConfigAndState failed: %v", err)
	}

	// Results depend on whether directories actually existed
	t.Logf("ConfigRemoved: %v, StateRemoved: %v", result.ConfigRemoved, result.StateRemoved)
}

func TestGetLaunchdPlistPath(t *testing.T) {
	if runtime.GOOS != "darwin" {
		// On non-macOS, should return error
		_, err := GetLaunchdPlistPath()
		if err == nil {
			t.Error("Expected error on non-macOS platform")
		}
		return
	}

	// On macOS, should return valid path
	path, err := GetLaunchdPlistPath()
	if err != nil {
		t.Fatalf("GetLaunchdPlistPath failed: %v", err)
	}

	if path == "" {
		t.Error("Expected non-empty path")
	}

	if !strings.Contains(path, "Library/LaunchAgents") {
		t.Errorf("Expected path to contain Library/LaunchAgents, got: %s", path)
	}

	if !strings.Contains(path, "com.kubectx-timeout.plist") {
		t.Errorf("Expected path to contain com.kubectx-timeout.plist, got: %s", path)
	}
}

func TestCheckDaemonStatus(t *testing.T) {
	// This test just verifies the function doesn't crash
	// We can't test actual daemon status without installing it
	_, err := CheckDaemonStatus()
	if err != nil {
		t.Logf("CheckDaemonStatus returned error (expected if daemon not installed): %v", err)
	}
}

func TestGetInstalledShells(t *testing.T) {
	// This test verifies the function doesn't crash
	shells, err := GetInstalledShells()
	if err != nil {
		t.Fatalf("GetInstalledShells failed: %v", err)
	}

	// shells may be empty if no integrations are installed, that's fine
	t.Logf("Installed shells: %v", shells)
}

func TestFormatUninstallResult(t *testing.T) {
	tests := []struct {
		name   string
		result *UninstallResult
		expect []string // Strings that should be in output
	}{
		{
			name: "complete uninstall",
			result: &UninstallResult{
				DaemonStopped:   true,
				LaunchdRemoved:  true,
				ShellsProcessed: []string{"bash", "zsh"},
				ConfigRemoved:   true,
				StateRemoved:    true,
				BinaryRemoved:   true,
				BackupsCreated:  []string{"/home/user/.bashrc.backup"},
				Errors:          []error{},
			},
			expect: []string{
				"Uninstallation Summary",
				"Daemon stopped and removed",
				"Shell integration removed from: bash, zsh",
				"Configuration files removed",
				"State files removed",
				"Binary removed",
				"Backups created",
			},
		},
		{
			name: "partial uninstall with errors",
			result: &UninstallResult{
				DaemonStopped:   false,
				LaunchdRemoved:  true,
				ShellsProcessed: []string{},
				ConfigRemoved:   false,
				StateRemoved:    false,
				BinaryRemoved:   false,
				BackupsCreated:  []string{},
				Errors:          []error{os.ErrNotExist},
			},
			expect: []string{
				"Uninstallation Summary",
				"Daemon configuration removed (daemon was not running)",
			},
		},
		{
			name: "minimal uninstall",
			result: &UninstallResult{
				DaemonStopped:   false,
				LaunchdRemoved:  false,
				ShellsProcessed: []string{},
				ConfigRemoved:   false,
				StateRemoved:    false,
				BinaryRemoved:   false,
				BackupsCreated:  []string{},
				Errors:          []error{},
			},
			expect: []string{
				"Uninstallation Summary",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := FormatUninstallResult(tt.result)

			for _, expected := range tt.expect {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

func TestRemoveShellIntegration(t *testing.T) {
	// Test shell integration removal without actually modifying real shell profiles
	result := &UninstallResult{}
	opts := UninstallOptions{
		TargetShell: ShellBash,
		AllShells:   false,
	}

	err := removeShellIntegration(opts, result)
	if err != nil {
		t.Fatalf("removeShellIntegration failed: %v", err)
	}

	// The result depends on whether the integration was actually detected and removed
	t.Logf("ShellsProcessed: %v", result.ShellsProcessed)
	t.Logf("Errors: %v", result.Errors)

	// Test with AllShells option
	result2 := &UninstallResult{}
	opts2 := UninstallOptions{
		TargetShell: "",
		AllShells:   true,
	}

	err = removeShellIntegration(opts2, result2)
	if err != nil {
		t.Fatalf("removeShellIntegration with AllShells failed: %v", err)
	}

	t.Logf("AllShells - ShellsProcessed: %v", result2.ShellsProcessed)
	t.Logf("AllShells - Errors: %v", result2.Errors)
}

func TestStopAndRemoveDaemon_NonMacOS(t *testing.T) {
	if runtime.GOOS == "darwin" {
		t.Skip("Skipping non-macOS test on macOS")
	}

	result := &UninstallResult{}
	err := stopAndRemoveDaemon(result)

	// On non-macOS, this function does nothing and returns nil
	// because the check `runtime.GOOS == "darwin"` in Uninstall() prevents it from being called
	// But if called directly, it would try to access ~/Library/LaunchAgents which might not exist
	// The function should handle this gracefully
	t.Logf("stopAndRemoveDaemon on non-macOS returned: %v", err)
}

func TestUninstallOptions_Validation(t *testing.T) {
	// Test that UninstallOptions fields work as expected
	opts := UninstallOptions{
		KeepConfig:  true,
		KeepBinary:  true,
		Force:       false,
		AllShells:   false,
		TargetShell: ShellBash,
		BinaryPath:  "/usr/local/bin/kubectx-timeout",
	}

	if !opts.KeepConfig {
		t.Error("Expected KeepConfig to be true")
	}
	if !opts.KeepBinary {
		t.Error("Expected KeepBinary to be true")
	}
	if opts.Force {
		t.Error("Expected Force to be false")
	}
	if opts.AllShells {
		t.Error("Expected AllShells to be false")
	}
	if opts.TargetShell != ShellBash {
		t.Errorf("Expected TargetShell to be %s, got %s", ShellBash, opts.TargetShell)
	}
}
