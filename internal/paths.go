package internal

import (
	"os"
	"path/filepath"
)

// GetConfigDir returns the configuration directory following XDG Base Directory spec.
// Returns $XDG_CONFIG_HOME/kubectx-timeout if set, otherwise ~/.config/kubectx-timeout
func GetConfigDir() string {
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "kubectx-timeout")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to a reasonable default
		return filepath.Join("/tmp", "kubectx-timeout")
	}

	return filepath.Join(home, ".config", "kubectx-timeout")
}

// GetStateDir returns the state directory following XDG Base Directory spec.
// Returns $XDG_STATE_HOME/kubectx-timeout if set, otherwise ~/.local/state/kubectx-timeout
func GetStateDir() string {
	if xdgState := os.Getenv("XDG_STATE_HOME"); xdgState != "" {
		return filepath.Join(xdgState, "kubectx-timeout")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to a reasonable default
		return filepath.Join("/tmp", "kubectx-timeout")
	}

	return filepath.Join(home, ".local", "state", "kubectx-timeout")
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() string {
	return filepath.Join(GetConfigDir(), "config.yaml")
}

// GetStatePath returns the full path to the state file
func GetStatePath() string {
	return filepath.Join(GetStateDir(), "state.json")
}

// GetLogPath returns the full path to the log file
func GetLogPath() string {
	return filepath.Join(GetStateDir(), "daemon.log")
}
