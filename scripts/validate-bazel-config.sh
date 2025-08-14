#!/bin/bash
# Validate Bazel configuration for protobuf optimizations
# This script checks if the optimization is properly configured

set -euo pipefail

echo "🔧 Validating Bazel protobuf optimization configuration..."

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

if grep -q "toolchains_protoc" WORKSPACE; then
    echo -e "${GREEN}✅ toolchains_protoc dependency found${NC}"
else
    echo -e "${RED}❌ toolchains_protoc dependency missing${NC}"
    exit 1
fi

if grep -q "protoc_toolchains" WORKSPACE; then
    echo -e "${GREEN}✅ protoc_toolchains registration found${NC}"
else
    echo -e "${RED}❌ protoc_toolchains registration missing${NC}"
    exit 1
fi

# Validate .bazelrc configuration
echo ""
echo "📋 Checking .bazelrc configuration..."

if grep -q "incompatible_enable_proto_toolchain_resolution" .bazelrc; then
    echo -e "${GREEN}✅ Proto toolchain resolution enabled${NC}"
else
    echo -e "${RED}❌ Proto toolchain resolution flag missing${NC}"
    exit 1
fi

if grep -q "PROTOBUF_WAS_NOT_SUPPOSED_TO_BE_BUILT" .bazelrc; then
    echo -e "${GREEN}✅ Fail-fast flags for source builds configured${NC}"
else
    echo -e "${RED}❌ Fail-fast flags missing${NC}"
    exit 1
fi

# Check toolchain files
echo ""
echo "📋 Checking custom toolchain files..."

if [ -f "tools/toolchains/BUILD.bazel" ]; then
    echo -e "${GREEN}✅ Custom toolchain BUILD.bazel found${NC}"
else
    echo -e "${RED}❌ Custom toolchain BUILD.bazel missing${NC}"
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

# Test toolchain availability
echo ""
echo "📋 Testing toolchain registration..."

if bazel query "kind(toolchain, //tools/toolchains:all)" >/dev/null 2>&1; then
    echo -e "${GREEN}✅ Custom toolchains registered successfully${NC}"
    bazel query "kind(toolchain, //tools/toolchains:all)"
else
    echo -e "${YELLOW}⚠️  Could not query custom toolchains${NC}"
fi

# Test proto targets (archived, no longer in active build)
echo ""
echo "📋 Proto targets archived for 0.1 release - skipping validation..."
echo -e "${YELLOW}ℹ️  Proto examples moved to archive/ folder${NC}"

echo ""
echo "🎉 Configuration validation completed!"
echo ""
echo "To test Bazel build:"
echo "  1. Run: bazel clean --expunge"  
echo "  2. Run: bazel build //..."
echo "  3. Run: bazel test //..."