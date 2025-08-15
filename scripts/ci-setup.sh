#!/bin/bash
set -e

# CI Setup Script for Ephemos
# This script provides complex setup logic that can be reused across different CI jobs

echo "🔧 Setting up Ephemos CI environment..."

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
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
    
    else
        exit 1
    fi    
}

# Main setup logic
main() {
    echo "🚀 Starting CI setup for $(uname -s)..."
        
    # Verify everything is working
    verify_setup
    
    echo "🎉 CI setup completed successfully!"
}

# Run main function
main "$@"