#!/bin/bash
# Test demo setup and cleanup

set -euo pipefail

echo "🧪 Testing demo setup and cleanup..."

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Test setup
echo "🔧 Testing demo setup..."
if "$SCRIPT_DIR/setup-demo.sh"; then
    echo "✅ Demo setup test passed"
    
    # Test cleanup
    echo "🧹 Testing demo cleanup..."
    if "$SCRIPT_DIR/cleanup.sh"; then
        echo "✅ Demo cleanup test passed"
    else
        echo "❌ Demo cleanup test failed"
        exit 1
    fi
else
    echo "❌ Demo setup test failed"
    exit 1
fi

echo "🎉 All demo setup tests passed!"