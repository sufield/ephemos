#!/bin/bash

# Development dependencies installation script for Ephemos (with sudo)
# This script installs all required tools for HTTP over mTLS library development using sudo

set -e

echo "🔧 Installing Ephemos Development Dependencies (with sudo)..."
echo "============================================================="

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

# Function to install for different package managers (with sudo for system packages)
install_with_manager() {
    local package="$1"
    local manager="$2"
    
    echo -e "${YELLOW}Installing $package with $manager...${NC}"
    case "$manager" in
        "apt")
            sudo apt-get update && sudo apt-get install -y "$package"
            ;;
        "yum")
            sudo yum install -y "$package"
            ;;
        "dnf")
            sudo dnf install -y "$package"
            ;;
        "pacman")
            sudo pacman -S --noconfirm "$package"
            ;;
        "brew")
            brew install "$package"
            ;;
        "choco")
            choco install "$package"
            ;;
        *)
            echo -e "${RED}Unknown package manager: $manager${NC}"
            return 1
            ;;
    esac
}

# Detect package manager
detect_package_manager() {
    if command_exists apt-get; then
        echo "apt"
    elif command_exists yum; then
        echo "yum"
    elif command_exists dnf; then
        echo "dnf"
    elif command_exists pacman; then
        echo "pacman"
    elif command_exists brew; then
        echo "brew"
    elif command_exists choco; then
        echo "choco"
    else
        echo "unknown"
    fi
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
        echo "Consider upgrading Go from: https://golang.org/dl/"
    fi
else
    echo -e "${RED}✗ Go is not installed${NC}"
    echo "Installing Go..."
    PACKAGE_MANAGER=$(detect_package_manager)
    case "$PACKAGE_MANAGER" in
        "apt")
            install_with_manager "golang-go" "apt"
            ;;
        "yum"|"dnf")
            install_with_manager "golang" "$PACKAGE_MANAGER"
            ;;
        "pacman")
            install_with_manager "go" "pacman"
            ;;
        "brew")
            install_with_manager "go" "brew"
            ;;
        *)
            echo -e "${RED}✗ Could not detect package manager${NC}"
            echo "Please install Go manually from: https://golang.org/dl/"
            INSTALL_ERRORS=1
            ;;
    esac
fi

# 2. Install system development tools
echo -e "\n${BLUE}2. Installing system development tools...${NC}"
PACKAGE_MANAGER=$(detect_package_manager)
case "$PACKAGE_MANAGER" in
    "apt")
        install_with_manager "build-essential git curl" "apt"
        ;;
    "yum"|"dnf")
        install_with_manager "gcc make git curl" "$PACKAGE_MANAGER"
        ;;
    "pacman")
        install_with_manager "base-devel git curl" "pacman"
        ;;
    "brew")
        # Xcode command line tools provide build essentials on macOS
        if ! xcode-select -p &>/dev/null; then
            echo "Installing Xcode command line tools..."
            xcode-select --install 2>/dev/null || true
        fi
        ;;
    *)
        echo -e "${YELLOW}⚠️  Unknown package manager, skipping system tools${NC}"
        ;;
esac

# 3. Install Go development tools
echo -e "\n${BLUE}3. Installing Go development tools...${NC}"
if command_exists go; then
    # golangci-lint
    if ! command_exists golangci-lint; then
        echo "Installing golangci-lint..."
        if curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin; then
            echo -e "${GREEN}✓ golangci-lint installed${NC}"
        else
            echo -e "${YELLOW}⚠️  Failed to install golangci-lint${NC}"
        fi
    else
        echo -e "${GREEN}✓ golangci-lint already installed${NC}"
    fi
    
    # gosec
    if ! command_exists gosec; then
        echo "Installing gosec..."
        if go install github.com/securego/gosec/v2/cmd/gosec@latest; then
            echo -e "${GREEN}✓ gosec installed${NC}"
        else
            echo -e "${YELLOW}⚠️  Failed to install gosec${NC}"
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
            echo -e "${YELLOW}⚠️  Failed to install govulncheck${NC}"
        fi
    else
        echo -e "${GREEN}✓ govulncheck already installed${NC}"
    fi
else
    echo -e "${RED}✗ Cannot install Go tools - Go is not available${NC}"
    INSTALL_ERRORS=1
fi

# 4. Verify PATH setup
echo -e "\n${BLUE}4. Verifying PATH setup...${NC}"
if command_exists go; then
    GO_BIN_PATH="$(go env GOPATH)/bin"
    if [[ ":$PATH:" != *":$GO_BIN_PATH:"* ]]; then
        echo -e "${YELLOW}⚠️  $(go env GOPATH)/bin is not in PATH${NC}"
        echo "Add this to your shell profile (.bashrc, .zshrc, etc.):"
        echo "export PATH=\"\$PATH:\$(go env GOPATH)/bin\""
        
        # Try to add to common shell profiles
        for shell_profile in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
            if [[ -f "$shell_profile" ]] && ! grep -q 'export PATH.*go.*bin' "$shell_profile"; then
                echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> "$shell_profile"
                echo -e "${GREEN}✓ Added Go bin directory to $shell_profile${NC}"
                break
            fi
        done
    else
        echo -e "${GREEN}✓ Go bin directory is in PATH${NC}"
    fi
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
echo -e "\n============================================================="
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
    echo -e "${BLUE}💡 Next steps:${NC}"
    echo "1. Restart your terminal or run: source ~/.bashrc"
    echo "2. Run: make build"
    echo "3. Run: make test"
    echo "4. Start developing!"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some dependencies failed to install.${NC}"
    echo "Please check the error messages above and install missing dependencies manually."
    exit 1
fi