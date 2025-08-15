#!/bin/bash

# Development dependencies installation script for Ephemos
# This script installs all required tools for HTTP over mTLS library development
# 
# Usage:
#   ./install-deps.sh           # Install Go tools only (no sudo)
#   ./install-deps.sh --system  # Install system packages and Go (requires sudo)

set -e

# Parse command line arguments
INSTALL_SYSTEM=false
for arg in "$@"; do
    case $arg in
        --system)
            INSTALL_SYSTEM=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --system    Install system packages and Go (requires sudo)"
            echo "  --help      Show this help message"
            echo ""
            echo "Without --system flag, only Go development tools will be installed."
            echo "Use --system for fresh system setup or when Go is not installed."
            exit 0
            ;;
        *)
            echo "Unknown option: $arg"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Set title based on mode
if [ "$INSTALL_SYSTEM" = true ]; then
    echo "🔧 Installing Ephemos Development Dependencies (System + Go Tools)..."
    echo "====================================================================="
else
    echo "🔧 Installing Ephemos Development Dependencies (Go Tools Only)..."
    echo "================================================================="
fi

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

# Function to detect package manager
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

# Function to install with package manager (requires sudo for most)
install_with_manager() {
    local packages="$1"
    local manager="$2"
    
    echo -e "${YELLOW}Installing $packages with $manager...${NC}"
    case "$manager" in
        "apt")
            sudo apt-get update && sudo apt-get install -y $packages
            ;;
        "yum")
            sudo yum install -y $packages
            ;;
        "dnf")
            sudo dnf install -y $packages
            ;;
        "pacman")
            sudo pacman -S --noconfirm $packages
            ;;
        "brew")
            brew install $packages
            ;;
        "choco")
            choco install $packages
            ;;
        *)
            echo -e "${RED}Unknown package manager: $manager${NC}"
            return 1
            ;;
    esac
}

# Function to install Go from official source
install_go_official() {
    local GO_VERSION="1.24.1"
    local OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    local ARCH=$(uname -m)
    
    # Map architecture names
    case "$ARCH" in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            return 1
            ;;
    esac
    
    local GO_TARBALL="go${GO_VERSION}.${OS}-${ARCH}.tar.gz"
    local GO_URL="https://go.dev/dl/${GO_TARBALL}"
    
    echo -e "${YELLOW}Downloading Go ${GO_VERSION} from official source...${NC}"
    if curl -LO "$GO_URL"; then
        echo -e "${YELLOW}Installing Go to /usr/local...${NC}"
        sudo rm -rf /usr/local/go
        sudo tar -C /usr/local -xzf "$GO_TARBALL"
        rm "$GO_TARBALL"
        
        # Add to PATH in current session
        export PATH=$PATH:/usr/local/go/bin
        
        echo -e "${GREEN}✓ Go ${GO_VERSION} installed successfully${NC}"
        return 0
    else
        echo -e "${RED}Failed to download Go${NC}"
        return 1
    fi
}

# 1. Check/Install Go
echo -e "\n${BLUE}1. Checking Go installation...${NC}"
if command_exists go; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo -e "${GREEN}✓ Go is installed: $GO_VERSION${NC}"
    
    # Check if Go version is adequate (1.24+)
    REQUIRED_VERSION="1.24"
    if [[ $(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1) == "$REQUIRED_VERSION" ]]; then
        echo -e "${GREEN}✓ Go version is adequate${NC}"
    else
        echo -e "${YELLOW}⚠️  Go version $GO_VERSION may be too old. Recommended: ${REQUIRED_VERSION}+${NC}"
        if [ "$INSTALL_SYSTEM" = true ]; then
            echo "Attempting to upgrade Go..."
            install_go_official
        else
            echo "Run with --system flag to upgrade Go"
        fi
    fi
else
    echo -e "${RED}✗ Go is not installed${NC}"
    if [ "$INSTALL_SYSTEM" = true ]; then
        echo "Installing Go..."
        PACKAGE_MANAGER=$(detect_package_manager)
        
        # Try package manager first
        case "$PACKAGE_MANAGER" in
            "apt")
                # Ubuntu/Debian often have older Go versions, use official installer
                install_go_official
                ;;
            "yum"|"dnf")
                if install_with_manager "golang" "$PACKAGE_MANAGER"; then
                    echo -e "${GREEN}✓ Go installed via $PACKAGE_MANAGER${NC}"
                else
                    install_go_official
                fi
                ;;
            "pacman")
                install_with_manager "go" "pacman"
                ;;
            "brew")
                install_with_manager "go" "brew"
                ;;
            *)
                echo -e "${YELLOW}Package manager not detected, using official installer${NC}"
                install_go_official
                ;;
        esac
    else
        echo -e "${RED}Please install Go from: https://golang.org/dl/${NC}"
        echo "Or run this script with --system flag to install automatically"
        INSTALL_ERRORS=1
    fi
fi

# 2. Install system development tools (if requested)
if [ "$INSTALL_SYSTEM" = true ]; then
    echo -e "\n${BLUE}2. Installing system development tools...${NC}"
    PACKAGE_MANAGER=$(detect_package_manager)
    case "$PACKAGE_MANAGER" in
        "apt")
            install_with_manager "build-essential git curl make" "apt"
            ;;
        "yum"|"dnf")
            install_with_manager "gcc make git curl" "$PACKAGE_MANAGER"
            ;;
        "pacman")
            install_with_manager "base-devel git curl make" "pacman"
            ;;
        "brew")
            # Xcode command line tools provide build essentials on macOS
            if ! xcode-select -p &>/dev/null; then
                echo "Installing Xcode command line tools..."
                xcode-select --install 2>/dev/null || true
            fi
            install_with_manager "git curl make" "brew"
            ;;
        *)
            echo -e "${YELLOW}⚠️  Unknown package manager, skipping system tools${NC}"
            ;;
    esac
else
    echo -e "\n${BLUE}2. Skipping system tools (use --system to install)${NC}"
fi

# 3. Install Go development tools
echo -e "\n${BLUE}3. Installing Go development tools...${NC}"
if command_exists go; then
    # Ensure GOPATH/bin exists
    GOPATH=$(go env GOPATH)
    mkdir -p "$GOPATH/bin"
    
    # golangci-lint (using official installer for latest version)
    if ! command_exists golangci-lint; then
        echo "Installing golangci-lint..."
        if curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$GOPATH/bin" v1.55.2; then
            echo -e "${GREEN}✓ golangci-lint installed${NC}"
        else
            echo -e "${YELLOW}⚠️  Failed to install golangci-lint (optional)${NC}"
        fi
    else
        echo -e "${GREEN}✓ golangci-lint already installed${NC}"
    fi
    
    # gosec - Security checker
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
    
    # govulncheck - Vulnerability checker
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
    
    # gofumpt - Stricter formatter
    if ! command_exists gofumpt; then
        echo "Installing gofumpt..."
        if go install mvdan.cc/gofumpt@latest; then
            echo -e "${GREEN}✓ gofumpt installed${NC}"
        else
            echo -e "${YELLOW}⚠️  Failed to install gofumpt (optional)${NC}"
        fi
    else
        echo -e "${GREEN}✓ gofumpt already installed${NC}"
    fi
    
    # staticcheck - Advanced static analysis
    if ! command_exists staticcheck; then
        echo "Installing staticcheck..."
        if go install honnef.co/go/tools/cmd/staticcheck@latest; then
            echo -e "${GREEN}✓ staticcheck installed${NC}"
        else
            echo -e "${YELLOW}⚠️  Failed to install staticcheck (optional)${NC}"
        fi
    else
        echo -e "${GREEN}✓ staticcheck already installed${NC}"
    fi
else
    echo -e "${RED}✗ Cannot install Go tools - Go is not available${NC}"
    INSTALL_ERRORS=1
fi

# 4. Verify and setup PATH
echo -e "\n${BLUE}4. Verifying PATH setup...${NC}"
if command_exists go; then
    GO_BIN_PATH="$(go env GOPATH)/bin"
    GO_ROOT_BIN="/usr/local/go/bin"
    
    # Check GOPATH/bin
    if [[ ":$PATH:" != *":$GO_BIN_PATH:"* ]]; then
        echo -e "${YELLOW}⚠️  $GO_BIN_PATH is not in PATH${NC}"
        
        # Add to current session
        export PATH="$PATH:$GO_BIN_PATH"
        
        # Try to add to shell profiles
        PATH_EXPORT="export PATH=\"\$PATH:$GO_BIN_PATH\""
        UPDATED_PROFILE=false
        
        for shell_profile in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
            if [[ -f "$shell_profile" ]]; then
                if ! grep -q "$GO_BIN_PATH" "$shell_profile" 2>/dev/null; then
                    echo "$PATH_EXPORT" >> "$shell_profile"
                    echo -e "${GREEN}✓ Added Go bin directory to $shell_profile${NC}"
                    UPDATED_PROFILE=true
                    break
                fi
            fi
        done
        
        if [ "$UPDATED_PROFILE" = false ]; then
            echo "Add this to your shell profile (.bashrc, .zshrc, etc.):"
            echo "$PATH_EXPORT"
        fi
    else
        echo -e "${GREEN}✓ Go bin directory is in PATH${NC}"
    fi
    
    # Check /usr/local/go/bin (if using official installer)
    if [[ -d "$GO_ROOT_BIN" ]] && [[ ":$PATH:" != *":$GO_ROOT_BIN:"* ]]; then
        echo -e "${YELLOW}⚠️  $GO_ROOT_BIN is not in PATH${NC}"
        export PATH="$PATH:$GO_ROOT_BIN"
        
        PATH_EXPORT="export PATH=\"\$PATH:/usr/local/go/bin\""
        for shell_profile in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
            if [[ -f "$shell_profile" ]]; then
                if ! grep -q "/usr/local/go/bin" "$shell_profile" 2>/dev/null; then
                    echo "$PATH_EXPORT" >> "$shell_profile"
                    echo -e "${GREEN}✓ Added Go root to $shell_profile${NC}"
                    break
                fi
            fi
        done
    fi
fi

# 5. Test build
echo -e "\n${BLUE}5. Testing build...${NC}"
if command_exists make && [[ -f "Makefile" ]]; then
    echo "Running build test..."
    if make build; then
        echo -e "${GREEN}✓ Build test successful${NC}"
    else
        echo -e "${RED}✗ Build test failed${NC}"
        echo "This might be due to missing dependencies. Checking..."
        if command_exists go; then
            echo "Running go mod download..."
            go mod download
            # Retry build
            if make build; then
                echo -e "${GREEN}✓ Build test successful after downloading dependencies${NC}"
            else
                INSTALL_ERRORS=1
            fi
        else
            INSTALL_ERRORS=1
        fi
    fi
else
    echo -e "${YELLOW}⚠️  Make not found or no Makefile present${NC}"
    if [[ -f "go.mod" ]]; then
        echo "Testing with go build instead..."
        if go build ./...; then
            echo -e "${GREEN}✓ Go build successful${NC}"
        else
            echo -e "${RED}✗ Go build failed${NC}"
            INSTALL_ERRORS=1
        fi
    fi
fi

# Summary
echo ""
echo "================================================================="
if [ $INSTALL_ERRORS -eq 0 ]; then
    echo -e "${GREEN}🎉 All dependencies installed successfully!${NC}"
    echo ""
    echo -e "${GREEN}Installed tools:${NC}"
    command_exists go && echo "  ✓ go $(go version | awk '{print $3}')"
    command_exists golangci-lint && echo "  ✓ golangci-lint $(golangci-lint --version 2>/dev/null | head -1)"
    command_exists gosec && echo "  ✓ gosec"
    command_exists govulncheck && echo "  ✓ govulncheck"
    command_exists gofumpt && echo "  ✓ gofumpt"
    command_exists staticcheck && echo "  ✓ staticcheck"
    echo ""
    echo -e "${GREEN}Available commands:${NC}"
    echo "  make build       # Build CLI tools"
    echo "  make test        # Run tests"
    echo "  make lint        # Run linting"
    echo "  make fmt         # Format code"
    echo "  make clean       # Clean build artifacts"
    echo ""
    if [ "$INSTALL_SYSTEM" = false ]; then
        echo -e "${BLUE}💡 Tip:${NC} Run with --system flag to install system dependencies"
    fi
    echo ""
    echo -e "${BLUE}Next steps:${NC}"
    echo "1. Restart your terminal or run: source ~/.bashrc"
    echo "2. Run: make test"
    echo "3. Start developing!"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some dependencies failed to install${NC}"
    echo ""
    if [ "$INSTALL_SYSTEM" = false ]; then
        echo "Try running with --system flag for complete setup:"
        echo "  $0 --system"
    else
        echo "Please check the error messages above and install missing dependencies manually."
    fi
    echo ""
    echo "You can still work with available tools:"
    command_exists go && echo "  ✓ go $(go version | awk '{print $3}')"
    command_exists make && echo "  ✓ make"
    exit 1
fi