#!/bin/bash
# Complete demo workflow

set -euo pipefail

echo "🎬 Running complete Ephemos demo..."

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Setup demo
echo "🔧 Setting up demo environment..."
if ! "$SCRIPT_DIR/setup-demo.sh"; then
    echo "❌ Demo setup failed"
    exit 1
fi

# Run demo
echo "🚀 Running demo..."
if ! "$SCRIPT_DIR/run-demo.sh"; then
    echo "❌ Demo execution failed"
    # Still try to cleanup
    "$SCRIPT_DIR/cleanup.sh" || true
    exit 1
fi

# Cleanup
echo "🧹 Cleaning up demo environment..."
if ! "$SCRIPT_DIR/cleanup.sh"; then
    echo "⚠️ Demo cleanup had issues, but demo completed successfully"
    exit 0
fi

echo "🎉 Complete demo finished successfully!"