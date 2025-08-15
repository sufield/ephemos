#!/bin/bash

# Complete development environment setup for Ephemos
# This script sets up everything needed for Ephemos development

set -e

echo "🚀 Setting up Ephemos Development Environment"
echo "============================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to print section headers
print_section() {
    echo -e "\n${BLUE}$1${NC}"
    echo "$(printf '=%.0s' {1..50})"
}

# Check prerequisites
print_section "1. Checking Prerequisites"

# Check if we're in the right directory
if [[ ! -f "WORKSPACE" ]] || [[ ! -f ".bazelrc" ]]; then
    echo -e "${RED}✗ This script must be run from the Ephemos project root directory${NC}"
    echo "Please cd to the directory containing WORKSPACE and .bazelrc files"
    exit 1
fi

echo -e "${GREEN}✓ Running from Ephemos project root${NC}"

# Check Git
if ! command_exists git; then
    echo -e "${RED}✗ Git is required but not installed${NC}"
    echo "Please install Git first: https://git-scm.com/downloads"
    exit 1
fi

echo -e "${GREEN}✓ Git is available${NC}"

# Check Docker (for act)
if ! command_exists docker; then
    echo -e "${YELLOW}⚠️  Docker not found - act (local CI testing) will not work${NC}"
    echo "Install Docker for local CI testing: https://docs.docker.com/get-docker/"
else
    echo -e "${GREEN}✓ Docker is available${NC}"
fi

# Install development dependencies
print_section "2. Installing Development Dependencies"

echo "Running dependency installation script..."
if ./scripts/install-deps.sh; then
    echo -e "${GREEN}✓ Development dependencies installed${NC}"
else
    echo -e "${RED}✗ Dependency installation encountered issues${NC}"
    echo "Please check the output above and resolve any issues"
    echo "You may need to run: ./scripts/install-deps.sh --system"
    exit 1
fi

# Verify installations
print_section "3. Verifying Installation"

MISSING_TOOLS=()

# Check Go
if command_exists go; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    echo -e "${GREEN}✓ Go: $GO_VERSION${NC}"
else
    echo -e "${RED}✗ Go not found${NC}"
    MISSING_TOOLS+=("Go")
fi

# Check Bazel
if command_exists bazel; then
    BAZEL_VERSION=$(bazel version 2>/dev/null | grep "Build label" | awk '{print $3}' || echo "unknown")
    echo -e "${GREEN}✓ Bazel: $BAZEL_VERSION${NC}"
else
    echo -e "${RED}✗ Bazel not found${NC}"
    MISSING_TOOLS+=("Bazel")
fi

else
fi

# Check act
if command_exists act; then
    ACT_VERSION=$(act --version 2>/dev/null | head -n1 || echo "unknown")
    echo -e "${GREEN}✓ act: $ACT_VERSION${NC}"
else
    echo -e "${YELLOW}⚠️  act not found${NC}"
fi

# Check Go tools
export PATH="$PATH:$(go env GOPATH)/bin"
else
fi

else
fi

# Test build system
print_section "4. Testing Build System"

if command_exists bazel && [[ -f "bazel.sh" ]]; then
    echo "Testing Bazel build system..."
    if ./bazel.sh info >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Bazel workspace is functional${NC}"
    else
        echo -e "${YELLOW}⚠️  Bazel workspace has issues (this is often normal)${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Bazel not available for testing${NC}"
fi

# Test act setup
if command_exists act && command_exists docker && [[ -f ".actrc" ]]; then
    echo "Testing act setup..."
    if ./act -l >/dev/null 2>&1; then
        WORKFLOW_COUNT=$(./act -l 2>/dev/null | grep -c "^[[:space:]]*[0-9]" || echo "0")
        echo -e "${GREEN}✓ act can list $WORKFLOW_COUNT workflows${NC}"
    else
        echo -e "${YELLOW}⚠️  act setup has issues${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  act not available for testing${NC}"
fi

# Set up Git hooks (optional)
print_section "5. Setting up Git Hooks (Optional)"

if [[ -d ".git" ]]; then
    echo "Setting up pre-commit hook for local CI testing..."
    
    PRE_COMMIT_HOOK=".git/hooks/pre-commit"
    if [[ ! -f "$PRE_COMMIT_HOOK" ]] && command_exists act; then
        cat > "$PRE_COMMIT_HOOK" << 'EOF'
#!/bin/bash
# Pre-commit hook for Ephemos
# Runs basic checks before allowing commits

echo "🔍 Running pre-commit checks..."

# Run linting if available
if command -v bazel >/dev/null 2>&1 && [[ -f "bazel.sh" ]]; then
    echo "Running Bazel lint check..."
    if ! ./bazel.sh lint; then
        echo "❌ Lint check failed. Please fix issues before committing."
        exit 1
    fi
elif command -v make >/dev/null 2>&1 && [[ -f "Makefile" ]]; then
    echo "Running Make lint check..."
    if ! make lint; then
        echo "❌ Lint check failed. Please fix issues before committing."
        exit 1
    fi
fi

echo "✅ Pre-commit checks passed!"
EOF
        chmod +x "$PRE_COMMIT_HOOK"
        echo -e "${GREEN}✓ Pre-commit hook installed${NC}"
    else
        echo -e "${YELLOW}⚠️  Pre-commit hook not installed (already exists or act not available)${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  Not a git repository${NC}"
fi

# Setup IDE configuration
print_section "6. IDE Configuration"

echo "Setting up IDE configurations..."

# VS Code settings
if [[ -d ".vscode" ]] || command_exists code; then
    mkdir -p .vscode
    
    # Create settings.json if it doesn't exist
    if [[ ! -f ".vscode/settings.json" ]]; then
        cat > .vscode/settings.json << 'EOF'
{
    "go.lintTool": "golangci-lint",
    "go.useLanguageServer": true,
    "go.formatTool": "goimports",
    "files.associations": {
        "BUILD": "starlark",
        "BUILD.bazel": "starlark",
        "*.bzl": "starlark",
        "WORKSPACE": "starlark"
    },
    "terminal.integrated.env.linux": {
        "PATH": "${env:PATH}:${env:GOPATH}/bin"
    },
    "terminal.integrated.env.osx": {
        "PATH": "${env:PATH}:${env:GOPATH}/bin"
    }
}
EOF
        echo -e "${GREEN}✓ VS Code settings created${NC}"
    else
        echo -e "${GREEN}✓ VS Code settings already exist${NC}"
    fi
    
    # Create tasks.json for common operations
    if [[ ! -f ".vscode/tasks.json" ]]; then
        cat > .vscode/tasks.json << 'EOF'
{
    "version": "2.0.0",
    "tasks": [
        {
            "label": "Bazel Build",
            "type": "shell",
            "command": "./bazel.sh",
            "args": ["build"],
            "group": "build",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        },
        {
            "label": "Bazel Test",
            "type": "shell", 
            "command": "./bazel.sh",
            "args": ["test"],
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        },
        {
            "label": "Local CI Test",
            "type": "shell",
            "command": "./act",
            "args": ["-j", "test", "--pull=false"],
            "group": "test",
            "presentation": {
                "echo": true,
                "reveal": "always",
                "focus": false,
                "panel": "shared"
            }
        }
    ]
}
EOF
        echo -e "${GREEN}✓ VS Code tasks created${NC}"
    else
        echo -e "${GREEN}✓ VS Code tasks already exist${NC}"
    fi
else
    echo -e "${YELLOW}⚠️  VS Code not detected, skipping IDE setup${NC}"
fi

# Final summary
print_section "7. Setup Complete!"

if [[ ${#MISSING_TOOLS[@]} -eq 0 ]]; then
    echo -e "${GREEN}🎉 Development environment setup completed successfully!${NC}"
    echo ""
    echo -e "${BLUE}Available commands:${NC}"
    if command_exists bazel && [[ -f "bazel.sh" ]]; then
        echo "  ./bazel.sh build         # Build all targets"
        echo "  ./bazel.sh test          # Run tests"
        echo "  ./bazel.sh lint          # Run linting"
        echo "  ./bazel.sh security-all  # Run security scans"
    fi
    if command_exists act; then
        echo "  ./act -l                 # List workflows"
        echo "  ./act -j test            # Test locally"
    fi
    echo ""
    echo -e "${BLUE}Documentation:${NC}"
    echo "  docs/contributors/       # Contributor guides"
    echo "  docs/contributors/local-ci-testing.md  # Local CI testing"
    echo "  docs/contributors/build-system.md      # Bazel build system"
    echo ""
    echo -e "${GREEN}🚀 You're ready to start developing!${NC}"
else
    echo -e "${YELLOW}⚠️  Setup completed with some missing tools:${NC}"
    printf '  %s\n' "${MISSING_TOOLS[@]}"
    echo ""
    echo "To install missing system packages, run:"
    echo "  ./scripts/install-deps.sh --system"
    echo ""
    echo "You can still develop with the available tools."
fi

echo ""
echo -e "${BLUE}💡 Next steps:${NC}"
echo "1. Read: docs/contributors/README.md"
echo "2. Try: ./bazel.sh build"
echo "3. Test: ./bazel.sh test"
if command_exists act; then
    echo "4. Local CI: ./act -j test"
fi
echo "5. Start coding!"