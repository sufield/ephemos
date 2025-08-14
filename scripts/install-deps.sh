#!/bin/bash

# Development dependencies installation script for Ephemos
# This script installs all required tools for HTTP over mTLS library development

set -e

echo "🔧 Installing Ephemos Development Dependencies..."
echo "=================================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Track installation status
INSTALL_ERRORS=0

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 1. Check Go installation
echo -e "\n${BLUE}1. Checking Go installation...${NC}"
if command_exists go; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo -e "${GREEN}✓ Go is installed: $GO_VERSION${NC}"
    
    # Check if Go version is adequate (1.24+)
    if [[ $(printf '%s\n' "1.24" "$GO_VERSION" | sort -V | head -n1) == "1.24" ]]; then
        echo -e "${GREEN}✓ Go version is adequate${NC}"
    else
        echo -e "${YELLOW}⚠️  Go version $GO_VERSION may be too old. Recommended: 1.24+${NC}"
    fi
else
    echo -e "${RED}✗ Go is not installed${NC}"
    echo "Please install Go from: https://golang.org/dl/"
    INSTALL_ERRORS=1
fi

# 2. Install development tools (optional but recommended)
echo -e "\n${BLUE}2. Installing optional development tools...${NC}"

# golangci-lint
if ! command_exists golangci-lint; then
    echo "Installing golangci-lint..."
    if curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin; then
        echo -e "${GREEN}✓ golangci-lint installed${NC}"
    else
        echo -e "${YELLOW}⚠️  Failed to install golangci-lint (optional)${NC}"
    fi
else
    echo -e "${GREEN}✓ golangci-lint already installed${NC}"
fi

# Security tools
echo -e "\n${BLUE}3. Installing security tools...${NC}"

# gosec
if ! command_exists gosec; then
    echo "Installing gosec..."
    if go install github.com/securego/gosec/v2/cmd/gosec@latest; then
        echo -e "${GREEN}✓ gosec installed${NC}"
    else
        echo -e "${YELLOW}⚠️  Failed to install gosec (optional)${NC}"
    fi
else
    echo -e "${GREEN}✓ gosec already installed${NC}"
fi

# govulncheck
if ! command_exists govulncheck; then
    echo "Installing govulncheck..."
    if go install golang.org/x/vuln/cmd/govulncheck@latest; then
        echo -e "${GREEN}✓ govulncheck installed${NC}"
    else
        echo -e "${YELLOW}⚠️  Failed to install govulncheck (optional)${NC}"
    fi
else
    echo -e "${GREEN}✓ govulncheck already installed${NC}"
fi

# 4. Verify PATH setup
echo -e "\n${BLUE}4. Verifying PATH setup...${NC}"
GO_BIN_PATH="$(go env GOPATH)/bin"
if [[ ":$PATH:" != *":$GO_BIN_PATH:"* ]]; then
    echo -e "${YELLOW}⚠️  $(go env GOPATH)/bin is not in PATH${NC}"
    echo "Add this to your shell profile (.bashrc, .zshrc, etc.):"
    echo "export PATH=\"\$PATH:\$(go env GOPATH)/bin\""
else
    echo -e "${GREEN}✓ Go bin directory is in PATH${NC}"
fi

# 5. Test build
echo -e "\n${BLUE}5. Testing build...${NC}"
if command_exists make && [[ -f "Makefile" ]]; then
    if make build; then
        echo -e "${GREEN}✓ Build test successful${NC}"
    else
        echo -e "${RED}✗ Build test failed${NC}"
        INSTALL_ERRORS=1
    fi
else
    echo -e "${YELLOW}⚠️  No Makefile found to test build${NC}"
fi

# Summary
echo -e "\n=================================================="
if [ $INSTALL_ERRORS -eq 0 ]; then
    echo -e "${GREEN}🎉 All dependencies installed successfully!${NC}"
    echo ""
    echo "Available commands:"
    echo -e "${GREEN}✓ Build commands:${NC}"
    echo "  make build       # Build CLI tools"
    echo "  make test        # Run tests"
    echo "  make lint        # Run linting"
    echo "  make clean       # Clean build artifacts"
    echo ""
    echo "For additional help, see: docs/contributors/"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some tools failed to install but core dependencies are ready.${NC}"
    echo ""
    echo "You can still develop with the available tools:"
    echo "  make build       # Build main CLI tools"
    echo "  make test        # Run tests"
    exit 0
fi