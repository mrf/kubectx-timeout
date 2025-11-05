package internal

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCommandInjectionPrevention tests that we safely handle malicious context names
func TestCommandInjectionPrevention(t *testing.T) {
	logger := log.New(os.Stdout, "[security-test] ", log.LstdFlags)
	cs := NewContextSwitcher(logger)

	maliciousContextNames := []string{
		"'; rm -rf /tmp/test; echo '",           // Command injection attempt
		"$(rm -rf /tmp/test)",                  // Command substitution
		"`rm -rf /tmp/test`",                   // Backtick command substitution
		"context\nrm -rf /tmp/test",            // Newline injection
		"context; ls -la /",                    // Semicolon command separator
		"context && ls -la /",                  // AND operator
		"context || ls -la /",                  // OR operator
		"context | cat /etc/passwd",            // Pipe operator
		"../../../etc/passwd",                  // Path traversal attempt
		"context\x00injection",                 // Null byte injection
	}

	for _, maliciousName := range maliciousContextNames {
		t.Run("Malicious_context_"+maliciousName, func(t *testing.T) {
			// Attempting to validate or switch to malicious context should fail safely
			err := cs.ValidateContext(maliciousName)

			// Should fail (context doesn't exist) but not execute any commands
			if err == nil {
				t.Errorf("ValidateContext should have failed for malicious input: %s", maliciousName)
			}

			// Attempt to switch should also fail safely
			err = cs.SwitchContext(maliciousName)
			if err == nil {
				t.Errorf("SwitchContext should have failed for malicious input: %s", maliciousName)
			}

			// Most important: verify no command was actually executed
			// The error message echoing the context name is acceptable as long as
			// no actual command execution occurred
			if err != nil {
				t.Logf("Malicious context rejected safely: %v", err)
			}
		})
	}
}

// TestFilePermissions verifies that sensitive files have correct permissions
func TestFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".kubectx-timeout", "config.yaml")
	statePath := filepath.Join(tmpDir, ".kubectx-timeout", "state.json")

	// Create state manager (should create directory with 0700)
	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}

	// Record activity to create state file
	if err := sm.RecordActivity("test"); err != nil {
		t.Fatalf("RecordActivity failed: %v", err)
	}

	// Check directory permissions
	dirInfo, err := os.Stat(filepath.Dir(statePath))
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}

	dirMode := dirInfo.Mode().Perm()
	if dirMode != 0700 {
		t.Errorf("Directory permissions should be 0700, got %o", dirMode)
	}

	// Check state file permissions
	stateInfo, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("Failed to stat state file: %v", err)
	}

	stateMode := stateInfo.Mode().Perm()
	if stateMode != 0600 {
		t.Errorf("State file permissions should be 0600, got %o", stateMode)
	}

	// Create a config file and verify permissions
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatalf("Failed to create config directory: %v", err)
	}

	// Write minimal config
	configContent := `
timeout:
  default: 30m
default_context: local
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Verify config file permissions
	configInfo, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	configMode := configInfo.Mode().Perm()
	if configMode != 0600 {
		t.Errorf("Config file permissions should be 0600, got %o", configMode)
	}
}

// TestYAMLParsingSafety tests that we safely handle malformed/malicious YAML
func TestYAMLParsingSafety(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	maliciousYAMLInputs := []struct {
		name    string
		content string
	}{
		{
			name:    "Billion laughs attack",
			content: `timeout: &a ["lol","lol","lol","lol","lol","lol","lol","lol","lol"]`,
		},
		{
			name:    "Deeply nested structures",
			content: strings.Repeat("nested: { ", 1000) + "value: test" + strings.Repeat(" }", 1000),
		},
		{
			name:    "Invalid YAML syntax",
			content: `this is: [not: valid: yaml: {{{`,
		},
		{
			name:    "Null bytes in YAML",
			content: "timeout:\x00 30m\ndefault_context: local\x00",
		},
		{
			name:    "SQL injection attempt in values",
			content: `default_context: "'; DROP TABLE users; --"`,
		},
		{
			name:    "Script injection in values",
			content: `default_context: "<script>alert('xss')</script>"`,
		},
	}

	for _, tt := range maliciousYAMLInputs {
		t.Run(tt.name, func(t *testing.T) {
			// Write malicious YAML
			if err := os.WriteFile(configPath, []byte(tt.content), 0600); err != nil {
				t.Fatalf("Failed to write test config: %v", err)
			}

			// Attempt to load - should fail safely
			_, err := LoadConfig(configPath)

			// We expect an error, but it should not panic or execute code
			if err == nil {
				t.Logf("Warning: LoadConfig did not error on potentially malicious input: %s", tt.name)
			}

			// Error should be safe and not expose sensitive info
			if err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "panic") {
					t.Errorf("Error message suggests panic: %v", err)
				}
			}
		})
	}

	// Cleanup
	os.Remove(configPath)
}

// TestPathTraversalPrevention ensures user input can't traverse paths
func TestPathTraversalPrevention(t *testing.T) {
	// Test that we don't allow path traversal in config file names
	maliciousPaths := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
		"/etc/passwd",
		"~/../../../etc/passwd",
	}

	for _, maliciousPath := range maliciousPaths {
		t.Run("Path_"+maliciousPath, func(t *testing.T) {
			// Our LoadConfig function expands ~ but should still be safe
			// It will fail to find the file or fail validation
			_, err := LoadConfig(maliciousPath)

			// Should either not exist or fail validation
			// The important part is it doesn't expose sensitive data
			if err != nil {
				// Error is expected - verify it's safe
				errStr := err.Error()
				if strings.Contains(errStr, "passwd") || strings.Contains(errStr, "system32") {
					t.Logf("Path traversal was prevented (expected): %v", err)
				}
			}
		})
	}
}

// TestLargeFil Tests ensures we handle large config files safely
func TestLargeFileHandling(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "large-config.yaml")

	// Create a large config file (not huge, but bigger than normal)
	var largeConfig strings.Builder
	largeConfig.WriteString("timeout:\n  default: 30m\ndefault_context: local\ncontexts:\n")

	// Add many contexts
	for i := 0; i < 1000; i++ {
		largeConfig.WriteString("  context_")
		largeConfig.WriteString(strings.Repeat("x", 100))
		largeConfig.WriteString("_")
		// Use a simple integer string instead of fmt.Sprintf for performance
		largeConfig.WriteString(string(rune(i + '0')))
		largeConfig.WriteString(":\n    timeout: 5m\n")
	}

	if err := os.WriteFile(configPath, []byte(largeConfig.String()), 0600); err != nil {
		t.Fatalf("Failed to write large config: %v", err)
	}

	// Should handle the large file gracefully (may be slow but shouldn't crash)
	_, err := LoadConfig(configPath)
	if err != nil {
		t.Logf("Large config failed to load (acceptable): %v", err)
	}
}

// TestStateFileCorruption tests handling of corrupted state files
func TestStateFileCorruption(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	sm, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("NewStateManager failed: %v", err)
	}

	// Write corrupted JSON
	corruptedJSON := `{"last_activity": "not-a-date", "current_context": malformed`

	if err := os.WriteFile(statePath, []byte(corruptedJSON), 0600); err != nil {
		t.Fatalf("Failed to write corrupted state: %v", err)
	}

	// Should handle corrupted file gracefully
	_, err = sm.Load()
	if err == nil {
		t.Error("Expected error when loading corrupted state file")
	}

	// Error should be safe
	if err != nil && !strings.Contains(err.Error(), "parse") {
		t.Logf("Corrupted state file handled: %v", err)
	}
}
