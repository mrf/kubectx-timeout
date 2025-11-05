# Development Guide

This document provides detailed technical guidelines for developing kubectx-timeout. For general contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md).

## Table of Contents

- [Go Best Practices](#go-best-practices)
- [Security Guidelines](#security-guidelines)
- [Testing Practices](#testing-practices)
- [Common Pitfalls](#common-pitfalls)
- [Project-Specific Concerns](#project-specific-concerns)
- [Tool Configuration](#tool-configuration)
- [Code Quality Metrics](#code-quality-metrics)

---

## Go Best Practices

### Error Handling

Always handle errors explicitly with context:

```go
// GOOD: Clear error handling with context wrapping
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
    }
    // Continue processing
    return cfg, nil
}

// BAD: Swallowing errors
func LoadConfig(path string) *Config {
    data, _ := os.ReadFile(path)  // BAD: ignoring error
    // ...
}
```

### Interface Design

Keep interfaces small and focused:

```go
// GOOD: Small, focused interface
type ConfigLoader interface {
    Load() (*Config, error)
}

// BAD: Large, do-everything interface
type ConfigManager interface {
    Load() (*Config, error)
    Save(*Config) error
    Validate(*Config) error
    Reload() error
    Watch() error
    // ... too many methods
}
```

### Concurrency Patterns

Properly manage goroutine lifecycles:

```go
// GOOD: Proper lifecycle management with context
func (d *Daemon) Start(ctx context.Context) error {
    ticker := time.NewTicker(d.interval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            if err := d.checkTimeout(); err != nil {
                d.logger.Error("timeout check failed", "error", err)
            }
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}

// BAD: Goroutine leak potential
func (d *Daemon) Start() {
    go func() {
        for {
            time.Sleep(d.interval)  // No way to stop this
            d.checkTimeout()
        }
    }()
}
```

### Package Organization

```
kubectx-timeout/
├── cmd/
│   └── kubectx-timeout/    # Main application entry point
│       └── main.go
├── internal/               # Private application code
│   ├── config/            # Configuration loading/validation
│   ├── daemon/            # Core daemon logic
│   ├── tracker/           # Activity tracking
│   ├── switcher/          # Context switching
│   └── notify/            # Notification system
└── scripts/               # Build/install scripts
```

**Rules:**
- `internal/` packages cannot be imported by external projects
- `cmd/` contains only main packages (entry points)
- One package per directory
- Package names are lowercase, no underscores

### Naming Conventions

- **Packages**: lowercase, single word (`config`, not `config_loader`)
- **Interfaces**: noun or noun phrase (`Reader`, `ConfigLoader`)
- **Functions/Methods**: camelCase, start with verb (`LoadConfig`, `checkTimeout`)
- **Variables**: camelCase, descriptive (`configPath`, not `cp`)
- **Constants**: CamelCase or SCREAMING_SNAKE_CASE for exported

### Documentation

All exported types and functions must have godoc comments:

```go
// Package config provides configuration file loading and validation
// for the kubectx-timeout daemon.
package config

// Config represents the daemon's runtime configuration.
// It supports per-context timeout overrides and notification settings.
type Config struct {
    // DefaultTimeout is the fallback timeout duration when no
    // context-specific timeout is configured.
    DefaultTimeout time.Duration `yaml:"default_timeout"`
}

// Load reads and parses the configuration file from the given path.
// It returns an error if the file cannot be read or contains invalid YAML.
func Load(path string) (*Config, error) {
    // Implementation
}
```

---

## Security Guidelines

### Input Validation

All external inputs must be validated:

```go
// GOOD: Validate before use
func LoadConfig(path string) (*Config, error) {
    // Validate path
    if !filepath.IsAbs(path) {
        return nil, fmt.Errorf("config path must be absolute: %s", path)
    }

    // Check path traversal
    cleanPath := filepath.Clean(path)
    if !strings.HasPrefix(cleanPath, "/Users/") &&
       !strings.HasPrefix(cleanPath, "/etc/kubectx-timeout") {
        return nil, fmt.Errorf("config path outside allowed directories: %s", path)
    }

    data, err := os.ReadFile(cleanPath)
    // ...
}

// BAD: No validation
func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)  // Potential path traversal
    // ...
}
```

### Error Messages

Don't leak sensitive information in error messages:

```go
// GOOD: Safe error messages for users
func (s *Switcher) SwitchContext(name string) error {
    if err := s.validateContext(name); err != nil {
        // Log detailed error internally
        s.logger.Error("context validation failed",
            "context", name,
            "error", err)
        // Return safe error to user
        return fmt.Errorf("invalid context: %s", name)
    }
    // ...
}

// BAD: Leaking system details
func (s *Switcher) SwitchContext(name string) error {
    cmd := exec.Command("kubectl", "config", "use-context", name)
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("command failed: %v, user: %s, path: %s",
            err, os.Getenv("USER"), os.Getenv("PATH"))  // BAD: leaking env
    }
}
```

### File Permissions

Use restrictive file permissions:

```go
// GOOD: Explicit, restrictive permissions
func SaveState(path string, state *State) error {
    data, err := json.Marshal(state)
    if err != nil {
        return err
    }

    // Create with restricted permissions (owner read/write only)
    return os.WriteFile(path, data, 0600)
}

// BAD: Overly permissive
func SaveState(path string, state *State) error {
    data, err := json.Marshal(state)
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0666)  // BAD: world-readable
}
```

### Command Injection Prevention

Never use shell commands with user input:

```go
// GOOD: No user input in commands
func SwitchContext(contextName string) error {
    // Validate context name against allowed pattern
    if !validContextName.MatchString(contextName) {
        return fmt.Errorf("invalid context name format")
    }

    // Use exec.Command with separate arguments (no shell)
    cmd := exec.Command("kubectl", "config", "use-context", contextName)
    return cmd.Run()
}

// BAD: Shell injection risk
func SwitchContext(contextName string) error {
    cmd := exec.Command("sh", "-c",
        fmt.Sprintf("kubectl config use-context %s", contextName))
    return cmd.Run()  // BAD: shell injection if contextName is malicious
}
```

### Race Condition Prevention

Use proper locking for shared state:

```go
// GOOD: Proper locking for shared state
type StateManager struct {
    mu    sync.RWMutex
    state *State
}

func (sm *StateManager) UpdateLastActivity(t time.Time) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    sm.state.LastActivity = t
}

func (sm *StateManager) GetLastActivity() time.Time {
    sm.mu.RLock()
    defer sm.mu.RUnlock()
    return sm.state.LastActivity
}

// BAD: Unprotected concurrent access
type StateManager struct {
    state *State  // Multiple goroutines might access
}

func (sm *StateManager) UpdateLastActivity(t time.Time) {
    sm.state.LastActivity = t  // RACE CONDITION
}
```

### Secrets and Credentials

**STRICT RULES:**
- NO hardcoded credentials, API keys, or tokens
- NO credentials in logs (even debug logs)
- NO credentials in error messages
- Use environment variables or secure keychain
- Add `secrets/`, `*.key`, `*.pem` to .gitignore

```go
// GOOD: Never log sensitive data
func (n *Notifier) SendWebhook(url string, msg string) error {
    n.logger.Debug("sending webhook", "url", redactURL(url))
    // ...
}

// BAD: Logging sensitive information
func (n *Notifier) SendWebhook(url string, msg string) error {
    n.logger.Debug("sending webhook",
        "url", url,  // BAD: might contain API key in query params
        "msg", msg)
}
```

---

## Testing Practices

### Table-Driven Tests

Use table-driven tests for multiple scenarios:

```go
func TestTimeoutCalculation(t *testing.T) {
    tests := []struct {
        name           string
        lastActivity   time.Time
        timeout        time.Duration
        currentTime    time.Time
        expectExpired  bool
    }{
        {
            name:          "not expired - within timeout",
            lastActivity:  time.Now().Add(-10 * time.Minute),
            timeout:       30 * time.Minute,
            currentTime:   time.Now(),
            expectExpired: false,
        },
        {
            name:          "expired - timeout exceeded",
            lastActivity:  time.Now().Add(-40 * time.Minute),
            timeout:       30 * time.Minute,
            currentTime:   time.Now(),
            expectExpired: true,
        },
        {
            name:          "edge case - exactly at timeout",
            lastActivity:  time.Now().Add(-30 * time.Minute),
            timeout:       30 * time.Minute,
            currentTime:   time.Now(),
            expectExpired: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := isExpired(tt.lastActivity, tt.timeout, tt.currentTime)
            assert.Equal(t, tt.expectExpired, result)
        })
    }
}
```

### Mocking Dependencies

Use interfaces for testability:

```go
// Interface for mocking
type KubectlRunner interface {
    GetContexts() ([]string, error)
    UseContext(name string) error
}

// Test with mock
type mockKubectl struct {
    contexts []string
    err      error
}

func (m *mockKubectl) GetContexts() ([]string, error) {
    return m.contexts, m.err
}

func TestSwitcher_ValidContext(t *testing.T) {
    kubectl := &mockKubectl{
        contexts: []string{"local", "dev", "prod"},
    }

    switcher := NewSwitcher(kubectl)
    err := switcher.Switch("dev")

    assert.NoError(t, err)
}
```

### Integration Tests

Mark integration tests with build tags:

```go
// +build integration

func TestDaemon_FullLifecycle(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Set up isolated test environment
    tmpDir := t.TempDir()
    configPath := filepath.Join(tmpDir, "config.yaml")

    // Write test config
    writeTestConfig(t, configPath)

    // Start daemon
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    daemon := NewDaemon(configPath)

    // Test full lifecycle
    // ...
}
```

Run integration tests separately:
```bash
go test -tags=integration ./...
```

### Test File Organization

```
internal/config/
├── config.go
├── config_test.go        # Unit tests
├── testdata/             # Test fixtures
│   ├── valid.yaml
│   ├── invalid.yaml
│   └── empty.yaml
└── integration_test.go   # Integration tests
```

---

## Common Pitfalls

### Goroutine Leaks

```go
// BAD: Goroutine leak
func (d *Daemon) Start() {
    go d.monitor()  // Never stops
}

// GOOD: Controlled goroutine
func (d *Daemon) Start(ctx context.Context) {
    go func() {
        <-ctx.Done()
        // Cleanup
    }()
}
```

### Defer in Loops

```go
// BAD: Defer in loop
for _, file := range files {
    f, _ := os.Open(file)
    defer f.Close()  // Won't close until function returns
}

// GOOD: Immediate cleanup
for _, file := range files {
    func() {
        f, _ := os.Open(file)
        defer f.Close()
        // Process file
    }()
}
```

### Map Iteration Order

```go
// BAD: Assuming order
for k, v := range configMap {
    // Don't assume order!
}

// GOOD: Sort if order matters
keys := make([]string, 0, len(configMap))
for k := range configMap {
    keys = append(keys, k)
}
sort.Strings(keys)
for _, k := range keys {
    v := configMap[k]
    // Process in order
}
```

### Ignoring Errors

```go
// BAD: Ignoring errors
data, _ := os.ReadFile(path)

// GOOD: Handle or explicitly document why ignored
data, err := os.ReadFile(path)
if err != nil {
    // Handle appropriately
}

// Or if truly safe to ignore:
data, _ := os.ReadFile(path)  // OK: path is guaranteed to exist by previous check
```

---

## Project-Specific Concerns

### Configuration Loading

Watch for these issues:

```go
// BAD: No validation
func Load(path string) (*Config, error) {
    data, _ := os.ReadFile(path)
    var cfg Config
    yaml.Unmarshal(data, &cfg)
    return &cfg, nil
}

// GOOD: Proper validation and error handling
func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return DefaultConfig(), nil  // Graceful fallback
        }
        return nil, fmt.Errorf("failed to read config: %w", err)
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("invalid YAML: %w", err)
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    return &cfg, nil
}
```

### State Tracking

Ensure proper locking and consistent timestamps:

```go
// BAD: No locking
func (s *State) UpdateActivity() error {
    s.LastActivity = time.Now()
    return s.save()
}

// GOOD: File locking and UTC timestamps
func (s *State) UpdateActivity() error {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.LastActivity = time.Now().UTC()
    return s.save()
}
```

### Daemon Main Loop

Implement graceful shutdown:

```go
// BAD: No graceful shutdown
func (d *Daemon) Run() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        d.checkTimeout()  // If this panics, daemon crashes
    }
}

// GOOD: Graceful shutdown
func (d *Daemon) Run(ctx context.Context) error {
    ticker := time.NewTicker(d.config.CheckInterval)
    defer ticker.Stop()

    d.logger.Info("daemon started")
    defer d.logger.Info("daemon stopped")

    for {
        select {
        case <-ticker.C:
            if err := d.checkTimeout(); err != nil {
                d.logger.Error("timeout check failed", "error", err)
                // Don't crash on error
            }
        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

### Signal Handling

Proper signal handling for config reload and shutdown:

```go
// GOOD: Proper signal handling
func (d *Daemon) Run(ctx context.Context) error {
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

    ticker := time.NewTicker(d.config.CheckInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            d.checkTimeout()

        case sig := <-sigCh:
            switch sig {
            case syscall.SIGHUP:
                d.logger.Info("received SIGHUP, reloading config")
                if err := d.reloadConfig(); err != nil {
                    d.logger.Error("config reload failed", "error", err)
                }

            case syscall.SIGTERM, syscall.SIGINT:
                d.logger.Info("received shutdown signal", "signal", sig)
                return nil
            }

        case <-ctx.Done():
            return ctx.Err()
        }
    }
}
```

### Context Switching Safety

Validate contexts and prevent command injection:

```go
// BAD: Command injection risk
func Switch(contextName string) error {
    cmd := exec.Command("sh", "-c",
        fmt.Sprintf("kubectl config use-context %s", contextName))
    return cmd.Run()
}

// GOOD: Safe command execution
func Switch(contextName string) error {
    // Validate context name format
    if !validContextName.MatchString(contextName) {
        return fmt.Errorf("invalid context name: %s", contextName)
    }

    // Check context exists first
    exists, err := contextExists(contextName)
    if err != nil {
        return err
    }
    if !exists {
        return fmt.Errorf("context not found: %s", contextName)
    }

    // No shell, direct command execution
    cmd := exec.Command("kubectl", "config", "use-context", contextName)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("switch failed: %w (output: %s)", err, output)
    }

    return nil
}
```

---

## Tool Configuration

### golangci-lint Configuration

See `.golangci.yml` for the complete linter configuration. Key enabled linters:

- `errcheck` - Check error handling
- `gosimple` - Simplify code
- `govet` - Go vet examination
- `staticcheck` - Advanced static analysis
- `gosec` - Security checks
- `gocyclo` - Cyclomatic complexity (max: 10)
- `revive` - Fast, configurable linter

Run linter:
```bash
golangci-lint run ./...
```

### Pre-commit Hook

Install the pre-commit hook to catch issues early:

```bash
make setup-hooks
```

This runs:
1. Code formatting (`gofmt`, `goimports`)
2. Linting (`golangci-lint`)
3. Tests with race detector
4. Coverage check (80% minimum)
5. Security scan (`gosec`)

---

## Code Quality Metrics

### Track Over Time

- **Code Coverage**: Target 80% overall, 90% for core logic
- **Cyclomatic Complexity**: Max 10 per function
- **PR Merge Time**: Target < 48 hours
- **Test Execution Time**: Unit tests < 1s per package

### Generate Metrics

```bash
# Lines of code
find . -name "*.go" -not -path "./vendor/*" | xargs wc -l

# Test coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | grep total

# Cyclomatic complexity
gocyclo -over 10 .

# Generate full metrics report
make metrics
```

---

## Additional Resources

- [CONTRIBUTING.md](CONTRIBUTING.md) - General contribution guidelines
- [PR_CHECKLIST.md](PR_CHECKLIST.md) - Quick reference for code reviewers
- [Makefile](Makefile) - All available make commands
- [Effective Go](https://golang.org/doc/effective_go.html) - Official Go guidelines

---

**Last Updated**: 2025-11-04
