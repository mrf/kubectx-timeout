package internal

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// ContextSwitcher handles safe kubectl context switching
type ContextSwitcher struct {
	logger     *log.Logger
	maxRetries int
	retryDelay time.Duration
}

// NewContextSwitcher creates a new context switcher
func NewContextSwitcher(logger *log.Logger) *ContextSwitcher {
	return &ContextSwitcher{
		logger:     logger,
		maxRetries: 3,
		retryDelay: 1 * time.Second,
	}
}

// ListContexts returns a list of available kubectl contexts
func (cs *ContextSwitcher) ListContexts() ([]string, error) {
	return GetAvailableContexts()
}

// GetAvailableContexts returns a list of all available kubectl contexts (global helper)
func GetAvailableContexts() ([]string, error) {
	cmd := exec.Command("kubectl", "config", "get-contexts", "-o", "name")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list contexts: %w", err)
	}

	contextsStr := strings.TrimSpace(string(output))
	if contextsStr == "" {
		return []string{}, nil
	}

	contexts := strings.Split(contextsStr, "\n")
	return contexts, nil
}

// ValidateContext checks if a context exists in kubectl config
func (cs *ContextSwitcher) ValidateContext(contextName string) error {
	contexts, err := cs.ListContexts()
	if err != nil {
		return err
	}

	for _, ctx := range contexts {
		if ctx == contextName {
			return nil
		}
	}

	return fmt.Errorf("context '%s' does not exist in kubectl config", contextName)
}

// SwitchContext switches to the specified kubectl context with retry logic
func (cs *ContextSwitcher) SwitchContext(targetContext string) error {
	// Get current context
	currentContext, err := GetCurrentContext()
	if err != nil {
		return fmt.Errorf("failed to get current context: %w", err)
	}

	// Check if already on target context
	if currentContext == targetContext {
		cs.logger.Printf("Already on context '%s', no switch needed", targetContext)
		return nil
	}

	// Validate target context exists
	if err := cs.ValidateContext(targetContext); err != nil {
		return err
	}

	// Attempt to switch with retry logic
	var lastErr error
	for attempt := 1; attempt <= cs.maxRetries; attempt++ {
		cs.logger.Printf("Switching context from '%s' to '%s' (attempt %d/%d)",
			currentContext, targetContext, attempt, cs.maxRetries)

		err := cs.executeSwitch(targetContext)
		if err == nil {
			cs.logger.Printf("Successfully switched context to '%s'", targetContext)
			return nil
		}

		lastErr = err
		cs.logger.Printf("Context switch attempt %d failed: %v", attempt, err)

		// Wait before retry (except on last attempt)
		if attempt < cs.maxRetries {
			cs.logger.Printf("Retrying in %v...", cs.retryDelay)
			time.Sleep(cs.retryDelay)
		}
	}

	return fmt.Errorf("failed to switch context after %d attempts: %w", cs.maxRetries, lastErr)
}

// executeSwitch performs the actual context switch
func (cs *ContextSwitcher) executeSwitch(targetContext string) error {
	cmd := exec.Command("kubectl", "config", "use-context", targetContext)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("kubectl command failed: %w, stderr: %s", err, stderr.String())
	}

	cs.logger.Printf("kubectl output: %s", strings.TrimSpace(string(output)))
	return nil
}

// SwitchContextSafe is a wrapper that includes additional safety checks
func (cs *ContextSwitcher) SwitchContextSafe(targetContext string, neverSwitchTo []string) error {
	// Check if target is in never_switch_to list
	for _, ctx := range neverSwitchTo {
		if ctx == targetContext {
			return fmt.Errorf("cannot switch to context '%s': it is in the never_switch_to list", targetContext)
		}
	}

	return cs.SwitchContext(targetContext)
}
