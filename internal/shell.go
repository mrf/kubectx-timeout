package internal

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Shell types
const (
	ShellBash = "bash"
	ShellZsh  = "zsh"
	ShellFish = "fish"
)

// ShellIntegration markers for identification
const (
	IntegrationStartMarker = "# >>> kubectx-timeout shell integration >>>"
	IntegrationEndMarker   = "# <<< kubectx-timeout shell integration <<<"
)

// ShellProfile represents a shell configuration
type ShellProfile struct {
	Shell       string
	ProfilePath string
	BinaryPath  string
}

// DetectShell detects the user's current shell
func DetectShell() (string, error) {
	// Try $SHELL environment variable first
	shellEnv := os.Getenv("SHELL")
	if shellEnv != "" {
		base := filepath.Base(shellEnv)
		if isValidShell(base) {
			return base, nil
		}
	}

	// Try to detect from parent process
	ppid := os.Getppid()
	if ppid > 0 {
		// #nosec G204 -- ppid is from os.Getppid() system call, formatted as %d, not user input
		cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", ppid), "-o", "comm=")
		output, err := cmd.Output()
		if err == nil {
			shell := strings.TrimSpace(string(output))
			shell = filepath.Base(shell)
			// Remove leading dash if present (login shells)
			shell = strings.TrimPrefix(shell, "-")
			if isValidShell(shell) {
				return shell, nil
			}
		}
	}

	return "", fmt.Errorf("unable to detect shell")
}

// isValidShell checks if the shell is supported
func isValidShell(shell string) bool {
	switch shell {
	case ShellBash, ShellZsh, ShellFish:
		return true
	default:
		return false
	}
}

// GetShellProfilePath returns the profile path for the given shell
func GetShellProfilePath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	var profile string
	switch shell {
	case ShellBash:
		// Check for .bash_profile first, then .bashrc
		bashProfile := filepath.Join(home, ".bash_profile")
		bashrc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(bashProfile); err == nil {
			profile = bashProfile
		} else {
			profile = bashrc
		}
	case ShellZsh:
		profile = filepath.Join(home, ".zshrc")
	case ShellFish:
		profile = filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}

	return profile, nil
}

// GetShellIntegrationCode returns the shell integration code for the given shell
func GetShellIntegrationCode(shell string, binaryPath string) (string, error) {
	switch shell {
	case ShellBash:
		return fmt.Sprintf(`%s
# Function-based kubectl wrapper
# This is lighter weight than aliasing to a script
_kubectx_timeout_kubectl() {
    local kubectx_timeout_bin="${KUBECTX_TIMEOUT_BIN:-%s}"

    # Record activity in background (non-blocking)
    if [ -x "$kubectx_timeout_bin" ]; then
        "$kubectx_timeout_bin" record-activity >/dev/null 2>&1 &
    fi

    # Execute kubectl with all arguments
    command kubectl "$@"
}

# Create kubectl alias/function
# Use a function instead of alias for better compatibility
kubectl() {
    _kubectx_timeout_kubectl "$@"
}

# Export for use in subshells
export -f _kubectx_timeout_kubectl 2>/dev/null || true
%s
`, IntegrationStartMarker, binaryPath, IntegrationEndMarker), nil

	case ShellZsh:
		return fmt.Sprintf(`%s
# Function-based kubectl wrapper
# This is lighter weight than aliasing to a script
_kubectx_timeout_kubectl() {
    local kubectx_timeout_bin="${KUBECTX_TIMEOUT_BIN:-%s}"

    # Record activity in background (non-blocking)
    if [ -x "$kubectx_timeout_bin" ]; then
        "$kubectx_timeout_bin" record-activity >/dev/null 2>&1 &
    fi

    # Execute kubectl with all arguments
    command kubectl "$@"
}

# Create kubectl alias/function
# Use a function instead of alias for better compatibility
kubectl() {
    _kubectx_timeout_kubectl "$@"
}
%s
`, IntegrationStartMarker, binaryPath, IntegrationEndMarker), nil

	case ShellFish:
		return fmt.Sprintf(`%s
# Fish shell kubectl wrapper
function kubectl
    set kubectx_timeout_bin %s

    # Record activity in background (non-blocking)
    if test -x "$kubectx_timeout_bin"
        $kubectx_timeout_bin record-activity >/dev/null 2>&1 &
    end

    # Execute kubectl with all arguments
    command kubectl $argv
end
%s
`, IntegrationStartMarker, binaryPath, IntegrationEndMarker), nil

	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

// IsIntegrationInstalled checks if the integration is already installed
func IsIntegrationInstalled(profilePath string) (bool, error) {
	// #nosec G304 -- profilePath is constructed from user home dir and known profile names, not user input
	file, err := os.Open(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to open profile: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, IntegrationStartMarker) {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("failed to read profile: %w", err)
	}

	return false, nil
}

// InstallIntegration installs the shell integration to the profile file
func InstallIntegration(profilePath string, integrationCode string) error {
	// Check if already installed
	installed, err := IsIntegrationInstalled(profilePath)
	if err != nil {
		return fmt.Errorf("failed to check installation status: %w", err)
	}
	if installed {
		return fmt.Errorf("integration already installed in %s", profilePath)
	}

	// Ensure profile directory exists
	profileDir := filepath.Dir(profilePath)
	if err := os.MkdirAll(profileDir, 0750); err != nil {
		return fmt.Errorf("failed to create profile directory: %w", err)
	}

	// Create backup
	backupPath := profilePath + ".kubectx-timeout.backup"
	if _, err := os.Stat(profilePath); err == nil {
		// #nosec G304 -- profilePath is constructed from user home dir and known profile names, not user input
		content, err := os.ReadFile(profilePath)
		if err != nil {
			return fmt.Errorf("failed to read profile for backup: %w", err)
		}
		if err := os.WriteFile(backupPath, content, 0600); err != nil {
			return fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Append integration code
	// #nosec G304 -- profilePath is constructed from user home dir and known profile names, not user input
	file, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open profile: %w", err)
	}
	defer file.Close()

	// Add newlines before and after for readability
	content := fmt.Sprintf("\n%s\n", integrationCode)
	if _, err := file.WriteString(content); err != nil {
		return fmt.Errorf("failed to write integration code: %w", err)
	}

	return nil
}

// UninstallIntegration removes the shell integration from the profile file
func UninstallIntegration(profilePath string) error {
	// #nosec G304 -- profilePath is constructed from user home dir and known profile names, not user input
	file, err := os.Open(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to uninstall
		}
		return fmt.Errorf("failed to open profile: %w", err)
	}
	defer file.Close()

	var newContent strings.Builder
	scanner := bufio.NewScanner(file)
	inIntegration := false
	found := false

	for scanner.Scan() {
		line := scanner.Text()

		if strings.Contains(line, IntegrationStartMarker) {
			inIntegration = true
			found = true
			continue
		}

		if strings.Contains(line, IntegrationEndMarker) {
			inIntegration = false
			continue
		}

		if !inIntegration {
			newContent.WriteString(line)
			newContent.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read profile: %w", err)
	}

	if !found {
		return nil // Integration not found, nothing to remove
	}

	// Create backup
	backupPath := profilePath + ".kubectx-timeout.backup"
	// #nosec G304 -- profilePath is constructed from user home dir and known profile names, not user input
	content, err := os.ReadFile(profilePath)
	if err != nil {
		return fmt.Errorf("failed to read profile for backup: %w", err)
	}
	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Write new content
	if err := os.WriteFile(profilePath, []byte(newContent.String()), 0600); err != nil {
		return fmt.Errorf("failed to write profile: %w", err)
	}

	return nil
}

// VerifyInstallation checks if the installation was successful
func VerifyInstallation(profilePath string, binaryPath string) []string {
	var issues []string

	// Check if integration is in profile
	installed, err := IsIntegrationInstalled(profilePath)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Failed to check profile: %v", err))
		return issues
	}
	if !installed {
		issues = append(issues, "Integration code not found in shell profile")
		return issues
	}

	// Check if binary exists and is executable
	if _, err := os.Stat(binaryPath); err != nil {
		issues = append(issues, fmt.Sprintf("Binary not found at %s", binaryPath))
	} else if info, err := os.Stat(binaryPath); err == nil {
		if info.Mode().Perm()&0111 == 0 {
			issues = append(issues, fmt.Sprintf("Binary at %s is not executable", binaryPath))
		}
	}

	// Check if kubectl is available
	if _, err := exec.LookPath("kubectl"); err != nil {
		issues = append(issues, "kubectl not found in PATH")
	}

	return issues
}
