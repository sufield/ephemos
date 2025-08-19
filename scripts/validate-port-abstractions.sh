#!/bin/bash
# Port Abstraction Validation Script
# This script enforces that the core domain doesn't leak infrastructure types
# Usage: scripts/validate-port-abstractions.sh

set -e

CORE_PATH="./internal/core"
EXIT_CODE=0

echo "🔍 Validating port abstractions in core domain..."

# Check for direct usage of net/http types in core domain (excluding comments and test files)
echo "Checking for net/http infrastructure leaks..."
if grep -r --include="*.go" --exclude="*_test.go" '\*http\.Client\|http\.Response\|http\.Request' "$CORE_PATH" | grep -v "//"; then
    echo "❌ Found net/http infrastructure leaks in core domain"
    EXIT_CODE=1
else
    echo "✅ No net/http infrastructure leaks found"
fi

# Check for direct usage of net types in core domain (excluding comments and test files)
echo "Checking for net package infrastructure leaks..."
if grep -r --include="*.go" --exclude="*_test.go" 'net\.Listener\|net\.Conn\|net\.Addr' "$CORE_PATH" | grep -v "//"; then
    echo "❌ Found net package infrastructure leaks in core domain"
    EXIT_CODE=1
else
    echo "✅ No net package infrastructure leaks found"
fi

# Verify abstraction files exist
echo "Checking for required abstraction files..."
ABSTRACTION_FILES=(
    "$CORE_PATH/ports/http_abstractions.go"
    "$CORE_PATH/ports/network_abstractions.go"
)

for file in "${ABSTRACTION_FILES[@]}"; do
    if [[ -f "$file" ]]; then
        echo "✅ Found $file"
    else
        echo "❌ Missing abstraction file: $file"
        EXIT_CODE=1
    fi
done

# Verify ports use only approved dependencies
echo "Checking port dependencies..."
ALLOWED_IMPORTS="context|io|fmt|time|sync|errors"
if grep -r --include="*.go" --exclude="*_test.go" '^import.*"net/\|^import.*"net"' "$CORE_PATH/ports/" | grep -v -E "($ALLOWED_IMPORTS)"; then
    echo "❌ Ports contain disallowed network imports"
    EXIT_CODE=1
else
    echo "✅ Ports use only approved abstractions"
fi

if [[ $EXIT_CODE -eq 0 ]]; then
    echo "🎉 All port abstraction validations passed!"
else
    echo "💥 Port abstraction validation failed!"
fi

exit $EXIT_CODE