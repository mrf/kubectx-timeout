package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// State represents the current state of kubectl activity
type State struct {
	// LastActivity is the timestamp of the last kubectl command execution
	LastActivity time.Time `json:"last_activity"`
	
	// CurrentContext is the current kubectl context at time of last activity
	CurrentContext string `json:"current_context"`
	
	// Version is the state file format version for future compatibility
	Version int `json:"version"`
	
	mu sync.RWMutex
}

const stateVersion = 1

// StateManager handles reading and writing state to disk
type StateManager struct {
	path string
	mu   sync.Mutex
}

// NewStateManager creates a new state manager
func NewStateManager(path string) (*StateManager, error) {
	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}
	
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}
	
	return &StateManager{path: path}, nil
}

// Load reads the current state from disk
// If the file doesn't exist, returns a new empty state
func (sm *StateManager) Load() (*State, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	// Check if file exists
	if _, err := os.Stat(sm.path); os.IsNotExist(err) {
		// Return empty state
		return &State{
			Version: stateVersion,
		}, nil
	}
	
	// Read file
	data, err := os.ReadFile(sm.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}
	
	// Parse JSON
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}
	
	// Validate version
	if state.Version > stateVersion {
		return nil, fmt.Errorf("state file version %d is newer than supported version %d", state.Version, stateVersion)
	}
	
	return &state, nil
}

// Save writes the state to disk
func (sm *StateManager) Save(state *State) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	
	state.mu.Lock()
	defer state.mu.Unlock()
	
	// Ensure version is set
	state.Version = stateVersion
	
	// Marshal to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	
	// Write to temporary file first, then rename for atomic operation
	tmpPath := sm.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	
	// Atomic rename
	if err := os.Rename(tmpPath, sm.path); err != nil {
		return fmt.Errorf("failed to rename state file: %w", err)
	}
	
	return nil
}

// RecordActivity updates the state with current activity
func (sm *StateManager) RecordActivity(context string) error {
	// Load current state
	state, err := sm.Load()
	if err != nil {
		return fmt.Errorf("failed to load state: %w", err)
	}
	
	// Update state
	state.mu.Lock()
	state.LastActivity = time.Now()
	state.CurrentContext = context
	state.mu.Unlock()
	
	// Save state
	if err := sm.Save(state); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}
	
	return nil
}

// GetLastActivity returns the timestamp of the last kubectl activity
func (sm *StateManager) GetLastActivity() (time.Time, string, error) {
	state, err := sm.Load()
	if err != nil {
		return time.Time{}, "", err
	}
	
	state.mu.RLock()
	defer state.mu.RUnlock()
	
	return state.LastActivity, state.CurrentContext, nil
}

// TimeSinceLastActivity returns the duration since last activity
func (sm *StateManager) TimeSinceLastActivity() (time.Duration, error) {
	lastActivity, _, err := sm.GetLastActivity()
	if err != nil {
		return 0, err
	}
	
	// If no activity recorded yet, return a large duration
	if lastActivity.IsZero() {
		return 24 * time.Hour, nil
	}
	
	return time.Since(lastActivity), nil
}
