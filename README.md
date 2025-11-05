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

## Quick Start

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

## Status

**Current Phase**: Functional MVP

Core features implemented and working:
- ✅ Configuration management with intelligent defaults
- ✅ Activity tracking and state management
- ✅ Daemon with timeout detection
- ✅ Safe context switching with validation
- ✅ Security hardening and testing
- ✅ CI/CD pipeline

## Architecture

- **Language**: Go (for single-binary distribution and efficient daemon operation)
- **Platform**: macOS (using launchd for daemon management)
- **State**: File-based tracking in `~/.kubectx-timeout/`
- **Configuration**: YAML-based config in `~/.kubectx-timeout/config.yaml`

## Contributing

This is an early-stage project. Contributions, ideas, and feedback are welcome once the MVP is complete.

## License

See [LICENSE](LICENSE) file for details.
