package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// SAFETY NOTE: These tests run shell scripts in isolated temporary directories.
// They do NOT modify your actual kubectl config, shell profile, or PATH.
// All test artifacts are cleaned up automatically by t.TempDir().

// TestKubectlWrapperIntegration tests the kubectl shell wrapper in a real shell environment
func TestKubectlWrapperIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	shells := []string{"bash", "zsh"}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			// Check if shell is available
			if _, err := exec.LookPath(shell); err != nil {
				t.Skipf("Skipping test: %s not found in PATH", shell)
			}

			// Setup isolated test environment (auto-cleaned on test completion)
			tmpDir := t.TempDir()

			// Verify we're actually in a temp directory for safety
			if !strings.Contains(tmpDir, "TestKubectlWrapperIntegration") {
				t.Fatalf("Safety check failed: tmpDir doesn't look like a test directory: %s", tmpDir)
			}

			// Create mock kubectl script that only writes to log files (safe operation)
			mockKubectl := filepath.Join(tmpDir, "kubectl")
			mockScript := fmt.Sprintf(`#!/bin/bash
# SAFE: This mock only writes to temp directory
echo "mock-kubectl-called" >> %s/kubectl-calls.log
echo "$@" >> %s/kubectl-args.log
echo "kubectl output"
exit 0
`, tmpDir, tmpDir)
			if err := os.WriteFile(mockKubectl, []byte(mockScript), 0755); err != nil {
				t.Fatalf("Failed to create mock kubectl: %v", err)
			}

			// Create mock kubectx-timeout binary that only records activity to log
			mockBinary := filepath.Join(tmpDir, "kubectx-timeout")
			recordScript := fmt.Sprintf(`#!/bin/bash
# SAFE: This mock only writes to temp directory
if [ "$1" = "record-activity" ]; then
    echo "record-activity-called:$(date +%%s%%N)" >> "%s/record-calls.log"
    exit 0
fi
echo "Error: unexpected argument: $1" >&2
exit 1
`, tmpDir)
			if err := os.WriteFile(mockBinary, []byte(recordScript), 0755); err != nil {
				t.Fatalf("Failed to create mock binary: %v", err)
			}

			// Generate shell integration (validates code before execution)
			integration, err := GenerateShellIntegration(shell, mockBinary)
			if err != nil {
				t.Fatalf("GenerateShellIntegration failed: %v", err)
			}

			// Safety check: verify generated integration doesn't contain suspicious patterns
			if strings.Contains(integration, "rm -rf") || strings.Contains(integration, "/etc/") {
				t.Fatalf("Safety check failed: generated integration contains suspicious commands")
			}

			// Create test script that sources integration and calls kubectl
			// NOTE: PATH is only modified within this subprocess
			testScript := filepath.Join(tmpDir, "test.sh")
			script := fmt.Sprintf(`#!/bin/%s
set -e
# PATH modification is isolated to this subprocess only
export PATH=%s:$PATH

# Source the integration (tested code)
%s

# Call kubectl with some arguments (calls our mock)
kubectl get pods --namespace=default
`, shell, tmpDir, integration)

			if err := os.WriteFile(testScript, []byte(script), 0755); err != nil {
				t.Fatalf("Failed to create test script: %v", err)
			}

			// Execute the test script in isolated subprocess
			cmd := exec.Command(shell, testScript)
			cmd.Dir = tmpDir // Run in temp directory
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Test script failed: %v\nOutput: %s", err, output)
			}

			t.Logf("Script output: %s", output)

			// Verify kubectl was called
			kubectlCalls, err := os.ReadFile(filepath.Join(tmpDir, "kubectl-calls.log"))
			if err != nil {
				t.Fatalf("kubectl was not called")
			}
			if !strings.Contains(string(kubectlCalls), "mock-kubectl-called") {
				t.Error("kubectl wrapper did not call real kubectl command")
			}

			// Verify arguments were passed correctly
			kubectlArgs, err := os.ReadFile(filepath.Join(tmpDir, "kubectl-args.log"))
			if err != nil {
				t.Fatalf("kubectl args were not logged")
			}
			if !strings.Contains(string(kubectlArgs), "get pods") {
				t.Error("kubectl wrapper did not pass arguments correctly")
			}
			if !strings.Contains(string(kubectlArgs), "--namespace=default") {
				t.Error("kubectl wrapper did not pass all arguments")
			}

			// Verify record-activity was called (poll for up to 1 second)
			recordCallsPath := filepath.Join(tmpDir, "record-calls.log")
			var recordCalls []byte
			for i := 0; i < 20; i++ {
				recordCalls, err = os.ReadFile(recordCallsPath)
				if err == nil && len(recordCalls) > 0 {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			if err != nil || len(recordCalls) == 0 {
				t.Fatalf("record-activity was not called (waited 1s)")
			}
			if !strings.Contains(string(recordCalls), "record-activity-called") {
				t.Error("kubectl wrapper did not call record-activity")
			}
		})
	}
}

// TestKubectxWrapperIntegrationSuccess tests kubectx wrapper with successful context switch
func TestKubectxWrapperIntegrationSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	shells := []string{"bash", "zsh"}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			// Check if shell is available
			if _, err := exec.LookPath(shell); err != nil {
				t.Skipf("Skipping test: %s not found in PATH", shell)
			}

			// Setup isolated test environment
			tmpDir := t.TempDir()

			// Safety check
			if !strings.Contains(tmpDir, "TestKubectxWrapperIntegrationSuccess") {
				t.Fatalf("Safety check failed: tmpDir doesn't look like a test directory: %s", tmpDir)
			}

			t.Logf("Test tmpDir: %s", tmpDir)

			// Create mock kubectx script that succeeds and records timing
			mockKubectx := filepath.Join(tmpDir, "kubectx")
			mockScript := fmt.Sprintf(`#!/bin/bash
# SAFE: This mock only writes to temp directory
echo "kubectx-called:$(date +%%s%%N)" >> %s/kubectx-calls.log
echo "$@" >> %s/kubectx-args.log
echo "Switched to context \"PROD\"."
exit 0
`, tmpDir, tmpDir)
			if err := os.WriteFile(mockKubectx, []byte(mockScript), 0755); err != nil {
				t.Fatalf("Failed to create mock kubectx: %v", err)
			}

			// Create mock kubectx-timeout binary that records timing
			mockBinary := filepath.Join(tmpDir, "kubectx-timeout")
			recordScript := fmt.Sprintf(`#!/bin/bash
# SAFE: This mock only writes to temp directory
if [ "$1" = "record-activity" ]; then
    echo "record-activity-called:$(date +%%s%%N)" >> "%s/record-calls.log"
    exit 0
fi
echo "Error: unexpected argument: $1" >&2
exit 1
`, tmpDir)
			if err := os.WriteFile(mockBinary, []byte(recordScript), 0755); err != nil {
				t.Fatalf("Failed to create mock binary: %v", err)
			}

			// Generate shell integration
			integration, err := GenerateShellIntegration(shell, mockBinary)
			if err != nil {
				t.Fatalf("GenerateShellIntegration failed: %v", err)
			}

			// Safety check
			if strings.Contains(integration, "rm -rf") || strings.Contains(integration, "/etc/") {
				t.Fatalf("Safety check failed: generated integration contains suspicious commands")
			}

			// Create test script
			testScript := filepath.Join(tmpDir, "test.sh")
			script := fmt.Sprintf(`#!/bin/%s
set -e
export PATH=%s:$PATH

# Source the integration
%s

# Call kubectx with argument
kubectx PROD
exit_code=$?

# Verify exit code was preserved
if [ $exit_code -ne 0 ]; then
    echo "Exit code not preserved: $exit_code"
    exit 1
fi
`, shell, tmpDir, integration)

			if err := os.WriteFile(testScript, []byte(script), 0755); err != nil {
				t.Fatalf("Failed to create test script: %v", err)
			}

			// Execute the test script in isolated subprocess
			cmd := exec.Command(shell, testScript)
			cmd.Dir = tmpDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Test script failed: %v\nOutput: %s", err, output)
			}

			t.Logf("Script output: %s", output)

			// Verify kubectx was called
			kubectxCalls, err := os.ReadFile(filepath.Join(tmpDir, "kubectx-calls.log"))
			if err != nil {
				t.Fatalf("kubectx was not called")
			}
			if !strings.Contains(string(kubectxCalls), "kubectx-called") {
				t.Error("kubectx wrapper did not call real kubectx command")
			}

			// Verify arguments were passed
			kubectxArgs, err := os.ReadFile(filepath.Join(tmpDir, "kubectx-args.log"))
			if err != nil {
				t.Fatalf("kubectx args were not logged")
			}
			if !strings.Contains(string(kubectxArgs), "PROD") {
				t.Error("kubectx wrapper did not pass context argument")
			}

			// Verify record-activity was called (poll for up to 1 second)
			recordCallsPath := filepath.Join(tmpDir, "record-calls.log")
			var recordCalls []byte
			for i := 0; i < 20; i++ {
				recordCalls, err = os.ReadFile(recordCallsPath)
				if err == nil && len(recordCalls) > 0 {
					break
				}
				time.Sleep(50 * time.Millisecond)
			}
			if err != nil || len(recordCalls) == 0 {
				t.Fatalf("record-activity was not called after successful kubectx (waited 1s)")
			}
			if !strings.Contains(string(recordCalls), "record-activity-called") {
				t.Error("kubectx wrapper did not call record-activity after successful switch")
			}

			// Verify timing: kubectx should be called BEFORE record-activity
			kubectxTime := extractTimestamp(t, string(kubectxCalls), "kubectx-called:")
			recordTime := extractTimestamp(t, string(recordCalls), "record-activity-called:")

			if recordTime <= kubectxTime {
				t.Errorf("Race condition: record-activity (%d) was called before or at same time as kubectx (%d)",
					recordTime, kubectxTime)
			}
		})
	}
}

// TestKubectxWrapperIntegrationFailure tests kubectx wrapper when context switch fails
func TestKubectxWrapperIntegrationFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	shells := []string{"bash", "zsh"}
	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			// Check if shell is available
			if _, err := exec.LookPath(shell); err != nil {
				t.Skipf("Skipping test: %s not found in PATH", shell)
			}

			// Setup isolated test environment
			tmpDir := t.TempDir()

			// Safety check
			if !strings.Contains(tmpDir, "TestKubectxWrapperIntegrationFailure") {
				t.Fatalf("Safety check failed: tmpDir doesn't look like a test directory: %s", tmpDir)
			}

			// Create mock kubectx script that FAILS
			mockKubectx := filepath.Join(tmpDir, "kubectx")
			mockScript := fmt.Sprintf(`#!/bin/bash
# SAFE: This mock only writes to temp directory
echo "kubectx-failed:$(date +%%s%%N)" >> %s/kubectx-calls.log
echo "error: context 'INVALID' not found" >&2
exit 1
`, tmpDir)
			if err := os.WriteFile(mockKubectx, []byte(mockScript), 0755); err != nil {
				t.Fatalf("Failed to create mock kubectx: %v", err)
			}

			// Create mock kubectx-timeout binary
			mockBinary := filepath.Join(tmpDir, "kubectx-timeout")
			recordScript := fmt.Sprintf(`#!/bin/bash
# SAFE: This mock only writes to temp directory
if [ "$1" = "record-activity" ]; then
    echo "record-activity-called:$(date +%%s%%N)" >> %s/record-calls.log
    exit 0
fi
exit 1
`, tmpDir)
			if err := os.WriteFile(mockBinary, []byte(recordScript), 0755); err != nil {
				t.Fatalf("Failed to create mock binary: %v", err)
			}

			// Generate shell integration
			integration, err := GenerateShellIntegration(shell, mockBinary)
			if err != nil {
				t.Fatalf("GenerateShellIntegration failed: %v", err)
			}

			// Safety check
			if strings.Contains(integration, "rm -rf") || strings.Contains(integration, "/etc/") {
				t.Fatalf("Safety check failed: generated integration contains suspicious commands")
			}

			// Create test script that expects failure
			testScript := filepath.Join(tmpDir, "test.sh")
			script := fmt.Sprintf(`#!/bin/%s
export PATH=%s:$PATH

# Source the integration
%s

# Call kubectx with invalid argument (should fail)
kubectx INVALID 2>/dev/null
exit_code=$?

# Verify exit code indicates failure
if [ $exit_code -eq 0 ]; then
    echo "Exit code should be non-zero for failed kubectx"
    exit 1
fi

# Success - we properly propagated the error
exit 0
`, shell, tmpDir, integration)

			if err := os.WriteFile(testScript, []byte(script), 0755); err != nil {
				t.Fatalf("Failed to create test script: %v", err)
			}

			// Execute the test script in isolated subprocess
			cmd := exec.Command(shell, testScript)
			cmd.Dir = tmpDir
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("Test script failed: %v\nOutput: %s", err, output)
			}

			// Give background process time (if it ran)
			time.Sleep(100 * time.Millisecond)

			// Verify kubectx was called
			kubectxCalls, err := os.ReadFile(filepath.Join(tmpDir, "kubectx-calls.log"))
			if err != nil {
				t.Fatalf("kubectx was not called")
			}
			if !strings.Contains(string(kubectxCalls), "kubectx-failed") {
				t.Error("kubectx wrapper did not call real kubectx command")
			}

			// Verify record-activity was NOT called (since kubectx failed)
			recordCalls, _ := os.ReadFile(filepath.Join(tmpDir, "record-calls.log"))
			if strings.Contains(string(recordCalls), "record-activity-called") {
				t.Error("kubectx wrapper should NOT call record-activity when kubectx fails")
			}
		})
	}
}

// TestKubectxWrapperPreservesExitCode tests that exit codes are properly preserved
func TestKubectxWrapperPreservesExitCode(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	testCases := []struct {
		name     string
		exitCode int
	}{
		{"success", 0},
		{"general_error", 1},
		{"custom_error", 42},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shell := "bash"
			tmpDir := t.TempDir()

			// Safety check
			if !strings.Contains(tmpDir, "TestKubectxWrapperPreservesExitCode") {
				t.Fatalf("Safety check failed: tmpDir doesn't look like a test directory: %s", tmpDir)
			}

			// Create mock kubectx that exits with specific code
			mockKubectx := filepath.Join(tmpDir, "kubectx")
			mockScript := fmt.Sprintf(`#!/bin/bash
# SAFE: This mock just exits with a code
exit %d
`, tc.exitCode)
			if err := os.WriteFile(mockKubectx, []byte(mockScript), 0755); err != nil {
				t.Fatalf("Failed to create mock kubectx: %v", err)
			}

			// Create mock binary
			mockBinary := filepath.Join(tmpDir, "kubectx-timeout")
			recordScript := `#!/bin/bash
# SAFE: This mock just exits
if [ "$1" = "record-activity" ]; then
    exit 0
fi
exit 1
`
			if err := os.WriteFile(mockBinary, []byte(recordScript), 0755); err != nil {
				t.Fatalf("Failed to create mock binary: %v", err)
			}

			// Generate shell integration
			integration, err := GenerateShellIntegration(shell, mockBinary)
			if err != nil {
				t.Fatalf("GenerateShellIntegration failed: %v", err)
			}

			// Safety check
			if strings.Contains(integration, "rm -rf") || strings.Contains(integration, "/etc/") {
				t.Fatalf("Safety check failed: generated integration contains suspicious commands")
			}

			// Create test script that captures exit code
			testScript := filepath.Join(tmpDir, "test.sh")
			script := fmt.Sprintf(`#!/bin/%s
export PATH=%s:$PATH

# Source the integration
%s

# Call kubectx
kubectx test-context 2>/dev/null
actual_exit_code=$?

# Write exit code to file for verification
echo "$actual_exit_code" > %s/exit_code.txt
exit 0
`, shell, tmpDir, integration, tmpDir)

			if err := os.WriteFile(testScript, []byte(script), 0755); err != nil {
				t.Fatalf("Failed to create test script: %v", err)
			}

			// Execute the test script in isolated subprocess
			cmd := exec.Command(shell, testScript)
			cmd.Dir = tmpDir
			if _, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("Test script failed: %v", err)
			}

			// Verify exit code was preserved
			exitCodeBytes, err := os.ReadFile(filepath.Join(tmpDir, "exit_code.txt"))
			if err != nil {
				t.Fatalf("Failed to read exit code: %v", err)
			}

			actualExitCode := strings.TrimSpace(string(exitCodeBytes))
			expectedExitCode := fmt.Sprintf("%d", tc.exitCode)

			if actualExitCode != expectedExitCode {
				t.Errorf("Exit code not preserved: expected %s, got %s", expectedExitCode, actualExitCode)
			}
		})
	}
}

// Helper function to extract timestamp from log line
func extractTimestamp(t *testing.T, log string, prefix string) int64 {
	lines := strings.Split(log, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			timestampStr := strings.TrimPrefix(line, prefix)
			var timestamp int64
			if _, err := fmt.Sscanf(timestampStr, "%d", &timestamp); err != nil {
				t.Fatalf("Failed to parse timestamp from '%s': %v", line, err)
			}
			return timestamp
		}
	}
	t.Fatalf("Timestamp with prefix '%s' not found in log", prefix)
	return 0
}
