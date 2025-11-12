package internal

import (
	"log"
	"os"
	"testing"
)

func TestNewContextSwitcher(t *testing.T) {
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	cs := NewContextSwitcher(logger)

	if cs == nil {
		t.Fatal("NewContextSwitcher returned nil")
	}

	if cs.logger == nil {
		t.Error("ContextSwitcher has nil logger")
	}

	if cs.maxRetries != 3 {
		t.Errorf("expected maxRetries to be 3, got %d", cs.maxRetries)
	}
}

func TestListContexts(t *testing.T) {
	// Setup isolated test environment to avoid leaking real context names
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	cs := NewContextSwitcher(logger)

	contexts, err := cs.ListContexts()
	if err != nil {
		t.Fatalf("ListContexts failed: %v", err)
	}

	if len(contexts) == 0 {
		t.Fatal("No contexts found from isolated kubeconfig")
	}

	// Verify we got test contexts from isolated kubeconfig
	expectedContexts := []string{"test-default", "test-prod", "test-stage"}
	if len(contexts) != len(expectedContexts) {
		t.Errorf("Expected %d contexts, got %d", len(expectedContexts), len(contexts))
	}

	t.Logf("Found %d contexts", len(contexts))
	for _, ctx := range contexts {
		t.Logf("  - %s", ctx)
	}
}

func TestValidateContext(t *testing.T) {
	// Setup isolated test environment to avoid leaking real context names
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	cs := NewContextSwitcher(logger)

	// Test validating an existing context from isolated kubeconfig
	err := cs.ValidateContext("test-default")
	if err != nil {
		t.Errorf("ValidateContext failed for existing context 'test-default': %v", err)
	}

	// Test validating a non-existent context
	err = cs.ValidateContext("definitely-does-not-exist-context")
	if err == nil {
		t.Error("ValidateContext should have failed for non-existent context")
	}
}

func TestSwitchContextSameContext(t *testing.T) {
	// Setup isolated test environment
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	cs := NewContextSwitcher(logger)

	// Use test context from isolated kubeconfig
	currentContext := "test-default"

	// Try to switch to the same context (should be no-op)
	err := cs.SwitchContext(currentContext)
	if err != nil {
		t.Errorf("SwitchContext failed when switching to same context: %v", err)
	}

	// Verify we're still on the same context
	afterContext, err := GetCurrentContext()
	if err != nil {
		t.Fatalf("Failed to get context after switch: %v", err)
	}

	if afterContext != currentContext {
		t.Errorf("Context changed unexpectedly: %s -> %s", currentContext, afterContext)
	}
}

func TestSwitchContextNonExistent(t *testing.T) {
	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	cs := NewContextSwitcher(logger)

	// Try to switch to non-existent context
	err := cs.SwitchContext("definitely-does-not-exist-context")
	if err == nil {
		t.Error("SwitchContext should have failed for non-existent context")
	}
}

func TestSwitchContextSafe(t *testing.T) {
	// Setup isolated test environment
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	cs := NewContextSwitcher(logger)

	// Use test context from isolated kubeconfig
	currentContext := "test-default"

	// Test with never_switch_to list containing the target context
	neverSwitchTo := []string{"production", "prod", currentContext}

	err := cs.SwitchContextSafe(currentContext, neverSwitchTo)
	if err == nil {
		t.Error("SwitchContextSafe should have failed when target is in never_switch_to list")
	}

	if err != nil && err.Error() != "" {
		t.Logf("Expected error: %v", err)
	}
}

func TestSwitchContextWithRetry(t *testing.T) {
	// Setup isolated test environment
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	cs := NewContextSwitcher(logger)

	// Use test contexts from isolated kubeconfig
	currentContext := "test-default"
	targetContext := "test-prod"

	t.Logf("Testing context switch from '%s' to '%s'", currentContext, targetContext)

	// Perform the switch
	err := cs.SwitchContext(targetContext)
	if err != nil {
		t.Fatalf("SwitchContext failed: %v", err)
	}

	// Verify the switch
	afterContext, err := GetCurrentContext()
	if err != nil {
		t.Fatalf("Failed to get context after switch: %v", err)
	}

	if afterContext != targetContext {
		t.Errorf("Expected context '%s', got '%s'", targetContext, afterContext)
	}

	// Switch back to original context for cleanup
	t.Logf("Switching back to original context '%s'", currentContext)
	err = cs.SwitchContext(currentContext)
	if err != nil {
		t.Errorf("Failed to switch back to original context: %v", err)
	}
}
