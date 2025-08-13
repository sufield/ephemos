#!/bin/bash

# Bazel installation script for Ephemos development
# This script installs Bazel and sets it up for the project

set -e

echo "🔧 Installing Bazel Build System..."
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Bazel configuration
BAZEL_VERSION="7.4.1"
BAZEL_BINARY_URL="https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-linux-x86_64"
BAZEL_INSTALLER_URL="https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-installer-linux-x86_64.sh"

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to get OS information
get_os_info() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "linux"
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macos"
    elif [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
        echo "windows"
    else
        echo "unknown"
    fi
}

# Function to get architecture
get_arch() {
    case $(uname -m) in
        x86_64) echo "x86_64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "unknown" ;;
    esac
}

# Function to install Bazel using package manager
install_bazel_package_manager() {
    local os="$1"
    
    echo -e "${BLUE}Attempting to install Bazel using package manager...${NC}"
    
    case "$os" in
        "linux")
            if command_exists apt-get; then
                echo "Installing Bazel GPG key and repository..."
                if curl -fsSL https://bazel.build/bazel-release.pub.gpg | gpg --dearmor >bazel-archive-keyring.gpg 2>/dev/null; then
                    if sudo mv bazel-archive-keyring.gpg /usr/share/keyrings 2>/dev/null; then
                        echo "deb [arch=amd64 signed-by=/usr/share/keyrings/bazel-archive-keyring.gpg] https://storage.googleapis.com/bazel-apt stable jdk1.8" | sudo tee /etc/apt/sources.list.d/bazel.list >/dev/null
                        if sudo apt-get update >/dev/null 2>&1 && sudo apt-get install -y bazel >/dev/null 2>&1; then
                            echo -e "${GREEN}✓ Bazel installed via apt${NC}"
                            return 0
                        fi
                    fi
                fi
                echo -e "${YELLOW}⚠️  Package manager installation failed, trying binary installation${NC}"
            elif command_exists yum; then
                echo "Installing Bazel repository for YUM..."
                if sudo dnf copr enable -y vbatts/bazel 2>/dev/null || sudo yum-config-manager --add-repo https://copr.fedorainfracloud.org/coprs/vbatts/bazel/repo/epel-7/vbatts-bazel-epel-7.repo 2>/dev/null; then
                    if sudo yum install -y bazel 2>/dev/null; then
                        echo -e "${GREEN}✓ Bazel installed via yum${NC}"
                        return 0
                    fi
                fi
                echo -e "${YELLOW}⚠️  Package manager installation failed, trying binary installation${NC}"
            fi
            ;;
        "macos")
            if command_exists brew; then
                echo "Installing Bazel via Homebrew..."
                if brew install bazel; then
                    echo -e "${GREEN}✓ Bazel installed via Homebrew${NC}"
                    return 0
                fi
            fi
            echo -e "${YELLOW}⚠️  Homebrew installation failed, trying binary installation${NC}"
            ;;
    esac
    
    return 1
}

# Function to install Bazel binary directly
install_bazel_binary() {
    local os="$1"
    local arch="$2"
    
    echo -e "${BLUE}Installing Bazel binary directly...${NC}"
    
    # Create local bin directory
    mkdir -p "$HOME/bin"
    
    case "$os" in
        "linux")
            case "$arch" in
                "x86_64")
                    BINARY_URL="https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-linux-x86_64"
                    ;;
                "arm64")
                    BINARY_URL="https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-linux-arm64"
                    ;;
                *)
                    echo -e "${RED}✗ Unsupported architecture: $arch${NC}"
                    return 1
                    ;;
            esac
            ;;
        "macos")
            case "$arch" in
                "x86_64")
                    BINARY_URL="https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-darwin-x86_64"
                    ;;
                "arm64")
                    BINARY_URL="https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-darwin-arm64"
                    ;;
                *)
                    echo -e "${RED}✗ Unsupported architecture: $arch${NC}"
                    return 1
                    ;;
            esac
            ;;
        *)
            echo -e "${RED}✗ Unsupported OS: $os${NC}"
            return 1
            ;;
    esac
    
    echo "Downloading Bazel ${BAZEL_VERSION} for ${os}-${arch}..."
    if curl -L -o "$HOME/bin/bazel" "$BINARY_URL"; then
        chmod +x "$HOME/bin/bazel"
        echo -e "${GREEN}✓ Bazel binary installed to $HOME/bin/bazel${NC}"
        
        # Add to PATH if not already there
        if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
            echo -e "${YELLOW}Adding $HOME/bin to PATH...${NC}"
            export PATH="$HOME/bin:$PATH"
            
            # Add to shell profile
            for shell_profile in "$HOME/.bashrc" "$HOME/.zshrc" "$HOME/.profile"; do
                if [[ -f "$shell_profile" ]]; then
                    if ! grep -q 'export PATH="$HOME/bin:$PATH"' "$shell_profile"; then
                        echo 'export PATH="$HOME/bin:$PATH"' >> "$shell_profile"
                        echo -e "${GREEN}✓ Added PATH to $shell_profile${NC}"
                    fi
                fi
            done
        fi
        
        return 0
    else
        echo -e "${RED}✗ Failed to download Bazel binary${NC}"
        return 1
    fi
}

# Function to verify Bazel installation
verify_bazel() {
    echo -e "\n${BLUE}Verifying Bazel installation...${NC}"
    
    if command_exists bazel; then
        INSTALLED_VERSION=$(bazel version 2>/dev/null | grep "Build label" | awk '{print $3}' || echo "unknown")
        echo -e "${GREEN}✓ Bazel is installed: $INSTALLED_VERSION${NC}"
        
        if [[ "$INSTALLED_VERSION" == "$BAZEL_VERSION" ]]; then
            echo -e "${GREEN}✓ Version matches expected: $BAZEL_VERSION${NC}"
        else
            echo -e "${YELLOW}⚠️  Installed version ($INSTALLED_VERSION) differs from expected ($BAZEL_VERSION)${NC}"
        fi
        
        return 0
    else
        echo -e "${RED}✗ Bazel not found in PATH${NC}"
        return 1
    fi
}

# Function to set up Bazel for the project
setup_bazel_project() {
    echo -e "\n${BLUE}Setting up Bazel for Ephemos project...${NC}"
    
    # Check if we're in a git repository
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        echo -e "${RED}✗ Not in a git repository${NC}"
        return 1
    fi
    
    # Check for required files
    local missing_files=()
    
    if [[ ! -f "WORKSPACE" ]]; then
        missing_files+=("WORKSPACE")
    fi
    
    if [[ ! -f ".bazelrc" ]]; then
        missing_files+=(".bazelrc")
    fi
    
    if [[ ! -f "bazel.sh" ]]; then
        missing_files+=("bazel.sh")
    fi
    
    if [[ ${#missing_files[@]} -gt 0 ]]; then
        echo -e "${RED}✗ Missing required Bazel files: ${missing_files[*]}${NC}"
        echo "This script should be run from the Ephemos project root directory"
        return 1
    fi
    
    echo -e "${GREEN}✓ Bazel project files found${NC}"
    
    # Make bazel.sh executable
    chmod +x bazel.sh
    echo -e "${GREEN}✓ Made bazel.sh executable${NC}"
    
    # Test basic Bazel functionality
    echo "Testing Bazel workspace..."
    if bazel info workspace >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Bazel workspace is valid${NC}"
    else
        echo -e "${RED}✗ Bazel workspace validation failed${NC}"
        return 1
    fi
    
    return 0
}

# Function to install development tools for Bazel
install_bazel_dev_tools() {
    echo -e "\n${BLUE}Installing Bazel development tools...${NC}"
    
    # Install buildtools (buildifier, buildozer)
    if ! command_exists buildifier; then
        echo "Installing buildifier..."
        if go install github.com/bazelbuild/buildtools/buildifier@latest; then
            echo -e "${GREEN}✓ buildifier installed${NC}"
        else
            echo -e "${YELLOW}⚠️  Failed to install buildifier (optional)${NC}"
        fi
    else
        echo -e "${GREEN}✓ buildifier already installed${NC}"
    fi
    
    # Install gazelle (BUILD file generator)
    if ! command_exists gazelle; then
        echo "Installing gazelle..."
        if go install github.com/bazelbuild/bazel-gazelle/cmd/gazelle@latest; then
            echo -e "${GREEN}✓ gazelle installed${NC}"
        else
            echo -e "${YELLOW}⚠️  Failed to install gazelle (optional)${NC}"
        fi
    else
        echo -e "${GREEN}✓ gazelle already installed${NC}"
    fi
}

# Function to run initial Bazel setup
run_initial_bazel_setup() {
    echo -e "\n${BLUE}Running initial Bazel setup...${NC}"
    
    # Fetch external dependencies
    echo "Fetching external dependencies..."
    if bazel fetch //... >/dev/null 2>&1; then
        echo -e "${GREEN}✓ External dependencies fetched${NC}"
    else
        echo -e "${YELLOW}⚠️  Failed to fetch some dependencies (this is often normal)${NC}"
    fi
    
    # Test build
    echo "Testing build system..."
    if ./bazel.sh info >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Build system test successful${NC}"
    else
        echo -e "${YELLOW}⚠️  Build system test failed (may be dependency issues)${NC}"
    fi
}

# Main installation process
main() {
    echo -e "${BLUE}Starting Bazel installation for Ephemos...${NC}\n"
    
    # Check if Bazel is already installed
    if command_exists bazel; then
        CURRENT_VERSION=$(bazel version 2>/dev/null | grep "Build label" | awk '{print $3}' || echo "unknown")
        echo -e "${GREEN}✓ Bazel is already installed: $CURRENT_VERSION${NC}"
        
        if [[ "$CURRENT_VERSION" == "$BAZEL_VERSION" ]]; then
            echo -e "${GREEN}✓ Version is up to date${NC}"
        else
            echo -e "${YELLOW}⚠️  Installed version differs from expected${NC}"
            echo "Installed: $CURRENT_VERSION, Expected: $BAZEL_VERSION"
            echo "Continuing with current installation..."
        fi
    else
        # Detect system
        OS=$(get_os_info)
        ARCH=$(get_arch)
        
        echo "Detected system: $OS-$ARCH"
        
        # Try package manager first, then binary installation
        if ! install_bazel_package_manager "$OS"; then
            if ! install_bazel_binary "$OS" "$ARCH"; then
                echo -e "${RED}✗ Failed to install Bazel${NC}"
                exit 1
            fi
        fi
    fi
    
    # Verify installation
    if ! verify_bazel; then
        echo -e "${RED}✗ Bazel installation verification failed${NC}"
        exit 1
    fi
    
    # Set up project
    if ! setup_bazel_project; then
        echo -e "${RED}✗ Failed to set up Bazel project${NC}"
        exit 1
    fi
    
    # Install development tools
    install_bazel_dev_tools
    
    # Run initial setup
    run_initial_bazel_setup
    
    # Success message
    echo -e "\n=================================="
    echo -e "${GREEN}🎉 Bazel installation completed successfully!${NC}"
    echo ""
    echo "You can now use Bazel with the following commands:"
    echo "  ./bazel.sh build         # Build all targets"
    echo "  ./bazel.sh test          # Run all tests"
    echo "  ./bazel.sh lint          # Run linting"
    echo "  ./bazel.sh security      # Run security scans"
    echo "  bazel info               # Show Bazel information"
    echo ""
    echo "For more commands, run: ./bazel.sh help"
    echo ""
    echo -e "${BLUE}💡 Next steps:${NC}"
    echo "1. Restart your terminal or run: source ~/.bashrc"
    echo "2. Run: ./bazel.sh build"
    echo "3. Run: ./bazel.sh test"
    echo "4. Start developing with Bazel!"
}

# Run main function
main "$@"