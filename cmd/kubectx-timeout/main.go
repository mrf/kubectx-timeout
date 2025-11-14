package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrf/kubectx-timeout/internal"
)

const (
	version = "1.0.0"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "version":
		cmdVersion()
	case "init":
		cmdInit()
	case "daemon":
		cmdDaemon()
	case "install-shell":
		cmdInstallShell()
	case "uninstall-shell":
		cmdUninstallShell()
	case "record-activity":
		cmdRecordActivity()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`kubectx-timeout version %s

Usage:
  kubectx-timeout <command> [options]

Commands:
  version              Show version information
  init                 Initialize configuration file
  daemon               Run the timeout monitoring daemon
  install-shell        Install shell integration (kubectl wrapper)
  uninstall-shell      Remove shell integration
  record-activity      Record kubectl activity (used by shell integration)
  help                 Show this help message

Examples:
  # Initialize configuration
  kubectx-timeout init

  # Install shell integration for your current shell
  kubectx-timeout install-shell

  # Install for a specific shell
  kubectx-timeout install-shell bash
  kubectx-timeout install-shell zsh

  # Run daemon (usually via launchd, but can run manually)
  kubectx-timeout daemon

For more information, visit: https://github.com/mrf/kubectx-timeout
`, version)
}

func cmdVersion() {
	fmt.Printf("kubectx-timeout version %s\n", version)
}

func cmdDaemon() {
	fs := flag.NewFlagSet("daemon", flag.ExitOnError)
	defaultConfigPath := internal.GetConfigPath()
	defaultStatePath := internal.GetStatePath()

	configPath := fs.String("config", defaultConfigPath, "Path to configuration file")
	statePath := fs.String("state", defaultStatePath, "Path to state file")

	fs.Parse(os.Args[2:])

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

func cmdInit() {
	defaultConfigPath := internal.GetConfigPath()

	fs := flag.NewFlagSet("init", flag.ExitOnError)
	configPath := fs.String("config", defaultConfigPath, "Path to configuration file")
	fs.Parse(os.Args[2:])

	if err := initializeConfig(*configPath); err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}
	fmt.Println("\n✓ Configuration initialized successfully")
	fmt.Printf("  Config file: %s\n", *configPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review and customize the configuration file")
	fmt.Println("  2. Run: kubectx-timeout install-shell")
	fmt.Println("  3. Restart your shell or run: source ~/.bashrc (or ~/.zshrc)")
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

func cmdInstallShell() {
	// Detect the current binary path
	defaultBinaryPath := "/usr/local/bin/kubectx-timeout" // fallback default
	if execPath, err := os.Executable(); err == nil {
		// Get absolute path
		if absPath, err := filepath.Abs(execPath); err == nil {
			defaultBinaryPath = absPath
		}
	}

	fs := flag.NewFlagSet("install-shell", flag.ExitOnError)
	noConfirm := fs.Bool("yes", false, "Skip confirmation prompts")
	noReload := fs.Bool("no-reload", false, "Don't offer to reload shell")
	binaryPath := fs.String("binary", defaultBinaryPath, "Path to kubectx-timeout binary")

	fs.Parse(os.Args[2:])

	// Determine shell
	var targetShell string
	args := fs.Args()
	if len(args) > 0 {
		// Shell specified as argument
		targetShell = args[0]
		if !isValidShellArg(targetShell) {
			log.Fatalf("Unsupported shell: %s\nSupported shells: bash, zsh, fish", targetShell)
		}
	} else {
		// Auto-detect shell
		detected, err := internal.DetectShell()
		if err != nil {
			log.Fatalf("Failed to detect shell: %v\nPlease specify shell explicitly: kubectx-timeout install-shell <bash|zsh|fish>", err)
		}
		targetShell = detected
		fmt.Printf("Detected shell: %s\n", targetShell)
	}

	// Get profile path
	profilePath, err := internal.GetShellProfilePath(targetShell)
	if err != nil {
		log.Fatalf("Failed to get shell profile path: %v", err)
	}

	fmt.Printf("Shell profile: %s\n", profilePath)
	fmt.Printf("Binary path: %s\n", *binaryPath)

	// Check if already installed
	installed, err := internal.IsIntegrationInstalled(profilePath)
	if err != nil {
		log.Fatalf("Failed to check installation status: %v", err)
	}
	if installed {
		fmt.Println("\n✓ Shell integration is already installed")
		fmt.Printf("  To reinstall, first run: kubectx-timeout uninstall-shell %s\n", targetShell)
		return
	}

	// Get integration code
	integrationCode, err := internal.GetShellIntegrationCode(targetShell, *binaryPath)
	if err != nil {
		log.Fatalf("Failed to generate integration code: %v", err)
	}

	// Show preview
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("The following will be added to your shell profile:")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println(integrationCode)
	fmt.Println(strings.Repeat("=", 60))

	// Confirm unless --yes flag is set
	if !*noConfirm {
		fmt.Print("\nDo you want to proceed with the installation? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read input: %v", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Installation cancelled")
			return
		}
	}

	// Install integration
	fmt.Println("\nInstalling shell integration...")
	if err := internal.InstallIntegration(profilePath, integrationCode); err != nil {
		log.Fatalf("Failed to install integration: %v", err)
	}

	// Create backup notice
	backupPath := profilePath + ".kubectx-timeout.backup"
	fmt.Printf("✓ Backup created: %s\n", backupPath)
	fmt.Printf("✓ Integration installed to: %s\n", profilePath)

	// Verify installation
	fmt.Println("\nVerifying installation...")
	issues := internal.VerifyInstallation(profilePath, *binaryPath)
	if len(issues) > 0 {
		fmt.Println("\n⚠ Verification found some issues:")
		for _, issue := range issues {
			fmt.Printf("  - %s\n", issue)
		}
		fmt.Println("\nTroubleshooting:")
		fmt.Printf("  - Make sure the binary exists at: %s\n", *binaryPath)
		fmt.Println("  - Make sure kubectl is installed and in your PATH")
		fmt.Println("  - Restart your shell for changes to take effect")
	} else {
		fmt.Println("✓ Installation verified successfully")
	}

	// Offer to reload shell
	if !*noReload {
		fmt.Println("\nTo activate the integration:")
		switch targetShell {
		case "bash":
			fmt.Println("  source ~/.bashrc")
			if profilePath != filepath.Join(os.Getenv("HOME"), ".bashrc") {
				fmt.Printf("  Or: source %s\n", profilePath)
			}
		case "zsh":
			fmt.Println("  source ~/.zshrc")
		case "fish":
			fmt.Println("  source ~/.config/fish/config.fish")
		}
		fmt.Println("  Or: Start a new shell")
		fmt.Println("\nNote: The integration will be active in all new shells automatically")
	}

	fmt.Println("\n✓ Installation complete!")
}

func cmdUninstallShell() {
	fs := flag.NewFlagSet("uninstall-shell", flag.ExitOnError)
	noConfirm := fs.Bool("yes", false, "Skip confirmation prompts")

	fs.Parse(os.Args[2:])

	// Determine shell
	var targetShell string
	args := fs.Args()
	if len(args) > 0 {
		// Shell specified as argument
		targetShell = args[0]
		if !isValidShellArg(targetShell) {
			log.Fatalf("Unsupported shell: %s\nSupported shells: bash, zsh, fish", targetShell)
		}
	} else {
		// Auto-detect shell
		detected, err := internal.DetectShell()
		if err != nil {
			log.Fatalf("Failed to detect shell: %v\nPlease specify shell explicitly: kubectx-timeout uninstall-shell <bash|zsh|fish>", err)
		}
		targetShell = detected
		fmt.Printf("Detected shell: %s\n", targetShell)
	}

	// Get profile path
	profilePath, err := internal.GetShellProfilePath(targetShell)
	if err != nil {
		log.Fatalf("Failed to get shell profile path: %v", err)
	}

	fmt.Printf("Shell profile: %s\n", profilePath)

	// Check if installed
	installed, err := internal.IsIntegrationInstalled(profilePath)
	if err != nil {
		log.Fatalf("Failed to check installation status: %v", err)
	}
	if !installed {
		fmt.Println("\n✓ Shell integration is not installed (nothing to remove)")
		return
	}

	// Confirm unless --yes flag is set
	if !*noConfirm {
		fmt.Print("\nDo you want to remove the shell integration? [y/N]: ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Failed to read input: %v", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))
		if response != "y" && response != "yes" {
			fmt.Println("Uninstallation cancelled")
			return
		}
	}

	// Uninstall integration
	fmt.Println("\nRemoving shell integration...")
	if err := internal.UninstallIntegration(profilePath); err != nil {
		log.Fatalf("Failed to uninstall integration: %v", err)
	}

	// Create backup notice
	backupPath := profilePath + ".kubectx-timeout.backup"
	fmt.Printf("✓ Backup created: %s\n", backupPath)
	fmt.Printf("✓ Integration removed from: %s\n", profilePath)

	fmt.Println("\n✓ Uninstallation complete!")
	fmt.Println("  Restart your shell for changes to take effect")
}

func cmdRecordActivity() {
	defaultStatePath := internal.GetStatePath()
	defaultConfigPath := internal.GetConfigPath()

	fs := flag.NewFlagSet("record-activity", flag.ExitOnError)
	statePath := fs.String("state", defaultStatePath, "Path to state file")
	configPath := fs.String("config", defaultConfigPath, "Path to configuration file")
	fs.Parse(os.Args[2:])

	// Create activity tracker
	tracker, err := internal.NewActivityTracker(*statePath, *configPath)
	if err != nil {
		// Silent failure - don't break kubectl workflow
		// Error is logged but we exit 0
		log.Printf("Warning: failed to create activity tracker: %v", err)
		return
	}

	// Record activity
	if err := tracker.RecordActivity(); err != nil {
		// Silent failure - don't break kubectl workflow
		// Error is logged but we exit 0
		log.Printf("Warning: failed to record activity: %v", err)
	}
}

func isValidShellArg(shell string) bool {
	switch shell {
	case "bash", "zsh", "fish":
		return true
	default:
		return false
	}
}
