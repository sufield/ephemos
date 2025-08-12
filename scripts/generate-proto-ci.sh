#!/bin/bash
# CI-friendly protobuf generation script
# Falls back to pre-generated files if protoc is unavailable

set -e

PROTO_DIR="${1:-examples/proto}"
GO_OUT="${2:-examples/proto}"

echo "🔧 CI-friendly protobuf generation..."

# Check if protobuf files already exist
if [[ -f "${GO_OUT}/echo.pb.go" ]] && [[ -f "${GO_OUT}/echo_grpc.pb.go" ]]; then
    echo "✅ Protobuf files already exist, using existing files"
    exit 0
fi

# Try to use the regular generation script
if ./scripts/build/generate-proto.sh "$PROTO_DIR" "$GO_OUT" 2>/dev/null; then
    echo "✅ Protobuf generation completed successfully"
    exit 0
else
    echo "⚠️  Protobuf generation failed, checking for alternatives..."
    
    # In CI, we might have pre-generated files in a different location
    # or committed for CI compatibility (though not recommended)
    if [[ -f "ci/prebuilt/${GO_OUT##*/}/echo.pb.go" ]] && [[ -f "ci/prebuilt/${GO_OUT##*/}/echo_grpc.pb.go" ]]; then
        echo "📦 Using pre-built protobuf files for CI"
        mkdir -p "$GO_OUT"
        cp "ci/prebuilt/${GO_OUT##*/}/"*.pb.go "$GO_OUT/"
        exit 0
    fi
    
    echo "❌ Cannot generate or find protobuf files"
    echo "Please ensure protoc is installed or provide pre-generated files"
    exit 1
fi