#!/bin/bash

# check-shadowing.sh - Check for variable shadowing in Go code
# Usage: ./scripts/check-shadowing.sh [package-pattern]

set -e

YELLOW='\033[1;33m'
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Default to checking internal packages if no argument provided
PACKAGE_PATTERN="${1:-./internal/...}"

echo -e "${YELLOW}🔍 Checking for variable shadowing in: ${PACKAGE_PATTERN}${NC}"

# Check if shadow tool is installed
if ! command -v shadow >/dev/null 2>&1; then
    echo -e "${RED}❌ Shadow analyzer not found${NC}"
    echo "Installing shadow analyzer..."
    go install golang.org/x/tools/go/analysis/passes/shadow/cmd/shadow@latest
    echo -e "${GREEN}✅ Shadow analyzer installed${NC}"
fi

# Run shadow analysis and filter for shadowing issues only
SHADOW_OUTPUT=$(go vet -vettool=$(which shadow) ${PACKAGE_PATTERN} 2>&1 | grep "declaration of" || true)

if [ -n "$SHADOW_OUTPUT" ]; then
    echo -e "${RED}❌ Variable shadowing detected:${NC}"
    echo ""
    
    # Format output for better readability
    echo "$SHADOW_OUTPUT" | while IFS= read -r line; do
        # Extract file path and line number
        FILE_INFO=$(echo "$line" | cut -d':' -f1-2)
        MESSAGE=$(echo "$line" | cut -d':' -f3-)
        echo -e "  ${RED}•${NC} ${FILE_INFO}${MESSAGE}"
    done
    
    echo ""
    echo -e "${YELLOW}💡 Tips to fix shadowing:${NC}"
    echo "  • Use more specific variable names (e.g., 'validationErr' instead of 'err')"
    echo "  • Extract intermediate variables to break up long chains"
    echo "  • Avoid reusing variable names in nested scopes"
    echo ""
    echo "  For detailed examples, see: docs/contributing/CODE_QUALITY_TOOLS.md"
    
    exit 1
else
    echo -e "${GREEN}✅ No variable shadowing detected${NC}"
fi

echo ""
echo -e "${YELLOW}ℹ️  To check specific packages:${NC}"
echo "  ./scripts/check-shadowing.sh ./internal/core/services/"
echo "  ./scripts/check-shadowing.sh ./internal/adapters/..."
echo "  ./scripts/check-shadowing.sh ./pkg/..."