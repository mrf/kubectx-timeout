package internal

import (
	"os"
	"testing"
)

func TestGetConfigDir(t *testing.T) {
	tests := []struct {
		name           string
		xdgConfigHome  string
		home           string
		expectedSuffix string
	}{
		{
			name:           "XDG_CONFIG_HOME set",
			xdgConfigHome:  "/custom/config",
			home:           "/home/user",
			expectedSuffix: "/custom/config/kubectx-timeout",
		},
		{
			name:           "XDG_CONFIG_HOME not set",
			xdgConfigHome:  "",
			home:           "/home/user",
			expectedSuffix: "/home/user/.config/kubectx-timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			oldXDG := os.Getenv("XDG_CONFIG_HOME")
			oldHome := os.Getenv("HOME")
			defer func() {
				os.Setenv("XDG_CONFIG_HOME", oldXDG)
				os.Setenv("HOME", oldHome)
			}()

			// Set test environment
			os.Setenv("HOME", tt.home)
			if tt.xdgConfigHome != "" {
				os.Setenv("XDG_CONFIG_HOME", tt.xdgConfigHome)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}

			result := GetConfigDir()
			if result != tt.expectedSuffix {
				t.Errorf("GetConfigDir() = %v, want %v", result, tt.expectedSuffix)
			}
		})
	}
}

func TestGetStateDir(t *testing.T) {
	tests := []struct {
		name           string
		xdgStateHome   string
		home           string
		expectedSuffix string
	}{
		{
			name:           "XDG_STATE_HOME set",
			xdgStateHome:   "/custom/state",
			home:           "/home/user",
			expectedSuffix: "/custom/state/kubectx-timeout",
		},
		{
			name:           "XDG_STATE_HOME not set",
			xdgStateHome:   "",
			home:           "/home/user",
			expectedSuffix: "/home/user/.local/state/kubectx-timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore environment
			oldXDG := os.Getenv("XDG_STATE_HOME")
			oldHome := os.Getenv("HOME")
			defer func() {
				os.Setenv("XDG_STATE_HOME", oldXDG)
				os.Setenv("HOME", oldHome)
			}()

			// Set test environment
			os.Setenv("HOME", tt.home)
			if tt.xdgStateHome != "" {
				os.Setenv("XDG_STATE_HOME", tt.xdgStateHome)
			} else {
				os.Unsetenv("XDG_STATE_HOME")
			}

			result := GetStateDir()
			if result != tt.expectedSuffix {
				t.Errorf("GetStateDir() = %v, want %v", result, tt.expectedSuffix)
			}
		})
	}
}

func TestGetConfigPath(t *testing.T) {
	tests := []struct {
		name           string
		xdgConfigHome  string
		home           string
		expectedSuffix string
	}{
		{
			name:           "XDG path",
			xdgConfigHome:  "/custom/config",
			home:           "/home/user",
			expectedSuffix: "/custom/config/kubectx-timeout/config.yaml",
		},
		{
			name:           "Default XDG path",
			xdgConfigHome:  "",
			home:           "/home/user",
			expectedSuffix: "/home/user/.config/kubectx-timeout/config.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldXDG := os.Getenv("XDG_CONFIG_HOME")
			oldHome := os.Getenv("HOME")
			defer func() {
				os.Setenv("XDG_CONFIG_HOME", oldXDG)
				os.Setenv("HOME", oldHome)
			}()

			os.Setenv("HOME", tt.home)
			if tt.xdgConfigHome != "" {
				os.Setenv("XDG_CONFIG_HOME", tt.xdgConfigHome)
			} else {
				os.Unsetenv("XDG_CONFIG_HOME")
			}

			result := GetConfigPath()
			if result != tt.expectedSuffix {
				t.Errorf("GetConfigPath() = %v, want %v", result, tt.expectedSuffix)
			}
		})
	}
}

func TestGetStatePath(t *testing.T) {
	tests := []struct {
		name           string
		xdgStateHome   string
		home           string
		expectedSuffix string
	}{
		{
			name:           "XDG state path",
			xdgStateHome:   "/custom/state",
			home:           "/home/user",
			expectedSuffix: "/custom/state/kubectx-timeout/state.json",
		},
		{
			name:           "Default XDG state path",
			xdgStateHome:   "",
			home:           "/home/user",
			expectedSuffix: "/home/user/.local/state/kubectx-timeout/state.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldXDG := os.Getenv("XDG_STATE_HOME")
			oldHome := os.Getenv("HOME")
			defer func() {
				os.Setenv("XDG_STATE_HOME", oldXDG)
				os.Setenv("HOME", oldHome)
			}()

			os.Setenv("HOME", tt.home)
			if tt.xdgStateHome != "" {
				os.Setenv("XDG_STATE_HOME", tt.xdgStateHome)
			} else {
				os.Unsetenv("XDG_STATE_HOME")
			}

			result := GetStatePath()
			if result != tt.expectedSuffix {
				t.Errorf("GetStatePath() = %v, want %v", result, tt.expectedSuffix)
			}
		})
	}
}

func TestGetKubeconfigPath(t *testing.T) {
	tests := []struct {
		name         string
		kubeconfig   string
		home         string
		expectedPath string
	}{
		{
			name:         "KUBECONFIG env set to single path",
			kubeconfig:   "/custom/path/kubeconfig.yaml",
			home:         "/home/user",
			expectedPath: "/custom/path/kubeconfig.yaml",
		},
		{
			name:         "KUBECONFIG env with multiple paths (colon-separated)",
			kubeconfig:   "/first/path/config:/second/path/config:/third/path/config",
			home:         "/home/user",
			expectedPath: "/first/path/config",
		},
		{
			name:         "KUBECONFIG not set, use default",
			kubeconfig:   "",
			home:         "/home/user",
			expectedPath: "/home/user/.kube/config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldKubeconfig := os.Getenv("KUBECONFIG")
			oldHome := os.Getenv("HOME")
			defer func() {
				if oldKubeconfig != "" {
					os.Setenv("KUBECONFIG", oldKubeconfig)
				} else {
					os.Unsetenv("KUBECONFIG")
				}
				os.Setenv("HOME", oldHome)
			}()

			os.Setenv("HOME", tt.home)
			if tt.kubeconfig != "" {
				os.Setenv("KUBECONFIG", tt.kubeconfig)
			} else {
				os.Unsetenv("KUBECONFIG")
			}

			result := GetKubeconfigPath()
			if result != tt.expectedPath {
				t.Errorf("GetKubeconfigPath() = %v, want %v", result, tt.expectedPath)
			}
		})
	}
}
