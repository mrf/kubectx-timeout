#!/bin/bash
# Shell integration for kubectx-timeout
# Source this file in your .bashrc or .zshrc to enable kubectl activity tracking

# Function-based kubectl wrapper
# This is lighter weight than aliasing to a script
_kubectx_timeout_kubectl() {
    local kubectx_timeout_bin="${KUBECTX_TIMEOUT_BIN:-/usr/local/bin/kubectx-timeout}"
    
    # Record activity in background (non-blocking)
    if [ -x "$kubectx_timeout_bin" ]; then
        "$kubectx_timeout_bin" record-activity >/dev/null 2>&1 &
    fi
    
    # Execute kubectl with all arguments
    command kubectl "$@"
}

# Create kubectl alias/function
# Use a function instead of alias for better compatibility
kubectl() {
    _kubectx_timeout_kubectl "$@"
}

# Export for use in subshells
export -f _kubectx_timeout_kubectl
