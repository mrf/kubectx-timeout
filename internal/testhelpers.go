package internal

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestKubeconfig creates an isolated kubeconfig for testing
// Returns a cleanup function that restores the original KUBECONFIG
func setupTestKubeconfig(t *testing.T, tmpDir string) func() {
	t.Helper()

	// Save original KUBECONFIG
	originalKubeconfig := os.Getenv("KUBECONFIG")

	// Create test kubeconfig with fake contexts
	testKubeconfig := filepath.Join(tmpDir, "test-kubeconfig.yaml")
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
	if err := os.WriteFile(testKubeconfig, []byte(kubeconfigContent), 0600); err != nil {
		t.Fatalf("Failed to create test kubeconfig: %v", err)
	}

	// Set KUBECONFIG to test file
	if err := os.Setenv("KUBECONFIG", testKubeconfig); err != nil {
		t.Fatalf("Failed to set KUBECONFIG: %v", err)
	}

	t.Logf("Using isolated test kubeconfig: %s", testKubeconfig)

	// Return cleanup function
	return func() {
		if originalKubeconfig != "" {
			if err := os.Setenv("KUBECONFIG", originalKubeconfig); err != nil {
				t.Errorf("Failed to restore KUBECONFIG: %v", err)
			}
		} else {
			if err := os.Unsetenv("KUBECONFIG"); err != nil {
				t.Errorf("Failed to unset KUBECONFIG: %v", err)
			}
		}
		t.Logf("Restored original KUBECONFIG")
	}
}
