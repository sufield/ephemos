#!/bin/bash
# Security scan script for Bazel

set -euo pipefail

echo "🔒 Running security scans on Bazel-built binaries..."

# Check if binaries directory exists
if [ ! -d "bazel-bin" ]; then
    echo "❌ No bazel-bin directory found. Run 'bazel build //...' first"
    exit 1
fi

# Find all built binaries
binaries=$(find bazel-bin -type f -executable -name "*" | grep -E "(ephemos-cli|config-validator|echo-server|echo-client)" || true)

if [ -z "$binaries" ]; then
    echo "⚠️ No binaries found to scan"
    exit 0
fi

echo "📋 Found binaries to scan:"
echo "$binaries"
echo ""

# Basic security checks
echo "🔍 Performing basic security checks..."

for binary in $binaries; do
    echo "Checking $binary..."
    
    # Check for executable stack
    if command -v execstack >/dev/null 2>&1; then
        if execstack -q "$binary" 2>/dev/null | grep -q "X"; then
            echo "⚠️ Warning: $binary has executable stack"
        else
            echo "✅ $binary: No executable stack"
        fi
    fi
    
    # Check for RELRO
    if command -v readelf >/dev/null 2>&1; then
        if readelf -l "$binary" 2>/dev/null | grep -q "GNU_RELRO"; then
            echo "✅ $binary: RELRO enabled"
        else
            echo "⚠️ Warning: $binary has no RELRO protection"
        fi
    fi
    
    # Check file permissions
    perms=$(stat -c "%a" "$binary")
    if [ "$perms" = "755" ] || [ "$perms" = "750" ]; then
        echo "✅ $binary: Secure permissions ($perms)"
    else
        echo "⚠️ Warning: $binary has unusual permissions ($perms)"
    fi
    
    echo ""
done

echo "✅ Security scan completed"