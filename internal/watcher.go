package internal

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// KubeconfigWatcher monitors ~/.kube/config for changes
type KubeconfigWatcher struct {
	kubeconfigPath string
	stateManager   *StateManager
	logger         *log.Logger
	ctx            context.Context
}

// NewKubeconfigWatcher creates a new kubeconfig watcher
func NewKubeconfigWatcher(stateManager *StateManager, logger *log.Logger, ctx context.Context) (*KubeconfigWatcher, error) {
	// Get kubeconfig path - check KUBECONFIG env var first, then default to ~/.kube/config
	kubeconfigPath := os.Getenv("KUBECONFIG")
	if kubeconfigPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfigPath = filepath.Join(home, ".kube", "config")
	}

	return &KubeconfigWatcher{
		kubeconfigPath: kubeconfigPath,
		stateManager:   stateManager,
		logger:         logger,
		ctx:            ctx,
	}, nil
}

// Watch starts monitoring the kubeconfig file for changes
// This runs in a separate goroutine and uses fswatch on macOS (FSEvents API)
// If fswatch is not available, it degrades gracefully and logs a warning
func (w *KubeconfigWatcher) Watch() {
	// Check if fswatch is available
	if !w.isFswatchAvailable() {
		w.logger.Println("fswatch not found - kubeconfig file monitoring disabled")
		w.logger.Println("Install fswatch for automatic context switch detection: brew install fswatch")
		return
	}

	// Check if kubeconfig file exists
	if _, err := os.Stat(w.kubeconfigPath); os.IsNotExist(err) {
		w.logger.Printf("Kubeconfig file not found at %s - file monitoring disabled", w.kubeconfigPath)
		return
	}

	w.logger.Printf("Starting kubeconfig file monitoring at %s", w.kubeconfigPath)

	// Start fswatch process
	if err := w.watchWithFswatch(); err != nil {
		w.logger.Printf("fswatch monitoring stopped: %v", err)
	}
}

// isFswatchAvailable checks if fswatch is installed and available
func (w *KubeconfigWatcher) isFswatchAvailable() bool {
	// Only use fswatch on macOS where FSEvents API is available
	if runtime.GOOS != "darwin" {
		return false
	}

	// Check if fswatch binary exists
	_, err := exec.LookPath("fswatch")
	return err == nil
}

// watchWithFswatch uses fswatch to monitor the kubeconfig file
func (w *KubeconfigWatcher) watchWithFswatch() error {
	// Use fswatch with FSEvents API on macOS
	// -0: Use NUL character as separator (more reliable for paths with spaces)
	// -1: Exit after first event set (we restart the loop to handle context cancellation)
	// --event Created,Updated,Renamed: Only watch for relevant events
	// -r: Recursive (needed even for single file to detect changes reliably)
	// -l 0.5: Latency of 0.5 seconds (debounce rapid changes)

	for {
		select {
		case <-w.ctx.Done():
			w.logger.Println("Kubeconfig file monitoring stopped (context canceled)")
			return nil
		default:
			// Continue monitoring
		}

		// Start fswatch process
		cmd := exec.CommandContext(w.ctx, "fswatch",
			"-0",              // NUL separator
			"-1",              // Exit after first set of events
			"--event=Created", // Watch for file creation
			"--event=Updated", // Watch for file updates
			"--event=Renamed", // Watch for file renames
			"-l", "0.5",       // 0.5 second latency (debounce)
			w.kubeconfigPath,
		)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		if err := cmd.Start(); err != nil {
			return fmt.Errorf("failed to start fswatch: %w", err)
		}

		// Read events from fswatch
		scanner := bufio.NewScanner(stdout)
		scanner.Split(scanNullTerminated)

		for scanner.Scan() {
			// File was modified, check for context change
			if err := w.handleConfigChange(); err != nil {
				w.logger.Printf("Error handling config change: %v", err)
			}
		}

		if err := scanner.Err(); err != nil {
			w.logger.Printf("Error reading fswatch output: %v", err)
		}

		// Wait for process to exit
		if err := cmd.Wait(); err != nil {
			// Check if context was canceled
			if w.ctx.Err() != nil {
				return nil
			}
			// Otherwise log error and retry
			w.logger.Printf("fswatch process exited with error: %v, retrying in 5 seconds...", err)
			time.Sleep(5 * time.Second)
		}
	}
}

// handleConfigChange is called when the kubeconfig file changes
// It checks if the context actually changed and records activity if so
func (w *KubeconfigWatcher) handleConfigChange() error {
	// Get current context
	currentContext, err := GetCurrentContext()
	if err != nil {
		// If we can't get current context, skip this change
		// This can happen during transient states when the file is being written
		return nil
	}

	// Get last recorded context
	_, lastContext, err := w.stateManager.GetLastActivity()
	if err != nil {
		// If we can't get last activity, record fresh activity
		w.logger.Printf("Detected context switch to '%s' (no previous state)", currentContext)
		return w.stateManager.RecordActivity(currentContext)
	}

	// Check if context actually changed
	if lastContext != currentContext {
		w.logger.Printf("Detected context switch from '%s' to '%s' via file monitoring", lastContext, currentContext)
		return w.stateManager.RecordActivity(currentContext)
	}

	// Context didn't change, but file was modified (might be other kubeconfig changes)
	// Still record activity to extend timeout
	w.logger.Printf("Detected kubeconfig modification while in context '%s' (extending timeout)", currentContext)
	return w.stateManager.RecordActivity(currentContext)
}

// scanNullTerminated is a split function for bufio.Scanner that splits on NUL bytes
func scanNullTerminated(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Find NUL terminator
	for i := 0; i < len(data); i++ {
		if data[i] == 0 {
			return i + 1, data[0:i], nil
		}
	}

	// If we're at EOF, return what we have
	if atEOF {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}
