#!/bin/bash
# Wrapper script to run the Go proto generator
# Can be called from Makefile or manually

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Change to project root
cd "$PROJECT_ROOT"

# Check if Go binary exists (built by Bazel)
BAZEL_BINARY="bazel-bin/scripts/proto_generator"
GO_BUILD_BINARY="/tmp/ephemos-proto-generator"

if [ -f "$BAZEL_BINARY" ]; then
    echo "Using Bazel-built proto generator..."
    "$BAZEL_BINARY" "$@"
elif command -v bazel >/dev/null 2>&1; then
    echo "Building and running proto generator with Bazel..."
    bazel build //scripts:proto_generator
    "$BAZEL_BINARY" "$@"
else
    echo "Building and running proto generator with Go..."
    go build -o "$GO_BUILD_BINARY" scripts/proto/main.go
    "$GO_BUILD_BINARY" "$@"
fi