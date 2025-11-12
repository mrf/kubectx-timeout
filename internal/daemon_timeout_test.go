package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestDaemonTimeoutTriggersSwitch tests that the daemon actually switches context after timeout
func TestDaemonTimeoutTriggersSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow test in short mode")
	}

	// Setup isolated test environment
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	// Use test contexts from isolated kubeconfig
	prodContext := "test-prod"
	safeContext := "test-default"

	t.Logf("Testing timeout: %s (prod) -> %s (safe)", prodContext, safeContext)

	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	switcher := NewContextSwitcher(logger)

	// Setup config and state files
	configPath := filepath.Join(tmpDir, "config.yaml")
	statePath := filepath.Join(tmpDir, "state.json")
	logPath := filepath.Join(tmpDir, "daemon.log")

	// Create config with SHORT timeout for fast testing
	configContent := fmt.Sprintf(`timeout:
  default: 2s
  check_interval: 500ms
  contexts:
    %s: 2s

default_context: %s

daemon:
  enabled: true
  log_file: %s

safety:
  never_switch_to: []
  never_switch_from: []
`, prodContext, safeContext, logPath)

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create daemon
	daemon, err := NewDaemon(configPath, statePath)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	// Start daemon in background
	daemonCtx, daemonCancel := context.WithCancel(context.Background())
	defer daemonCancel()

	daemon.ctx = daemonCtx
	daemon.cancel = daemonCancel

	daemonErrCh := make(chan error, 1)
	go func() {
		daemonErrCh <- daemon.Run()
	}()
	defer func() {
		daemon.Shutdown()
		<-daemonErrCh // Wait for daemon to stop
	}()

	// Give daemon time to start
	time.Sleep(200 * time.Millisecond)

	// Step 1: Switch to production context
	t.Logf("Switching to production context: %s", prodContext)
	if err := switcher.SwitchContextSafe(prodContext, []string{}); err != nil {
		t.Fatalf("Failed to switch to prod context: %v", err)
	}

	// Verify we're on prod context
	currentCtx, _ := GetCurrentContext()
	if currentCtx != prodContext {
		t.Fatalf("Not on prod context after switch, got: %s", currentCtx)
	}

	// Step 2: Record activity in prod context (simulate kubectl command)
	stateManager, err := NewStateManager(statePath)
	if err != nil {
		t.Fatalf("Failed to create state manager: %v", err)
	}

	t.Logf("Recording activity in context: %s", prodContext)
	if err := stateManager.RecordActivity(prodContext); err != nil {
		t.Fatalf("Failed to record activity: %v", err)
	}

	// Verify state was recorded
	lastActivity, lastContext, err := stateManager.GetLastActivity()
	if err != nil {
		t.Fatalf("Failed to get last activity: %v", err)
	}
	t.Logf("State: last_activity=%v, current_context=%s", lastActivity, lastContext)

	if lastContext != prodContext {
		t.Errorf("State has wrong context: expected %s, got %s", prodContext, lastContext)
	}

	// Step 3: Wait for timeout to exceed (2s timeout + 500ms check interval + buffer)
	waitTime := 3 * time.Second
	t.Logf("Waiting %v for timeout to trigger...", waitTime)
	time.Sleep(waitTime)

	// Step 4: Verify daemon switched to safe context
	currentCtx, err = GetCurrentContext()
	if err != nil {
		t.Fatalf("Failed to get current context after timeout: %v", err)
	}

	t.Logf("Current context after timeout: %s", currentCtx)

	if currentCtx != safeContext {
		// Read daemon logs for debugging
		logs, _ := os.ReadFile(logPath)
		t.Errorf("Daemon did not switch context after timeout!\nExpected: %s\nActual: %s\n\nDaemon logs:\n%s",
			safeContext, currentCtx, string(logs))
	}

	// Cleanup happens automatically via defer restoreKubeconfig()
}

// TestDaemonDoesNotSwitchWhenActive tests that daemon doesn't switch if activity is ongoing
func TestDaemonDoesNotSwitchWhenActive(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow test in short mode")
	}

	// Setup isolated test environment
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	// Use test contexts from isolated kubeconfig
	prodContext := "test-prod"
	safeContext := "test-default"

	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	switcher := NewContextSwitcher(logger)

	// Setup config and state files
	configPath := filepath.Join(tmpDir, "config.yaml")
	statePath := filepath.Join(tmpDir, "state.json")
	logPath := filepath.Join(tmpDir, "daemon.log")

	configContent := fmt.Sprintf(`timeout:
  default: 1s
  check_interval: 300ms

default_context: %s

daemon:
  enabled: true
  log_file: %s

safety:
  never_switch_to: []
  never_switch_from: []
`, safeContext, logPath)

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create daemon
	daemon, err := NewDaemon(configPath, statePath)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	// Start daemon
	daemonCtx, daemonCancel := context.WithCancel(context.Background())
	defer daemonCancel()

	daemon.ctx = daemonCtx
	daemon.cancel = daemonCancel

	daemonErrCh := make(chan error, 1)
	go func() {
		daemonErrCh <- daemon.Run()
	}()
	defer func() {
		daemon.Shutdown()
		<-daemonErrCh
	}()

	time.Sleep(200 * time.Millisecond)

	// Switch to prod context
	if err := switcher.SwitchContextSafe(prodContext, []string{}); err != nil {
		t.Fatalf("Failed to switch to prod context: %v", err)
	}

	// Record activity and keep recording every 500ms (before timeout)
	stateManager, _ := NewStateManager(statePath)

	for i := 0; i < 5; i++ {
		stateManager.RecordActivity(prodContext)
		t.Logf("Recording activity (iteration %d)", i+1)
		time.Sleep(500 * time.Millisecond)
	}

	// After 5 iterations (2.5s), we should STILL be on prod context
	// because we kept recording activity
	currentCtx, _ := GetCurrentContext()
	if currentCtx != prodContext {
		logs, _ := os.ReadFile(logPath)
		t.Errorf("Daemon switched context despite ongoing activity!\nExpected: %s\nActual: %s\n\nLogs:\n%s",
			prodContext, currentCtx, string(logs))
	}

	// Cleanup happens automatically via defer restoreKubeconfig()
}

// TestDaemonRecordsActivityAfterSwitch tests whether daemon updates state after switching
func TestDaemonRecordsActivityAfterSwitch(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping slow test in short mode")
	}

	// Setup isolated test environment
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	// Use test contexts from isolated kubeconfig
	prodContext := "test-prod"
	safeContext := "test-default"

	logger := log.New(os.Stdout, "[test] ", log.LstdFlags)
	switcher := NewContextSwitcher(logger)

	// Setup config and state files
	configPath := filepath.Join(tmpDir, "config.yaml")
	statePath := filepath.Join(tmpDir, "state.json")
	logPath := filepath.Join(tmpDir, "daemon.log")

	configContent := fmt.Sprintf(`timeout:
  default: 1s
  check_interval: 300ms

default_context: %s

daemon:
  enabled: true
  log_file: %s

safety:
  never_switch_to: []
  never_switch_from: []
`, safeContext, logPath)

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create daemon
	daemon, err := NewDaemon(configPath, statePath)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	// Start daemon
	daemonCtx, daemonCancel := context.WithCancel(context.Background())
	defer daemonCancel()

	daemon.ctx = daemonCtx
	daemon.cancel = daemonCancel

	daemonErrCh := make(chan error, 1)
	go func() {
		daemonErrCh <- daemon.Run()
	}()
	defer func() {
		daemon.Shutdown()
		<-daemonErrCh
	}()

	time.Sleep(200 * time.Millisecond)

	// Switch to prod and record activity
	switcher.SwitchContextSafe(prodContext, []string{})
	stateManager, _ := NewStateManager(statePath)
	stateManager.RecordActivity(prodContext)

	// Wait for timeout
	t.Logf("Waiting for timeout...")
	time.Sleep(2 * time.Second)

	// Verify switched to safe context
	currentCtx, _ := GetCurrentContext()
	if currentCtx != safeContext {
		t.Fatalf("Daemon didn't switch to safe context: got %s", currentCtx)
	}

	// Check state file - what context does it have recorded?
	_, stateContext, err := stateManager.GetLastActivity()
	if err != nil {
		t.Fatalf("Failed to get state: %v", err)
	}

	t.Logf("After daemon switch - kubectl context: %s, state context: %s", currentCtx, stateContext)

	// CRITICAL CHECK: After daemon switches context, it should update the state file
	// to reflect the new context. Otherwise state is out of sync with reality.
	if stateContext != safeContext {
		logs, _ := os.ReadFile(logPath)
		t.Errorf("BUG: State file not updated after daemon switch!\nExpected state context: %s\nActual state context: %s\n\nLogs:\n%s",
			safeContext, stateContext, string(logs))
	}

	// Wait one more check interval to ensure daemon doesn't try to switch again
	time.Sleep(500 * time.Millisecond)

	currentCtx2, _ := GetCurrentContext()
	if currentCtx2 != safeContext {
		t.Errorf("Context changed again after daemon switch! Now at: %s", currentCtx2)
	}

	// Cleanup happens automatically via defer restoreKubeconfig()
}
