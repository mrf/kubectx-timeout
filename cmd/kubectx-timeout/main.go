package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/markferree/kubectx-timeout/internal"
)

const (
	version = "1.0.0"
)

func main() {
	// Parse command-line flags
	var (
		configPath  = flag.String("config", "~/.kubectx-timeout/config.yaml", "Path to configuration file")
		statePath   = flag.String("state", "~/.kubectx-timeout/state.json", "Path to state file")
		showVersion = flag.Bool("version", false, "Show version information")
		initMode    = flag.Bool("init", false, "Initialize configuration")
	)
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("kubectx-timeout version %s\n", version)
		os.Exit(0)
	}

	// Handle init mode
	if *initMode {
		if err := initializeConfig(*configPath); err != nil {
			log.Fatalf("Failed to initialize configuration: %v", err)
		}
		fmt.Println("Configuration initialized successfully")
		os.Exit(0)
	}

	// Create daemon
	daemon, err := internal.NewDaemon(*configPath, *statePath)
	if err != nil {
		log.Fatalf("Failed to create daemon: %v", err)
	}

	// Run daemon
	if err := daemon.Run(); err != nil {
		log.Fatalf("Daemon exited with error: %v", err)
	}
}

// initializeConfig creates a default configuration file
func initializeConfig(configPath string) error {
	// Expand ~ to home directory
	if len(configPath) > 0 && configPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, configPath[1:])
	}

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists at %s", configPath)
	}

	// Create config directory
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Get available contexts
	contexts, err := internal.GetAvailableContexts()
	if err != nil {
		return fmt.Errorf("failed to get available contexts: %w", err)
	}

	if len(contexts) == 0 {
		return fmt.Errorf("no kubectl contexts available - please configure kubectl first")
	}

	// Show available contexts
	fmt.Println("Available kubectl contexts:")
	for i, ctx := range contexts {
		fmt.Printf("  %d. %s\n", i+1, ctx)
	}

	// Get current context as default suggestion
	current, err := internal.GetCurrentContext()
	if err == nil && current != "" {
		fmt.Printf("\nCurrent context: %s\n", current)
	}

	// Create default config with a valid context
	config := internal.DefaultConfig()
	if len(contexts) > 0 {
		config.DefaultContext = contexts[0]
	}

	// Save config (would need a SaveConfig function in internal package)
	// For now, just create a basic YAML file
	configContent := fmt.Sprintf(`# kubectx-timeout configuration
timeout:
  default: 30m          # Default timeout for all contexts
  check_interval: 30s   # How often to check for inactivity

default_context: %s    # Context to switch to after timeout

# Context-specific timeouts (optional)
contexts:
  # production:
  #   timeout: 5m

daemon:
  enabled: true
  log_level: info
  log_file: daemon.log
  log_max_size: 10
  log_max_backups: 5

notifications:
  enabled: true
  method: both  # terminal, macos, or both

safety:
  check_active_kubectl: true
  validate_default_context: true
  # never_switch_from:
  #   - production
  # never_switch_to:
  #   - production

state_file: state.json

shell:
  generate_wrapper: true
  shells:
    - bash
    - zsh
`, config.DefaultContext)

	if err := os.WriteFile(configPath, []byte(configContent), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("\nConfiguration file created at: %s\n", configPath)
	return nil
}
