#!/bin/bash
#
# Install protobuf tools with guaranteed success
# This script ensures protoc-gen-go and protoc-gen-go-grpc are installed and available
#

set -euo pipefail

echo "🔧 Installing Go protobuf tools..."

# Step 1: Determine GOBIN
if [ -z "$(go env GOBIN)" ]; then
  export GOBIN="$(go env GOPATH)/bin"
  echo "📋 GOBIN not set, using GOPATH/bin: $GOBIN"
else
  export GOBIN="$(go env GOBIN)"
  echo "📋 Using existing GOBIN: $GOBIN"
fi

# Step 2: Create GOBIN directory
mkdir -p "$GOBIN"
echo "✅ GOBIN directory ensured: $GOBIN"

# Step 3: Add GOBIN to PATH
export PATH="$GOBIN:$PATH"
echo "✅ Added GOBIN to PATH"

# Step 4: Install protoc-gen-go
echo ""
echo "📥 Installing protoc-gen-go..."
if go install google.golang.org/protobuf/cmd/protoc-gen-go@latest; then
  echo "✅ protoc-gen-go installed successfully"
else
  echo "❌ Failed to install protoc-gen-go"
  echo "Retrying with verbose output..."
  go install -v google.golang.org/protobuf/cmd/protoc-gen-go@latest || exit 1
fi

# Step 5: Install protoc-gen-go-grpc
echo ""
echo "📥 Installing protoc-gen-go-grpc..."
if go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest; then
  echo "✅ protoc-gen-go-grpc installed successfully"
else
  echo "❌ Failed to install protoc-gen-go-grpc"
  echo "Retrying with verbose output..."
  go install -v google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest || exit 1
fi

# Step 6: Verify installations
echo ""
echo "🔍 Verifying installations..."

# Check files exist
if [ ! -f "$GOBIN/protoc-gen-go" ]; then
  echo "❌ protoc-gen-go not found at $GOBIN/protoc-gen-go"
  echo "Contents of GOBIN:"
  ls -la "$GOBIN"
  exit 1
fi

if [ ! -f "$GOBIN/protoc-gen-go-grpc" ]; then
  echo "❌ protoc-gen-go-grpc not found at $GOBIN/protoc-gen-go-grpc"
  echo "Contents of GOBIN:"
  ls -la "$GOBIN"
  exit 1
fi

# Check they're in PATH
if ! which protoc-gen-go >/dev/null 2>&1; then
  echo "❌ protoc-gen-go not in PATH"
  echo "PATH: $PATH"
  exit 1
fi

if ! which protoc-gen-go-grpc >/dev/null 2>&1; then
  echo "❌ protoc-gen-go-grpc not in PATH"
  echo "PATH: $PATH"
  exit 1
fi

echo "✅ protoc-gen-go found at: $(which protoc-gen-go)"
echo "✅ protoc-gen-go-grpc found at: $(which protoc-gen-go-grpc)"

# Step 7: Export for GitHub Actions (if running in CI)
if [ -n "${GITHUB_PATH:-}" ]; then
  echo "$GOBIN" >> "$GITHUB_PATH"
  echo "✅ Added GOBIN to GITHUB_PATH for future steps"
fi

if [ -n "${GITHUB_ENV:-}" ]; then
  echo "GOBIN=$GOBIN" >> "$GITHUB_ENV"
  echo "✅ Exported GOBIN to GITHUB_ENV"
fi

echo ""
echo "🎉 All protobuf tools installed and verified successfully!"
echo ""
echo "To use these tools, ensure your PATH includes: $GOBIN"
echo "export PATH=\"$GOBIN:\$PATH\""