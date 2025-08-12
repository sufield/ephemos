#!/bin/bash

# Lightweight script to ensure protoc and Go protobuf tools are available
# This script is designed for CI environments and automated builds

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

echo "🔧 Ensuring protobuf dependencies are available..."

# Check and install Go protobuf tools
echo "Installing Go protobuf tools..."
if ! command_exists protoc-gen-go; then
    echo "Installing protoc-gen-go..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
fi

if ! command_exists protoc-gen-go-grpc; then
    echo "Installing protoc-gen-go-grpc..."
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
fi

# Ensure Go bin is in PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# Try to install protoc if we can
if ! command_exists protoc; then
    echo "protoc not found, trying to install..."
    
    # Try to install protoc with available package managers
    # Skip sudo attempts in CI environments to avoid breaking builds
    if [[ "${CI:-}" == "true" ]] || [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
        echo -e "${YELLOW}⚠️  CI environment detected - skipping protoc installation${NC}"
        echo "CI should handle protoc installation via workflow setup actions"
    elif command_exists brew; then
        echo "Installing protoc with brew..."
        brew install protobuf
    elif command_exists apt-get && sudo -n true 2>/dev/null; then
        echo "Installing protoc with apt..."
        sudo apt-get update && sudo apt-get install -y protobuf-compiler
    elif command_exists yum && sudo -n true 2>/dev/null; then
        echo "Installing protoc with yum..."
        sudo yum install -y protobuf-compiler
    elif command_exists dnf && sudo -n true 2>/dev/null; then
        echo "Installing protoc with dnf..."
        sudo dnf install -y protobuf-compiler
    else
        echo -e "${YELLOW}⚠️  Cannot auto-install protoc without sudo or brew${NC}"
        echo "Please install manually:"
        echo "  Ubuntu/Debian: sudo apt-get install -y protobuf-compiler"
        echo "  CentOS/RHEL: sudo yum install -y protobuf-compiler"
        echo "  macOS: brew install protobuf"
    fi
fi

# Verify tools are available
MISSING_TOOLS=0
if ! command_exists protoc; then
    echo -e "${YELLOW}⚠️  protoc: not available${NC}"
    MISSING_TOOLS=1
fi

if ! command_exists protoc-gen-go; then
    echo -e "${YELLOW}⚠️  protoc-gen-go: not available${NC}"
    MISSING_TOOLS=1
fi

if ! command_exists protoc-gen-go-grpc; then
    echo -e "${YELLOW}⚠️  protoc-gen-go-grpc: not available${NC}"  
    MISSING_TOOLS=1
fi

if [ $MISSING_TOOLS -eq 0 ]; then
    echo -e "${GREEN}✅ All protobuf tools are available${NC}"
    echo "protoc: $(which protoc)"
    echo "protoc-gen-go: $(which protoc-gen-go)"
    echo "protoc-gen-go-grpc: $(which protoc-gen-go-grpc)"
    exit 0
else
    # In CI environments, continue if we have Go tools even without protoc
    if command_exists protoc-gen-go && command_exists protoc-gen-go-grpc; then
        echo -e "${YELLOW}⚠️  protoc missing, but Go tools available. Build may still work if protobuf files are pre-generated.${NC}"
        exit 0
    else
        echo -e "${RED}❌ Critical protobuf tools missing${NC}"
        exit 1
    fi
fi