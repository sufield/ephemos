#!/bin/bash
# Test SPIRE lifecycle (install, start, stop, cleanup)

set -euo pipefail

echo "🧪 Testing SPIRE lifecycle..."

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Test SPIRE installation
echo "📦 Testing SPIRE installation..."
if "$SCRIPT_DIR/install-spire.sh"; then
    echo "✅ SPIRE installation test passed"
else
    echo "❌ SPIRE installation test failed"
    exit 1
fi

# Test SPIRE start
echo "🚀 Testing SPIRE start..."
if "$SCRIPT_DIR/start-spire.sh"; then
    echo "✅ SPIRE start test passed"
    
    # Give SPIRE time to start
    sleep 2
    
    # Test SPIRE stop
    echo "🛑 Testing SPIRE stop..."
    if "$SCRIPT_DIR/stop-spire.sh"; then
        echo "✅ SPIRE stop test passed"
        
        # Test cleanup
        echo "🧹 Testing SPIRE cleanup..."
        if "$SCRIPT_DIR/cleanup-spire-entries.sh"; then
            echo "✅ SPIRE cleanup test passed"
        else
            echo "❌ SPIRE cleanup test failed"
            exit 1
        fi
    else
        echo "❌ SPIRE stop test failed"
        exit 1
    fi
else
    echo "❌ SPIRE start test failed"
    exit 1
fi

echo "🎉 All SPIRE lifecycle tests passed!"