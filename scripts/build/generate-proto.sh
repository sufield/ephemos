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

# Secure PATH setup
readonly ORIGINAL_PATH="$PATH"
export PATH="$PATH:$(go env GOPATH)/bin:/usr/bin:/usr/local/bin"

# Check for protoc
if ! command -v protoc >/dev/null 2>&1; then
    echo "Warning: protoc not found in PATH"
    
    # Check if files already exist
    if [[ -f "${GO_OUT}/echo.pb.go" ]] && [[ -f "${GO_OUT}/echo_grpc.pb.go" ]]; then
        echo "✅ Protobuf files already exist, skipping generation"
        exit 0
    else
        echo "❌ protoc not found and protobuf files don't exist" >&2
        echo "Please install protoc: apt-get install protobuf-compiler" >&2
        exit 1
    fi
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