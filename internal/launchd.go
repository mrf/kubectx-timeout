package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	// LaunchdLabel is the service label for the daemon
	LaunchdLabel = "com.kubectx-timeout"

	// LaunchdPlistTemplate is the template for the launchd plist file
	LaunchdPlistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <!-- Service label (must match filename) -->
    <key>Label</key>
    <string>{{.Label}}</string>

    <!-- Program to run -->
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>daemon</string>
    </array>

    <!-- Run automatically on login -->
    <key>RunAtLoad</key>
    <true/>

    <!-- Keep alive - restart if it crashes -->
    <key>KeepAlive</key>
    <true/>

    <!-- Standard output path (XDG Base Directory compliant) -->
    <key>StandardOutPath</key>
    <string>{{.StdoutPath}}</string>

    <!-- Standard error path (XDG Base Directory compliant) -->
    <key>StandardErrorPath</key>
    <string>{{.StderrPath}}</string>

    <!-- Working directory -->
    <key>WorkingDirectory</key>
    <string>{{.HomeDir}}</string>

    <!-- Environment variables -->
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>{{.Path}}</string>
        <key>HOME</key>
        <string>{{.HomeDir}}</string>
    </dict>

    <!-- Throttle interval to prevent rapid restarts (10 seconds) -->
    <key>ThrottleInterval</key>
    <integer>10</integer>

    <!-- Process type -->
    <key>ProcessType</key>
    <string>Background</string>

    <!-- Nice value (lower priority) -->
    <key>Nice</key>
    <integer>1</integer>
</dict>
</plist>
`
)

// LaunchdManager handles launchd operations for macOS
type LaunchdManager struct {
	label      string
	plistPath  string
	binaryPath string
}

// NewLaunchdManager creates a new launchd manager instance
func NewLaunchdManager(binaryPath string) (*LaunchdManager, error) {
	// Verify we're on macOS
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("launchd is only available on macOS")
	}

	// Get user home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Construct plist path in ~/Library/LaunchAgents/
	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", LaunchdLabel+".plist")

	// If no binary path specified, try to find the current executable
	if binaryPath == "" {
		execPath, err := os.Executable()
		if err != nil {
			return nil, fmt.Errorf("failed to determine executable path: %w", err)
		}
		// Resolve symlinks
		binaryPath, err = filepath.EvalSymlinks(execPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve executable path: %w", err)
		}
	}

	return &LaunchdManager{
		label:      LaunchdLabel,
		plistPath:  plistPath,
		binaryPath: binaryPath,
	}, nil
}

// Install installs the launchd plist and loads the daemon
func (lm *LaunchdManager) Install() error {
	// Check if already installed
	if lm.IsInstalled() {
		return fmt.Errorf("daemon is already installed at %s", lm.plistPath)
	}

	// Ensure LaunchAgents directory exists
	launchAgentsDir := filepath.Dir(lm.plistPath)
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Ensure state directory exists
	stateDir := GetStateDir()
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Generate plist content
	plistContent, err := lm.generatePlist()
	if err != nil {
		return fmt.Errorf("failed to generate plist: %w", err)
	}

	// Write plist file
	if err := os.WriteFile(lm.plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write plist file: %w", err)
	}

	// Load the daemon
	if err := lm.Load(); err != nil {
		// If load fails, clean up the plist file
		os.Remove(lm.plistPath)
		return fmt.Errorf("failed to load daemon: %w", err)
	}

	return nil
}

// Uninstall unloads the daemon and removes the plist file
func (lm *LaunchdManager) Uninstall() error {
	// Check if installed
	if !lm.IsInstalled() {
		return fmt.Errorf("daemon is not installed")
	}

	// Unload if running
	if lm.IsRunning() {
		if err := lm.Unload(); err != nil {
			return fmt.Errorf("failed to unload daemon: %w", err)
		}
	}

	// Remove plist file
	if err := os.Remove(lm.plistPath); err != nil {
		return fmt.Errorf("failed to remove plist file: %w", err)
	}

	return nil
}

// Start starts the daemon (alias for Load)
func (lm *LaunchdManager) Start() error {
	if !lm.IsInstalled() {
		return fmt.Errorf("daemon is not installed. Run 'kubectx-timeout daemon-install' first")
	}

	if lm.IsRunning() {
		return fmt.Errorf("daemon is already running")
	}

	return lm.Load()
}

// Stop stops the daemon (alias for Unload)
func (lm *LaunchdManager) Stop() error {
	if !lm.IsInstalled() {
		return fmt.Errorf("daemon is not installed")
	}

	if !lm.IsRunning() {
		return fmt.Errorf("daemon is not running")
	}

	return lm.Unload()
}

// Restart restarts the daemon
func (lm *LaunchdManager) Restart() error {
	if !lm.IsInstalled() {
		return fmt.Errorf("daemon is not installed. Run 'kubectx-timeout daemon-install' first")
	}

	// Stop if running (ignore error if not running)
	if lm.IsRunning() {
		if err := lm.Unload(); err != nil {
			return fmt.Errorf("failed to stop daemon: %w", err)
		}
	}

	// Start
	if err := lm.Load(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	return nil
}

// Load loads the daemon using launchctl
func (lm *LaunchdManager) Load() error {
	cmd := exec.Command("launchctl", "load", lm.plistPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// Unload unloads the daemon using launchctl
func (lm *LaunchdManager) Unload() error {
	cmd := exec.Command("launchctl", "unload", lm.plistPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl unload failed: %w\nOutput: %s", err, string(output))
	}
	return nil
}

// IsInstalled checks if the plist file exists
func (lm *LaunchdManager) IsInstalled() bool {
	_, err := os.Stat(lm.plistPath)
	return err == nil
}

// IsRunning checks if the daemon is currently running
func (lm *LaunchdManager) IsRunning() bool {
	cmd := exec.Command("launchctl", "list", lm.label)
	err := cmd.Run()
	return err == nil
}

// GetStatus returns the daemon status information
func (lm *LaunchdManager) GetStatus() (string, error) {
	installed := lm.IsInstalled()
	running := lm.IsRunning()

	var status strings.Builder
	status.WriteString("Daemon Status:\n")
	status.WriteString(fmt.Sprintf("  Installed: %v\n", installed))
	status.WriteString(fmt.Sprintf("  Running: %v\n", running))
	status.WriteString(fmt.Sprintf("  Plist Path: %s\n", lm.plistPath))
	status.WriteString(fmt.Sprintf("  Binary Path: %s\n", lm.binaryPath))

	if installed && running {
		// Get detailed status from launchctl
		cmd := exec.Command("launchctl", "list", lm.label)
		output, err := cmd.CombinedOutput()
		if err == nil {
			status.WriteString(fmt.Sprintf("\nLaunchctl Info:\n%s", string(output)))
		}
	}

	return status.String(), nil
}

// GetPID returns the process ID of the running daemon, or 0 if not running
func (lm *LaunchdManager) GetPID() (int, error) {
	if !lm.IsRunning() {
		return 0, nil
	}

	cmd := exec.Command("launchctl", "list", lm.label)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to get daemon PID: %w", err)
	}

	// Parse output to get PID
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "PID") {
			fields := strings.Fields(line)
			if len(fields) >= 3 {
				var pid int
				fmt.Sscanf(fields[2], "%d", &pid)
				return pid, nil
			}
		}
	}

	return 0, nil
}

// generatePlist generates the plist file content
func (lm *LaunchdManager) generatePlist() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	stateDir := GetStateDir()
	stdoutPath := filepath.Join(stateDir, "daemon.stdout.log")
	stderrPath := filepath.Join(stateDir, "daemon.stderr.log")

	// Get PATH from environment, or use a sensible default
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		pathEnv = "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin"
	}

	// Simple template replacement (not using text/template to avoid dependencies)
	plist := LaunchdPlistTemplate
	plist = strings.ReplaceAll(plist, "{{.Label}}", lm.label)
	plist = strings.ReplaceAll(plist, "{{.BinaryPath}}", lm.binaryPath)
	plist = strings.ReplaceAll(plist, "{{.StdoutPath}}", stdoutPath)
	plist = strings.ReplaceAll(plist, "{{.StderrPath}}", stderrPath)
	plist = strings.ReplaceAll(plist, "{{.HomeDir}}", homeDir)
	plist = strings.ReplaceAll(plist, "{{.Path}}", pathEnv)

	return plist, nil
}

// GetPlistPath returns the path to the plist file
func (lm *LaunchdManager) GetPlistPath() string {
	return lm.plistPath
}
