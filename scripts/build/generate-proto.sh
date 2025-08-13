#!/bin/bash
# Secure protobuf generation script for Ephemos with verbose diagnostics
# Usage: ./generate-proto.sh <PROTO_DIR> <GO_OUT>
# Environment: EPHEMOS_VERBOSE=1 for detailed diagnostics

set -euo pipefail  # Exit on error, undefined vars, pipe failures

# Enable verbose mode if requested
VERBOSE=${EPHEMOS_VERBOSE:-0}
readonly SCRIPT_NAME="$(basename "$0")"

# Diagnostic logging function with line numbers
log_diagnostic() {
    local level="$1"
    local message="$2"
    local line_number="${BASH_LINENO[1]:-unknown}"
    local timestamp="$(date '+%Y-%m-%d %H:%M:%S')"
    
    # In non-verbose mode, only show essential messages
    if [[ $VERBOSE -eq 0 ]] && [[ "$level" == "INFO" || "$level" == "DEBUG" ]]; then
        return 0
    fi
    
    case "$level" in
        "INFO")  echo "[$timestamp] [$SCRIPT_NAME:$line_number] INFO:  $message" ;;
        "WARN")  echo "[$timestamp] [$SCRIPT_NAME:$line_number] WARN:  $message" >&2 ;;
        "ERROR") echo "[$timestamp] [$SCRIPT_NAME:$line_number] ERROR: $message" >&2 ;;
        "DEBUG") [[ $VERBOSE -eq 1 ]] && echo "[$timestamp] [$SCRIPT_NAME:$line_number] DEBUG: $message" ;;
    esac
}

# Verbose error exit function
fail_with_diagnostics() {
    local exit_code="$1"
    local error_message="$2"
    local cause="${3:-Unknown cause}"
    local fix_suggestion="${4:-No specific fix available}"
    local line_number="${BASH_LINENO[1]:-unknown}"
    
    log_diagnostic "ERROR" "CRITICAL FAILURE at line $line_number"
    log_diagnostic "ERROR" "Message: $error_message"
    log_diagnostic "ERROR" "Cause: $cause"
    log_diagnostic "ERROR" "How to fix: $fix_suggestion"
    log_diagnostic "ERROR" "Exit code: $exit_code"
    
    if [[ $VERBOSE -eq 1 ]]; then
        log_diagnostic "DEBUG" "Full environment dump:"
        log_diagnostic "DEBUG" "  PWD: $(pwd)"
        log_diagnostic "DEBUG" "  PATH: $PATH"
        log_diagnostic "DEBUG" "  GOPATH: ${GOPATH:-unset}"
        log_diagnostic "DEBUG" "  GOBIN: ${GOBIN:-unset}"
        log_diagnostic "DEBUG" "  GO111MODULE: ${GO111MODULE:-unset}"
        log_diagnostic "DEBUG" "  CI: ${CI:-unset}"
        log_diagnostic "DEBUG" "  GITHUB_ACTIONS: ${GITHUB_ACTIONS:-unset}"
    fi
    
    exit "$exit_code"
}

if [[ $VERBOSE -eq 1 ]]; then
    log_diagnostic "INFO" "Starting protobuf generation with verbose diagnostics"
    log_diagnostic "DEBUG" "Script: $SCRIPT_NAME, Args: $*"
else
    echo "Generating protobuf files..."
fi

# Step 1: Input validation with verbose diagnostics
log_diagnostic "INFO" "Step 1: Validating command line arguments"
if [[ $# -ne 2 ]]; then
    fail_with_diagnostics 1 \
        "Invalid number of arguments: expected 2, got $#" \
        "Script called with wrong number of parameters" \
        "Usage: $0 <PROTO_DIR> <GO_OUT>"
fi

readonly PROTO_DIR="$1"
readonly GO_OUT="$2"
log_diagnostic "INFO" "✓ Step 1 SUCCESS: Arguments validated (PROTO_DIR='$PROTO_DIR', GO_OUT='$GO_OUT')"

# Step 2: Validate input paths
log_diagnostic "INFO" "Step 2: Validating input paths"
if [[ ! -d "$PROTO_DIR" ]]; then
    fail_with_diagnostics 1 \
        "PROTO_DIR '$PROTO_DIR' does not exist" \
        "Missing or invalid proto directory path" \
        "Create the directory or provide correct path: mkdir -p '$PROTO_DIR'"
fi
log_diagnostic "DEBUG" "Proto directory exists: $PROTO_DIR"

# Secure path construction
readonly PROTO_FILE="${PROTO_DIR}/echo.proto"
if [[ ! -f "$PROTO_FILE" ]]; then
    fail_with_diagnostics 1 \
        "Proto file '$PROTO_FILE' does not exist" \
        "Required echo.proto file is missing from proto directory" \
        "Create echo.proto file or check if filename is correct in directory '$PROTO_DIR'"
fi
log_diagnostic "INFO" "✓ Step 2 SUCCESS: Proto file found at '$PROTO_FILE'"

# Step 3: Create output directory if needed
log_diagnostic "INFO" "Step 3: Ensuring output directory exists"
if [[ ! -d "$GO_OUT" ]]; then
    log_diagnostic "WARN" "Output directory '$GO_OUT' does not exist, creating it"
    if ! mkdir -p "$GO_OUT"; then
        fail_with_diagnostics 1 \
            "Failed to create output directory '$GO_OUT'" \
            "Permission denied or invalid path" \
            "Check directory permissions or run with appropriate privileges"
    fi
    log_diagnostic "INFO" "Created output directory: $GO_OUT"
fi
log_diagnostic "INFO" "✓ Step 3 SUCCESS: Output directory ready at '$GO_OUT'"

log_diagnostic "INFO" "Configuration summary:"
log_diagnostic "INFO" "  Proto directory: $PROTO_DIR"
log_diagnostic "INFO" "  Output directory: $GO_OUT"
log_diagnostic "INFO" "  Proto file: $PROTO_FILE"

# Step 4: Setup Go environment with verbose diagnostics
log_diagnostic "INFO" "Step 4: Setting up Go environment"
readonly ORIGINAL_PATH="$PATH"

# Check if Go is available
if ! command -v go >/dev/null 2>&1; then
    fail_with_diagnostics 1 \
        "Go toolchain not found in PATH" \
        "Go is not installed or not in PATH" \
        "Install Go from https://golang.org/dl/ and ensure it's in PATH"
fi

export GO111MODULE=on
log_diagnostic "DEBUG" "Set GO111MODULE=on"

# Get and validate GOPATH/GOBIN
if ! GOPATH_VALUE=$(go env GOPATH 2>/dev/null); then
    fail_with_diagnostics 1 \
        "Failed to get GOPATH from 'go env'" \
        "Go environment is corrupted or not properly configured" \
        "Check Go installation and run 'go env' to verify configuration"
fi

export GOBIN="${GOPATH_VALUE}/bin"
export PATH="$PATH:$GOBIN:/usr/bin:/usr/local/bin"
log_diagnostic "INFO" "✓ Step 4 SUCCESS: Go environment configured"
log_diagnostic "DEBUG" "  GO111MODULE: $GO111MODULE"
log_diagnostic "DEBUG" "  GOPATH: $GOPATH_VALUE" 
log_diagnostic "DEBUG" "  GOBIN: $GOBIN"
log_diagnostic "DEBUG" "  Updated PATH: $PATH"

# Step 5: Check for protoc with detailed diagnostics
log_diagnostic "INFO" "Step 5: Checking for protoc compiler"
if ! command -v protoc >/dev/null 2>&1; then
    log_diagnostic "WARN" "protoc not found in PATH"
    
    # Check if generated files already exist as fallback
    if [[ -f "${GO_OUT}/echo.pb.go" ]] && [[ -f "${GO_OUT}/echo_grpc.pb.go" ]]; then
        log_diagnostic "INFO" "✓ Step 5 SUCCESS: Protobuf files already exist, skipping generation"
        log_diagnostic "INFO" "  Found: ${GO_OUT}/echo.pb.go"
        log_diagnostic "INFO" "  Found: ${GO_OUT}/echo_grpc.pb.go"
        exit 0
    else
        # Provide detailed installation instructions based on environment
        fix_instructions=""
        if [[ "${CI:-}" == "true" ]] || [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
            fix_instructions="CI detected: Check .github/actions/setup-protobuf action or add 'apt-get install protobuf-compiler' to workflow"
        else
            fix_instructions="Install protoc: Ubuntu/Debian: 'sudo apt-get install -y protobuf-compiler', macOS: 'brew install protobuf', or run 'make setup'"
        fi
        
        fail_with_diagnostics 1 \
            "protoc not found and generated files don't exist" \
            "Protocol Buffer compiler is not installed" \
            "$fix_instructions"
    fi
else
    protoc_version=""
    if protoc_version=$(protoc --version 2>/dev/null); then
        log_diagnostic "INFO" "✓ Step 5 SUCCESS: Found protoc at $(which protoc)"
        log_diagnostic "DEBUG" "  Version: $protoc_version"
    else
        fail_with_diagnostics 1 \
            "protoc found but version check failed" \
            "protoc binary is corrupted or incompatible" \
            "Reinstall protoc or check if binary has proper permissions"
    fi
fi

# Step 6: Install/verify protoc-gen-go with detailed diagnostics
log_diagnostic "INFO" "Step 6: Checking and installing Go protobuf plugins"

# Function to install Go plugin with retries and diagnostics
install_go_plugin() {
    local plugin_name="$1"
    local package_path="$2"
    local line_number="${BASH_LINENO[0]}"
    
    log_diagnostic "INFO" "Step 6a: Checking for $plugin_name"
    
    if command -v "$plugin_name" >/dev/null 2>&1; then
        plugin_path=$(which "$plugin_name")
        log_diagnostic "INFO" "✓ Found $plugin_name at: $plugin_path"
        return 0
    fi
    
    log_diagnostic "WARN" "$plugin_name not found, installing from $package_path"
    
    # Verify GOBIN is set and in PATH before installing
    if [[ ! -d "$GOBIN" ]]; then
        if ! mkdir -p "$GOBIN"; then
            fail_with_diagnostics 1 \
                "Cannot create GOBIN directory '$GOBIN'" \
                "Permission denied or invalid path" \
                "Check directory permissions or set GOBIN to a writable location"
        fi
        log_diagnostic "DEBUG" "Created GOBIN directory: $GOBIN"
    fi
    
    # Install with retries and detailed error reporting
    max_retries=3
    for attempt in $(seq 1 $max_retries); do
        log_diagnostic "INFO" "Installing $plugin_name (attempt $attempt/$max_retries)"
        
        if go install "$package_path@latest" 2>&1; then
            log_diagnostic "INFO" "✓ $plugin_name installed successfully"
            
            # Verify installation worked
            if command -v "$plugin_name" >/dev/null 2>&1; then
                installed_path=$(which "$plugin_name")
                log_diagnostic "INFO" "✓ Step 6a SUCCESS: $plugin_name verified at $installed_path"
                return 0
            else
                log_diagnostic "ERROR" "$plugin_name installation completed but binary not found in PATH"
                log_diagnostic "DEBUG" "Expected binary at: $GOBIN/$plugin_name"
                log_diagnostic "DEBUG" "Current PATH: $PATH"
                
                if [[ $attempt -eq $max_retries ]]; then
                    fail_with_diagnostics 1 \
                        "$plugin_name installed but not accessible" \
                        "GOBIN directory not in PATH or binary not created" \
                        "Add \$GOBIN to PATH: export PATH=\"\$PATH:\$GOBIN\" or check Go module proxy settings"
                fi
            fi
        else
            go_error=$?
            log_diagnostic "WARN" "$plugin_name install attempt $attempt failed (exit code: $go_error)"
            
            if [[ $attempt -eq $max_retries ]]; then
                fail_with_diagnostics 1 \
                    "All $plugin_name install attempts failed after $max_retries tries" \
                    "Network issues, Go proxy problems, or invalid package path" \
                    "Check internet connection, verify Go proxy settings (GOPROXY), or run 'go env' to check Go configuration"
            else
                log_diagnostic "INFO" "Retrying in 2 seconds..."
                sleep 2
            fi
        fi
    done
}

# Install both required plugins
install_go_plugin "protoc-gen-go" "google.golang.org/protobuf/cmd/protoc-gen-go"
install_go_plugin "protoc-gen-go-grpc" "google.golang.org/grpc/cmd/protoc-gen-go-grpc"

log_diagnostic "INFO" "✓ Step 6 SUCCESS: All Go protobuf plugins ready"

# Step 7: Generate protobuf files with comprehensive diagnostics
log_diagnostic "INFO" "Step 7: Generating protobuf Go files"

# Pre-generation validation
expected_pb_file="${GO_OUT}/echo.pb.go"
expected_grpc_file="${GO_OUT}/echo_grpc.pb.go"

log_diagnostic "DEBUG" "Protoc command will be:"
log_diagnostic "DEBUG" "  protoc \\"
log_diagnostic "DEBUG" "    --go_out='$GO_OUT' \\"
log_diagnostic "DEBUG" "    --go_opt=paths=source_relative \\"
log_diagnostic "DEBUG" "    --go-grpc_out='$GO_OUT' \\"
log_diagnostic "DEBUG" "    --go-grpc_opt=paths=source_relative \\"
log_diagnostic "DEBUG" "    -I '$PROTO_DIR' \\"
log_diagnostic "DEBUG" "    '$PROTO_FILE'"

# Run protoc with detailed error capture
log_diagnostic "INFO" "Executing protoc command..."
if protoc_output=$(protoc \
    --go_out="$GO_OUT" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$GO_OUT" \
    --go-grpc_opt=paths=source_relative \
    -I "$PROTO_DIR" \
    "$PROTO_FILE" 2>&1); then
    
    log_diagnostic "INFO" "✓ protoc command completed successfully"
    
    # Verify generated files exist and are non-empty
    validation_failed=0
    
    if [[ ! -f "$expected_pb_file" ]]; then
        log_diagnostic "ERROR" "Expected protobuf file not generated: $expected_pb_file"
        validation_failed=1
    elif [[ ! -s "$expected_pb_file" ]]; then
        log_diagnostic "ERROR" "Generated protobuf file is empty: $expected_pb_file"
        validation_failed=1
    else
        pb_size=$(stat -c%s "$expected_pb_file")
        log_diagnostic "INFO" "✓ Generated echo.pb.go ($pb_size bytes)"
    fi
    
    if [[ ! -f "$expected_grpc_file" ]]; then
        log_diagnostic "ERROR" "Expected gRPC file not generated: $expected_grpc_file"
        validation_failed=1
    elif [[ ! -s "$expected_grpc_file" ]]; then
        log_diagnostic "ERROR" "Generated gRPC file is empty: $expected_grpc_file"
        validation_failed=1
    else
        grpc_size=$(stat -c%s "$expected_grpc_file")
        log_diagnostic "INFO" "✓ Generated echo_grpc.pb.go ($grpc_size bytes)"
    fi
    
    if [[ $validation_failed -eq 1 ]]; then
        fail_with_diagnostics 1 \
            "Protoc completed but required files not generated or empty" \
            "protoc-gen-go or protoc-gen-go-grpc plugins malfunctioned" \
            "Reinstall plugins: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    fi
    
    log_diagnostic "INFO" "✓ Step 7 SUCCESS: Protobuf generation completed successfully!"
    
else
    protoc_exit_code=$?
    log_diagnostic "ERROR" "protoc command failed with exit code: $protoc_exit_code"
    if [[ -n "$protoc_output" ]]; then
        log_diagnostic "ERROR" "protoc output: $protoc_output"
    fi
    
    # Analyze potential causes
    probable_cause="Unknown protoc error"
    fix_suggestion="Check protoc installation and plugin availability"
    
    if echo "$protoc_output" | grep -q "protoc-gen-go"; then
        probable_cause="protoc-gen-go plugin not found or not executable"
        fix_suggestion="Ensure protoc-gen-go is in PATH: export PATH=\"\$PATH:\$(go env GOPATH)/bin\""
    elif echo "$protoc_output" | grep -q "protoc-gen-go-grpc"; then
        probable_cause="protoc-gen-go-grpc plugin not found or not executable"
        fix_suggestion="Ensure protoc-gen-go-grpc is in PATH: export PATH=\"\$PATH:\$(go env GOPATH)/bin\""
    elif echo "$protoc_output" | grep -q "No such file"; then
        probable_cause="Proto file or import path not found"
        fix_suggestion="Verify proto file exists and import paths are correct"
    fi
    
    fail_with_diagnostics $protoc_exit_code \
        "Protobuf generation failed" \
        "$probable_cause" \
        "$fix_suggestion"
fi

# Step 8: Final cleanup and verification
log_diagnostic "INFO" "Step 8: Final cleanup and verification"

# Restore original PATH
export PATH="$ORIGINAL_PATH"
log_diagnostic "DEBUG" "Restored original PATH"

log_diagnostic "INFO" "✓ SUCCESS: All steps completed successfully!"
log_diagnostic "INFO" "Generated files:"
log_diagnostic "INFO" "  - $expected_pb_file"
log_diagnostic "INFO" "  - $expected_grpc_file"

# Simple success message for non-verbose mode
if [[ $VERBOSE -eq 0 ]]; then
    echo "✅ Protobuf generation completed successfully!"
    echo "   Generated: echo.pb.go, echo_grpc.pb.go"
fi