#!/bin/bash
set -e

# CI Setup Script for Ephemos
# This script provides complex setup logic that can be reused across different CI jobs

echo "🔧 Setting up Ephemos CI environment..."

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to install protobuf compiler based on OS
install_protoc() {
    if [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "📦 Installing protobuf compiler for Linux..."
        sudo apt-get update -qq
        sudo apt-get install -y protobuf-compiler
    elif [[ "$OSTYPE" == "darwin"* ]]; then
        echo "📦 Installing protobuf compiler for macOS..."
        brew install protobuf
    else
        echo "❌ Unsupported OS: $OSTYPE"
        exit 1
    fi
}

# Function to install Go protobuf tools
install_go_protobuf_tools() {
    echo "📦 Installing Go protobuf tools..."
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
}

# Function to verify installations
verify_setup() {
    echo "✅ Verifying setup..."
    
    # Check Go
    if command_exists go; then
        echo "   Go: $(go version)"
    else
        echo "❌ Go not found"
        exit 1
    fi
    
    # Check protoc
    if command_exists protoc; then
        echo "   protoc: $(protoc --version)"
    else
        echo "❌ protoc not found"
        exit 1
    fi
    
    # Check Go protobuf tools
    if command_exists protoc-gen-go && command_exists protoc-gen-go-grpc; then
        echo "   ✅ Go protobuf tools installed"
    else
        echo "❌ Go protobuf tools not found"
        exit 1
    fi
}

# Main setup logic
main() {
    echo "🚀 Starting CI setup for $(uname -s)..."
    
    # Install protoc if not present
    if ! command_exists protoc; then
        install_protoc
    else
        echo "✅ protoc already installed: $(protoc --version)"
    fi
    
    # Install Go protobuf tools
    install_go_protobuf_tools
    
    # Generate protobuf code
    echo "🔨 Generating protobuf code..."
    make proto
    
    # Verify everything is working
    verify_setup
    
    echo "🎉 CI setup completed successfully!"
}

# Run main function
main "$@"