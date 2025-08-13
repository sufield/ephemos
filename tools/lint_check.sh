#!/bin/bash
# Lint check script for Bazel

set -euo pipefail

echo "🧹 Running lint checks..."

# Check if Go is available
if ! command -v go >/dev/null 2>&1; then
    echo "❌ Go not found in PATH"
    exit 1
fi

# Run go fmt check
echo "📝 Checking Go formatting..."
if [ -n "$(gofmt -l .)" ]; then
    echo "❌ Go code is not formatted. Run 'gofmt -w .' to fix."
    gofmt -l .
    exit 1
else
    echo "✅ Go code is properly formatted"
fi

# Run go vet
echo "🔍 Running go vet..."
if go vet ./...; then
    echo "✅ go vet passed"
else
    echo "❌ go vet failed"
    exit 1
fi

# Check for common issues
echo "🔍 Checking for common issues..."

# Check for TODO/FIXME comments
if grep -r "TODO\|FIXME" --include="*.go" . >/dev/null 2>&1; then
    echo "⚠️ Found TODO/FIXME comments:"
    grep -rn "TODO\|FIXME" --include="*.go" . || true
fi

# Check for debug prints
if grep -r "fmt\.Print\|log\.Print" --include="*.go" . >/dev/null 2>&1; then
    echo "⚠️ Found debug print statements:"
    grep -rn "fmt\.Print\|log\.Print" --include="*.go" . || true
fi

echo "✅ Lint checks completed"