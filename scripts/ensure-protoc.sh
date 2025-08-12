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

# Verify tools are available
if command_exists protoc && command_exists protoc-gen-go && command_exists protoc-gen-go-grpc; then
    echo -e "${GREEN}✅ All protobuf tools are available${NC}"
    echo "protoc: $(which protoc)"
    echo "protoc-gen-go: $(which protoc-gen-go)"
    echo "protoc-gen-go-grpc: $(which protoc-gen-go-grpc)"
    exit 0
else
    echo -e "${RED}❌ Some protobuf tools are still missing${NC}"
    
    if ! command_exists protoc; then
        echo -e "${YELLOW}protoc not found. Install with:${NC}"
        echo "  Ubuntu: sudo apt-get install -y protobuf-compiler"
        echo "  macOS: brew install protobuf"
    fi
    
    if ! command_exists protoc-gen-go; then
        echo -e "${YELLOW}protoc-gen-go not found${NC}"
    fi
    
    if ! command_exists protoc-gen-go-grpc; then
        echo -e "${YELLOW}protoc-gen-go-grpc not found${NC}"
    fi
    
    exit 1
fi