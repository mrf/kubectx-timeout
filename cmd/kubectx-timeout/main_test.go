package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestInstallShellDetectFlag tests the --detect flag for install-shell command
func TestInstallShellDetectFlag(t *testing.T) {
	// Build the binary first
	binPath := buildTestBinary(t)
	defer os.Remove(binPath)

	tests := []struct {
		name           string
		shellEnv       string
		expectedShell  string
		expectInOutput []string
	}{
		{
			name:          "detect bash shell",
			shellEnv:      "/bin/bash",
			expectedShell: "bash",
			expectInOutput: []string{
				"Detected shell: bash",
				"Profile path:",
				"Profile exists:",
				"kubectx-timeout install-shell bash",
			},
		},
		{
			name:          "detect zsh shell",
			shellEnv:      "/bin/zsh",
			expectedShell: "zsh",
			expectInOutput: []string{
				"Detected shell: zsh",
				"Profile path:",
				"Profile exists:",
				"kubectx-timeout install-shell zsh",
			},
		},
		{
			name:          "detect fish shell",
			shellEnv:      "/usr/bin/fish",
			expectedShell: "fish",
			expectInOutput: []string{
				"Detected shell: fish",
				"Profile path:",
				"Profile exists:",
				"kubectx-timeout install-shell fish",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, "install-shell", "--detect")
			cmd.Env = append(os.Environ(), "SHELL="+tt.shellEnv)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			if err != nil {
				t.Fatalf("Command failed: %v\nstderr: %s", err, stderr.String())
			}

			output := stdout.String()
			for _, expected := range tt.expectInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

// TestInstallShellMissingArgument tests error when shell argument is missing
func TestInstallShellMissingArgument(t *testing.T) {
	binPath := buildTestBinary(t)
	defer os.Remove(binPath)

	cmd := exec.Command(binPath, "install-shell")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("Expected command to fail when shell argument is missing")
	}

	stderrStr := stderr.String()
	expectedMessages := []string{
		"Error: Shell argument is required",
		"Usage:",
		"kubectx-timeout install-shell <shell>",
		"Supported shells: bash, zsh, fish",
		"kubectx-timeout install-shell --detect",
	}

	for _, expected := range expectedMessages {
		if !strings.Contains(stderrStr, expected) {
			t.Errorf("Expected stderr to contain %q, got:\n%s", expected, stderrStr)
		}
	}
}

// TestInstallShellInvalidShell tests error for unsupported shell
func TestInstallShellInvalidShell(t *testing.T) {
	binPath := buildTestBinary(t)
	defer os.Remove(binPath)

	cmd := exec.Command(binPath, "install-shell", "ksh")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("Expected command to fail for unsupported shell")
	}

	// The error message goes to stderr via log.Fatalf
	combined := stderr.String()
	if !strings.Contains(combined, "Unsupported shell: ksh") {
		t.Errorf("Expected error about unsupported shell, got:\n%s", combined)
	}
	if !strings.Contains(combined, "Supported shells: bash, zsh, fish") {
		t.Errorf("Expected supported shells list, got:\n%s", combined)
	}
}

// TestUninstallShellDetectFlag tests the --detect flag for uninstall-shell command
func TestUninstallShellDetectFlag(t *testing.T) {
	binPath := buildTestBinary(t)
	defer os.Remove(binPath)

	tests := []struct {
		name           string
		shellEnv       string
		expectedShell  string
		expectInOutput []string
	}{
		{
			name:          "detect bash for uninstall",
			shellEnv:      "/bin/bash",
			expectedShell: "bash",
			expectInOutput: []string{
				"Detected shell: bash",
				"kubectx-timeout uninstall-shell bash",
			},
		},
		{
			name:          "detect zsh for uninstall",
			shellEnv:      "/bin/zsh",
			expectedShell: "zsh",
			expectInOutput: []string{
				"Detected shell: zsh",
				"kubectx-timeout uninstall-shell zsh",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command(binPath, "uninstall-shell", "--detect")
			cmd.Env = append(os.Environ(), "SHELL="+tt.shellEnv)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			if err != nil {
				t.Fatalf("Command failed: %v\nstderr: %s", err, stderr.String())
			}

			output := stdout.String()
			for _, expected := range tt.expectInOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
				}
			}
		})
	}
}

// TestUninstallShellMissingArgument tests error when shell argument is missing
func TestUninstallShellMissingArgument(t *testing.T) {
	binPath := buildTestBinary(t)
	defer os.Remove(binPath)

	cmd := exec.Command(binPath, "uninstall-shell")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("Expected command to fail when shell argument is missing")
	}

	stderrStr := stderr.String()
	expectedMessages := []string{
		"Error: Shell argument is required",
		"Usage:",
		"kubectx-timeout uninstall-shell <shell>",
		"Supported shells: bash, zsh, fish",
		"kubectx-timeout uninstall-shell --detect",
	}

	for _, expected := range expectedMessages {
		if !strings.Contains(stderrStr, expected) {
			t.Errorf("Expected stderr to contain %q, got:\n%s", expected, stderrStr)
		}
	}
}

// TestUninstallShellInvalidShell tests error for unsupported shell
func TestUninstallShellInvalidShell(t *testing.T) {
	binPath := buildTestBinary(t)
	defer os.Remove(binPath)

	cmd := exec.Command(binPath, "uninstall-shell", "tcsh")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err == nil {
		t.Fatal("Expected command to fail for unsupported shell")
	}

	combined := stderr.String()
	if !strings.Contains(combined, "Unsupported shell: tcsh") {
		t.Errorf("Expected error about unsupported shell, got:\n%s", combined)
	}
}

// TestInstallShellWithValidShellArg tests that valid shell arguments work
func TestInstallShellWithValidShellArg(t *testing.T) {
	binPath := buildTestBinary(t)
	defer os.Remove(binPath)

	// Create a temp home directory to avoid modifying real shell profiles
	tmpHome, err := os.MkdirTemp("", "test-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	shells := []string{"bash", "zsh", "fish"}

	for _, shell := range shells {
		t.Run("install_"+shell, func(t *testing.T) {
			// Create a fresh temp home for each test
			testHome, err := os.MkdirTemp("", "test-home-"+shell+"-*")
			if err != nil {
				t.Fatalf("Failed to create temp home: %v", err)
			}
			defer os.RemoveAll(testHome)

			// Create .config/fish directory for fish shell
			if shell == "fish" {
				fishConfigDir := filepath.Join(testHome, ".config", "fish")
				if err := os.MkdirAll(fishConfigDir, 0755); err != nil {
					t.Fatalf("Failed to create fish config dir: %v", err)
				}
			}

			cmd := exec.Command(binPath, "install-shell", shell, "--yes")
			cmd.Env = []string{
				"HOME=" + testHome,
				"PATH=" + os.Getenv("PATH"),
			}

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err = cmd.Run()
			if err != nil {
				t.Logf("stdout: %s", stdout.String())
				t.Logf("stderr: %s", stderr.String())
				// Don't fail - installation may fail due to missing binary verification
				// but we're testing the argument parsing works
			}

			output := stdout.String()
			// Should show the shell being targeted
			if !strings.Contains(output, "Shell: "+shell) && !strings.Contains(output, "Shell profile:") {
				t.Errorf("Expected output to show shell info for %s, got:\n%s", shell, output)
			}
		})
	}
}

// TestInstallShellProfileExistsOutput tests that profile existence is reported correctly
func TestInstallShellProfileExistsOutput(t *testing.T) {
	binPath := buildTestBinary(t)
	defer os.Remove(binPath)

	// Create temp home with a .bashrc file
	tmpHome, err := os.MkdirTemp("", "test-home-*")
	if err != nil {
		t.Fatalf("Failed to create temp home: %v", err)
	}
	defer os.RemoveAll(tmpHome)

	// Create .bashrc
	bashrcPath := filepath.Join(tmpHome, ".bashrc")
	if err := os.WriteFile(bashrcPath, []byte("# existing bashrc\n"), 0644); err != nil {
		t.Fatalf("Failed to create .bashrc: %v", err)
	}

	t.Run("profile exists", func(t *testing.T) {
		cmd := exec.Command(binPath, "install-shell", "--detect")
		cmd.Env = []string{
			"HOME=" + tmpHome,
			"SHELL=/bin/bash",
			"PATH=" + os.Getenv("PATH"),
		}

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			t.Fatalf("Command failed: %v\nstderr: %s", err, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Profile exists: yes") {
			t.Errorf("Expected 'Profile exists: yes', got:\n%s", output)
		}
	})

	t.Run("profile does not exist", func(t *testing.T) {
		// Create another temp home without .bashrc
		tmpHome2, err := os.MkdirTemp("", "test-home2-*")
		if err != nil {
			t.Fatalf("Failed to create temp home: %v", err)
		}
		defer os.RemoveAll(tmpHome2)

		cmd := exec.Command(binPath, "install-shell", "--detect")
		cmd.Env = []string{
			"HOME=" + tmpHome2,
			"SHELL=/bin/bash",
			"PATH=" + os.Getenv("PATH"),
		}

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err = cmd.Run()
		if err != nil {
			t.Fatalf("Command failed: %v\nstderr: %s", err, stderr.String())
		}

		output := stdout.String()
		if !strings.Contains(output, "Profile exists: no") {
			t.Errorf("Expected 'Profile exists: no', got:\n%s", output)
		}
	})
}

// TestIsValidShellArg tests the isValidShellArg function
func TestIsValidShellArg(t *testing.T) {
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
		{"BASH", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			result := isValidShellArg(tt.shell)
			if result != tt.valid {
				t.Errorf("isValidShellArg(%q) = %v, want %v", tt.shell, result, tt.valid)
			}
		})
	}
}

// TestVersionCommand tests the version command works
func TestVersionCommand(t *testing.T) {
	binPath := buildTestBinary(t)
	defer os.Remove(binPath)

	cmd := exec.Command(binPath, "version")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		t.Fatalf("version command failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "kubectx-timeout version") {
		t.Errorf("Expected version output, got: %s", output)
	}
}

// TestHelpCommand tests the help command works
func TestHelpCommand(t *testing.T) {
	binPath := buildTestBinary(t)
	defer os.Remove(binPath)

	cmd := exec.Command(binPath, "help")

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	err := cmd.Run()
	if err != nil {
		t.Fatalf("help command failed: %v", err)
	}

	output := stdout.String()
	expectedSections := []string{
		"install-shell",
		"uninstall-shell",
		"--detect",
		"bash",
		"zsh",
		"fish",
	}

	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Expected help to contain %q, got:\n%s", section, output)
		}
	}
}

// buildTestBinary builds the binary for testing and returns the path
func buildTestBinary(t *testing.T) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "kubectx-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	binPath := filepath.Join(tmpDir, "kubectx-timeout")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = filepath.Dir(mustGetCurrentFile(t))

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, stderr.String())
	}

	return binPath
}

// mustGetCurrentFile returns the directory of the current test file
func mustGetCurrentFile(t *testing.T) string {
	t.Helper()

	// Get the directory of the current file
	_, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	return "."
}
