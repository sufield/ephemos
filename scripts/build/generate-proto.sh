#!/bin/bash
# Secure protobuf generation script for Ephemos
# Usage: ./generate-proto.sh <PROTO_DIR> <GO_OUT>

set -euo pipefail  # Exit on error, undefined vars, pipe failures

# Input validation
if [[ $# -ne 2 ]]; then
    echo "Usage: $0 <PROTO_DIR> <GO_OUT>" >&2
    exit 1
fi

readonly PROTO_DIR="$1"
readonly GO_OUT="$2"

# Validate input paths
if [[ ! -d "$PROTO_DIR" ]]; then
    echo "Error: PROTO_DIR '$PROTO_DIR' does not exist" >&2
    exit 1
fi

# Secure path construction
readonly PROTO_FILE="${PROTO_DIR}/echo.proto"
if [[ ! -f "$PROTO_FILE" ]]; then
    echo "Error: Proto file '$PROTO_FILE' does not exist" >&2
    exit 1
fi

echo "Generating protobuf code..."
echo "  Proto dir: $PROTO_DIR"
echo "  Output dir: $GO_OUT"

# Secure PATH setup with proper Go environment
readonly ORIGINAL_PATH="$PATH"
export GO111MODULE=on
export GOBIN=$(go env GOPATH)/bin
export PATH="$PATH:$GOBIN:/usr/bin:/usr/local/bin"

# Check for protoc
if ! command -v protoc >/dev/null 2>&1; then
    echo "Warning: protoc not found in PATH"
    
    # Check if files already exist
    if [[ -f "${GO_OUT}/echo.pb.go" ]] && [[ -f "${GO_OUT}/echo_grpc.pb.go" ]]; then
        echo "✅ Protobuf files already exist, skipping generation"
        exit 0
    else
        echo "❌ protoc not found and protobuf files don't exist" >&2
        echo "" >&2
        if [[ "${CI:-}" == "true" ]] || [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
            echo "CI environment detected. Protoc should be installed by GitHub Actions." >&2
            echo "If you see this error in CI, check the setup-protobuf action." >&2
        else
            echo "Install protoc with:" >&2
            echo "  Ubuntu/Debian: sudo apt-get update && sudo apt-get install -y protobuf-compiler" >&2
            echo "  CentOS/RHEL: sudo yum install -y protobuf-compiler" >&2
            echo "  macOS: brew install protobuf" >&2
            echo "  Windows: choco install protoc" >&2
            echo "" >&2
            echo "For automated setup:" >&2
            echo "  make setup          # Smart setup (Go tools only)" >&2
            echo "  ./scripts/install-deps-sudo.sh  # Full setup (requires sudo)" >&2
        fi
        exit 1
    fi
fi

# Check for Go protobuf tools (GOBIN already set above)
if ! command -v protoc-gen-go >/dev/null 2>&1; then
    echo "Warning: protoc-gen-go not found, installing with retries..."
    for i in {1..3}; do
        if go install google.golang.org/protobuf/cmd/protoc-gen-go@latest; then
            echo "✅ protoc-gen-go installed successfully"
            break
        else
            echo "⚠️ protoc-gen-go install attempt $i failed"
            if [ $i -eq 3 ]; then
                echo "❌ All protoc-gen-go install attempts failed"
                exit 1
            fi
            sleep 2
        fi
    done
fi

if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
    echo "Warning: protoc-gen-go-grpc not found, installing with retries..."
    for i in {1..3}; do
        if go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest; then
            echo "✅ protoc-gen-go-grpc installed successfully"
            break
        else
            echo "⚠️ protoc-gen-go-grpc install attempt $i failed"
            if [ $i -eq 3 ]; then
                echo "❌ All protoc-gen-go-grpc install attempts failed"
                exit 1
            fi
            sleep 2
        fi
    done
fi

echo "Found protoc at: $(which protoc)"

# Secure protobuf generation
if protoc \
    --go_out="$GO_OUT" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$GO_OUT" \
    --go-grpc_opt=paths=source_relative \
    -I "$PROTO_DIR" \
    "$PROTO_FILE"; then
    echo "✅ Protobuf generation completed successfully!"
else
    echo "❌ Protobuf generation failed" >&2
    exit 1
fi

# Restore original PATH
export PATH="$ORIGINAL_PATH"