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
}

// NewDaemon creates a new daemon instance
func NewDaemon(configPath string, statePath string) (*Daemon, error) {
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

	return &Daemon{
		config:       config,
		stateManager: sm,
		switcher:     switcher,
		ctx:          ctx,
		cancel:       cancel,
		logger:       logger,
	}, nil
}

// Run starts the daemon main loop
func (d *Daemon) Run() error {
	if !d.config.Daemon.Enabled {
		d.logger.Println("Daemon is disabled in configuration")
		return nil
	}

	d.logger.Printf("Starting kubectx-timeout daemon (check interval: %v, default timeout: %v)",
		d.config.Timeout.CheckInterval,
		d.config.Timeout.Default)

	// Create ticker for periodic checks
	ticker := time.NewTicker(d.config.Timeout.CheckInterval)
	defer ticker.Stop()

	// Setup signal handling for graceful shutdown and config reload
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Main event loop
	for {
		select {
		case <-d.ctx.Done():
			d.logger.Println("Daemon context cancelled, shutting down...")
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
	return nil
}

// ReloadConfig reloads the daemon configuration
func (d *Daemon) ReloadConfig() error {
	// Load new configuration
	config, err := LoadConfig("~/.kubectx-timeout/config.yaml")
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Update daemon config
	d.config = config

	return nil
}

// Shutdown gracefully shuts down the daemon
func (d *Daemon) Shutdown() {
	d.logger.Println("Shutting down daemon...")
	d.cancel()
}
