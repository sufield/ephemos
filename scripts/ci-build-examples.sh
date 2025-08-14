#!/bin/bash
# CI-specific build script for examples with aggressive cache clearing
set -e

echo "🏗️  Starting CI build for examples with cache management..."

# Clear all Go caches aggressively
echo "🧹 Clearing all Go caches..."
go clean -cache || true
go clean -testcache || true 
go clean -modcache || true
echo "✅ All caches cleared"

# Verify Go environment
echo "🔍 Verifying Go environment..."
go version
go env GOROOT
go env GOPATH
go env GOCACHE

# Download dependencies fresh
echo "📦 Downloading dependencies..."
go mod download
go mod tidy

# Build the main package first to ensure it compiles
echo "🔨 Building main ephemos package..."
go build -v ./pkg/ephemos || {
    echo "❌ Failed to build pkg/ephemos"
    echo "Checking file contents:"
    ls -la pkg/ephemos/
    echo "Checking for missing functions:"
    grep -n "TransportServer\|newTransportServer\|mount" pkg/ephemos/*.go || echo "Functions not found"
    exit 1
}


# Clear cache again after protobuf generation
echo "🧹 Clearing cache after protobuf generation..."
go clean -cache || true

# Build examples
echo "🏗️  Building examples..."
make examples || {
    echo "❌ Failed to build examples"
    echo "Checking pkg/ephemos again:"
    ls -la pkg/ephemos/
    echo "Checking imports in examples:"
    grep -n "github.com/sufield/ephemos/pkg/ephemos" examples/*/main.go || true
    exit 1
}

echo "✅ CI build completed successfully!"