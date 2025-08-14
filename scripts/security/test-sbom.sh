#!/bin/bash
# Test SBOM generation and validation

set -euo pipefail

echo "🧪 Testing SBOM generation and validation..."

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Create temporary directory for test
TEST_DIR=$(mktemp -d)
cleanup() {
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

cd "$TEST_DIR"

# Test SBOM generation
echo "📋 Testing SBOM generation..."
if "$SCRIPT_DIR/generate-sbom.sh"; then
    echo "✅ SBOM generation test passed"
else
    echo "❌ SBOM generation test failed"
    exit 1
fi

# Check if SBOM files were created
if [ -f "ephemos-sbom.spdx.json" ] && [ -f "ephemos-sbom.cyclonedx.json" ]; then
    echo "✅ SBOM files created successfully"
else
    echo "❌ SBOM files not found"
    exit 1
fi

# Test SBOM validation
echo "🔍 Testing SBOM validation..."
if "$SCRIPT_DIR/validate-sbom.sh"; then
    echo "✅ SBOM validation test passed"
else
    echo "❌ SBOM validation test failed"
    exit 1
fi

echo "🎉 All SBOM tests passed!"