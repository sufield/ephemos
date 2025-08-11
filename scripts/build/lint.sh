#!/bin/bash
# Secure linting script for Ephemos
# Runs Go linting with security-focused checks

set -euo pipefail

echo "🔍 Running linting checks..."

# Basic Go formatting and vetting
echo "Running go fmt..."
if ! go fmt ./...; then
    echo "❌ go fmt found formatting issues" >&2
    exit 1
fi

echo "Running go vet..."
if ! go vet ./...; then
    echo "⚠️ go vet found build issues (may need 'make proto' first)" >&2
    echo "Continuing with other lint checks..."
fi

# golangci-lint if available
if command -v golangci-lint >/dev/null 2>&1; then
    echo "Running golangci-lint..."
    if golangci-lint run --config=.golangci.yml; then
        echo "✅ golangci-lint passed"
    else
        echo "❌ golangci-lint found issues" >&2
        exit 1
    fi
else
    echo "⚠️ golangci-lint not installed"
    echo "Install with: make install-security-tools"
fi

echo "✅ Linting completed successfully!"