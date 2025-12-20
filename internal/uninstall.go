package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// UninstallOptions contains options for uninstallation
type UninstallOptions struct {
	KeepConfig  bool   // Keep configuration and state files
	KeepBinary  bool   // Keep the binary file
	Force       bool   // Skip confirmations
	AllShells   bool   // Remove from all detected shell profiles
	TargetShell string // Specific shell to target (bash, zsh, fish)
	BinaryPath  string // Path to the binary to remove
}

// UninstallResult tracks what was removed during uninstallation
type UninstallResult struct {
	DaemonStopped   bool
	LaunchdRemoved  bool
	ShellsProcessed []string
	ConfigRemoved   bool
	StateRemoved    bool
	BinaryRemoved   bool
	BackupsCreated  []string
	Errors          []error
}

// Uninstall performs a complete uninstallation of kubectx-timeout
func Uninstall(opts UninstallOptions) (*UninstallResult, error) {
	result := &UninstallResult{
		ShellsProcessed: []string{},
		BackupsCreated:  []string{},
		Errors:          []error{},
	}

	// Step 1: Stop and remove daemon (macOS launchd)
	if runtime.GOOS == "darwin" {
		if err := stopAndRemoveDaemon(result); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("daemon removal: %w", err))
		}
	}

	// Step 2: Remove shell integration
	if err := removeShellIntegration(opts, result); err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("shell integration removal: %w", err))
	}

	// Step 3: Clean up state and config files (if not keeping)
	if !opts.KeepConfig {
		if err := removeConfigAndState(result); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("config/state removal: %w", err))
		}
	}

	// Step 4: Remove binary (if not keeping)
	if !opts.KeepBinary && opts.BinaryPath != "" {
		if err := removeBinary(opts.BinaryPath, result); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("binary removal: %w", err))
		}
	}

	return result, nil
}

// stopAndRemoveDaemon stops the running daemon and removes launchd configuration
func stopAndRemoveDaemon(result *UninstallResult) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.kubectx-timeout.plist")

	// Check if plist exists
	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		// No daemon installed, nothing to do
		return nil
	}

	// Try to unload the daemon
	// #nosec G204 -- plistPath is constructed from user home dir and known filename, not user input
	cmd := exec.Command("launchctl", "unload", plistPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If unload fails, try to stop it by label
		// #nosec G204 -- label is hardcoded, not user input
		stopCmd := exec.Command("launchctl", "stop", "com.kubectx-timeout")
		// #nosec G104 -- Intentionally ignoring error, daemon might not be running
		stopCmd.Run() // Ignore error, daemon might not be running

		// Continue even if unload failed - the plist might not be loaded
		result.DaemonStopped = false
	} else {
		result.DaemonStopped = true
	}

	// Log output for debugging
	if len(output) > 0 && strings.TrimSpace(string(output)) != "" {
		// Silently ignore "Could not find specified service" errors
		if !strings.Contains(string(output), "Could not find specified service") {
			result.Errors = append(result.Errors, fmt.Errorf("launchctl output: %s", output))
		}
	}

	// Remove the plist file
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist: %w", err)
	}

	result.LaunchdRemoved = true
	return nil
}

// removeShellIntegration removes the kubectl wrapper from shell profiles
func removeShellIntegration(opts UninstallOptions, result *UninstallResult) error {
	var shellsToProcess []string

	if opts.AllShells {
		// Process all supported shells
		shellsToProcess = []string{ShellBash, ShellZsh, ShellFish}
	} else if opts.TargetShell != "" {
		// Process specific shell
		shellsToProcess = []string{opts.TargetShell}
	} else {
		// Auto-detect current shell
		detected, err := DetectShell()
		if err != nil {
			// If detection fails, try all shells
			shellsToProcess = []string{ShellBash, ShellZsh, ShellFish}
		} else {
			shellsToProcess = []string{detected}
		}
	}

	for _, shell := range shellsToProcess {
		profilePath, err := GetShellProfilePath(shell)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to get %s profile path: %w", shell, err))
			continue
		}

		// Check if profile exists
		if _, err := os.Stat(profilePath); os.IsNotExist(err) {
			// Profile doesn't exist, skip
			continue
		}

		// Check if integration is installed
		installed, err := IsIntegrationInstalled(profilePath)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to check %s integration: %w", shell, err))
			continue
		}

		if !installed {
			// Not installed, skip
			continue
		}

		// Uninstall the integration
		if err := UninstallIntegration(profilePath); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to uninstall %s integration: %w", shell, err))
			continue
		}

		result.ShellsProcessed = append(result.ShellsProcessed, shell)
		backupPath := profilePath + ".kubectx-timeout.backup"
		result.BackupsCreated = append(result.BackupsCreated, backupPath)
	}

	return nil
}

// removeConfigAndState removes configuration and state directories
func removeConfigAndState(result *UninstallResult) error {
	configDir := GetConfigDir()
	stateDir := GetStateDir()

	// Remove config directory
	if _, err := os.Stat(configDir); err == nil {
		if err := os.RemoveAll(configDir); err != nil {
			return fmt.Errorf("failed to remove config directory: %w", err)
		}
		result.ConfigRemoved = true
	}

	// Remove state directory
	if _, err := os.Stat(stateDir); err == nil {
		if err := os.RemoveAll(stateDir); err != nil {
			return fmt.Errorf("failed to remove state directory: %w", err)
		}
		result.StateRemoved = true
	}

	return nil
}

// removeBinary removes the kubectx-timeout binary
func removeBinary(binaryPath string, result *UninstallResult) error {
	// Verify the path looks like a kubectx-timeout binary
	if !strings.Contains(binaryPath, "kubectx-timeout") {
		return fmt.Errorf("refusing to remove binary that doesn't appear to be kubectx-timeout: %s", binaryPath)
	}

	// Check if binary exists
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		// Binary doesn't exist, nothing to do
		return nil
	}

	// Remove the binary
	if err := os.Remove(binaryPath); err != nil {
		return fmt.Errorf("failed to remove binary: %w", err)
	}

	result.BinaryRemoved = true
	return nil
}

// GetLaunchdPlistPath returns the path to the launchd plist file
func GetLaunchdPlistPath() (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("launchd is only available on macOS")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(home, "Library", "LaunchAgents", "com.kubectx-timeout.plist"), nil
}

// CheckDaemonStatus checks if the daemon is currently running
func CheckDaemonStatus() (bool, error) {
	if runtime.GOOS != "darwin" {
		// On non-macOS systems, check if process is running
		// #nosec G204 -- command is hardcoded, not user input
		cmd := exec.Command("pgrep", "-f", "kubectx-timeout daemon")
		err := cmd.Run()
		return err == nil, nil
	}

	// On macOS, check launchd status
	// #nosec G204 -- command and label are hardcoded, not user input
	cmd := exec.Command("launchctl", "list", "com.kubectx-timeout")
	err := cmd.Run()
	if err != nil {
		// If launchctl returns error, daemon is not running
		return false, nil
	}

	return true, nil
}

// GetInstalledShells returns a list of shells that have the integration installed
func GetInstalledShells() ([]string, error) {
	var installed []string
	shells := []string{ShellBash, ShellZsh, ShellFish}

	for _, shell := range shells {
		profilePath, err := GetShellProfilePath(shell)
		if err != nil {
			continue
		}

		// Check if profile exists
		if _, err := os.Stat(profilePath); os.IsNotExist(err) {
			continue
		}

		// Check if integration is installed
		isInstalled, err := IsIntegrationInstalled(profilePath)
		if err != nil {
			continue
		}

		if isInstalled {
			installed = append(installed, shell)
		}
	}

	return installed, nil
}

// FormatUninstallResult returns a formatted string describing the uninstallation result
func FormatUninstallResult(result *UninstallResult) string {
	var sb strings.Builder

	sb.WriteString("\nUninstallation Summary:\n")
	sb.WriteString(strings.Repeat("=", 60) + "\n")

	// Daemon
	if result.LaunchdRemoved {
		if result.DaemonStopped {
			sb.WriteString("✓ Daemon stopped and removed\n")
		} else {
			sb.WriteString("✓ Daemon configuration removed (daemon was not running)\n")
		}
	}

	// Shell integration
	if len(result.ShellsProcessed) > 0 {
		sb.WriteString(fmt.Sprintf("✓ Shell integration removed from: %s\n",
			strings.Join(result.ShellsProcessed, ", ")))
	}

	// Config and state
	if result.ConfigRemoved {
		sb.WriteString("✓ Configuration files removed\n")
	}
	if result.StateRemoved {
		sb.WriteString("✓ State files removed\n")
	}

	// Binary
	if result.BinaryRemoved {
		sb.WriteString("✓ Binary removed\n")
	}

	// Backups
	if len(result.BackupsCreated) > 0 {
		sb.WriteString("\nBackups created:\n")
		for _, backup := range result.BackupsCreated {
			sb.WriteString(fmt.Sprintf("  - %s\n", backup))
		}
	}

	// Errors
	if len(result.Errors) > 0 {
		sb.WriteString("\nWarnings/Errors:\n")
		for _, err := range result.Errors {
			// Skip "Could not find specified service" errors
			if !strings.Contains(err.Error(), "Could not find specified service") {
				sb.WriteString(fmt.Sprintf("  ⚠ %v\n", err))
			}
		}
	}

	sb.WriteString(strings.Repeat("=", 60) + "\n")

	return sb.String()
}
