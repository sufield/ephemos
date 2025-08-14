#!/bin/bash
#
# Bazel Syntax Validation Script
# This script validates all BUILD.bazel and .bzl files for syntax errors
# and formatting issues before commits.
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check if tools are installed
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v bazel >/dev/null 2>&1; then
        log_error "Bazel is not installed. Please install Bazelisk or Bazel."
        log_info "Install Bazelisk: curl -LO https://github.com/bazelbuild/bazelisk/releases/latest/download/bazelisk-linux-amd64"
        log_info "chmod +x bazelisk-linux-amd64 && sudo mv bazelisk-linux-amd64 /usr/local/bin/bazel"
        exit 1
    fi
    
    if ! command -v buildifier >/dev/null 2>&1; then
        log_error "Buildifier is not installed. Please install it for syntax checking."
        log_info "Install: curl -LO https://github.com/bazelbuild/buildtools/releases/latest/download/buildifier-linux-amd64"
        log_info "chmod +x buildifier-linux-amd64 && sudo mv buildifier-linux-amd64 /usr/local/bin/buildifier"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Find all Bazel build files
find_bazel_files() {
    find "$REPO_ROOT" -name "BUILD.bazel" -o -name "*.bzl" | sort
}

# Validate syntax with buildifier
validate_syntax_with_buildifier() {
    log_info "Running Buildifier syntax and lint checks..."
    
    local files
    files=$(find_bazel_files)
    local file_count
    file_count=$(echo "$files" | wc -l)
    
    log_info "Found $file_count Bazel files to validate"
    
    local errors=0
    local warnings=0
    
    while IFS= read -r file; do
        if [[ -n "$file" ]]; then
            log_info "Checking: $(basename "$file")"
            
            # Check formatting and syntax
            if ! buildifier --mode=check --lint=warn "$file" 2>/dev/null; then
                log_warn "$(basename "$file") has formatting or lint issues"
                # Show the actual issues
                buildifier --mode=check --lint=warn "$file" 2>&1 | while IFS= read -r line; do
                    if [[ "$line" =~ "# reformat" ]]; then
                        log_warn "  Formatting issue: needs reformatting"
                        warnings=$((warnings + 1))
                    elif [[ "$line" =~ "error:" ]]; then
                        log_error "  Syntax error: $line"
                        errors=$((errors + 1))
                    elif [[ "$line" =~ "warning:" || "$line" =~ "Loaded symbol" ]]; then
                        log_warn "  Lint warning: $line"
                        warnings=$((warnings + 1))
                    fi
                done || true
            else
                log_success "  ✅ $(basename "$file") passed"
            fi
        fi
    done <<< "$files"
    
    return $((errors > 0 ? 1 : 0))
}

# Validate workspace parsing with Bazel
validate_bazel_workspace() {
    log_info "Validating Bazel workspace parsing..."
    
    cd "$REPO_ROOT"
    
    # Test basic workspace loading
    if bazel query --ui_event_filters=-info --noshow_progress '//...' >/dev/null 2>&1; then
        log_success "✅ Bazel workspace parsing successful"
        return 0
    else
        log_error "❌ Bazel workspace parsing failed"
        log_info "Running with verbose output to show errors:"
        bazel query '//...' 2>&1 | head -20
        return 1
    fi
}

# Validate build targets (without actually building)
validate_build_targets() {
    log_info "Validating build target definitions..."
    
    cd "$REPO_ROOT"
    
    # Run analysis phase only (no building)
    if bazel build --nobuild --ui_event_filters=-info --noshow_progress '//...' >/dev/null 2>&1; then
        log_success "✅ Build target validation successful"
        return 0
    else
        log_error "❌ Build target validation failed"
        log_info "Running with verbose output to show errors:"
        bazel build --nobuild '//...' 2>&1 | head -20
        return 1
    fi
}

# Fix formatting issues automatically
fix_formatting() {
    if [[ "${1:-}" == "--fix" ]]; then
        log_info "Fixing formatting issues automatically..."
        
        local files
        files=$(find_bazel_files)
        
        while IFS= read -r file; do
            if [[ -n "$file" ]]; then
                buildifier --lint=fix "$file"
                log_info "Fixed formatting for: $(basename "$file")"
            fi
        done <<< "$files"
        
        log_success "Formatting fixes applied"
        return 0
    fi
    
    return 1
}

# Main function
main() {
    log_info "🔍 Starting Bazel syntax validation"
    echo "Repository: $REPO_ROOT"
    echo "Tools: $(bazel version | head -1), $(buildifier --version | head -1)"
    echo ""
    
    # Check if --fix flag is provided
    if fix_formatting "$@"; then
        return 0
    fi
    
    # Run all validation steps
    local exit_code=0
    
    check_prerequisites || exit_code=1
    
    if [[ $exit_code -eq 0 ]]; then
        validate_syntax_with_buildifier || exit_code=1
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        validate_bazel_workspace || exit_code=1
    fi
    
    if [[ $exit_code -eq 0 ]]; then
        validate_build_targets || exit_code=1
    fi
    
    echo ""
    if [[ $exit_code -eq 0 ]]; then
        log_success "🎉 All Bazel syntax validation checks passed!"
        log_info "Tip: Run with --fix to automatically format files"
    else
        log_error "❌ Bazel syntax validation failed"
        log_info "Fix the errors above and run again"
        log_info "Use 'buildifier --lint=fix file.bazel' to fix individual files"
    fi
    
    return $exit_code
}

# Run main function with all arguments
main "$@"