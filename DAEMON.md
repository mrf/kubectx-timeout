# Daemon Lifecycle Management

This document describes the daemon lifecycle management features for kubectx-timeout on macOS using launchd.

## Overview

The kubectx-timeout daemon runs in the background to monitor kubectl activity and automatically switch contexts after periods of inactivity. On macOS, the daemon is managed using launchd, Apple's service management framework.

## Features

- **Automatic startup**: Daemon starts automatically on user login
- **Single instance**: Ensures only one daemon instance runs at a time using PID file locking
- **Graceful shutdown**: Handles SIGINT and SIGTERM signals for clean shutdown
- **Configuration reload**: Supports SIGHUP signal to reload configuration without restart
- **Process supervision**: launchd automatically restarts the daemon if it crashes
- **Logging**: Separate stdout and stderr logs in XDG-compliant state directory

## Installation

### Prerequisites

- macOS system (launchd is macOS-specific)
- kubectx-timeout binary installed (e.g., in `/usr/local/bin/`)
- Configuration file initialized (`kubectx-timeout init`)

### Install Daemon

To install the daemon as a launchd service:

```bash
kubectx-timeout daemon-install
```

This command:
1. Creates a launchd plist file at `~/Library/LaunchAgents/com.kubectx-timeout.plist`
2. Configures the daemon to start automatically on login
3. Starts the daemon immediately
4. Sets up logging to `~/.local/state/kubectx-timeout/`

#### Custom Binary Path

If kubectx-timeout is installed in a non-standard location:

```bash
kubectx-timeout daemon-install --binary /path/to/kubectx-timeout
```

### Uninstall Daemon

To remove the daemon:

```bash
kubectx-timeout daemon-uninstall
```

This will:
1. Stop the daemon if running
2. Remove the launchd plist file
3. Daemon will no longer start automatically

## Daemon Control

### Start Daemon

```bash
kubectx-timeout daemon-start
```

Starts the daemon if it's not already running. The daemon must be installed first.

### Stop Daemon

```bash
kubectx-timeout daemon-stop
```

Stops the running daemon gracefully. The daemon will:
1. Complete any in-progress operations
2. Release the PID file
3. Exit cleanly

### Restart Daemon

```bash
kubectx-timeout daemon-restart
```

Stops and then starts the daemon. Useful after configuration changes or updates.

### Check Status

```bash
kubectx-timeout daemon-status
```

Shows:
- Whether daemon is installed
- Whether daemon is running
- Plist file location
- Binary path
- Detailed launchctl information (if running)
- Log file locations

## Manual Operation

For testing or development, you can run the daemon manually without launchd:

```bash
kubectx-timeout daemon
```

This runs the daemon in the foreground with output to stdout/stderr. Press Ctrl+C to stop.

### Manual Operation with Custom Paths

```bash
kubectx-timeout daemon --config /path/to/config.yaml --state /path/to/state.json
```

## Architecture

### Single Instance Enforcement

The daemon uses a PID file (`~/.local/state/kubectx-timeout/daemon.pid`) to ensure only one instance runs at a time:

1. On startup, daemon attempts to acquire the PID file
2. If PID file exists, daemon checks if the process is still running
3. If process is running, daemon exits with error
4. If process is not running (stale PID), daemon removes stale PID and continues
5. On shutdown, daemon releases the PID file

### Graceful Shutdown

The daemon handles shutdown signals properly:

- **SIGINT/SIGTERM**: Triggers graceful shutdown
  1. Stops accepting new operations
  2. Cancels context to signal all goroutines
  3. Releases PID file
  4. Logs shutdown message
  5. Exits

- **SIGHUP**: Reloads configuration without restarting
  1. Reloads config file from disk
  2. Updates daemon configuration
  3. Continues running with new config

### Launchd Integration

The daemon integrates with launchd using a plist file with the following features:

- **Label**: `com.kubectx-timeout`
- **RunAtLoad**: Start automatically on login
- **KeepAlive**: Restart if daemon crashes
- **ThrottleInterval**: Wait 10 seconds before restart to prevent rapid restarts
- **ProcessType**: Background process (low priority)
- **Nice**: Priority level 1 (slightly lower than default)

### Logging

Daemon logs are written to:

- stdout: `~/.local/state/kubectx-timeout/daemon.stdout.log`
- stderr: `~/.local/state/kubectx-timeout/daemon.stderr.log`

To view logs:

```bash
# Follow stdout log
tail -f ~/.local/state/kubectx-timeout/daemon.stdout.log

# Follow stderr log
tail -f ~/.local/state/kubectx-timeout/daemon.stderr.log

# View both with timestamps
tail -f ~/.local/state/kubectx-timeout/daemon.*.log
```

## Launchd Plist Configuration

The generated plist file (`~/Library/LaunchAgents/com.kubectx-timeout.plist`) contains:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.kubectx-timeout</string>

    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/kubectx-timeout</string>
        <string>daemon</string>
    </array>

    <key>RunAtLoad</key>
    <true/>

    <key>KeepAlive</key>
    <true/>

    <key>StandardOutPath</key>
    <string>~/.local/state/kubectx-timeout/daemon.stdout.log</string>

    <key>StandardErrorPath</key>
    <string>~/.local/state/kubectx-timeout/daemon.stderr.log</string>

    <key>WorkingDirectory</key>
    <string>~</string>

    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
        <key>HOME</key>
        <string>~</string>
    </dict>

    <key>ThrottleInterval</key>
    <integer>10</integer>

    <key>ProcessType</key>
    <string>Background</string>

    <key>Nice</key>
    <integer>1</integer>
</dict>
</plist>
```

## Troubleshooting

### Daemon Won't Start

1. Check if already running:
   ```bash
   kubectx-timeout daemon-status
   ```

2. Check for stale PID file:
   ```bash
   rm ~/.local/state/kubectx-timeout/daemon.pid
   kubectx-timeout daemon-start
   ```

3. Check launchd status:
   ```bash
   launchctl list | grep kubectx-timeout
   ```

### Daemon Crashes on Startup

1. Check error logs:
   ```bash
   cat ~/.local/state/kubectx-timeout/daemon.stderr.log
   ```

2. Verify configuration is valid:
   ```bash
   cat ~/.config/kubectx-timeout/config.yaml
   ```

3. Run daemon manually to see errors:
   ```bash
   kubectx-timeout daemon
   ```

### Daemon Not Starting Automatically

1. Verify plist is installed:
   ```bash
   ls -l ~/Library/LaunchAgents/com.kubectx-timeout.plist
   ```

2. Reload launchd:
   ```bash
   launchctl unload ~/Library/LaunchAgents/com.kubectx-timeout.plist
   launchctl load ~/Library/LaunchAgents/com.kubectx-timeout.plist
   ```

3. Check launchd errors:
   ```bash
   tail -f /var/log/system.log | grep kubectx-timeout
   ```

### Configuration Not Taking Effect

1. Reload configuration:
   ```bash
   # Send SIGHUP to reload config
   pkill -HUP kubectx-timeout
   ```

2. Or restart daemon:
   ```bash
   kubectx-timeout daemon-restart
   ```

### Multiple Instances Running

This should not happen due to PID file locking, but if it does:

1. Stop all instances:
   ```bash
   pkill kubectx-timeout
   ```

2. Remove stale PID file:
   ```bash
   rm ~/.local/state/kubectx-timeout/daemon.pid
   ```

3. Start daemon:
   ```bash
   kubectx-timeout daemon-start
   ```

## Advanced Usage

### Debugging

Run daemon with debug logging by editing the config file:

```yaml
daemon:
  enabled: true
  log_level: debug
```

Then restart:

```bash
kubectx-timeout daemon-restart
```

### Custom Launchd Configuration

To customize the plist after installation:

1. Unload the service:
   ```bash
   launchctl unload ~/Library/LaunchAgents/com.kubectx-timeout.plist
   ```

2. Edit the plist file:
   ```bash
   vim ~/Library/LaunchAgents/com.kubectx-timeout.plist
   ```

3. Reload the service:
   ```bash
   launchctl load ~/Library/LaunchAgents/com.kubectx-timeout.plist
   ```

### Running as System Daemon

For system-wide installation (requires root):

1. Copy plist to system location:
   ```bash
   sudo cp ~/Library/LaunchAgents/com.kubectx-timeout.plist /Library/LaunchDaemons/
   ```

2. Update plist to use absolute paths
3. Load with sudo:
   ```bash
   sudo launchctl load /Library/LaunchDaemons/com.kubectx-timeout.plist
   ```

**Note**: System-wide installation is not recommended as kubectx-timeout is designed for per-user kubectl contexts.

## File Locations

| File | Location | Description |
|------|----------|-------------|
| Configuration | `~/.config/kubectx-timeout/config.yaml` | Daemon configuration |
| State | `~/.local/state/kubectx-timeout/state.json` | Activity tracking state |
| PID file | `~/.local/state/kubectx-timeout/daemon.pid` | Process ID file |
| stdout log | `~/.local/state/kubectx-timeout/daemon.stdout.log` | Standard output |
| stderr log | `~/.local/state/kubectx-timeout/daemon.stderr.log` | Error output |
| Plist | `~/Library/LaunchAgents/com.kubectx-timeout.plist` | launchd configuration |

## Security Considerations

1. **PID file**: Only writable by user, prevents privilege escalation
2. **Configuration**: Mode 0600, readable only by user
3. **Logs**: Mode 0644, readable by user and group
4. **Plist**: Mode 0644, in user's LaunchAgents directory
5. **Binary**: Should have appropriate permissions (0755)

## See Also

- [Configuration Guide](examples/config.example.yaml)
- [README](README.md)
- [DEVELOPMENT](DEVELOPMENT.md)
- `man launchd.plist` - launchd plist file format
- `man launchctl` - launchd service management
