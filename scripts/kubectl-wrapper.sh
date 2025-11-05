#!/bin/bash
# kubectl wrapper for kubectx-timeout
# This wrapper records kubectl activity to track timeout

# Path to kubectx-timeout binary
KUBECTX_TIMEOUT_BIN="${KUBECTX_TIMEOUT_BIN:-/usr/local/bin/kubectx-timeout}"

# Path to real kubectl
REAL_KUBECTL="${REAL_KUBECTL:-$(which -a kubectl | grep -v "$(dirname "${BASH_SOURCE[0]}")" | head -1)}"

# Record activity before executing kubectl
# We do this in the background to not slow down kubectl commands
if [ -x "$KUBECTX_TIMEOUT_BIN" ]; then
    "$KUBECTX_TIMEOUT_BIN" record-activity >/dev/null 2>&1 &
fi

# Execute the real kubectl with all arguments
exec "$REAL_KUBECTL" "$@"
