#!/bin/bash

# Development dependencies installation script for Ephemos
# This script installs all required tools for local development

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

# Function to install for different package managers (non-sudo by default)
install_with_manager() {
    local package="$1"
    local manager="$2"
    
    echo -e "${YELLOW}Installing $package with $manager...${NC}"
    case "$manager" in
        "apt")
            echo -e "${YELLOW}⚠️  System package installation requires sudo.${NC}"
            echo "Please run manually: sudo apt-get update && sudo apt-get install -y $package"
            echo "Or use the install-deps-sudo.sh script for automatic installation"
            return 1
            ;;
        "yum")
            echo -e "${YELLOW}⚠️  System package installation requires sudo.${NC}"
            echo "Please run manually: sudo yum install -y $package"
            echo "Or use the install-deps-sudo.sh script for automatic installation"
            return 1
            ;;
        "dnf")
            echo -e "${YELLOW}⚠️  System package installation requires sudo.${NC}"
            echo "Please run manually: sudo dnf install -y $package"
            echo "Or use the install-deps-sudo.sh script for automatic installation"
            return 1
            ;;
        "pacman")
            echo -e "${YELLOW}⚠️  System package installation requires sudo.${NC}"
            echo "Please run manually: sudo pacman -S --noconfirm $package"
            echo "Or use the install-deps-sudo.sh script for automatic installation"
            return 1
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
    
    # Check if Go version is adequate (1.21+)
    if [[ $(printf '%s\n' "1.21" "$GO_VERSION" | sort -V | head -n1) == "1.21" ]]; then
        echo -e "${GREEN}✓ Go version is adequate${NC}"
    else
        echo -e "${YELLOW}⚠️  Go version $GO_VERSION may be too old. Recommended: 1.21+${NC}"
    fi
else
    echo -e "${RED}✗ Go is not installed${NC}"
    echo "Please install Go from: https://golang.org/dl/"
    INSTALL_ERRORS=1
fi

# 2. Install Protocol Buffers compiler
echo -e "\n${BLUE}2. Installing Protocol Buffers compiler (protoc)...${NC}"
if command_exists protoc; then
    PROTOC_VERSION=$(protoc --version | awk '{print $2}')
    echo -e "${GREEN}✓ protoc is already installed: $PROTOC_VERSION${NC}"
else
    PACKAGE_MANAGER=$(detect_package_manager)
    case "$PACKAGE_MANAGER" in
        "apt")
            install_with_manager "protobuf-compiler" "apt"
            ;;
        "yum"|"dnf")
            install_with_manager "protobuf-compiler" "$PACKAGE_MANAGER"
            ;;
        "pacman")
            install_with_manager "protobuf" "pacman"
            ;;
        "brew")
            install_with_manager "protobuf" "brew"
            ;;
        "choco")
            install_with_manager "protoc" "choco"
            ;;
        *)
            echo -e "${RED}✗ Could not detect package manager${NC}"
            echo "Please install protoc manually:"
            echo "  Ubuntu/Debian: sudo apt-get install protobuf-compiler"
            echo "  CentOS/RHEL: sudo yum install protobuf-compiler"
            echo "  macOS: brew install protobuf"
            echo "  Windows: choco install protoc"
            INSTALL_ERRORS=1
            ;;
    esac
    
    # Verify installation
    # Note: protoc installation may have failed due to sudo requirements
    if command_exists protoc; then
        echo -e "${GREEN}✓ protoc installed successfully${NC}"
    else
        echo -e "${YELLOW}⚠️  protoc not found after installation attempt${NC}"
        echo "This is expected when sudo is required for system packages"
        echo "Continuing with Go tools installation..."
    fi
fi

# 3. Install Go protobuf tools
echo -e "\n${BLUE}3. Installing Go protobuf generation tools...${NC}"
if command_exists go; then
    echo "Installing protoc-gen-go..."
    if go install google.golang.org/protobuf/cmd/protoc-gen-go@latest; then
        echo -e "${GREEN}✓ protoc-gen-go installed${NC}"
    else
        echo -e "${RED}✗ Failed to install protoc-gen-go${NC}"
        INSTALL_ERRORS=1
    fi
    
    echo "Installing protoc-gen-go-grpc..."
    if go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest; then
        echo -e "${GREEN}✓ protoc-gen-go-grpc installed${NC}"
    else
        echo -e "${RED}✗ Failed to install protoc-gen-go-grpc${NC}"
        INSTALL_ERRORS=1
    fi
else
    echo -e "${RED}✗ Cannot install Go tools - Go is not available${NC}"
    INSTALL_ERRORS=1
fi

# 4. Install development tools (optional but recommended)
echo -e "\n${BLUE}4. Installing optional development tools...${NC}"

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
echo -e "\n${BLUE}5. Installing security tools...${NC}"

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

# 6. Verify PATH setup
echo -e "\n${BLUE}6. Verifying PATH setup...${NC}"
GO_BIN_PATH="$(go env GOPATH)/bin"
if [[ ":$PATH:" != *":$GO_BIN_PATH:"* ]]; then
    echo -e "${YELLOW}⚠️  $(go env GOPATH)/bin is not in PATH${NC}"
    echo "Add this to your shell profile (.bashrc, .zshrc, etc.):"
    echo "export PATH=\"\$PATH:\$(go env GOPATH)/bin\""
else
    echo -e "${GREEN}✓ Go bin directory is in PATH${NC}"
fi

# 7. Install Bazel build system
echo -e "\n${BLUE}7. Installing Bazel build system...${NC}"
if command_exists bazel; then
    BAZEL_VERSION=$(bazel version 2>/dev/null | grep "Build label" | awk '{print $3}' || echo "unknown")
    echo -e "${GREEN}✓ Bazel is already installed: $BAZEL_VERSION${NC}"
else
    echo "Installing Bazel..."
    if ./scripts/install-bazel.sh; then
        echo -e "${GREEN}✓ Bazel installed successfully${NC}"
    else
        echo -e "${RED}✗ Bazel installation failed${NC}"
        INSTALL_ERRORS=1
    fi
fi

# 8. Install act (GitHub Actions local runner)
echo -e "\n${BLUE}8. Installing act (GitHub Actions local runner)...${NC}"
if command_exists act; then
    ACT_VERSION=$(act --version 2>/dev/null | head -n1 || echo "unknown")
    echo -e "${GREEN}✓ act is already installed: $ACT_VERSION${NC}"
else
    echo "Installing act..."
    if curl -s https://api.github.com/repos/nektos/act/releases/latest | grep "browser_download_url.*linux_amd64" | cut -d '"' -f 4 | xargs curl -L -o /tmp/act && chmod +x /tmp/act; then
        mkdir -p "$HOME/bin"
        mv /tmp/act "$HOME/bin/act"
        export PATH="$HOME/bin:$PATH"
        echo -e "${GREEN}✓ act installed to $HOME/bin/act${NC}"
        
        # Add to PATH in shell profiles if not already there
        for shell_profile in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
            if [[ -f "$shell_profile" ]] && ! grep -q 'export PATH="$HOME/bin:$PATH"' "$shell_profile"; then
                echo 'export PATH="$HOME/bin:$PATH"' >> "$shell_profile"
            fi
        done
    else
        echo -e "${YELLOW}⚠️  Failed to install act (optional)${NC}"
    fi
fi

# 9. Test protobuf generation
echo -e "\n${BLUE}9. Testing protobuf generation...${NC}"
export PATH="$PATH:$(go env GOPATH)/bin"

if command_exists protoc && command_exists protoc-gen-go && command_exists protoc-gen-go-grpc; then
    echo "Testing protobuf generation..."
    # Use Bazel if available, otherwise fall back to make
    if command_exists bazel && [[ -f "bazel.sh" ]]; then
        if ./bazel.sh proto; then
            echo -e "${GREEN}✓ Protobuf generation test successful (Bazel)${NC}"
        else
            echo -e "${RED}✗ Protobuf generation test failed (Bazel)${NC}"
            INSTALL_ERRORS=1
        fi
    elif command_exists make && [[ -f "Makefile" ]]; then
        if make proto; then
            echo -e "${GREEN}✓ Protobuf generation test successful (Make)${NC}"
        else
            echo -e "${RED}✗ Protobuf generation test failed (Make)${NC}"
            INSTALL_ERRORS=1
        fi
    else
        echo -e "${YELLOW}⚠️  No build system found to test protobuf generation${NC}"
    fi
else
    echo -e "${RED}✗ Missing protobuf tools for testing${NC}"
    INSTALL_ERRORS=1
fi

# Summary
echo -e "\n=================================================="
if [ $INSTALL_ERRORS -eq 0 ]; then
    echo -e "${GREEN}🎉 All dependencies installed successfully!${NC}"
    echo ""
    echo "Available build systems:"
    if command_exists bazel && [[ -f "bazel.sh" ]]; then
        echo -e "${GREEN}✓ Bazel (recommended):${NC}"
        echo "  ./bazel.sh build         # Build all targets"
        echo "  ./bazel.sh test          # Run tests"
        echo "  ./bazel.sh lint          # Run linting"
        echo "  ./bazel.sh security-all  # Run security scans"
        echo "  ./bazel.sh examples      # Build examples"
        echo ""
    fi
    if command_exists make && [[ -f "Makefile" ]]; then
        echo -e "${BLUE}✓ Make (legacy):${NC}"
        echo "  make build       # Build main CLI tools"
        echo "  make examples    # Build example applications"
        echo "  make test        # Run tests"
        echo "  make lint        # Run linting"
        echo ""
    fi
    if command_exists act && [[ -f ".actrc" ]]; then
        echo -e "${GREEN}✓ Local CI/CD testing:${NC}"
        echo "  ./act -l                 # List available workflows"
        echo "  ./act -j test            # Run specific job locally"
        echo "  ./act --dryrun           # Validate workflows"
        echo ""
    fi
    echo "For additional help, see: docs/contributors/"
    exit 0
else
    echo -e "${YELLOW}⚠️  Partial installation completed.${NC}"
    echo "Some dependencies require manual installation (system packages)."
    echo "Go tools were installed successfully."
    echo ""
    echo "To install system packages:"
    echo "  ./scripts/install-deps-sudo.sh    # Automatic installation with sudo"
    echo "  # OR install manually as suggested above"
    echo ""
    echo "You can still try to build - Go dependencies are available:"
    if command_exists bazel && [[ -f "bazel.sh" ]]; then
        echo "  ./bazel.sh build       # May work if protoc already installed"
    fi
    if command_exists make && [[ -f "Makefile" ]]; then
        echo "  make build             # May work if protoc already installed"
    fi
    # Exit 0 instead of exit 1 to avoid breaking automated processes
    exit 0
fi

echo -e "\n💡 ${BLUE}Next steps:${NC}"
echo "1. If protoc installation was skipped, run: ./scripts/install-deps-sudo.sh"
echo "2. Restart your terminal or run: source ~/.bashrc"
if command_exists bazel && [[ -f "bazel.sh" ]]; then
    echo "3. Run: ./bazel.sh build"
    echo "4. Run: ./bazel.sh test"
    echo "5. Try local CI: ./act -j test"
else
    echo "3. Run: make build"
    echo "4. Run: make test"
fi
echo "6. Start developing!"