package internal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Test default values - should auto-detect or use placeholder
	if cfg.DefaultContext == "" {
		t.Error("expected default_context to be set")
	}
	t.Logf("Default context detected as: %s", cfg.DefaultContext)

	if cfg.Timeout.Default != 30*time.Minute {
		t.Errorf("expected default timeout to be 30m, got %v", cfg.Timeout.Default)
	}

	if cfg.Timeout.CheckInterval != 30*time.Second {
		t.Errorf("expected check_interval to be 30s, got %v", cfg.Timeout.CheckInterval)
	}

	if !cfg.Daemon.Enabled {
		t.Error("expected daemon to be enabled by default")
	}

	if cfg.Daemon.LogLevel != "info" {
		t.Errorf("expected log_level to be 'info', got '%s'", cfg.Daemon.LogLevel)
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	// Load from non-existent file should return default config
	cfg, err := LoadConfig("/tmp/nonexistent-kubectx-timeout-config.yaml")
	if err != nil {
		t.Fatalf("LoadConfig failed on missing file: %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig returned nil for missing file")
	}

	// Should have default values - context will be auto-detected or placeholder
	if cfg.DefaultContext == "" {
		t.Error("expected default_context to be set")
	}
	t.Logf("Default context: %s", cfg.DefaultContext)
}

func TestLoadConfigValid(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `
timeout:
  default: 15m
  check_interval: 15s

default_context: test-context

daemon:
  enabled: true
  log_level: debug
  log_file: test.log
  log_max_size: 5
  log_max_backups: 3

contexts:
  production:
    timeout: 5m
  dev:
    timeout: 1h
`

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify loaded values
	if cfg.DefaultContext != "test-context" {
		t.Errorf("expected default_context to be 'test-context', got '%s'", cfg.DefaultContext)
	}

	if cfg.Timeout.Default != 15*time.Minute {
		t.Errorf("expected default timeout to be 15m, got %v", cfg.Timeout.Default)
	}

	if cfg.Daemon.LogLevel != "debug" {
		t.Errorf("expected log_level to be 'debug', got '%s'", cfg.Daemon.LogLevel)
	}

	// Verify context-specific timeouts
	if ctx, ok := cfg.Contexts["production"]; !ok {
		t.Error("expected 'production' context to be loaded")
	} else if ctx.Timeout != 5*time.Minute {
		t.Errorf("expected production timeout to be 5m, got %v", ctx.Timeout)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `
this is not: [valid yaml content
  because: it's malformed
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0600); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected LoadConfig to fail on invalid YAML")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		wantError bool
	}{
		{
			name: "valid default config",
			config: func() *Config {
				cfg := DefaultConfig()
				// Ensure we have a valid context for testing
				if cfg.DefaultContext == "CONFIGURE_ME" || cfg.DefaultContext == "" {
					cfg.DefaultContext = "test-context"
				}
				return cfg
			}(),
			wantError: false,
		},
		{
			name: "missing default_context",
			config: &Config{
				Timeout: TimeoutConfig{
					Default:       30 * time.Minute,
					CheckInterval: 30 * time.Second,
				},
				Daemon: DaemonConfig{LogLevel: "info"},
				Notifications: NotificationConfig{Method: "both"},
			},
			wantError: true,
		},
		{
			name: "negative default timeout",
			config: &Config{
				DefaultContext: "local",
				Timeout: TimeoutConfig{
					Default:       -1 * time.Minute,
					CheckInterval: 30 * time.Second,
				},
				Daemon: DaemonConfig{LogLevel: "info"},
				Notifications: NotificationConfig{Method: "both"},
			},
			wantError: true,
		},
		{
			name: "invalid log level",
			config: &Config{
				DefaultContext: "local",
				Timeout: TimeoutConfig{
					Default:       30 * time.Minute,
					CheckInterval: 30 * time.Second,
				},
				Daemon: DaemonConfig{LogLevel: "invalid"},
				Notifications: NotificationConfig{Method: "both"},
			},
			wantError: true,
		},
		{
			name: "invalid notification method",
			config: &Config{
				DefaultContext: "local",
				Timeout: TimeoutConfig{
					Default:       30 * time.Minute,
					CheckInterval: 30 * time.Second,
				},
				Daemon: DaemonConfig{LogLevel: "info"},
				Notifications: NotificationConfig{Method: "invalid"},
			},
			wantError: true,
		},
		{
			name: "check_interval greater than default",
			config: &Config{
				DefaultContext: "local",
				Timeout: TimeoutConfig{
					Default:       5 * time.Minute,
					CheckInterval: 10 * time.Minute,
				},
				Daemon: DaemonConfig{LogLevel: "info"},
				Notifications: NotificationConfig{Method: "both"},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestGetTimeoutForContext(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Contexts = map[string]Context{
		"production": {Timeout: 5 * time.Minute},
		"dev":        {Timeout: 1 * time.Hour},
	}

	tests := []struct {
		name        string
		contextName string
		want        time.Duration
	}{
		{
			name:        "context with specific timeout",
			contextName: "production",
			want:        5 * time.Minute,
		},
		{
			name:        "context without specific timeout",
			contextName: "staging",
			want:        30 * time.Minute, // default
		},
		{
			name:        "another context with specific timeout",
			contextName: "dev",
			want:        1 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetTimeoutForContext(tt.contextName)
			if got != tt.want {
				t.Errorf("GetTimeoutForContext(%s) = %v, want %v", tt.contextName, got, tt.want)
			}
		})
	}
}
