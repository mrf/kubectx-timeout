package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Daemon represents the timeout monitoring daemon
type Daemon struct {
	config       *Config
	stateManager *StateManager
	switcher     *ContextSwitcher
	ctx          context.Context
	cancel       context.CancelFunc
	logger       *log.Logger
	pidFile      *PIDFile
}

// NewDaemon creates a new daemon instance
func NewDaemon(configPath string, statePath string) (*Daemon, error) {
	return NewDaemonWithPIDFile(configPath, statePath, nil)
}

// NewDaemonWithPIDFile creates a new daemon instance with a custom PID file
// If pidFile is nil, uses the default PID file location
func NewDaemonWithPIDFile(configPath string, statePath string, pidFile *PIDFile) (*Daemon, error) {
	// Load configuration
	config, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create state manager
	sm, err := NewStateManager(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Create logger
	logger := log.New(os.Stdout, "[kubectx-timeout] ", log.LstdFlags)

	// Create context switcher
	switcher := NewContextSwitcher(logger)

	// Create PID file manager if not provided
	if pidFile == nil {
		pidFile = NewPIDFile()
	}

	daemon := &Daemon{
		config:       config,
		stateManager: sm,
		switcher:     switcher,
		ctx:          ctx,
		cancel:       cancel,
		logger:       logger,
		pidFile:      pidFile,
	}

	// Check if context changed while daemon was down
	// If so, record fresh activity to prevent immediate timeout
	if err := daemon.checkContextChangeOnStartup(); err != nil {
		logger.Printf("Warning: failed to check context change on startup: %v", err)
		// Don't fail daemon creation, just log warning
	}

	return daemon, nil
}

// checkContextChangeOnStartup resets the activity timer on daemon startup to prevent
// immediate timeout due to stale timestamps while the daemon was not running
func (d *Daemon) checkContextChangeOnStartup() error {
	// Get current context
	currentContext, err := GetCurrentContext()
	if err != nil {
		// If we can't get current context, skip this check
		return nil
	}

	// Get last recorded context and timestamp from state
	lastActivity, lastContext, err := d.stateManager.GetLastActivity()
	if err != nil {
		// If we can't load state, record fresh activity
		d.logger.Printf("No previous state found, recording initial activity for context '%s'", currentContext)
		if err := d.stateManager.RecordActivity(currentContext); err != nil {
			return fmt.Errorf("failed to record activity: %w", err)
		}
		return nil
	}

	// Check for zero/uninitialized timestamp (first run or corrupted state)
	if lastActivity.IsZero() {
		d.logger.Printf("No previous activity timestamp found, recording initial activity for context '%s'", currentContext)
		if err := d.stateManager.RecordActivity(currentContext); err != nil {
			return fmt.Errorf("failed to record activity: %w", err)
		}
		return nil
	}

	// Check if context changed while daemon was down
	if lastContext != "" && lastContext != currentContext {
		d.logger.Printf("Context changed from '%s' to '%s' while daemon was down, resetting activity timer",
			lastContext, currentContext)
		if err := d.stateManager.RecordActivity(currentContext); err != nil {
			return fmt.Errorf("failed to record activity: %w", err)
		}
		return nil
	}

	// Check if the last activity timestamp is stale (older than timeout)
	// This prevents immediate timeout when daemon restarts after being down for a while
	timeout := d.config.GetTimeoutForContext(currentContext)
	timeSinceActivity := time.Since(lastActivity)
	if timeSinceActivity > timeout {
		d.logger.Printf("Daemon was down for %v (longer than timeout %v), resetting activity timer for context '%s'",
			timeSinceActivity.Round(time.Second), timeout, currentContext)
		if err := d.stateManager.RecordActivity(currentContext); err != nil {
			return fmt.Errorf("failed to record activity: %w", err)
		}
	}

	return nil
}

// Run starts the daemon main loop
func (d *Daemon) Run() error {
	if !d.config.Daemon.Enabled {
		d.logger.Println("Daemon is disabled in configuration")
		return nil
	}

	// Acquire PID file to ensure single instance
	if err := d.pidFile.Acquire(); err != nil {
		return fmt.Errorf("failed to acquire PID file: %w", err)
	}
	// Ensure PID file is released on exit
	defer d.pidFile.Release()

	d.logger.Printf("Starting kubectx-timeout daemon (PID: %d, check interval: %v, default timeout: %v)",
		os.Getpid(),
		d.config.Timeout.CheckInterval,
		d.config.Timeout.Default)

	// Create ticker for periodic checks
	ticker := time.NewTicker(d.config.Timeout.CheckInterval)
	defer ticker.Stop()

	// Setup signal handling for graceful shutdown and config reload
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Start kubeconfig file watcher in separate goroutine
	// This provides backup detection for context switches from any tool
	watcher, err := NewKubeconfigWatcher(d.stateManager, d.logger, d.ctx)
	if err != nil {
		d.logger.Printf("Warning: failed to create kubeconfig watcher: %v", err)
		// Don't fail daemon startup, just log warning and continue without file monitoring
	} else {
		go watcher.Watch()
	}

	// Main event loop
	for {
		select {
		case <-d.ctx.Done():
			d.logger.Println("Daemon context canceled, shutting down...")
			return nil

		case sig := <-sigChan:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				d.logger.Printf("Received %v signal, shutting down gracefully...", sig)
				d.Shutdown()
				return nil

			case syscall.SIGHUP:
				d.logger.Println("Received SIGHUP signal, reloading configuration...")
				if err := d.ReloadConfig(); err != nil {
					d.logger.Printf("Failed to reload config: %v", err)
				} else {
					d.logger.Println("Configuration reloaded successfully")
				}
			}

		case <-ticker.C:
			// Periodic timeout check
			if err := d.checkTimeout(); err != nil {
				d.logger.Printf("Error checking timeout: %v", err)
			}
		}
	}
}

// checkTimeout checks if timeout has been exceeded and switches context if needed
func (d *Daemon) checkTimeout() error {
	// Get time since last activity
	timeSince, err := d.stateManager.TimeSinceLastActivity()
	if err != nil {
		return fmt.Errorf("failed to get time since last activity: %w", err)
	}

	// Get current context
	currentContext, err := GetCurrentContext()
	if err != nil {
		// If we can't get current context, log and continue
		d.logger.Printf("Warning: failed to get current context: %v", err)
		return nil
	}

	// Check if context is in never_switch_from list
	for _, ctx := range d.config.Safety.NeverSwitchFrom {
		if ctx == currentContext {
			d.logger.Printf("Current context '%s' is in never_switch_from list, skipping timeout check", currentContext)
			return nil
		}
	}

	// If current context is already the default, no need to switch
	if currentContext == d.config.DefaultContext {
		return nil
	}

	// Get timeout for current context
	timeout := d.config.GetTimeoutForContext(currentContext)

	// Check if timeout exceeded
	if timeSince >= timeout {
		d.logger.Printf("Timeout exceeded for context '%s' (inactive for %v, timeout is %v)",
			currentContext, timeSince.Round(time.Second), timeout)

		// Trigger context switch
		if err := d.switchContext(currentContext, d.config.DefaultContext); err != nil {
			return fmt.Errorf("failed to switch context: %w", err)
		}
	}

	return nil
}

// switchContext switches from one context to another
func (d *Daemon) switchContext(fromContext, toContext string) error {
	// Use the safe switcher with safety checks
	if err := d.switcher.SwitchContextSafe(toContext, d.config.Safety.NeverSwitchTo); err != nil {
		return fmt.Errorf("context switch failed: %w", err)
	}

	d.logger.Printf("Successfully switched context from '%s' to '%s'", fromContext, toContext)

	// Record activity in the new context to keep state file in sync
	// This prevents the daemon from immediately trying to switch again
	if err := d.stateManager.RecordActivity(toContext); err != nil {
		d.logger.Printf("Warning: failed to record activity after context switch: %v", err)
		// Don't return error - the switch was successful
	}

	return nil
}

// ReloadConfig reloads the daemon configuration
func (d *Daemon) ReloadConfig() error {
	// Load new configuration from XDG path
	config, err := LoadConfig(GetConfigPath())
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Update daemon config
	d.config = config

	return nil
}

// Shutdown gracefully shuts down the daemon
func (d *Daemon) Shutdown() {
	d.logger.Println("Shutting down daemon gracefully...")

	// Cancel context to signal shutdown
	d.cancel()

	// Release PID file
	if err := d.pidFile.Release(); err != nil {
		d.logger.Printf("Warning: failed to release PID file: %v", err)
	}

	d.logger.Println("Daemon shutdown complete")
}
