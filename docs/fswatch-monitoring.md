# File System Monitoring with fswatch

## Overview

`kubectx-timeout` includes optional file system monitoring to detect kubectl context switches made outside the shell wrapper. This provides comprehensive coverage for context changes made through:

- IDE plugins (VSCode Kubernetes extension, IntelliJ IDEA, etc.)
- GUI tools (Lens, K9s, etc.)
- Direct `kubectx` commands
- Manual kubeconfig file edits
- Any other tool that modifies `~/.kube/config`

## How It Works

### Architecture

The file monitoring feature:

1. **Uses fswatch** - A cross-platform file change monitor that uses native OS APIs
2. **Monitors kubeconfig** - Watches `~/.kube/config` (or `$KUBECONFIG` path) for modifications
3. **Detects context changes** - When the file changes, checks if the active context changed
4. **Resets timeout** - Records activity and extends the timeout when a context switch is detected
5. **Runs in background** - Operates in a separate goroutine alongside periodic timeout checking

### FSEvents API (macOS)

On macOS, fswatch uses the **FSEvents API**, which provides:

- **Efficient monitoring** - Low CPU and battery overhead
- **Kernel-level events** - File changes detected at the OS level
- **Coalescing** - Multiple rapid changes are batched together
- **Debouncing** - Built-in 0.5-second latency to handle transient writes

### Detection Flow

```
User switches context in IDE
    ↓
~/.kube/config is modified
    ↓
FSEvents API notifies fswatch
    ↓
fswatch outputs event to stdout
    ↓
Daemon reads event (NUL-separated)
    ↓
handleConfigChange() checks if context actually changed
    ↓
If changed: Record activity with new context
    ↓
Timeout is reset for the new context
```

## Installation

### Prerequisites

- **macOS only** - File monitoring currently requires macOS and the FSEvents API
- **Homebrew** - Recommended installation method

### Installing fswatch

```bash
# Install via Homebrew
brew install fswatch

# Verify installation
fswatch --version
```

### Enabling in kubectx-timeout

File monitoring is **automatically enabled** when:

1. Running on macOS
2. fswatch is installed and in PATH
3. Daemon is running

To enable after installing fswatch:

```bash
# Restart the daemon to enable file monitoring
kubectx-timeout restart-daemon

# Check logs to verify it's enabled
tail -f ~/.local/state/kubectx-timeout/daemon.log | grep -i fswatch
```

## Configuration

### No Configuration Required

File monitoring works automatically with **no configuration needed**. The daemon:

- Auto-detects fswatch availability
- Uses `$KUBECONFIG` environment variable if set
- Falls back to `~/.kube/config` if `KUBECONFIG` is not set
- Gracefully degrades if fswatch is not available

### Environment Variables

The kubeconfig path is determined by:

```bash
# Priority order:
1. $KUBECONFIG environment variable
2. ~/.kube/config (default)
```

Example with custom kubeconfig path:

```bash
# Set in your shell profile
export KUBECONFIG="$HOME/.kube/custom-config"

# Restart daemon
kubectx-timeout restart-daemon
```

## Graceful Degradation

The daemon is designed to work **with or without** fswatch:

### With fswatch installed

```
[kubectx-timeout] Starting kubeconfig file monitoring at /Users/you/.kube/config
```

### Without fswatch

```
[kubectx-timeout] fswatch not found - kubeconfig file monitoring disabled
[kubectx-timeout] Install fswatch for automatic context switch detection: brew install fswatch
```

**Important**: The daemon continues to function normally using shell wrapper detection even if fswatch is not available.

## Behavior Details

### What Triggers Activity Recording

The watcher records activity when:

1. **Context actually changed** - Different context detected in kubeconfig
2. **File modified in same context** - Any kubeconfig modification extends timeout

### Debouncing

File changes are debounced with a **0.5-second latency** to handle:

- Atomic writes (temp file + rename)
- Multiple rapid edits
- Tool-specific write patterns

### Startup Behavior

On daemon startup:

1. Checks if running on macOS
2. Checks if fswatch binary exists in PATH
3. Checks if kubeconfig file exists
4. Spawns watcher goroutine if all checks pass
5. Logs status message (enabled or disabled)

### Runtime Behavior

The watcher:

- Runs continuously in a separate goroutine
- Automatically restarts if fswatch process exits unexpectedly
- Handles context cancellation for clean shutdown
- Uses null-terminated output parsing for reliable path handling

## Advanced Usage

### Testing File Monitoring

Manually test the file watcher:

```bash
# In terminal 1: Watch daemon logs
tail -f ~/.local/state/kubectx-timeout/daemon.log

# In terminal 2: Make a context switch
kubectl config use-context staging

# You should see in logs:
# Detected context switch from 'local' to 'staging' via file monitoring
```

### Testing fswatch Directly

Test fswatch outside of kubectx-timeout:

```bash
# Monitor kubeconfig with fswatch
fswatch -0 -1 --event=Updated ~/.kube/config &

# In another terminal, modify kubeconfig
kubectl config use-context production

# fswatch should output the file path
```

### Custom KUBECONFIG Handling

If you use multiple kubeconfig files:

```bash
# Set KUBECONFIG to monitor specific file
export KUBECONFIG="$HOME/.kube/prod-config"

# Daemon will monitor the file specified in KUBECONFIG
kubectx-timeout restart-daemon

# Verify in logs
tail -100 ~/.local/state/kubectx-timeout/daemon.log | grep "file monitoring"
# Should show: Starting kubeconfig file monitoring at /Users/you/.kube/prod-config
```

## Troubleshooting

### File Monitoring Not Starting

**Check fswatch installation:**
```bash
which fswatch
# Should output: /opt/homebrew/bin/fswatch (or similar)

fswatch --version
# Should output version info
```

**Check platform:**
```bash
uname -s
# Should output: Darwin (macOS)
```

**Check daemon logs:**
```bash
tail -100 ~/.local/state/kubectx-timeout/daemon.log | grep -i fswatch
```

Expected messages:
- ✅ `Starting kubeconfig file monitoring at ...`
- ⚠️ `fswatch not found - kubeconfig file monitoring disabled`
- ⚠️ `Kubeconfig file not found at ... - file monitoring disabled`

### File Monitoring Stops Unexpectedly

**Check daemon logs for errors:**
```bash
tail -100 ~/.local/state/kubectx-timeout/daemon.log | grep -i "fswatch\|monitoring"
```

**Restart daemon:**
```bash
kubectx-timeout restart-daemon
```

The watcher includes automatic restart logic, so temporary failures should self-heal.

### Context Changes Not Detected

**Verify fswatch is monitoring:**
```bash
# Check if fswatch process is running
ps aux | grep fswatch
```

**Test fswatch manually:**
```bash
# Start fswatch in foreground
fswatch -0 -1 --event=Created --event=Updated --event=Renamed ~/.kube/config

# In another terminal, modify kubeconfig
kubectl config use-context staging

# fswatch should output the path to stdout
```

**Check kubeconfig path:**
```bash
# Verify the path being monitored
tail -100 ~/.local/state/kubectx-timeout/daemon.log | grep "Starting kubeconfig"

# Compare with your actual kubeconfig
echo $KUBECONFIG
ls -la ~/.kube/config
```

### High CPU Usage

File monitoring should have **minimal CPU overhead**. If you see high CPU:

**Check fswatch latency setting:**
The watcher uses `-l 0.5` (500ms debounce). If needed, you can verify this in the logs.

**Check for rapid kubeconfig changes:**
```bash
# Monitor how often kubeconfig changes
fswatch -l 0.5 ~/.kube/config
# Leave running and observe output frequency
```

**Disable if necessary:**
Uninstall fswatch to disable file monitoring:
```bash
brew uninstall fswatch
kubectx-timeout restart-daemon
```

## Implementation Details

### Source Files

- **`internal/watcher.go`** - Main watcher implementation
- **`internal/watcher_test.go`** - Comprehensive test suite
- **`internal/daemon.go`** - Integration into daemon lifecycle

### Key Components

#### KubeconfigWatcher struct
```go
type KubeconfigWatcher struct {
    kubeconfigPath string          // Path to monitor
    stateManager   *StateManager   // Records activity
    logger         *log.Logger     // Logging
    ctx            context.Context // Shutdown coordination
}
```

#### Main Functions

- **`NewKubeconfigWatcher()`** - Creates watcher instance
- **`Watch()`** - Main monitoring loop (runs in goroutine)
- **`isFswatchAvailable()`** - Checks for fswatch on macOS
- **`watchWithFswatch()`** - Spawns and manages fswatch process
- **`handleConfigChange()`** - Detects and records context changes
- **`scanNullTerminated()`** - Parses fswatch NUL-separated output

### Integration Points

The watcher integrates with:

- **StateManager** - Records activity when context changes
- **GetCurrentContext()** - Checks current kubectl context
- **Daemon context** - Coordinated shutdown via context cancellation

### Error Handling

- **Startup errors** - Logged but don't prevent daemon startup
- **Runtime errors** - Logged with automatic retry after 5 seconds
- **Context cancellation** - Clean shutdown without errors

## Performance Considerations

### CPU Usage

- **FSEvents API** - Kernel-level monitoring, minimal CPU overhead
- **Debouncing** - 500ms latency reduces event frequency
- **Event filtering** - Only Created, Updated, Renamed events monitored

### Battery Impact

- **Idle state** - No polling, only event-driven
- **Event coalescing** - Multiple changes batched by FSEvents
- **Minimal wake-ups** - Only when kubeconfig actually changes

### Memory Usage

- **Single goroutine** - Small memory footprint
- **No buffering** - Events processed immediately
- **Context caching** - Previous context stored to detect changes

## Platform Support

### Current Support

- ✅ **macOS** - Full support via FSEvents API
- ❌ **Linux** - Not yet implemented
- ❌ **Windows** - Not yet implemented

### Future Platforms

File monitoring could be extended to:

- **Linux** - Using inotify
- **Windows** - Using ReadDirectoryChangesW or FSW

Pull requests welcome for platform support!

## FAQ

### Q: Is fswatch required?

**A:** No, fswatch is optional. The daemon works perfectly fine without it, using shell wrapper detection.

### Q: What's the benefit of fswatch?

**A:** It detects context switches from IDE plugins, GUI tools, and direct kubectx commands that bypass the shell wrapper.

### Q: Does it work on Linux?

**A:** Not yet. File monitoring is currently macOS-only. Shell wrapper detection works on all platforms.

### Q: Will it drain my battery?

**A:** No. The FSEvents API is designed for low overhead and only generates events when files actually change.

### Q: What if fswatch crashes?

**A:** The watcher automatically restarts fswatch after a 5-second delay. The daemon continues running normally.

### Q: Can I disable file monitoring?

**A:** Yes, simply uninstall fswatch: `brew uninstall fswatch`

### Q: Does it monitor merged kubeconfig files?

**A:** It monitors the single file specified by `$KUBECONFIG`. If you use `KUBECONFIG=file1:file2:file3`, only the first file is monitored.

### Q: What about kubeconfig in non-standard locations?

**A:** Set `KUBECONFIG` environment variable and restart the daemon. It will monitor the specified path.

## Related Documentation

- [README.md](../README.md) - Main documentation
- [CONTRIBUTING.md](../CONTRIBUTING.md) - Development guidelines
- [notifications.md](notifications.md) - Notification system documentation
