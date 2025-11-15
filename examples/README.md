# Example Configurations

This directory contains example configuration files for `kubectx-timeout`. Choose the configuration that best matches your use case, copy it to `~/.config/kubectx-timeout/config.yaml`, and customize it for your environment.

## Quick Start

1. Choose an example configuration from the table below
2. Copy it to the config location:
   ```bash
   cp examples/<example-file> ~/.config/kubectx-timeout/config.yaml
   ```
3. Edit the file to customize for your contexts
4. **(Optional but recommended)** Install fswatch for enhanced context switch detection:
   ```bash
   brew install fswatch  # macOS only
   ```
5. Start the daemon:
   ```bash
   kubectx-timeout daemon
   ```

## Available Examples

| File | Use Case | Best For |
|------|----------|----------|
| [`config.minimal.yaml`](config.minimal.yaml) | Absolute minimal config | Quick testing, first-time setup |
| [`config.basic.yaml`](config.basic.yaml) | Simple single-timeout setup | Users with a few contexts, simple needs |
| [`config.example.yaml`](config.example.yaml) | Comprehensive reference | Understanding all options, general use |
| [`config.multi-context.yaml`](config.multi-context.yaml) | Different timeouts per context | Multiple environments with varying risk levels |
| [`config.enterprise.yaml`](config.enterprise.yaml) | Maximum security | Enterprise/regulated environments, strict compliance |
| [`config.local-dev.yaml`](config.local-dev.yaml) | Local development focus | Developers working primarily with local clusters |

## Configuration Comparison

### config.minimal.yaml
**Lines:** ~15 | **Complexity:** Minimal

Just the bare essentials to get started:
- Single global timeout (30m)
- Safe default context
- All other settings use defaults

**Use when:** You want to test kubectx-timeout quickly or prefer minimal configuration.

```bash
cp examples/config.minimal.yaml ~/.config/kubectx-timeout/config.yaml
# Edit default_context to your safe context name
```

---

### config.basic.yaml
**Lines:** ~70 | **Complexity:** Simple

A straightforward setup with one timeout for all contexts:
- Single global timeout applies to everything
- Clear explanations for each setting
- No per-context complexity

**Use when:** You have a few contexts and want the same timeout policy for all of them.

**Example scenario:** You work with production, staging, and local contexts but want them all to timeout after 30 minutes of inactivity.

---

### config.example.yaml
**Lines:** ~280+ | **Complexity:** Moderate (comprehensive reference)

The most detailed documentation of all configuration options:
- Every option explained in detail
- Examples and recommendations for each setting
- Troubleshooting section included
- References to other example files

**Use when:** You want to understand all available options or need a well-documented starting point.

**Example scenario:** You're setting up kubectx-timeout for the first time and want to understand what each option does.

---

### config.multi-context.yaml
**Lines:** ~150 | **Complexity:** Moderate

Different timeout policies for different contexts:
- 5m timeout for production contexts
- 15m for staging
- 1h for development
- 24h for local (effectively disabled)
- Extensive safety lists

**Use when:** You work with multiple environments and want risk-based timeout policies.

**Example scenario:** You occasionally access production (needs short timeout), regularly use staging (moderate timeout), and mostly work in dev/local (longer timeouts).

---

### config.enterprise.yaml
**Lines:** ~280 | **Complexity:** Advanced

Maximum security for enterprise/production environments:
- Very short timeouts (2-5m for production)
- Explicit configuration for every context
- Defense-in-depth safety features
- Detailed audit logging
- Compliance-focused settings
- Security checklist and best practices

**Use when:** You work in a regulated environment with strict security policies.

**Example scenario:** Your organization has SOC2/ISO27001 requirements, manages critical production infrastructure, or requires detailed audit trails for compliance.

**Key features:**
- 2-minute timeout for production contexts
- Never-switch lists for all production contexts
- Extended log retention (30 backups)
- Security deployment checklist
- Incident response guidelines

---

### config.local-dev.yaml
**Lines:** ~180 | **Complexity:** Moderate

Optimized for developers working primarily with local clusters:
- 24h timeouts for local contexts (effectively disabled)
- 5m timeouts for rare production access
- Battery-friendly polling (1m intervals)
- Convenient defaults for local work

**Use when:** You do most development locally and only occasionally need production access.

**Example scenario:** You work all day in docker-desktop or minikube, and only switch to production for quick debugging or verification. The short production timeout (5m) ensures you're quickly switched back to your safe local environment.

**Philosophy:** Maximize convenience for local work, maximize safety for rare production access.

---

## Choosing the Right Configuration

### Decision Tree

```
Do you work ONLY with local clusters?
├─ Yes → config.local-dev.yaml (or disable daemon entirely)
└─ No ↓

Do you work in an enterprise/regulated environment?
├─ Yes → config.enterprise.yaml
└─ No ↓

Do you need different timeouts for different contexts?
├─ Yes → config.multi-context.yaml
└─ No ↓

Do you want detailed documentation of all options?
├─ Yes → config.example.yaml
└─ No → config.basic.yaml
```

### By Experience Level

- **New to kubectx-timeout:** Start with `config.basic.yaml` or `config.minimal.yaml`
- **Understand the basics, need more control:** Use `config.multi-context.yaml`
- **Want to understand all options:** Use `config.example.yaml`
- **Enterprise deployment:** Use `config.enterprise.yaml`

### By Use Case

- **Solo developer, mostly local work:** `config.local-dev.yaml`
- **Team lead, mixed environments:** `config.multi-context.yaml`
- **SRE/DevOps, production access:** `config.enterprise.yaml`
- **Consultant, multiple client environments:** `config.multi-context.yaml` or `config.enterprise.yaml`

## Common Customizations

After copying an example, you'll typically need to:

1. **Set your default_context** (REQUIRED)
   ```yaml
   default_context: docker-desktop  # Change to your safe context
   ```

   Find your contexts: `kubectl config get-contexts`

2. **Adjust timeouts** for your workflow
   ```yaml
   timeout:
     default: 30m  # Increase/decrease based on your preference
   ```

3. **Add your context names** to per-context timeouts
   ```yaml
   contexts:
     your-production-context:  # Use your actual context names
       timeout: 5m
   ```

4. **Configure safety lists**
   ```yaml
   safety:
     never_switch_to:
       - your-production-context  # Add your production contexts
   ```

## Testing Your Configuration

After setting up your configuration:

```bash
# 1. Validate configuration (daemon will report errors)
kubectx-timeout daemon

# 2. Set a very short timeout for testing
# Edit config: timeout.default: 30s

# 3. Run a kubectl command
kubectl get pods

# 4. Wait 30 seconds - you should see a context switch

# 5. Check logs for confirmation
tail -f ~/.local/state/kubectx-timeout/daemon.log

# 6. Restore your normal timeout
# Edit config: timeout.default: 30m
```

## File Locations

### Configuration
- **Default:** `~/.config/kubectx-timeout/config.yaml`
- **Custom:** `$XDG_CONFIG_HOME/kubectx-timeout/config.yaml` (if XDG_CONFIG_HOME is set)

### State and Logs
- **State:** `~/.local/state/kubectx-timeout/state.json`
- **Logs:** `~/.local/state/kubectx-timeout/daemon.log`
- **Custom:** `$XDG_STATE_HOME/kubectx-timeout/` (if XDG_STATE_HOME is set)

## Additional Resources

- **Main README:** [`../README.md`](../README.md) - Overview and installation
- **Contributing:** [`../CONTRIBUTING.md`](../CONTRIBUTING.md) - Development guidelines
- **Development:** [`../DEVELOPMENT.md`](../DEVELOPMENT.md) - Detailed development docs

## Need Help?

- Check the troubleshooting section in `config.example.yaml`
- Review the main README for setup instructions
- Open an issue on GitHub for questions or bugs

## LaunchD Integration

This directory also contains:

- [`com.kubectx-timeout.plist`](com.kubectx-timeout.plist) - macOS launchd configuration for automatic daemon startup

See the main README for launchd setup instructions.
