#!/bin/bash
# Migration script from Makefiles and shell scripts to Bazel

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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
        log_error "Bazel is not installed."
        echo ""
        echo "To install Bazel:"
        echo "1. Visit: https://bazel.build/install"
        echo "2. Or run: curl -fsSL https://github.com/bazelbuild/bazel/releases/download/7.4.1/bazel-7.4.1-installer-linux-x86_64.sh -o bazel-installer.sh && chmod +x bazel-installer.sh && ./bazel-installer.sh --user"
        echo "3. Add ~/bin to your PATH"
        return 1
    fi
    return 0
}

# Show migration plan
show_migration_plan() {
    echo "Ephemos Bazel Migration Plan"
    echo "============================"
    echo ""
    echo "This script will help you migrate from Makefiles and shell scripts to Bazel."
    echo ""
    echo "Migration steps:"
    echo "1. 🔍 Check prerequisites"
    echo "2. 🏗️ Verify Bazel configuration"
    echo "3. 🧪 Test basic build"
    echo "4. 📋 Compare with current system"
    echo "5. 📝 Show usage examples"
    echo ""
}

# Test Bazel build
test_bazel_build() {
    log_info "Testing Bazel build..."
    
    
    # Test main library build
    log_info "Testing library build..."
    if bazel build //pkg/ephemos:ephemos; then
        log_success "Library build works"
    else
        log_error "Library build failed"
        return 1
    fi
    
    # Test binary builds
    log_info "Testing binary builds..."
    if bazel build //cmd/ephemos-cli:ephemos-cli; then
        log_success "CLI build works"
    else
        log_error "CLI build failed"
        return 1
    fi
    
    return 0
}

# Compare with current build system
compare_systems() {
    echo ""
    log_info "Comparing build systems..."
    echo ""
    
    echo "┌─────────────────────┬─────────────────────┬─────────────────────┐"
    echo "│ Task                │ Current (Make)      │ New (Bazel)         │"
    echo "├─────────────────────┼─────────────────────┼─────────────────────┤"
    echo "│ Build everything    │ make build          │ ./bazel.sh build    │"
    echo "│ Generate protobuf   │ make proto          │ ./bazel.sh proto    │"
    echo "│ Run tests           │ make test           │ ./bazel.sh test     │"
    echo "│ Run with coverage   │ make ci-test        │ ./bazel.sh coverage │"
    echo "│ Build examples      │ make examples       │ ./bazel.sh examples │"
    echo "│ Security scan       │ make security-scan  │ ./bazel.sh security │"
    echo "│ Lint code          │ make lint           │ ./bazel.sh lint     │"
    echo "│ Clean builds       │ make clean          │ ./bazel.sh clean    │"
    echo "│ Build demo         │ make demo           │ ./bazel.sh demo     │"
    echo "└─────────────────────┴─────────────────────┴─────────────────────┘"
    echo ""
}

# Show benefits
show_benefits() {
    echo ""
    log_info "Benefits of migrating to Bazel:"
    echo ""
    echo "🚀 Performance:"
    echo "   • Incremental builds (only rebuild what changed)"
    echo "   • Parallel execution"
    echo "   • Build result caching"
    echo "   • Remote execution support"
    echo ""
    echo "🔒 Reliability:"
    echo "   • Hermetic builds (reproducible)"
    echo "   • Explicit dependency management"  
    echo "   • Sandboxed execution"
    echo "   • Better error handling"
    echo ""
    echo "🛠️ Maintainability:"
    echo "   • Declarative build rules"
    echo "   • Language-agnostic"
    echo "   • Centralized configuration"
    echo "   • Tool integration"
    echo ""
    echo "📈 Scalability:"
    echo "   • Handles large codebases"
    echo "   • Monorepo support"
    echo "   • Distributed builds"
    echo "   • Advanced testing features"
    echo ""
}

# Show usage examples
show_usage_examples() {
    echo ""
    log_info "Usage examples:"
    echo ""
    echo "# Build everything"
    echo "./bazel.sh build"
    echo ""
    echo "# Run tests with coverage"
    echo "./bazel.sh coverage"
    echo ""
    echo "# Build only the CLI"
    echo "./bazel.sh build-cli"
    echo ""
    echo "# Run security scans"
    echo "./bazel.sh security"
    echo ""
    echo "# Generate and update BUILD files"
    echo "./bazel.sh gazelle"
    echo ""
    echo "# Run the demo"
    echo "./bazel.sh demo"
    echo ""
}

# Main migration workflow
main() {
    show_migration_plan
    
    # Step 1: Check prerequisites
    log_info "Step 1: Checking prerequisites..."
    if ! check_bazel; then
        log_error "Bazel is required for migration"
        exit 1
    fi
    log_success "Bazel is installed: $(bazel version | head -1)"
    
    # Step 2: Verify configuration
    log_info "Step 2: Verifying Bazel configuration..."
    if [ ! -f "WORKSPACE" ]; then
        log_error "WORKSPACE file not found"
        exit 1
    fi
    if [ ! -f ".bazelrc" ]; then
        log_error ".bazelrc file not found"
        exit 1
    fi
    if [ ! -f "bazel.sh" ]; then
        log_error "bazel.sh wrapper script not found"
        exit 1
    fi
    log_success "Bazel configuration files found"
    
    # Step 3: Test build (optional, might fail due to checksums)
    log_info "Step 3: Testing basic build..."
    if test_bazel_build; then
        log_success "Bazel build system is working!"
    else
        log_warning "Bazel build needs checksum fixes (see docs/build-systems/BAZEL.md)"
        log_info "You can still proceed with migration planning"
    fi
    
    # Step 4: Show comparison
    log_info "Step 4: Comparing build systems..."
    compare_systems
    
    # Step 5: Show benefits and usage
    show_benefits
    show_usage_examples
    
    echo ""
    log_success "Migration analysis complete!"
    echo ""
    echo "Next steps:"
    echo "1. Fix WORKSPACE checksums if needed (see error messages above)"
    echo "2. Test: ./bazel.sh build"
    echo "3. Update CI workflows to use Bazel (see .github/workflows/bazel-ci.yml)"
    echo "4. Gradually replace Makefile usage with Bazel commands"
    echo "5. Remove shell scripts after verifying Bazel equivalents work"
    echo ""
    echo "📖 See docs/build-systems/BAZEL.md for detailed migration guide"
}

# Run main function
main "$@"