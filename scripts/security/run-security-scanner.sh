#!/bin/bash
# Wrapper script to run the Go security scanner
# Can be called from Makefile or manually

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

# Check if Go binary exists (built by Bazel)
BAZEL_BINARY="bazel-bin/scripts/security/security_scanner"
GO_BUILD_BINARY="/tmp/ephemos-security-scanner"

if [ -f "$BAZEL_BINARY" ]; then
    echo "Using Bazel-built security scanner..."
    "$BAZEL_BINARY" "$@"
elif command -v bazel >/dev/null 2>&1; then
    echo "Building and running security scanner with Bazel..."
    bazel build //scripts/security:security_scanner
    "$BAZEL_BINARY" "$@"
else
    echo "Building and running security scanner with Go..."
    go build -o "$GO_BUILD_BINARY" scripts/security/go/main.go
    "$GO_BUILD_BINARY" "$@"
fi