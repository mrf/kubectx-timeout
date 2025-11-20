# kubectx-timeout

A macOS daemon that automatically switches your kubectl context to a safe default after a period of inactivity, preventing accidental commands against production clusters.

[![Go Report Card](https://goreportcard.com/badge/github.com/mrf/kubectx-timeout)](https://goreportcard.com/report/github.com/mrf/kubectx-timeout)

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
3. **Optional**: File system monitoring (fswatch) detects context switches from any tool
4. When inactivity exceeds the configured timeout, the daemon switches to your default safe context
5. You're notified of the switch and can continue working safely

## Installation

### Quick Start

```bash
# 1. Clone and build
git clone https://github.com/YOUR-USERNAME/kubectx-timeout.git
cd kubectx-timeout
make build

# 2. Install binary
sudo cp bin/kubectx-timeout /usr/local/bin/

# 3. Initialize configuration (interactive, selects your safe default context)
kubectx-timeout init

# 4. Install shell integration (auto-detects your shell: bash/zsh/fish)
kubectx-timeout install-shell

# 5. Install and start the daemon (macOS launchd)
kubectx-timeout install-daemon
kubectx-timeout start-daemon

# 6. (Optional but recommended) Install fswatch for automatic context switch detection
brew install fswatch

# 7. Restart your shell
source ~/.bashrc  # or ~/.zshrc

# You're done! kubectl activity is now tracked.
```

### Installation Steps Explained

#### 1. Build and Install Binary

```bash
make build
sudo cp bin/kubectx-timeout /usr/local/bin/
```

#### 2. Initialize Configuration


The `init` command creates an interactive configuration file:

```bash
kubectx-timeout init
```

This will:
- Show available kubectl contexts
- Let you select a safe default context
- Create `~/.config/kubectx-timeout/config.yaml` with sensible defaults

#### 3. Install Shell Integration

The shell integration wraps kubectl to track activity:

```bash
# Auto-detect current shell
kubectx-timeout install-shell

# Or specify shell explicitly
kubectx-timeout install-shell bash
kubectx-timeout install-shell zsh
kubectx-timeout install-shell fish
```

This modifies your shell profile (`.bashrc`, `.zshrc`, or `config.fish`) to wrap kubectl commands.

#### 4. Set Up Daemon (macOS)

Install the launchd agent for automatic daemon startup:

```bash
# Install daemon configuration
kubectx-timeout install-daemon

# Start the daemon
kubectx-timeout start-daemon

# Check daemon status
kubectx-timeout daemon-status
```

**Other daemon commands:**
- `kubectx-timeout stop-daemon` - Stop the daemon
- `kubectx-timeout restart-daemon` - Restart the daemon
- `kubectx-timeout uninstall-daemon` - Remove daemon configuration

**Manual daemon control** (if not using launchd):
```bash
# Run in foreground (for testing)
kubectx-timeout daemon

# Run in background
kubectx-timeout daemon &
```

### Verification

After installation, verify everything is working:

```bash
# Check the binary is installed
kubectx-timeout version

# Run a kubectl command (activity will be tracked)
kubectl get pods

# Check daemon status
kubectx-timeout daemon-status

# View logs
tail -f ~/.local/state/kubectx-timeout/daemon.log
```

### Uninstallation

The CLI provides a complete uninstallation command:

```bash
# Interactive uninstallation (keeps binary and config by default)
kubectx-timeout uninstall

# Complete removal including binary
kubectx-timeout uninstall --all

# Remove everything but keep configuration files
kubectx-timeout uninstall --all --keep-config

# Non-interactive (useful for scripts)
kubectx-timeout uninstall --all --yes
```

The uninstall command will:
1. Stop and remove the launchd daemon
2. Remove shell integration from all detected shells
3. Optionally remove configuration and state files
4. Optionally remove the binary (with `--all`)
5. Clean up backup files

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

### Daemon Management

The daemon is managed through launchd on macOS for automatic startup and process supervision.

**Quick Start:**

```bash
# Install daemon as launchd service
kubectx-timeout daemon-install

# The daemon will start automatically and on every login
kubectx-timeout daemon-status
```

**Management Commands:**

```bash
kubectx-timeout daemon-install   # Install daemon as launchd service
kubectx-timeout daemon-uninstall # Remove daemon service
kubectx-timeout daemon-start     # Start the daemon
kubectx-timeout daemon-stop      # Stop the daemon
kubectx-timeout daemon-restart   # Restart the daemon
kubectx-timeout daemon-status    # Show detailed status
```

**Manual Mode (for testing/development):**

```bash
# Start daemon in foreground
kubectx-timeout daemon

# Start daemon in background
kubectx-timeout daemon &

# Reload configuration (send SIGHUP)
pkill -HUP kubectx-timeout

# Stop daemon
pkill kubectx-timeout
```

For detailed documentation on daemon management, architecture, troubleshooting, and advanced usage, see [DAEMON.md](DAEMON.md).

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

### File System Monitoring (Optional)

For enhanced detection of context switches made outside the shell wrapper (e.g., IDE plugins, GUI tools, direct kubeconfig edits), `kubectx-timeout` supports **fswatch-based monitoring** on macOS.

#### How It Works

When fswatch is installed, the daemon:
1. Monitors `~/.kube/config` (or `$KUBECONFIG` path) for file modifications
2. Uses the macOS FSEvents API for efficient, low-overhead file watching
3. Detects context switches from ANY tool that modifies the kubeconfig
4. Automatically records activity and resets the timeout when a context change is detected

#### Installing fswatch

```bash
# macOS (Homebrew)
brew install fswatch
```

After installation, restart the daemon to enable file monitoring:

```bash
kubectx-timeout restart-daemon
```

#### Behavior

- **If fswatch is available**: Automatic monitoring is enabled at daemon startup
- **If fswatch is not installed**: Daemon continues to work normally using only shell-wrapper-based detection
- **Graceful degradation**: The daemon logs an informative message and continues without file monitoring

The file watcher runs in a separate goroutine alongside the periodic timeout checker, providing comprehensive coverage:
- **Shell wrapper**: Detects kubectl commands
- **File monitoring**: Detects context switches from IDE plugins, kubectx, GUI tools, manual edits

#### Platform Support

File system monitoring is currently **macOS-only** as it uses the FSEvents API. On other platforms, the daemon will automatically skip file monitoring and rely solely on shell wrapper detection.

### Timeout Detection

The daemon uses a simple, efficient polling mechanism:
1. Checks the state file every `check_interval` (default: 30 seconds)
2. Uses mtime-based caching to avoid unnecessary file reads (battery optimization)
3. Compares current time to `last_activity`
4. If time elapsed > timeout for current context:
   - Validates the target (default) context exists
   - Checks if kubectl is currently running (optional safety check)
   - Switches to the default context using `kubectl config use-context`
   - Sends a notification (macOS notification + terminal output)

**Battery Optimization**: The daemon is designed to be battery-friendly:
- File modification time (mtime) is checked before reading the full state file
- Cached values are used when the file hasn't changed
- Default 30s check interval is a good balance (can be increased for longer battery life)
- Timeout checking is not time-critical, so longer intervals (60s+) are acceptable

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
- ✅ Daemon with timeout detection and fswatch support
- ✅ Safe context switching with validation
- ✅ Security hardening and testing
- ✅ CI/CD pipeline
- ✅ XDG Base Directory compliance

In progress:
- 🚧 Comprehensive logging and observability
- 🚧 Complete user documentation
- 🚧 Full unit test coverage

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
# Check daemon status
kubectx-timeout daemon-status

# View logs
tail -f ~/.local/state/kubectx-timeout/daemon.stderr.log
tail -f ~/.local/state/kubectx-timeout/daemon.stdout.log

# Try restarting
kubectx-timeout daemon-restart
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
tail -100 ~/.local/state/kubectx-timeout/daemon.stderr.log
```

### File System Monitoring Not Working

```bash
# Check if fswatch is installed
which fswatch
fswatch --version

# Install fswatch (macOS)
brew install fswatch

# Check daemon logs for fswatch status
tail -100 ~/.local/state/kubectx-timeout/daemon.log | grep -i fswatch

# Restart daemon after installing fswatch
kubectx-timeout restart-daemon

# Verify KUBECONFIG path
echo $KUBECONFIG  # Should show path to config, or be empty (uses ~/.kube/config)

# Test fswatch manually
fswatch -1 ~/.kube/config &
# Make a change to kubeconfig in another terminal
# You should see output from fswatch
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

For maintainers creating releases, see the [Release Process](CONTRIBUTING.md#release-process) section in CONTRIBUTING.md and [VERSIONING.md](VERSIONING.md) for versioning strategy.

## License

See [LICENSE](LICENSE) file for details.

## Security

Found a security issue? Please email [security contact] instead of opening a public issue.
