# kubectx-timeout

A macOS daemon that automatically switches your kubectl context to a safe default after a period of inactivity, preventing accidental commands against production clusters.

## Overview

Have you ever forgotten you were in a production kubectl context and run a potentially dangerous command? `kubectx-timeout` solves this by monitoring your kubectl activity and automatically switching to a safe default context after a configurable timeout period.

## Goals

- **Safety First**: Prevent accidental operations on production clusters by enforcing automatic context timeouts
- **Simple & Reliable**: Lightweight daemon with minimal dependencies, designed for local macOS use
- **Configurable**: Per-context timeout settings with sensible defaults
- **Non-intrusive**: Runs quietly in the background via launchd, only acts when needed
- **Transparent**: Logs all context switches and provides status commands

## How It Works

1. A shell wrapper tracks kubectl command activity by writing timestamps to a state file
2. A background daemon monitors this activity via a periodic check
3. When inactivity exceeds the configured timeout, the daemon switches to your default safe context
4. You're notified of the switch and can continue working safely

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/YOUR-USERNAME/kubectx-timeout.git
cd kubectx-timeout

# Build the binary
make build

# Install to /usr/local/bin
sudo cp bin/kubectx-timeout /usr/local/bin/

# Initialize configuration
kubectx-timeout init
```

### Quick Setup

After installation, run the interactive setup:

```bash
# 1. Run the initialization wizard (guided setup)
kubectx-timeout init

# 2. Install shell integration
kubectx-timeout install-shell bash  # or zsh

# 3. Restart your shell or source your profile
source ~/.bashrc  # or ~/.zshrc

# 4. Start using kubectl - activity is now tracked!
kubectl get pods

# 5. Check status
kubectx-timeout status

# 6. (Optional) Run daemon in foreground to test
kubectx-timeout daemon
```

## Configuration

### File Locations

`kubectx-timeout` follows the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html) for clean, organized file management:

#### Configuration Files
- **Default**: `~/.config/kubectx-timeout/config.yaml`
- **Custom**: Set `$XDG_CONFIG_HOME` to override (uses `$XDG_CONFIG_HOME/kubectx-timeout/config.yaml`)

#### State Files
- **Default**: `~/.local/state/kubectx-timeout/state.json`
- **Custom**: Set `$XDG_STATE_HOME` to override (uses `$XDG_STATE_HOME/kubectx-timeout/`)
- **Log files**: Stored alongside state in `~/.local/state/kubectx-timeout/daemon.log`

#### Why XDG?

The XDG Base Directory specification provides:
- **Clean home directory**: No dotfiles cluttering `~/`
- **Standard locations**: Follows modern Unix/Linux conventions
- **User control**: Respects `$XDG_*` environment variables
- **Separation**: Config and state are kept in different directories

### Configuration File

The configuration file (`config.yaml`) controls all daemon behavior:

```yaml
# Global timeout settings
timeout:
  default: 30m          # Default timeout for all contexts
  check_interval: 30s   # How often to check for inactivity

# Context to switch to after timeout
default_context: local  # Should be a safe, non-production context

# Context-specific timeout overrides (optional)
contexts:
  production:
    timeout: 5m         # Production gets a shorter timeout

# Daemon behavior
daemon:
  enabled: true
  log_level: info       # debug, info, warn, error
  log_file: daemon.log
  log_max_size: 10      # MB
  log_max_backups: 5

# Notifications when context switch occurs
notifications:
  enabled: true
  method: both          # terminal, macos, or both

# Safety features
safety:
  check_active_kubectl: true
  validate_default_context: true
  never_switch_to:      # Extra safety
    - production
    - prod

# State file location (relative to state directory)
state_file: state.json

# Shell integration settings
shell:
  generate_wrapper: true
  shells:
    - bash
    - zsh
```

See [`examples/config.example.yaml`](examples/config.example.yaml) for a fully documented example.

### Minimal Configuration

For quick setup, you only need to specify your default (safe) context:

```yaml
timeout:
  default: 30m

default_context: local  # Change to your safe context
```

## Usage

### Command Reference

```bash
# Initialize configuration (interactive setup)
kubectx-timeout init

# Install shell integration
kubectx-timeout install-shell bash    # Install for bash
kubectx-timeout install-shell zsh     # Install for zsh

# Run daemon (usually via launchd, but can run manually)
kubectx-timeout daemon

# Check version
kubectx-timeout --version

# Get help
kubectx-timeout --help

# Use custom config/state paths
kubectx-timeout --config /custom/path/config.yaml --state /custom/path/state.json daemon
```

### launchd Integration (Recommended)

For automatic startup on macOS, use launchd:

```bash
# 1. Copy the example plist
cp examples/com.kubectx-timeout.plist ~/Library/LaunchAgents/

# 2. Edit the plist to replace YOUR_USERNAME with your actual username
nano ~/Library/LaunchAgents/com.kubectx-timeout.plist

# 3. Load the daemon
launchctl load ~/Library/LaunchAgents/com.kubectx-timeout.plist

# 4. Check it's running
launchctl list | grep kubectx-timeout
```

The daemon will now start automatically on login.

### Manual Daemon Management

```bash
# Start daemon (foreground)
kubectx-timeout daemon

# Start daemon (background)
kubectx-timeout daemon &

# Reload configuration (send SIGHUP)
killall -HUP kubectx-timeout

# Stop daemon
killall kubectx-timeout
```

## How It Works in Detail

### Activity Tracking

When you run `kubectl` commands, the shell wrapper:
1. Records the current timestamp to the state file
2. Records the current context name
3. Executes the actual kubectl command

The state file is a simple JSON file:
```json
{
  "last_activity": "2025-11-05T15:30:00Z",
  "current_context": "production"
}
```

### Timeout Detection

The daemon:
1. Checks the state file every `check_interval` (default: 30 seconds)
2. Compares current time to `last_activity`
3. If time elapsed > timeout for current context:
   - Validates the target (default) context exists
   - Checks if kubectl is currently running (optional safety check)
   - Switches to the default context using `kubectl config use-context`
   - Sends a notification (macOS notification + terminal output)

### Safety Features

- **Context Validation**: Ensures target context exists before switching
- **Active Command Detection**: Optionally prevents switching during active kubectl operations
- **Never-Switch Lists**: Contexts you never want to auto-switch from or to
- **Secure Execution**: Uses `exec.Command` (not shell) to prevent injection attacks

## Status

**Current Phase**: Functional MVP

Core features implemented and working:
- ✅ Configuration management with intelligent defaults
- ✅ Activity tracking and state management
- ✅ Daemon with timeout detection
- ✅ Safe context switching with validation
- ✅ Security hardening and testing
- ✅ CI/CD pipeline
- ✅ XDG Base Directory compliance

## Architecture

- **Language**: Go 1.21+ (for single-binary distribution and efficient daemon operation)
- **Platform**: macOS (using launchd for daemon management)
- **Dependencies**: Only kubectl (no external libraries for runtime)
- **File Locations**: Following [XDG Base Directory](https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html) specification
  - **Configuration**: `~/.config/kubectx-timeout/` (or `$XDG_CONFIG_HOME/kubectx-timeout/`)
  - **State & Logs**: `~/.local/state/kubectx-timeout/` (or `$XDG_STATE_HOME/kubectx-timeout/`)

### Project Structure

```
kubectx-timeout/
├── cmd/kubectx-timeout/    # Main application entry point
│   └── main.go
├── internal/               # Private application code
│   ├── config.go          # Configuration loading & validation
│   ├── daemon.go          # Core daemon logic
│   ├── paths.go           # XDG path management
│   ├── state.go           # State file management
│   ├── switcher.go        # Context switching
│   └── tracker.go         # Activity tracking & shell integration
├── examples/              # Example configurations
│   ├── config.example.yaml
│   ├── config.minimal.yaml
│   └── com.kubectx-timeout.plist
└── Makefile              # Build & development tasks
```

## Troubleshooting

### Daemon Not Starting

```bash
# Check if daemon is running
ps aux | grep kubectx-timeout

# Check launchd status
launchctl list | grep kubectx-timeout

# View logs
tail -f ~/.local/state/kubectx-timeout/daemon.log

# View launchd stderr/stdout
tail -f ~/.local/state/kubectx-timeout/daemon.stderr.log
tail -f ~/.local/state/kubectx-timeout/daemon.stdout.log
```

### Activity Not Being Tracked

```bash
# Verify shell integration installed
cat ~/.bashrc | grep kubectx-timeout  # or ~/.zshrc

# Test kubectl wrapper manually
which kubectl  # Should show your wrapper function, not /usr/local/bin/kubectl

# Check state file
cat ~/.local/state/kubectx-timeout/state.json
```

### Context Not Switching

```bash
# Check timeout configuration
cat ~/.config/kubectx-timeout/config.yaml

# Verify default context exists
kubectl config get-contexts

# Check daemon logs for errors
tail -100 ~/.local/state/kubectx-timeout/daemon.log
```

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for contribution guidelines and [DEVELOPMENT.md](DEVELOPMENT.md) for detailed development practices.

### Quick Start for Developers

```bash
# Clone repository
git clone https://github.com/YOUR-USERNAME/kubectx-timeout.git
cd kubectx-timeout

# Install development tools
make setup-tools

# Run tests
make test

# Check coverage
make coverage

# Build binary
make build

# Run all pre-commit checks
make pre-commit
```

## Contributing

Contributions are welcome! This is an early-stage project, so please:

1. Read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines
2. Check existing issues before creating new ones
3. Follow the code standards and testing requirements
4. Open PRs against the `main` branch

## License

See [LICENSE](LICENSE) file for details.

## Security

Found a security issue? Please email [security contact] instead of opening a public issue.
