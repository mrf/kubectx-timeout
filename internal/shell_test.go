package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectShell(t *testing.T) {
	tests := []struct {
		name        string
		shellEnv    string
		expectError bool
	}{
		{
			name:        "bash detected",
			shellEnv:    "/bin/bash",
			expectError: false,
		},
		{
			name:        "zsh detected",
			shellEnv:    "/bin/zsh",
			expectError: false,
		},
		{
			name:        "fish detected",
			shellEnv:    "/usr/bin/fish",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original SHELL
			originalShell := os.Getenv("SHELL")
			defer os.Setenv("SHELL", originalShell)

			// Set test SHELL
			os.Setenv("SHELL", tt.shellEnv)

			shell, err := DetectShell()
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError {
				expected := filepath.Base(tt.shellEnv)
				if shell != expected {
					t.Errorf("Expected shell %s, got %s", expected, shell)
				}
			}
		})
	}
}

func TestIsValidShell(t *testing.T) {
	tests := []struct {
		shell string
		valid bool
	}{
		{"bash", true},
		{"zsh", true},
		{"fish", true},
		{"sh", false},
		{"ksh", false},
		{"tcsh", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			result := isValidShell(tt.shell)
			if result != tt.valid {
				t.Errorf("isValidShell(%s) = %v, want %v", tt.shell, result, tt.valid)
			}
		})
	}
}

func TestGetShellProfilePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		shell        string
		expectedPath string
		expectError  bool
	}{
		{
			shell: ShellBash,
			// For bash, we need to check what actually exists
			// The function prefers .bash_profile over .bashrc
			expectedPath: func() string {
				bashProfile := filepath.Join(home, ".bash_profile")
				bashrc := filepath.Join(home, ".bashrc")
				if _, err := os.Stat(bashProfile); err == nil {
					return bashProfile
				}
				return bashrc
			}(),
			expectError: false,
		},
		{
			shell:        ShellZsh,
			expectedPath: filepath.Join(home, ".zshrc"),
			expectError:  false,
		},
		{
			shell:        ShellFish,
			expectedPath: filepath.Join(home, ".config", "fish", "config.fish"),
			expectError:  false,
		},
		{
			shell:       "unsupported",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			path, err := GetShellProfilePath(tt.shell)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError && path != tt.expectedPath {
				t.Errorf("Expected path %s, got %s", tt.expectedPath, path)
			}
		})
	}
}

// TestGetShellProfilePathBashPreference specifically tests the bash profile preference logic
// This test validates that .bash_profile is preferred over .bashrc when both exist
func TestGetShellProfilePathBashPreference(t *testing.T) {
	// Save original HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	tests := []struct {
		name           string
		createFiles    []string
		expectedSuffix string
		description    string
	}{
		{
			name:           "bash_profile exists alone",
			createFiles:    []string{".bash_profile"},
			expectedSuffix: ".bash_profile",
			description:    "Should return .bash_profile when only it exists",
		},
		{
			name:           "bashrc exists alone",
			createFiles:    []string{".bashrc"},
			expectedSuffix: ".bashrc",
			description:    "Should return .bashrc when only it exists",
		},
		{
			name:           "both files exist",
			createFiles:    []string{".bash_profile", ".bashrc"},
			expectedSuffix: ".bash_profile",
			description:    "Should prefer .bash_profile when both exist",
		},
		{
			name:           "neither file exists",
			createFiles:    []string{},
			expectedSuffix: ".bashrc",
			description:    "Should return .bashrc as fallback when neither exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh temp directory for this test case
			testHome, err := os.MkdirTemp("", "bash-test-*")
			if err != nil {
				t.Fatalf("Failed to create test home dir: %v", err)
			}
			defer os.RemoveAll(testHome)

			// Set HOME to test directory
			os.Setenv("HOME", testHome)

			// Create the specified files
			for _, filename := range tt.createFiles {
				filePath := filepath.Join(testHome, filename)
				if err := os.WriteFile(filePath, []byte("# test content"), 0644); err != nil {
					t.Fatalf("Failed to create %s: %v", filename, err)
				}
			}

			// Call the function
			result, err := GetShellProfilePath(ShellBash)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify the result
			expectedPath := filepath.Join(testHome, tt.expectedSuffix)
			if result != expectedPath {
				t.Errorf("%s\nExpected: %s\nGot:      %s", tt.description, expectedPath, result)
			}

			// Log for clarity
			t.Logf("Test: %s\nCreated: %v\nReturned: %s",
				tt.description, tt.createFiles, filepath.Base(result))
		})
	}

	// Restore original HOME
	os.Setenv("HOME", originalHome)
}

func TestGetShellIntegrationCode(t *testing.T) {
	binaryPath := "/usr/local/bin/kubectx-timeout"

	tests := []struct {
		shell       string
		expectError bool
	}{
		{ShellBash, false},
		{ShellZsh, false},
		{ShellFish, false},
		{"unsupported", true},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			code, err := GetShellIntegrationCode(tt.shell, binaryPath)
			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tt.expectError {
				// Check that code contains expected markers
				if !strings.Contains(code, IntegrationStartMarker) {
					t.Errorf("Code missing start marker")
				}
				if !strings.Contains(code, IntegrationEndMarker) {
					t.Errorf("Code missing end marker")
				}
				// Check that code contains binary path
				if !strings.Contains(code, binaryPath) {
					t.Errorf("Code missing binary path")
				}
				// Check that code contains kubectl reference
				if !strings.Contains(code, "kubectl") {
					t.Errorf("Code missing kubectl reference")
				}
			}
		})
	}
}

func TestInstallAndUninstallIntegration(t *testing.T) {
	// Create a temporary directory for test
	tmpDir, err := os.MkdirTemp("", "shell-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profilePath := filepath.Join(tmpDir, ".testrc")
	binaryPath := "/usr/local/bin/kubectx-timeout"

	// Test installation
	t.Run("install integration", func(t *testing.T) {
		code, err := GetShellIntegrationCode(ShellBash, binaryPath)
		if err != nil {
			t.Fatalf("Failed to get integration code: %v", err)
		}

		err = InstallIntegration(profilePath, code)
		if err != nil {
			t.Fatalf("Failed to install integration: %v", err)
		}

		// Verify installation
		installed, err := IsIntegrationInstalled(profilePath)
		if err != nil {
			t.Fatalf("Failed to check installation: %v", err)
		}
		if !installed {
			t.Errorf("Integration not detected after installation")
		}

		// Note: Backup is only created if profile already existed
		// In this test, we're creating a new file, so no backup is expected

		// Verify content
		content, err := os.ReadFile(profilePath)
		if err != nil {
			t.Fatalf("Failed to read profile: %v", err)
		}
		if !strings.Contains(string(content), IntegrationStartMarker) {
			t.Errorf("Profile missing start marker")
		}
		if !strings.Contains(string(content), IntegrationEndMarker) {
			t.Errorf("Profile missing end marker")
		}
	})

	// Test duplicate installation prevention
	t.Run("prevent duplicate installation", func(t *testing.T) {
		code, err := GetShellIntegrationCode(ShellBash, binaryPath)
		if err != nil {
			t.Fatalf("Failed to get integration code: %v", err)
		}

		err = InstallIntegration(profilePath, code)
		if err == nil {
			t.Errorf("Expected error for duplicate installation, got none")
		}
		if !strings.Contains(err.Error(), "already installed") {
			t.Errorf("Expected 'already installed' error, got: %v", err)
		}
	})

	// Test uninstallation
	t.Run("uninstall integration", func(t *testing.T) {
		err := UninstallIntegration(profilePath)
		if err != nil {
			t.Fatalf("Failed to uninstall integration: %v", err)
		}

		// Verify uninstallation
		installed, err := IsIntegrationInstalled(profilePath)
		if err != nil {
			t.Fatalf("Failed to check installation: %v", err)
		}
		if installed {
			t.Errorf("Integration still detected after uninstallation")
		}

		// Verify content doesn't contain markers
		content, err := os.ReadFile(profilePath)
		if err != nil {
			t.Fatalf("Failed to read profile: %v", err)
		}
		if strings.Contains(string(content), IntegrationStartMarker) {
			t.Errorf("Profile still contains start marker")
		}
		if strings.Contains(string(content), IntegrationEndMarker) {
			t.Errorf("Profile still contains end marker")
		}
	})

	// Test uninstalling when not installed
	t.Run("uninstall when not installed", func(t *testing.T) {
		err := UninstallIntegration(profilePath)
		if err != nil {
			t.Errorf("Unexpected error when uninstalling non-installed integration: %v", err)
		}
	})
}

func TestInstallIntegrationPreservesExistingContent(t *testing.T) {
	// Create a temporary directory for test
	tmpDir, err := os.MkdirTemp("", "shell-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profilePath := filepath.Join(tmpDir, ".testrc")
	binaryPath := "/usr/local/bin/kubectx-timeout"

	// Create profile with existing content
	existingContent := "# Existing content\nexport PATH=$PATH:/usr/local/bin\nalias ll='ls -la'\n"
	err = os.WriteFile(profilePath, []byte(existingContent), 0600)
	if err != nil {
		t.Fatalf("Failed to create profile: %v", err)
	}

	// Install integration
	code, err := GetShellIntegrationCode(ShellBash, binaryPath)
	if err != nil {
		t.Fatalf("Failed to get integration code: %v", err)
	}

	err = InstallIntegration(profilePath, code)
	if err != nil {
		t.Fatalf("Failed to install integration: %v", err)
	}

	// Read profile and verify existing content is preserved
	content, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("Failed to read profile: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "# Existing content") {
		t.Errorf("Existing content was not preserved")
	}
	if !strings.Contains(contentStr, "export PATH=$PATH:/usr/local/bin") {
		t.Errorf("Existing PATH export was not preserved")
	}
	if !strings.Contains(contentStr, "alias ll='ls -la'") {
		t.Errorf("Existing alias was not preserved")
	}
	if !strings.Contains(contentStr, IntegrationStartMarker) {
		t.Errorf("Integration was not added")
	}
}

func TestVerifyInstallation(t *testing.T) {
	// Create a temporary directory for test
	tmpDir, err := os.MkdirTemp("", "shell-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profilePath := filepath.Join(tmpDir, ".testrc")
	binaryPath := filepath.Join(tmpDir, "kubectx-timeout")

	// Test verification without installation
	t.Run("verify without installation", func(t *testing.T) {
		issues := VerifyInstallation(profilePath, binaryPath)
		if len(issues) == 0 {
			t.Errorf("Expected issues when integration not installed")
		}
		found := false
		for _, issue := range issues {
			if strings.Contains(issue, "not found in shell profile") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected issue about integration not found in profile")
		}
	})

	// Test verification with installation but no binary
	t.Run("verify with installation but no binary", func(t *testing.T) {
		code, err := GetShellIntegrationCode(ShellBash, binaryPath)
		if err != nil {
			t.Fatalf("Failed to get integration code: %v", err)
		}

		err = InstallIntegration(profilePath, code)
		if err != nil {
			t.Fatalf("Failed to install integration: %v", err)
		}

		issues := VerifyInstallation(profilePath, binaryPath)
		if len(issues) == 0 {
			t.Errorf("Expected issues when binary doesn't exist")
		}
		found := false
		for _, issue := range issues {
			if strings.Contains(issue, "Binary not found") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected issue about binary not found: %v", issues)
		}
	})

	// Test verification with installation and binary
	t.Run("verify with installation and binary", func(t *testing.T) {
		// Create a dummy binary
		err := os.WriteFile(binaryPath, []byte("#!/bin/bash\necho test"), 0755)
		if err != nil {
			t.Fatalf("Failed to create binary: %v", err)
		}

		issues := VerifyInstallation(profilePath, binaryPath)
		// Should only complain about kubectl if it's not in PATH
		// We can't guarantee kubectl is installed in test environment
		for _, issue := range issues {
			if !strings.Contains(issue, "kubectl") {
				t.Errorf("Unexpected issue: %s", issue)
			}
		}
	})
}

func TestIsIntegrationInstalled(t *testing.T) {
	// Create a temporary directory for test
	tmpDir, err := os.MkdirTemp("", "shell-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	profilePath := filepath.Join(tmpDir, ".testrc")

	// Test with non-existent file
	t.Run("non-existent file", func(t *testing.T) {
		installed, err := IsIntegrationInstalled(profilePath)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if installed {
			t.Errorf("Expected not installed for non-existent file")
		}
	})

	// Test with empty file
	t.Run("empty file", func(t *testing.T) {
		err := os.WriteFile(profilePath, []byte(""), 0600)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		installed, err := IsIntegrationInstalled(profilePath)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if installed {
			t.Errorf("Expected not installed for empty file")
		}
	})

	// Test with file containing marker
	t.Run("file with marker", func(t *testing.T) {
		content := "# Some content\n" + IntegrationStartMarker + "\n# More content\n"
		err := os.WriteFile(profilePath, []byte(content), 0600)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		installed, err := IsIntegrationInstalled(profilePath)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !installed {
			t.Errorf("Expected installed when marker present")
		}
	})

	// Test with file without marker
	t.Run("file without marker", func(t *testing.T) {
		content := "# Some content\nexport PATH=$PATH:/usr/local/bin\n"
		err := os.WriteFile(profilePath, []byte(content), 0600)
		if err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		installed, err := IsIntegrationInstalled(profilePath)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if installed {
			t.Errorf("Expected not installed when marker not present")
		}
	})
}
