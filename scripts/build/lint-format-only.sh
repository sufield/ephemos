#!/bin/bash
# Format-only linting script for Ephemos
# Runs only formatting checks without build validation

set -euo pipefail

echo "🎨 Running format-only linting checks..."

# Go formatting
echo "Running go fmt..."
if ! go fmt ./...; then
    echo "❌ go fmt found formatting issues" >&2
    exit 1
fi

echo "✅ Format-only linting completed successfully!"