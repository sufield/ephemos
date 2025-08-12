#!/bin/bash

# Quick dependency check script for Ephemos
# Verifies all required tools are available before building

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🔍 Checking Ephemos development dependencies..."

MISSING_DEPS=0

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check Go
if command_exists go; then
    echo -e "${GREEN}✓ Go: $(go version | awk '{print $3}')${NC}"
else
    echo -e "${RED}✗ Go: not installed${NC}"
    MISSING_DEPS=1
fi

# Check protoc
if command_exists protoc; then
    echo -e "${GREEN}✓ protoc: $(protoc --version | awk '{print $2}')${NC}"
else
    echo -e "${RED}✗ protoc: not installed${NC}"
    MISSING_DEPS=1
fi

# Check Go protobuf tools
export PATH="$PATH:$(go env GOPATH)/bin"

if command_exists protoc-gen-go; then
    echo -e "${GREEN}✓ protoc-gen-go: installed${NC}"
else
    echo -e "${RED}✗ protoc-gen-go: not installed${NC}"
    MISSING_DEPS=1
fi

if command_exists protoc-gen-go-grpc; then
    echo -e "${GREEN}✓ protoc-gen-go-grpc: installed${NC}"
else
    echo -e "${RED}✗ protoc-gen-go-grpc: not installed${NC}"
    MISSING_DEPS=1
fi

# Summary
if [ $MISSING_DEPS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All required dependencies are available!${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Missing dependencies detected!${NC}"
    echo ""
    echo -e "${YELLOW}The build system will attempt to install these automatically.${NC}"
    echo "If automatic installation fails, install manually:"
    echo ""
    echo "Ubuntu/Debian:"
    echo "  sudo apt-get update && sudo apt-get install -y protobuf-compiler"
    echo ""
    echo "CentOS/RHEL:"
    echo "  sudo yum install -y protobuf-compiler"
    echo ""
    echo "macOS:"
    echo "  brew install protobuf"
    echo ""
    echo "Go tools (all platforms):"
    echo "  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    echo "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    exit 1
fi