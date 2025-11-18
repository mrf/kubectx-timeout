package internal

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// requireFswatch skips the test if fswatch is not available or not on macOS
// fswatch is required for file monitoring tests and only works on macOS
func requireFswatch(t *testing.T) {
	t.Helper()

	// Check if we're on macOS
	if runtime.GOOS != "darwin" {
		t.Skip("fswatch file monitoring only works on macOS (requires FSEvents API)")
	}

	// Check if fswatch is installed
	if _, err := exec.LookPath("fswatch"); err != nil {
		t.Skip("fswatch not installed - install with: brew install fswatch")
	}
}

// TestWatchKubeconfigDetectsChanges verifies that the file watcher detects
// context changes when the kubeconfig file is modified
func TestWatchKubeconfigDetectsChanges(t *testing.T) {
	requireFswatch(t)
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	configPath := filepath.Join(tmpDir, "config.yaml")
	statePath := filepath.Join(tmpDir, "state.json")

	// Create config
	configContent := `
timeout:
  default: 30m
  check_interval: 1s
default_context: test-default
daemon:
  enabled: true
  log_level: info
notifications:
  enabled: false
safety:
  check_active_kubectl: false
  validate_default_context: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Create daemon
	daemon, err := NewDaemon(configPath, statePath)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	// Start daemon in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- daemon.Run()
	}()

	// Give the file watcher time to start
	time.Sleep(200 * time.Millisecond)

	// Get initial activity timestamp
	initialActivity, initialContext, err := daemon.stateManager.GetLastActivity()
	if err != nil {
		t.Fatalf("Failed to get initial activity: %v", err)
	}
	t.Logf("Initial context: %s, initial activity: %v", initialContext, initialActivity)

	// Wait a bit to ensure we can detect a time difference
	time.Sleep(100 * time.Millisecond)

	// Modify the kubeconfig to switch context
	kubeconfigPath := GetKubeconfigPath()
	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: test-prod
clusters:
- cluster:
    server: https://fake-cluster-1.example.com
  name: fake-cluster-prod
- cluster:
    server: https://fake-cluster-2.example.com
  name: fake-cluster-stage
- cluster:
    server: https://fake-cluster-3.example.com
  name: fake-cluster-test
contexts:
- context:
    cluster: fake-cluster-prod
    user: fake-user
  name: test-prod
- context:
    cluster: fake-cluster-stage
    user: fake-user
  name: test-stage
- context:
    cluster: fake-cluster-test
    user: fake-user
  name: test-default
users:
- name: fake-user
  user:
    token: fake-token-for-testing
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("Failed to write kubeconfig: %v", err)
	}

	// Give the file watcher time to detect the change and record activity
	time.Sleep(500 * time.Millisecond)

	// Check that activity was recorded
	newActivity, newContext, err := daemon.stateManager.GetLastActivity()
	if err != nil {
		t.Fatalf("Failed to get new activity: %v", err)
	}

	// Verify context changed
	if newContext != "test-prod" {
		t.Errorf("Expected context to be 'test-prod', got '%s'", newContext)
	}

	// Verify activity timestamp was updated
	if !newActivity.After(initialActivity) {
		t.Errorf("Expected activity timestamp to be updated, but it wasn't. Initial: %v, New: %v", initialActivity, newActivity)
	}

	t.Logf("File watcher successfully detected context change from '%s' to '%s'", initialContext, newContext)
	t.Logf("Activity updated from %v to %v", initialActivity, newActivity)

	// Shutdown daemon
	daemon.Shutdown()

	// Wait for shutdown
	select {
	case <-errChan:
	case <-time.After(2 * time.Second):
		t.Error("Daemon did not shut down within timeout")
	}
}

// TestWatchKubeconfigExtendsTimeoutOnModification verifies that the file watcher
// records activity when the kubeconfig is modified, even if context stays the same
// This is intentional - any kubeconfig modification indicates K8s activity and should extend timeout
func TestWatchKubeconfigExtendsTimeoutOnModification(t *testing.T) {
	requireFswatch(t)
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	configPath := filepath.Join(tmpDir, "config.yaml")
	statePath := filepath.Join(tmpDir, "state.json")

	configContent := `
timeout:
  default: 30m
  check_interval: 1s
default_context: test-default
daemon:
  enabled: true
  log_level: info
notifications:
  enabled: false
safety:
  check_active_kubectl: false
  validate_default_context: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	daemon, err := NewDaemon(configPath, statePath)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- daemon.Run()
	}()

	time.Sleep(200 * time.Millisecond)

	// Get initial activity timestamp
	initialActivity, initialContext, err := daemon.stateManager.GetLastActivity()
	if err != nil {
		t.Fatalf("Failed to get initial activity: %v", err)
	}

	// Wait to ensure we can detect time differences
	time.Sleep(100 * time.Millisecond)

	// Modify the kubeconfig but keep the same context (e.g., adding a new cluster)
	kubeconfigPath := GetKubeconfigPath()
	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: test-default
clusters:
- cluster:
    server: https://fake-cluster-1.example.com
  name: fake-cluster-prod
- cluster:
    server: https://fake-cluster-2.example.com
  name: fake-cluster-stage
- cluster:
    server: https://fake-cluster-3.example.com
  name: fake-cluster-test
- cluster:
    server: https://new-fake-cluster.example.com
  name: new-cluster
contexts:
- context:
    cluster: fake-cluster-prod
    user: fake-user
  name: test-prod
- context:
    cluster: fake-cluster-stage
    user: fake-user
  name: test-stage
- context:
    cluster: fake-cluster-test
    user: fake-user
  name: test-default
- context:
    cluster: new-cluster
    user: fake-user
  name: new-context
users:
- name: fake-user
  user:
    token: fake-token-for-testing
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("Failed to write kubeconfig: %v", err)
	}

	// Give the file watcher time to process the event
	time.Sleep(500 * time.Millisecond)

	// Check that activity WAS updated (any kubeconfig modification extends timeout)
	newActivity, newContext, err := daemon.stateManager.GetLastActivity()
	if err != nil {
		t.Fatalf("Failed to get new activity: %v", err)
	}

	// Verify context didn't change
	if newContext != initialContext {
		t.Errorf("Context should not have changed, got '%s', want '%s'", newContext, initialContext)
	}

	// Verify activity timestamp WAS updated (kubeconfig modification should extend timeout)
	if !newActivity.After(initialActivity) {
		t.Errorf("Activity timestamp should have been updated when kubeconfig was modified")
		t.Logf("Initial: %v, New: %v", initialActivity, newActivity)
	}

	t.Logf("File watcher correctly extended timeout on kubeconfig modification in context '%s'", newContext)

	daemon.Shutdown()
	select {
	case <-errChan:
	case <-time.After(2 * time.Second):
		t.Error("Daemon did not shut down within timeout")
	}
}

// TestWatchKubeconfigGracefulDegradation verifies that the daemon continues
// to run even if file watching fails
func TestWatchKubeconfigGracefulDegradation(t *testing.T) {
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	configPath := filepath.Join(tmpDir, "config.yaml")
	statePath := filepath.Join(tmpDir, "state.json")

	configContent := `
timeout:
  default: 30m
  check_interval: 1s
default_context: test-default
daemon:
  enabled: true
  log_level: info
notifications:
  enabled: false
safety:
  check_active_kubectl: false
  validate_default_context: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Point KUBECONFIG to a non-existent file
	nonExistentKubeconfig := filepath.Join(tmpDir, "nonexistent", "kubeconfig.yaml")
	if err := os.Setenv("KUBECONFIG", nonExistentKubeconfig); err != nil {
		t.Fatalf("Failed to set KUBECONFIG: %v", err)
	}

	daemon, err := NewDaemon(configPath, statePath)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	// Daemon should still run even if file watching fails
	errChan := make(chan error, 1)
	go func() {
		errChan <- daemon.Run()
	}()

	// Give it time to start
	time.Sleep(300 * time.Millisecond)

	// Daemon should still be running despite file watching failure
	select {
	case err := <-errChan:
		t.Errorf("Daemon exited unexpectedly: %v", err)
	default:
		t.Log("Daemon is still running despite file watching failure (expected)")
	}

	daemon.Shutdown()
	select {
	case <-errChan:
	case <-time.After(2 * time.Second):
		t.Error("Daemon did not shut down within timeout")
	}
}

// TestWatchKubeconfigHandlesFileRecreation verifies that the watcher
// can handle the kubeconfig file being removed and recreated
func TestWatchKubeconfigHandlesFileRecreation(t *testing.T) {
	requireFswatch(t)
	tmpDir := t.TempDir()
	restoreKubeconfig := setupTestKubeconfig(t, tmpDir)
	defer restoreKubeconfig()

	configPath := filepath.Join(tmpDir, "config.yaml")
	statePath := filepath.Join(tmpDir, "state.json")

	configContent := `
timeout:
  default: 30m
  check_interval: 1s
default_context: test-default
daemon:
  enabled: true
  log_level: info
notifications:
  enabled: false
safety:
  check_active_kubectl: false
  validate_default_context: false
`
	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	daemon, err := NewDaemon(configPath, statePath)
	if err != nil {
		t.Fatalf("NewDaemon failed: %v", err)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- daemon.Run()
	}()

	time.Sleep(200 * time.Millisecond)

	kubeconfigPath := GetKubeconfigPath()

	// Remove the kubeconfig file (simulating atomic write pattern)
	if err := os.Remove(kubeconfigPath); err != nil {
		t.Fatalf("Failed to remove kubeconfig: %v", err)
	}

	// Wait for the removal to be detected
	time.Sleep(200 * time.Millisecond)

	// Recreate with a different context
	kubeconfigContent := `apiVersion: v1
kind: Config
current-context: test-stage
clusters:
- cluster:
    server: https://fake-cluster-2.example.com
  name: fake-cluster-stage
contexts:
- context:
    cluster: fake-cluster-stage
    user: fake-user
  name: test-stage
users:
- name: fake-user
  user:
    token: fake-token-for-testing
`
	if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("Failed to recreate kubeconfig: %v", err)
	}

	// Give the watcher time to detect the recreation and process the change
	time.Sleep(500 * time.Millisecond)

	// Verify the daemon is still running
	select {
	case err := <-errChan:
		t.Errorf("Daemon exited unexpectedly: %v", err)
	default:
		t.Log("Daemon is still running after file recreation (expected)")
	}

	daemon.Shutdown()
	select {
	case <-errChan:
	case <-time.After(2 * time.Second):
		t.Error("Daemon did not shut down within timeout")
	}
}

// TestDaemonGetKubeconfigPath tests the helper function for getting kubeconfig path (daemon-specific)
func TestDaemonGetKubeconfigPath(t *testing.T) {
	// Save original KUBECONFIG
	originalKubeconfig := os.Getenv("KUBECONFIG")
	defer func() {
		if originalKubeconfig != "" {
			os.Setenv("KUBECONFIG", originalKubeconfig)
		} else {
			os.Unsetenv("KUBECONFIG")
		}
	}()

	// Test with KUBECONFIG set
	testPath := "/tmp/test-kubeconfig.yaml"
	if err := os.Setenv("KUBECONFIG", testPath); err != nil {
		t.Fatalf("Failed to set KUBECONFIG: %v", err)
	}

	path := GetKubeconfigPath()
	if path != testPath {
		t.Errorf("Expected path '%s', got '%s'", testPath, path)
	}

	// Test with multiple paths (colon-separated)
	multiPath := "/path/one:/path/two:/path/three"
	if err := os.Setenv("KUBECONFIG", multiPath); err != nil {
		t.Fatalf("Failed to set KUBECONFIG: %v", err)
	}

	path = GetKubeconfigPath()
	if path != "/path/one" {
		t.Errorf("Expected first path '/path/one', got '%s'", path)
	}

	// Test with KUBECONFIG unset (should use default)
	if err := os.Unsetenv("KUBECONFIG"); err != nil {
		t.Fatalf("Failed to unset KUBECONFIG: %v", err)
	}

	path = GetKubeconfigPath()
	if path == "" {
		t.Error("Expected non-empty default path")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("Expected absolute path, got '%s'", path)
	}

	t.Logf("Default kubeconfig path: %s", path)
}
