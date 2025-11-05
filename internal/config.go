package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the kubectx-timeout configuration
type Config struct {
	Timeout        TimeoutConfig      `yaml:"timeout"`
	DefaultContext string             `yaml:"default_context"`
	Contexts       map[string]Context `yaml:"contexts,omitempty"`
	Daemon         DaemonConfig       `yaml:"daemon"`
	Notifications  NotificationConfig `yaml:"notifications"`
	Safety         SafetyConfig       `yaml:"safety"`
	StateFile      string             `yaml:"state_file"`
	Shell          ShellConfig        `yaml:"shell"`
}

// TimeoutConfig holds global timeout settings
type TimeoutConfig struct {
	Default       time.Duration `yaml:"default"`
	CheckInterval time.Duration `yaml:"check_interval"`
}

// Context holds context-specific timeout settings
type Context struct {
	Timeout       time.Duration `yaml:"timeout"`
	ConfirmSwitch bool          `yaml:"confirm_switch,omitempty"`
}

// DaemonConfig holds daemon behavior settings
type DaemonConfig struct {
	Enabled       bool   `yaml:"enabled"`
	LogLevel      string `yaml:"log_level"`
	LogFile       string `yaml:"log_file"`
	LogMaxSize    int    `yaml:"log_max_size"`
	LogMaxBackups int    `yaml:"log_max_backups"`
}

// NotificationConfig holds notification settings
type NotificationConfig struct {
	Enabled bool   `yaml:"enabled"`
	Method  string `yaml:"method"`
	Message string `yaml:"message,omitempty"`
}

// SafetyConfig holds safety feature settings
type SafetyConfig struct {
	CheckActiveKubectl     bool     `yaml:"check_active_kubectl"`
	NeverSwitchFrom        []string `yaml:"never_switch_from,omitempty"`
	NeverSwitchTo          []string `yaml:"never_switch_to,omitempty"`
	ValidateDefaultContext bool     `yaml:"validate_default_context"`
}

// ShellConfig holds shell integration settings
type ShellConfig struct {
	GenerateWrapper bool     `yaml:"generate_wrapper"`
	Shells          []string `yaml:"shells"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	// Try to detect a safe default context
	defaultCtx := detectSafeDefaultContext()

	return &Config{
		Timeout: TimeoutConfig{
			Default:       30 * time.Minute,
			CheckInterval: 30 * time.Second,
		},
		DefaultContext: defaultCtx,
		Daemon: DaemonConfig{
			Enabled:       true,
			LogLevel:      "info",
			LogFile:       "daemon.log",
			LogMaxSize:    10,
			LogMaxBackups: 5,
		},
		Notifications: NotificationConfig{
			Enabled: true,
			Method:  "both",
		},
		Safety: SafetyConfig{
			CheckActiveKubectl:     true,
			ValidateDefaultContext: true,
		},
		StateFile: "state.json",
		Shell: ShellConfig{
			GenerateWrapper: true,
			Shells:          []string{"bash", "zsh"},
		},
	}
}

// detectSafeDefaultContext tries to find a safe default context from available kubectl contexts
func detectSafeDefaultContext() string {
	// Get all available contexts
	contexts, err := GetAvailableContexts()
	if err != nil || len(contexts) == 0 {
		return "CONFIGURE_ME"
	}

	// Patterns that indicate a safe/dev context (in priority order)
	safePatterns := []string{
		"local",
		"docker-desktop",
		"minikube",
		"kind-",
		"dev",
		"development",
		"test",
	}

	// Patterns that indicate dangerous/production contexts
	dangerousPatterns := []string{
		"prod",
		"production",
		"stage",
		"staging",
		"prd",
	}

	// First pass: look for explicitly safe contexts
	for _, pattern := range safePatterns {
		for _, ctx := range contexts {
			ctxLower := strings.ToLower(ctx)
			// Check if context name contains the safe pattern
			if strings.Contains(ctxLower, pattern) {
				// But make sure it doesn't also contain a dangerous pattern
				isDangerous := false
				for _, danger := range dangerousPatterns {
					if strings.Contains(ctxLower, danger) {
						isDangerous = true
						break
					}
				}
				if !isDangerous {
					return ctx
				}
			}
		}
	}

	// No obviously safe context found - require configuration
	return "CONFIGURE_ME"
}

// LoadConfig loads configuration from the specified file path
// If the file doesn't exist, returns default configuration
// If the file exists but is invalid, returns an error
func LoadConfig(path string) (*Config, error) {
	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, return default config
		return DefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Start with default config and unmarshal on top of it
	// This ensures any missing fields get default values
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check required fields
	if c.DefaultContext == "" {
		return fmt.Errorf("default_context is required")
	}

	// Check if default context needs to be configured
	if c.DefaultContext == "CONFIGURE_ME" {
		return fmt.Errorf("default_context must be configured - run 'kubectx-timeout init' to set up")
	}

	// Validate timeout durations
	if c.Timeout.Default <= 0 {
		return fmt.Errorf("timeout.default must be positive")
	}
	if c.Timeout.CheckInterval <= 0 {
		return fmt.Errorf("timeout.check_interval must be positive")
	}
	if c.Timeout.CheckInterval > c.Timeout.Default {
		return fmt.Errorf("timeout.check_interval must be less than timeout.default")
	}

	// Validate log level
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[c.Daemon.LogLevel] {
		return fmt.Errorf("daemon.log_level must be one of: debug, info, warn, error")
	}

	// Validate notification method
	validMethods := map[string]bool{
		"terminal": true,
		"macos":    true,
		"both":     true,
	}
	if !validMethods[c.Notifications.Method] {
		return fmt.Errorf("notifications.method must be one of: terminal, macos, both")
	}

	// Validate context-specific timeouts
	for name, ctx := range c.Contexts {
		if ctx.Timeout <= 0 {
			return fmt.Errorf("timeout for context '%s' must be positive", name)
		}
	}

	// Check for conflicts in safety settings
	if c.Safety.ValidateDefaultContext {
		for _, ctx := range c.Safety.NeverSwitchTo {
			if ctx == c.DefaultContext {
				return fmt.Errorf("default_context '%s' is in never_switch_to list", c.DefaultContext)
			}
		}
	}

	return nil
}

// GetTimeoutForContext returns the timeout duration for a specific context
// If the context has a specific timeout configured, returns that
// Otherwise returns the default timeout
func (c *Config) GetTimeoutForContext(contextName string) time.Duration {
	if ctx, ok := c.Contexts[contextName]; ok {
		return ctx.Timeout
	}
	return c.Timeout.Default
}
