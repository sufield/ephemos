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

# Check make
if command_exists make; then
    echo -e "${GREEN}✓ make: available${NC}"
else
    echo -e "${YELLOW}⚠ make: not installed (recommended)${NC}"
fi

# Check git
if command_exists git; then
    echo -e "${GREEN}✓ git: available${NC}"
else
    echo -e "${YELLOW}⚠ git: not installed (recommended for version info)${NC}"
fi

# Summary
if [ $MISSING_DEPS -eq 0 ]; then
    echo -e "\n${GREEN}🎉 All required dependencies are available!${NC}"
    exit 0
else
    echo -e "\n${RED}❌ Missing dependencies detected!${NC}"
    echo ""
    echo -e "${YELLOW}Required dependencies:${NC}"
    echo "  - Go 1.24 or later"
    echo ""
    echo "Installation options:"
    echo "  Ubuntu/Debian: sudo apt-get update && sudo apt-get install -y golang-go"
    echo "  CentOS/RHEL:   sudo yum install -y golang"
    echo "  macOS:         brew install go"
    echo "  Or download from: https://golang.org/dl/"
    exit 1
fi