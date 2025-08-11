#!/bin/bash
# CI linting script for Ephemos
# Comprehensive linting for continuous integration

set -euo pipefail

echo "🔍 Running CI linting checks..."

# Run basic linting
./scripts/build/lint.sh

# Additional CI-specific checks
echo "Running additional CI checks..."

# Check for TODO/FIXME comments (warning only)
echo "Checking for TODO/FIXME comments..."
if grep -r "TODO\|FIXME" --include="*.go" .; then
    echo "⚠️ Found TODO/FIXME comments (review recommended)"
fi

# Check for debug prints
echo "Checking for debug prints..."
if grep -r "fmt\.Print\|log\.Print" --include="*.go" . --exclude-dir=examples; then
    echo "⚠️ Found debug print statements (review recommended)"
fi

# Check for hardcoded domains (security check)
echo "Checking for hardcoded domains..."
# Exclude test files, imports, and common safe domains
if grep -r "spiffe://[a-zA-Z0-9.-]\+\.\(com\|net\|org\|io\)" --include="*.go" . \
    --exclude="*_test.go" \
    --exclude-dir=".git" \
    --exclude-dir="vendor" \
    | grep -v "example\.org\|example\.com\|test\.com\|test\.local\|localhost\|company\.com\|your\.domain\|your\.production\.domain\|prod\.company\.com"; then
    echo "❌ Found hardcoded production SPIFFE domains (security risk)" >&2
    exit 1
fi

echo "✅ CI linting completed successfully!"