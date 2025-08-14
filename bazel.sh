#!/bin/bash
# Bazel wrapper script to replace Makefile functionality

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️ $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️ $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if Bazel is installed
check_bazel() {
    if ! command -v bazel >/dev/null 2>&1; then
        log_error "Bazel is not installed. Please install Bazel first."
        echo "Visit: https://bazel.build/install"
        exit 1
    fi
}

# Show help
show_help() {
    echo "Ephemos Bazel Build System"
    echo "=========================="
    echo ""
    echo "Usage: $0 <command>"
    echo ""
    echo "Build commands:"
    echo "  build         - Build all targets"
    echo "  build-cli     - Build CLI binary only"
    echo ""
    echo "Test commands:"
    echo "  test          - Run all tests"
    echo "  test-unit     - Run unit tests only"
    echo "  coverage      - Run tests with coverage"
    echo ""
    echo "Quality commands:"
    echo "  lint          - Run linting checks"
    echo "  security      - Run security scans"
    echo "  security-all  - Run all security checks"
    echo "  format        - Format BUILD files"
    echo ""
    echo "Development commands:"
    echo "  setup         - Complete development environment setup"
    echo "  clean         - Clean build artifacts"
    echo "  deps          - Update dependencies"
    echo "  gazelle       - Update BUILD files"
    echo "  check-deps    - Check dependencies"
    echo "  install-deps  - Install development dependencies"
    echo "  install-bazel - Install/update Bazel build system"
    echo ""
    echo "Demo commands:"
    echo "  demo          - Run complete demo"
    echo "  demo-setup    - Setup demo environment"
    echo "  demo-cleanup  - Cleanup demo environment"
    echo ""
    echo "Script commands:"
    echo "  scripts-build - Run build script tests"
    echo "  scripts-security - Run security script tests"
    echo "  scripts-demo  - Run demo script tests"
    echo ""
    echo "Utility commands:"
    echo "  version       - Show version info"
    echo "  info          - Show build info"
    echo "  help          - Show this help"
}

# Main command handler
case "${1:-help}" in
    "build")
        log_info "Building all targets..."
        bazel build //...
        log_success "Build completed!"
        ;;
    
    "build-cli")
        log_info "Building CLI..."
        bazel build //cmd/ephemos-cli:ephemos-cli
        log_success "CLI built!"
        ;;
    
    
    "test")
        log_info "Running all tests..."
        bazel test //...
        log_success "All tests passed!"
        ;;
    
    "test-unit")
        log_info "Running unit tests..."
        bazel test //pkg/ephemos:ephemos_test
        log_success "Unit tests passed!"
        ;;
    
    "coverage")
        log_info "Running tests with coverage..."
        bazel coverage //...
        log_success "Coverage analysis completed!"
        echo "Coverage report: bazel-out/_coverage/_coverage_report.dat"
        ;;
    
    "lint")
        log_info "Running lint checks..."
        bazel test //:lint_check
        log_success "Lint checks passed!"
        ;;
    
    "security")
        log_info "Running security scans..."
        bazel build //...
        bazel test //:security_scan
        log_success "Security scans completed!"
        ;;
    
    "security-all")
        log_info "Running all security checks..."
        bazel run //scripts/security:security_scan_all
        log_success "All security checks completed!"
        ;;
    
    "format")
        log_info "Formatting BUILD files..."
        bazel run //:gazelle-fix
        log_success "BUILD files formatted!"
        ;;
    
    "clean")
        log_info "Cleaning build artifacts..."
        bazel clean
        log_success "Clean completed!"
        ;;
    
    "deps")
        log_info "Updating dependencies..."
        bazel run //:gazelle-update-repos
        log_success "Dependencies updated!"
        ;;
    
    "gazelle")
        log_info "Updating BUILD files..."
        bazel run //:gazelle
        log_success "BUILD files updated!"
        ;;
    
    "check-deps")
        log_info "Checking dependencies..."
        bazel run //scripts:check_deps
        log_success "Dependency check completed!"
        ;;
    
    "install-deps")
        log_info "Installing development dependencies..."
        bazel run //scripts:install_deps
        log_success "Development dependencies installation completed!"
        ;;
    
    "install-bazel")
        log_info "Installing Bazel build system..."
        bazel run //scripts:install_bazel
        log_success "Bazel installation completed!"
        ;;
    
    "setup")
        log_info "Setting up development environment..."
        bazel run //scripts:setup_dev_environment
        log_success "Development environment setup completed!"
        ;;
    
    "demo")
        log_info "Running complete demo..."
        bazel run //scripts/demo:full_demo
        log_success "Demo completed!"
        ;;
    
    "demo-setup")
        log_info "Setting up demo environment..."
        bazel run //scripts/demo:setup_demo
        log_success "Demo setup completed!"
        ;;
    
    "demo-cleanup")
        log_info "Cleaning up demo environment..."
        bazel run //scripts/demo:cleanup
        log_success "Demo cleanup completed!"
        ;;
    
    "examples")
        log_info "Building examples..."
        bazel build //examples/...
        log_success "Examples built!"
        ;;
    
    "scripts-build")
        log_info "Running build script tests..."
        bazel test //scripts:build_tests
        log_success "Build script tests passed!"
        ;;
    
    "scripts-security")
        log_info "Running security script tests..."
        bazel test //scripts/security:security_tests
        log_success "Security script tests passed!"
        ;;
    
    "scripts-demo")
        log_info "Running demo script tests..."
        bazel test //scripts/demo:demo_tests
        log_success "Demo script tests passed!"
        ;;
    
    "version")
        if [ -f "bazel-bin/cmd/ephemos-cli/ephemos-cli_/ephemos-cli" ]; then
            ./bazel-bin/cmd/ephemos-cli/ephemos-cli_/ephemos-cli version
        elif [ -f "bazel-bin/cmd/ephemos-cli/ephemos-cli" ]; then
            ./bazel-bin/cmd/ephemos-cli/ephemos-cli version
        else
            log_warning "CLI not built yet. Run '$0 build-cli' first."
        fi
        ;;
    
    "info")
        log_info "Build information:"
        bazel info
        ;;
    
    "help"|"")
        show_help
        ;;
    
    *)
        log_error "Unknown command: $1"
        echo ""
        show_help
        exit 1
        ;;
esac