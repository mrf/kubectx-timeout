package internal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ActivityTracker tracks kubectl command activity
type ActivityTracker struct {
	stateManager *StateManager
	configPath   string
}

// NewActivityTracker creates a new activity tracker
func NewActivityTracker(statePath string, configPath string) (*ActivityTracker, error) {
	sm, err := NewStateManager(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	return &ActivityTracker{
		stateManager: sm,
		configPath:   configPath,
	}, nil
}

// GetCurrentContext returns the current kubectl context
func GetCurrentContext() (string, error) {
	cmd := exec.Command("kubectl", "config", "current-context")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current context: %w", err)
	}

	context := strings.TrimSpace(string(output))
	if context == "" {
		return "", fmt.Errorf("no current context set")
	}

	return context, nil
}

// RecordActivity records kubectl activity with the current context
func (at *ActivityTracker) RecordActivity() error {
	// Get current context
	context, err := GetCurrentContext()
	if err != nil {
		// If we can't get the context, still record activity with empty context
		// This ensures we don't break the user's kubectl workflow
		context = "unknown"
	}

	// Record activity
	if err := at.stateManager.RecordActivity(context); err != nil {
		return fmt.Errorf("failed to record activity: %w", err)
	}

	return nil
}

// GetLastActivity returns the last activity timestamp and context
func (at *ActivityTracker) GetLastActivity() (ActivityInfo, error) {
	lastActivity, context, err := at.stateManager.GetLastActivity()
	if err != nil {
		return ActivityInfo{}, fmt.Errorf("failed to get last activity: %w", err)
	}

	return ActivityInfo{
		LastActivity:   lastActivity,
		CurrentContext: context,
	}, nil
}

// ActivityInfo contains information about kubectl activity
type ActivityInfo struct {
	LastActivity   time.Time
	CurrentContext string
}

// GenerateShellIntegration generates shell integration code for the given shell
func GenerateShellIntegration(shell string, binaryPath string) (string, error) {
	if binaryPath == "" {
		binaryPath = "/usr/local/bin/kubectx-timeout"
	}

	switch shell {
	case "bash", "zsh":
		return fmt.Sprintf(`# kubectx-timeout shell integration for %s
# Add this to your ~/.%src

kubectl() {
    # Record activity before executing kubectl
    if [ -x "%s" ]; then
        "%s" record-activity >/dev/null 2>&1 &
    fi
    
    # Execute the real kubectl
    command kubectl "$@"
}
`, shell, shell, binaryPath, binaryPath), nil

	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

// InstallShellIntegration installs shell integration to the user's profile
func InstallShellIntegration(shell string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	var profilePath string
	switch shell {
	case "bash":
		profilePath = home + "/.bashrc"
	case "zsh":
		profilePath = home + "/.zshrc"
	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}

	// Generate integration code
	integration, err := GenerateShellIntegration(shell, "/usr/local/bin/kubectx-timeout")
	if err != nil {
		return err
	}

	// Check if already installed
	content, err := os.ReadFile(profilePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read profile: %w", err)
	}

	if strings.Contains(string(content), "kubectx-timeout shell integration") {
		return fmt.Errorf("shell integration already installed in %s", profilePath)
	}

	// Append to profile
	f, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open profile: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("\n" + integration); err != nil {
		return fmt.Errorf("failed to write to profile: %w", err)
	}

	return nil
}
