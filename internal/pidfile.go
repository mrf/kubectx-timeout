package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// PIDFile manages a PID file to ensure single daemon instance
type PIDFile struct {
	path string
}

// NewPIDFile creates a new PID file manager using the default state directory
func NewPIDFile() *PIDFile {
	stateDir := GetStateDir()
	pidPath := filepath.Join(stateDir, "daemon.pid")
	return &PIDFile{path: pidPath}
}

// NewPIDFileWithPath creates a new PID file manager with a custom path
// Useful for testing to avoid conflicts with the system daemon
func NewPIDFileWithPath(path string) *PIDFile {
	return &PIDFile{path: path}
}

// Acquire creates the PID file and writes the current process ID
// Returns an error if another instance is already running
func (p *PIDFile) Acquire() error {
	// Ensure state directory exists
	stateDir := filepath.Dir(p.path)
	if err := os.MkdirAll(stateDir, 0750); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Check if PID file already exists
	if _, err := os.Stat(p.path); err == nil {
		// PID file exists, check if process is still running
		existingPID, err := p.ReadPID()
		if err == nil && p.isProcessRunning(existingPID) {
			return fmt.Errorf("daemon is already running with PID %d", existingPID)
		}
		// Stale PID file, remove it
		_ = os.Remove(p.path) // Ignore error on cleanup
	}

	// Write current PID to file
	pid := os.Getpid()
	pidStr := strconv.Itoa(pid)
	if err := os.WriteFile(p.path, []byte(pidStr+"\n"), 0600); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	return nil
}

// Release removes the PID file
func (p *PIDFile) Release() error {
	if err := os.Remove(p.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove PID file: %w", err)
	}
	return nil
}

// ReadPID reads the PID from the PID file
func (p *PIDFile) ReadPID() (int, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return 0, fmt.Errorf("failed to read PID file: %w", err)
	}

	pidStr := strings.TrimSpace(string(data))
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

// isProcessRunning checks if a process with the given PID is running
func (p *PIDFile) isProcessRunning(pid int) bool {
	// Send signal 0 to check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds, so we need to send a signal to check
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// GetPath returns the path to the PID file
func (p *PIDFile) GetPath() string {
	return p.path
}
