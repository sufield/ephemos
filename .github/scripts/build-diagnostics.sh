#!/bin/bash
# Enhanced Build Diagnostics for CI/CD
# Provides fail-fast build validation with comprehensive error reporting

set -euo pipefail

# Source the common diagnostic library
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/ci-diagnostics.sh"

# Build-specific diagnostic functions
validate_build_environment() {
    local job_name="$1"
    local matrix_os="${2:-ubuntu-latest}"
    
    init_diagnostics "$job_name" "build-environment-validation"
    
    log_diagnostic "INFO" "🔧 Validating build environment for $matrix_os"
    
    # Start performance monitoring for environment validation
    local perf_start
    perf_start=$(monitor_step_performance "environment-validation" 300 120)
    
    # Validate basic prerequisites
    validate_step_prerequisites "build-environment" \
        "go" "workspace" "go-mod"
    
    # Validate Go environment specifically for builds
    validate_go_build_environment "$matrix_os"
    
    # Validate project structure
    validate_project_structure
    
    # Check system resources
    monitor_system_resources "build-environment-validation"
    
    complete_step_performance_monitoring "environment-validation" "$perf_start" 300 120
    
    log_diagnostic "SUCCESS" "✅ Build environment validation completed successfully"
}

validate_go_build_environment() {
    local matrix_os="$1"
    
    log_diagnostic "INFO" "🔍 Validating Go build environment"
    
    # Check Go version
    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    log_diagnostic "INFO" "Go version detected: $go_version"
    
    # Validate GOROOT exists and contains standard library
    local goroot
    goroot=$(go env GOROOT)
    if [[ ! -d "$goroot" ]]; then
        fail_with_comprehensive_diagnostics 1 \
            "GOROOT directory does not exist" \
            "Go installation is incomplete or corrupted" \
            "Reinstall Go or fix the GOROOT path" \
            "GOROOT: $goroot"
    fi
    
    # Check for critical standard library packages
    local stdlib_packages=("context" "crypto" "net" "os")
    local missing_stdlib=0
    
    for pkg in "${stdlib_packages[@]}"; do
        if [[ ! -d "$goroot/src/$pkg" ]]; then
            log_diagnostic "ERROR" "Missing standard library package: $pkg"
            missing_stdlib=1
        fi
    done
    
    if [[ $missing_stdlib -eq 1 ]]; then
        fail_with_comprehensive_diagnostics 1 \
            "Go standard library packages missing" \
            "Go installation is incomplete or corrupted on $matrix_os" \
            "Use official setup-go action or reinstall Go with complete standard library" \
            "GOROOT: $goroot, OS: $matrix_os"
    fi
    
    # Test basic Go compilation
    log_diagnostic "DEBUG" "Testing basic Go compilation capability"
    local test_file="diagnostic_test_compilation.go"
    cat > "$test_file" << 'EOF'
package main
import (
    "context"
    "fmt"
    "os"
)
func main() {
    ctx := context.Background()
    fmt.Printf("Go compilation test successful on %s\n", os.Getenv("RUNNER_OS"))
    _ = ctx
}
EOF
    
    if execute_with_diagnostics "go-compilation-test" "Testing basic Go compilation" \
        go build -o diagnostic_test "$test_file"; then
        rm -f diagnostic_test diagnostic_test.exe "$test_file"
        log_diagnostic "SUCCESS" "✅ Go compilation test passed"
    else
        rm -f "$test_file"
        fail_with_comprehensive_diagnostics 1 \
            "Basic Go compilation failed" \
            "Go toolchain or standard library issues on $matrix_os" \
            "Check Go installation, GOROOT, and standard library completeness" \
            "This indicates fundamental Go environment problems"
    fi
    
    # Validate Go module functionality
    log_diagnostic "DEBUG" "Validating Go module functionality"
    if ! execute_with_diagnostics "go-mod-verify" "Verifying Go module integrity" \
        go mod verify; then
        fail_with_comprehensive_diagnostics 1 \
            "Go module verification failed" \
            "Module checksums don't match or modules are corrupted" \
            "Run 'go mod tidy' or check module proxy connectivity" \
            "This may indicate dependency corruption or network issues"
    fi
    
    log_diagnostic "SUCCESS" "✅ Go build environment validated successfully"
}

validate_project_structure() {
    log_diagnostic "INFO" "🏗️ Validating project structure"
    
    # Check critical directories
    local required_dirs=("pkg/ephemos" "internal" "examples")
    local missing_dirs=0
    
    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            log_diagnostic "ERROR" "Missing required directory: $dir"
            missing_dirs=1
        else
            log_diagnostic "DEBUG" "Found required directory: $dir"
        fi
    done
    
    if [[ $missing_dirs -eq 1 ]]; then
        fail_with_comprehensive_diagnostics 1 \
            "Required project directories missing" \
            "Repository checkout is incomplete or corrupted" \
            "Check repository integrity and ensure full checkout" \
            "This may indicate git checkout issues or repository corruption"
    fi
    
    # Check critical Go files
    local required_files=("pkg/ephemos/ephemos.go" "go.mod" "go.sum")
    local missing_files=0
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            log_diagnostic "ERROR" "Missing required file: $file"
            missing_files=1
        else
            log_diagnostic "DEBUG" "Found required file: $file"
        fi
    done
    
    if [[ $missing_files -eq 1 ]]; then
        fail_with_comprehensive_diagnostics 1 \
            "Required project files missing" \
            "Repository checkout is incomplete or files were not committed" \
            "Ensure all required files are committed and repository is fully checked out" \
            "This indicates incomplete repository state"
    fi
    
    log_diagnostic "SUCCESS" "✅ Project structure validated successfully"
}

perform_enhanced_build() {
    local build_type="${1:-standard}"
    local target_os="${2:-current}"
    
    log_diagnostic "INFO" "🏗️ Starting enhanced build process: $build_type ($target_os)"
    
    # Start performance monitoring for the build
    local perf_start
    perf_start=$(monitor_step_performance "enhanced-build" 600 300)
    
    # Pre-build validation
    pre_build_validation "$build_type"
    
    # Clean build environment
    clean_build_environment
    
    # Execute the build with diagnostics
    case "$build_type" in
        "standard")
            perform_standard_build
            ;;
        "examples")
            perform_examples_build
            ;;
        "cross-platform")
            perform_cross_platform_build "$target_os"
            ;;
        "release")
            perform_release_build
            ;;
        *)
            fail_with_comprehensive_diagnostics 1 \
                "Unknown build type: $build_type" \
                "Invalid build type specified" \
                "Use one of: standard, examples, cross-platform, release" \
                "Available build types defined in build-diagnostics.sh"
            ;;
    esac
    
    # Post-build validation
    post_build_validation "$build_type"
    
    complete_step_performance_monitoring "enhanced-build" "$perf_start" 600 300
    
    log_diagnostic "SUCCESS" "✅ Enhanced build completed successfully: $build_type"
}

pre_build_validation() {
    local build_type="$1"
    
    log_diagnostic "INFO" "🔍 Pre-build validation for: $build_type"
    
    # Ensure protobuf files are present for builds that need them
    if [[ "$build_type" != "proto-only" ]]; then
        validate_step_prerequisites "pre-build" "proto-files"
    fi
    
    # Check Go module consistency
    if ! execute_with_diagnostics "go-mod-tidy-check" "Checking Go module consistency" \
        go mod tidy -diff; then
        log_diagnostic "WARN" "Go modules are not tidy - this may cause build issues"
        
        # Auto-fix module issues
        log_diagnostic "INFO" "Auto-fixing Go module issues"
        execute_with_diagnostics "go-mod-tidy-fix" "Fixing Go module issues" \
            go mod tidy
    fi
    
    # Validate no undefined references before building
    log_diagnostic "DEBUG" "Pre-validating package compilation"
    if ! execute_with_diagnostics "pre-compile-check" "Pre-compilation validation" \
        go build -o /dev/null ./pkg/ephemos; then
        fail_with_comprehensive_diagnostics 1 \
            "Pre-build compilation check failed" \
            "Package contains compilation errors or missing dependencies" \
            "Fix compilation errors in pkg/ephemos before proceeding" \
            "This prevents wasting time on a build that will definitely fail"
    fi
    
    log_diagnostic "SUCCESS" "✅ Pre-build validation completed"
}

clean_build_environment() {
    log_diagnostic "INFO" "🧹 Cleaning build environment"
    
    # Clean Go caches to ensure fresh build
    execute_with_diagnostics "clean-go-cache" "Cleaning Go build cache" \
        go clean -cache
    
    execute_with_diagnostics "clean-go-testcache" "Cleaning Go test cache" \
        go clean -testcache
    
    execute_with_diagnostics "clean-go-modcache" "Cleaning Go module cache" \
        go clean -modcache
    
    # Remove old binaries
    if [[ -d "bin" ]]; then
        log_diagnostic "DEBUG" "Removing old binaries from bin/ directory"
        rm -rf bin/*
    fi
    
    log_diagnostic "SUCCESS" "✅ Build environment cleaned"
}

perform_standard_build() {
    log_diagnostic "INFO" "🔨 Performing standard build"
    
    # Build main package
    execute_with_diagnostics "build-main-pkg" "Building main Ephemos package" \
        go build -v ./pkg/ephemos
    
    # Build CLI
    execute_with_diagnostics "build-ephemos-cli" "Building Ephemos CLI" \
        go build -v -o bin/ephemos ./cmd/ephemos-cli
    
    # Build config validator
    execute_with_diagnostics "build-config-validator" "Building config validator" \
        go build -v -o bin/config-validator ./cmd/config-validator
    
    # Validate built binaries
    validate_build_artifacts "standard-build" \
        "bin/ephemos" "bin/config-validator"
    
    log_diagnostic "SUCCESS" "✅ Standard build completed"
}

perform_examples_build() {
    log_diagnostic "INFO" "🔨 Building examples"
    
    # Build echo server example
    execute_with_diagnostics "build-echo-server" "Building echo server example" \
        go build -v -o bin/echo-server ./examples/echo-server
    
    # Build echo client example
    execute_with_diagnostics "build-echo-client" "Building echo client example" \
        go build -v -o bin/echo-client ./examples/echo-client
    
    # Validate example binaries
    validate_build_artifacts "examples-build" \
        "bin/echo-server" "bin/echo-client"
    
    log_diagnostic "SUCCESS" "✅ Examples build completed"
}

perform_cross_platform_build() {
    local target_os="$1"
    
    log_diagnostic "INFO" "🔨 Performing cross-platform build for: $target_os"
    
    local platforms
    case "$target_os" in
        "all")
            platforms=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")
            ;;
        "linux")
            platforms=("linux/amd64" "linux/arm64")
            ;;
        "darwin"|"macos")
            platforms=("darwin/amd64" "darwin/arm64")
            ;;
        "windows")
            platforms=("windows/amd64")
            ;;
        *)
            platforms=("$target_os")
            ;;
    esac
    
    mkdir -p dist
    
    for platform in "${platforms[@]}"; do
        local os="${platform%/*}"
        local arch="${platform#*/}"
        local ext=""
        
        if [[ "$os" == "windows" ]]; then
            ext=".exe"
        fi
        
        log_diagnostic "INFO" "Building for $os/$arch"
        
        # Build main CLI
        GOOS="$os" GOARCH="$arch" \
            execute_with_diagnostics "cross-build-cli-$os-$arch" \
            "Cross-compiling CLI for $os/$arch" \
            go build -v -o "dist/ephemos-$os-$arch$ext" ./cmd/ephemos-cli
        
        # Build examples
        GOOS="$os" GOARCH="$arch" \
            execute_with_diagnostics "cross-build-echo-server-$os-$arch" \
            "Cross-compiling echo-server for $os/$arch" \
            go build -v -o "dist/echo-server-$os-$arch$ext" ./examples/echo-server
    done
    
    log_diagnostic "SUCCESS" "✅ Cross-platform build completed for: $target_os"
}

perform_release_build() {
    log_diagnostic "INFO" "🔨 Performing release build with optimizations"
    
    # Build with release flags
    local build_flags=("-trimpath" "-ldflags" "-s -w")
    local version="${GITHUB_REF#refs/tags/}"
    
    if [[ -n "$version" && "$version" != "${GITHUB_REF}" ]]; then
        build_flags+=("-ldflags" "-X main.Version=$version")
        log_diagnostic "INFO" "Building release version: $version"
    fi
    
    # Build optimized binaries
    execute_with_diagnostics "release-build-cli" "Building optimized CLI binary" \
        go build "${build_flags[@]}" -v -o bin/ephemos ./cmd/ephemos-cli
    
    execute_with_diagnostics "release-build-config-validator" "Building optimized config validator" \
        go build "${build_flags[@]}" -v -o bin/config-validator ./cmd/config-validator
    
    execute_with_diagnostics "release-build-examples" "Building optimized examples" \
        go build "${build_flags[@]}" -v -o bin/echo-server ./examples/echo-server
    
    execute_with_diagnostics "release-build-examples" "Building optimized examples" \
        go build "${build_flags[@]}" -v -o bin/echo-client ./examples/echo-client
    
    # Validate release artifacts
    validate_build_artifacts "release-build" \
        "bin/ephemos" "bin/config-validator" "bin/echo-server" "bin/echo-client"
    
    log_diagnostic "SUCCESS" "✅ Release build completed"
}

post_build_validation() {
    local build_type="$1"
    
    log_diagnostic "INFO" "🔍 Post-build validation for: $build_type"
    
    # Test binary execution
    test_binary_execution
    
    # Check binary sizes (detect bloated binaries)
    validate_binary_sizes
    
    # Verify no debug symbols in release builds
    if [[ "$build_type" == "release" ]]; then
        validate_release_binary_properties
    fi
    
    log_diagnostic "SUCCESS" "✅ Post-build validation completed"
}

test_binary_execution() {
    log_diagnostic "DEBUG" "Testing binary execution"
    
    local binaries=("bin/ephemos" "bin/config-validator")
    
    for binary in "${binaries[@]}"; do
        if [[ -f "$binary" ]]; then
            # Test help output to ensure binary is functional
            if [[ "$RUNNER_OS" == "Windows" ]]; then
                binary="${binary}.exe"
            fi
            
            execute_with_diagnostics "test-${binary##*/}" \
                "Testing $binary execution" \
                timeout 10 "$binary" --help
        fi
    done
    
    log_diagnostic "SUCCESS" "✅ Binary execution tests passed"
}

validate_binary_sizes() {
    log_diagnostic "DEBUG" "Validating binary sizes"
    
    local max_size_mb=50  # Maximum reasonable size for Go binaries
    
    for binary in bin/*; do
        if [[ -f "$binary" && -x "$binary" ]]; then
            local size_bytes
            size_bytes=$(stat -c%s "$binary" 2>/dev/null || stat -f%z "$binary" 2>/dev/null || echo "0")
            local size_mb=$((size_bytes / 1024 / 1024))
            
            if [[ $size_mb -gt $max_size_mb ]]; then
                log_diagnostic "WARN" "Large binary detected: $binary (${size_mb}MB)"
                log_diagnostic "WARN" "Consider build optimizations or check for embedded resources"
            else
                log_diagnostic "DEBUG" "Binary size OK: $binary (${size_mb}MB)"
            fi
        fi
    done
    
    log_diagnostic "SUCCESS" "✅ Binary size validation completed"
}

validate_release_binary_properties() {
    log_diagnostic "DEBUG" "Validating release binary properties"
    
    # Check that release binaries are stripped
    for binary in bin/*; do
        if [[ -f "$binary" && -x "$binary" ]]; then
            if command -v file >/dev/null 2>&1; then
                local file_info
                file_info=$(file "$binary")
                
                if echo "$file_info" | grep -q "not stripped"; then
                    log_diagnostic "WARN" "Release binary not stripped: $binary"
                else
                    log_diagnostic "DEBUG" "Release binary properly stripped: $binary"
                fi
            fi
        fi
    done
    
    log_diagnostic "SUCCESS" "✅ Release binary properties validated"
}

# Export build-specific functions
export -f validate_build_environment
export -f perform_enhanced_build
export -f validate_go_build_environment
export -f validate_project_structure

log_diagnostic "INFO" "🔧 Build Diagnostics Library loaded successfully"