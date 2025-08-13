#!/bin/bash
#
# Protobuf Installation Diagnostic Script
# Comprehensive diagnostics for protobuf setup issues
#

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
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

log_debug() {
    echo -e "${PURPLE}[DEBUG]${NC} $1"
}

# Main diagnostic function
main() {
    echo -e "${BLUE}🔍 Protobuf Installation Diagnostic Tool${NC}"
    echo "========================================"
    echo ""

    log_info "🔧 Environment Analysis"
    echo "Date: $(date)"
    echo "User: $(whoami)"
    echo "PWD: $(pwd)"
    echo "HOME: $HOME"
    echo ""

    # Check Go installation
    log_info "🚀 Go Installation Check"
    if command -v go >/dev/null 2>&1; then
        log_success "Go is installed"
        echo "  Version: $(go version)"
        echo "  GOPATH: $(go env GOPATH)"
        echo "  GOBIN: $(go env GOBIN)"
        echo "  GOROOT: $(go env GOROOT)"
        echo "  GOPROXY: $(go env GOPROXY)"
    else
        log_error "Go is not installed or not in PATH"
        echo "  Install Go from: https://golang.org/dl/"
        return 1
    fi
    echo ""

    # Check GOBIN setup
    log_info "📁 GOBIN Configuration Analysis"
    export GOBIN=${GOBIN:-$(go env GOPATH)/bin}
    echo "  GOBIN (effective): $GOBIN"
    
    if [ -d "$GOBIN" ]; then
        log_success "GOBIN directory exists: $GOBIN"
        echo "  Directory size: $(du -sh "$GOBIN" 2>/dev/null | cut -f1 || echo 'Unknown')"
        echo "  Permissions: $(ls -ld "$GOBIN" | awk '{print $1, $3, $4}')"
        echo "  Contents count: $(ls -1 "$GOBIN" 2>/dev/null | wc -l) files"
    else
        log_warn "GOBIN directory does not exist: $GOBIN"
        log_info "Creating GOBIN directory..."
        mkdir -p "$GOBIN" && log_success "GOBIN directory created" || log_error "Failed to create GOBIN directory"
    fi
    echo ""

    # Check PATH configuration  
    log_info "🛤️ PATH Configuration"
    echo "  Current PATH: $PATH"
    if echo "$PATH" | grep -q "$GOBIN"; then
        log_success "GOBIN is in PATH"
    else
        log_warn "GOBIN is NOT in PATH"
        log_info "Add this to your shell profile: export PATH=\"$GOBIN:\$PATH\""
    fi
    echo ""

    # Check protoc installation
    log_info "🔨 Protoc Compiler Check"
    if command -v protoc >/dev/null 2>&1; then
        protoc_path=$(which protoc)
        log_success "protoc found at: $protoc_path"
        echo "  Version: $(protoc --version)"
        echo "  Permissions: $(ls -la "$protoc_path" | awk '{print $1, $3, $4}')"
        
        # Test protoc functionality
        if protoc --version >/dev/null 2>&1; then
            log_success "protoc is functional"
        else
            log_error "protoc is not functional"
        fi
    else
        log_error "protoc not found"
        echo "  Install options:"
        echo "    Ubuntu: sudo apt-get install protobuf-compiler"
        echo "    macOS: brew install protobuf"
        echo "    Windows: choco install protoc"
    fi
    echo ""

    # Check Go protobuf plugins
    log_info "🔌 Go Protobuf Plugins Check"
    
    # Check protoc-gen-go
    if command -v protoc-gen-go >/dev/null 2>&1; then
        protoc_gen_go_path=$(which protoc-gen-go)
        log_success "protoc-gen-go found at: $protoc_gen_go_path"
        echo "  File info: $(ls -la "$protoc_gen_go_path" | awk '{print $1, $5, $6, $7, $8}')"
        if [ -x "$protoc_gen_go_path" ]; then
            log_success "protoc-gen-go is executable"
        else
            log_error "protoc-gen-go is not executable"
        fi
    else
        log_error "protoc-gen-go not found"
        log_info "Install with: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    fi

    # Check protoc-gen-go-grpc
    if command -v protoc-gen-go-grpc >/dev/null 2>&1; then
        protoc_gen_go_grpc_path=$(which protoc-gen-go-grpc)
        log_success "protoc-gen-go-grpc found at: $protoc_gen_go_grpc_path"
        echo "  File info: $(ls -la "$protoc_gen_go_grpc_path" | awk '{print $1, $5, $6, $7, $8}')"
        if [ -x "$protoc_gen_go_grpc_path" ]; then
            log_success "protoc-gen-go-grpc is executable"
        else
            log_error "protoc-gen-go-grpc is not executable"
        fi
    else
        log_error "protoc-gen-go-grpc not found"  
        log_info "Install with: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    fi
    echo ""

    # Network connectivity check
    log_info "🌐 Network Connectivity Check"
    if curl -s --max-time 10 https://proxy.golang.org/ >/dev/null; then
        log_success "Go proxy is accessible"
    else
        log_error "Go proxy is not accessible"
        log_info "Check your internet connection and firewall settings"
    fi
    echo ""

    # Check proto files
    log_info "📄 Proto Files Check"
    proto_dir="examples/proto"
    if [ -d "$proto_dir" ]; then
        log_success "Proto directory exists: $proto_dir"
        echo "  Contents:"
        ls -la "$proto_dir" | while IFS= read -r line; do
            echo "    $line"
        done
        
        # Check specific proto file
        if [ -f "$proto_dir/echo.proto" ]; then
            log_success "echo.proto found"
            echo "  Size: $(stat -c%s "$proto_dir/echo.proto" 2>/dev/null || echo 'Unknown') bytes"
        else
            log_warn "echo.proto not found in $proto_dir"
        fi
    else
        log_error "Proto directory not found: $proto_dir"
    fi
    echo ""

    # Test generation capability
    log_info "🧪 Generation Test"
    if command -v protoc >/dev/null 2>&1 && command -v protoc-gen-go >/dev/null 2>&1; then
        log_info "Attempting test protobuf generation..."
        
        # Create test proto file
        test_dir="/tmp/protobuf-test-$$"
        mkdir -p "$test_dir"
        
        cat > "$test_dir/test.proto" << 'EOF'
syntax = "proto3";
package test;
option go_package = "./test";

message TestMessage {
    string content = 1;
}
EOF
        
        # Try generation
        if cd "$test_dir" && protoc --go_out=. --go_opt=paths=source_relative test.proto 2>/dev/null; then
            log_success "Test protobuf generation successful"
            if [ -f "test.pb.go" ]; then
                log_success "Generated Go file created: test.pb.go"
            fi
        else
            log_error "Test protobuf generation failed"
        fi
        
        # Cleanup
        rm -rf "$test_dir"
        cd - >/dev/null
    else
        log_warn "Cannot perform generation test - missing tools"
    fi
    echo ""

    # Summary and recommendations
    log_info "📋 Summary and Recommendations"
    
    missing_tools=()
    if ! command -v protoc >/dev/null 2>&1; then
        missing_tools+=("protoc")
    fi
    if ! command -v protoc-gen-go >/dev/null 2>&1; then
        missing_tools+=("protoc-gen-go")
    fi
    if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
        missing_tools+=("protoc-gen-go-grpc")
    fi

    if [ ${#missing_tools[@]} -eq 0 ]; then
        log_success "🎉 All protobuf tools are installed and configured!"
        echo ""
        echo "✅ Ready for protobuf development"
        echo "📝 Test generation: protoc --go_out=. your_file.proto"
    else
        log_error "❌ Missing tools: ${missing_tools[*]}"
        echo ""
        echo "🔧 Quick fix commands:"
        if [[ " ${missing_tools[*]} " =~ " protoc " ]]; then
            echo "  sudo apt-get install protobuf-compiler  # Ubuntu"
            echo "  brew install protobuf                   # macOS"
        fi
        if [[ " ${missing_tools[*]} " =~ " protoc-gen-go " ]]; then
            echo "  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
        fi
        if [[ " ${missing_tools[*]} " =~ " protoc-gen-go-grpc " ]]; then
            echo "  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
        fi
        echo ""
        echo "🛤️ Don't forget to add GOBIN to PATH:"
        echo "  export PATH=\"$GOBIN:\$PATH\""
    fi
    echo ""
    echo "🔍 For more help, see: https://protobuf.dev/getting-started/gotutorial/"
}

# Run main function
main "$@"