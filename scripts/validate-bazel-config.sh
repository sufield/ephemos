#!/bin/bash
# Validate Bazel configuration for ephemos library
# This script checks if the build configuration is properly set

set -euo pipefail

echo "🔧 Validating Bazel configuration..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if bazel is available
if ! command -v bazel >/dev/null 2>&1; then
    echo -e "${YELLOW}Warning: bazel not found in PATH${NC}"
    echo "Install bazel to run full validation"
    exit 0
fi

echo "✅ Bazel found: $(bazel version | head -1)"

# Validate WORKSPACE configuration
echo ""
echo "📋 Checking WORKSPACE configuration..."

if [ -f "WORKSPACE" ]; then
    echo -e "${GREEN}✅ WORKSPACE file found${NC}"
else
    echo -e "${RED}❌ WORKSPACE file missing${NC}"
    exit 1
fi

# Test basic bazel query
echo ""
echo "📋 Testing Bazel workspace..."

if bazel query //... >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Bazel workspace parses successfully${NC}"
else
    echo -e "${RED}❌ Bazel workspace has syntax errors${NC}"
    echo "Run 'bazel query //...' for details"
    exit 1
fi



echo ""
echo "🎉 Configuration validation completed!"
echo ""
echo "To test Bazel build:"
echo "  1. Run: bazel clean --expunge"  
echo "  2. Run: bazel build //..."
echo "  3. Run: bazel test //..."