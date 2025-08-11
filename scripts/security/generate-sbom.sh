#!/bin/bash
# Generate Software Bill of Materials (SBOM) for Ephemos
# Uses Syft to create both SPDX and CycloneDX format SBOMs

set -e

# Colors for output
GREEN='\033[32m'
BLUE='\033[34m'
YELLOW='\033[33m'
RED='\033[31m'
RESET='\033[0m'
CHECKMARK='✅'
INFO='📋'
WARNING='⚠️'
ERROR='❌'

# Function to print status messages
print_info() {
    echo -e "${INFO} ${BLUE}$1${RESET}"
}

print_success() {
    echo -e "${CHECKMARK} ${GREEN}$1${RESET}"
}

print_warning() {
    echo -e "${WARNING} ${YELLOW}$1${RESET}"
}

print_error() {
    echo -e "${ERROR} ${RED}$1${RESET}"
}

# Get project root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SBOM_DIR="$PROJECT_ROOT/sbom"

print_info "Starting SBOM generation for Ephemos..."
print_info "Project root: $PROJECT_ROOT"

# Create SBOM directory
mkdir -p "$SBOM_DIR"

# Check if Syft is installed
if ! command -v syft >/dev/null 2>&1; then
    print_error "Syft is not installed. Installing..."
    
    # Install Syft
    print_info "Installing Syft via Go install..."
    if ! go install github.com/anchore/syft/cmd/syft@latest; then
        print_error "Failed to install Syft via Go install"
        print_info "Attempting to install via curl..."
        
        # Fallback to curl installation
        curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
        
        if ! command -v syft >/dev/null 2>&1; then
            print_error "Failed to install Syft. Please install manually:"
            print_error "  Go install: go install github.com/anchore/syft/cmd/syft@latest"
            print_error "  Or visit: https://github.com/anchore/syft#installation"
            exit 1
        fi
    fi
    
    print_success "Syft installed successfully"
fi

# Check Syft version
SYFT_VERSION=$(syft version --output text 2>/dev/null | head -1 || echo "unknown")
print_info "Using Syft version: $SYFT_VERSION"

# Change to project root for scanning
cd "$PROJECT_ROOT"

# Generate project metadata
PROJECT_NAME="ephemos"
PROJECT_VERSION=$(go mod edit -json | jq -r '.Module.Path' | sed 's/.*\///')
if [ -z "$PROJECT_VERSION" ] || [ "$PROJECT_VERSION" = "null" ]; then
    PROJECT_VERSION="dev"
fi
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

print_info "Project: $PROJECT_NAME"
print_info "Version: $PROJECT_VERSION"
print_info "Timestamp: $TIMESTAMP"

# Generate SPDX SBOM (JSON format)
print_info "Generating SPDX SBOM (JSON format)..."
SPDX_FILE="$SBOM_DIR/${PROJECT_NAME}-${PROJECT_VERSION}-sbom.spdx.json"

if syft . -o spdx-json --file "$SPDX_FILE"; then
    print_success "SPDX SBOM generated: $(basename "$SPDX_FILE")"
    
    # Add file size info
    SPDX_SIZE=$(du -h "$SPDX_FILE" | cut -f1)
    print_info "SPDX SBOM size: $SPDX_SIZE"
else
    print_error "Failed to generate SPDX SBOM"
    exit 1
fi

# Generate CycloneDX SBOM (JSON format)
print_info "Generating CycloneDX SBOM (JSON format)..."
CYCLONEDX_FILE="$SBOM_DIR/${PROJECT_NAME}-${PROJECT_VERSION}-sbom.cyclonedx.json"

if syft . -o cyclonedx-json --file "$CYCLONEDX_FILE"; then
    print_success "CycloneDX SBOM generated: $(basename "$CYCLONEDX_FILE")"
    
    # Add file size info
    CYCLONEDX_SIZE=$(du -h "$CYCLONEDX_FILE" | cut -f1)
    print_info "CycloneDX SBOM size: $CYCLONEDX_SIZE"
else
    print_error "Failed to generate CycloneDX SBOM"
    exit 1
fi

# Generate text summary for quick viewing
print_info "Generating text summary..."
TEXT_FILE="$SBOM_DIR/${PROJECT_NAME}-${PROJECT_VERSION}-sbom-summary.txt"

cat > "$TEXT_FILE" << EOF
Ephemos Software Bill of Materials (SBOM) Summary
==================================================

Project: $PROJECT_NAME
Version: $PROJECT_VERSION
Generated: $TIMESTAMP
Generator: Syft $SYFT_VERSION

Files Generated:
- SPDX Format (JSON): $(basename "$SPDX_FILE") ($SPDX_SIZE)
- CycloneDX Format (JSON): $(basename "$CYCLONEDX_FILE") ($CYCLONEDX_SIZE)

Component Summary:
==================
EOF

# Add component count summary
if command -v jq >/dev/null 2>&1; then
    if [ -f "$SPDX_FILE" ]; then
        COMPONENT_COUNT=$(jq '.packages | length' "$SPDX_FILE" 2>/dev/null || echo "N/A")
        echo "Total Components: $COMPONENT_COUNT" >> "$TEXT_FILE"
        
        # Top-level dependencies
        echo "" >> "$TEXT_FILE"
        echo "Direct Dependencies (from go.mod):" >> "$TEXT_FILE"
        echo "====================================" >> "$TEXT_FILE"
        
        # Parse go.mod for direct dependencies
        if [ -f "$PROJECT_ROOT/go.mod" ]; then
            grep -E "^\s*[a-zA-Z]" "$PROJECT_ROOT/go.mod" | grep -v "module\|go\|require\|replace\|exclude" | head -20 >> "$TEXT_FILE" || true
        fi
        
        # Security-relevant packages
        echo "" >> "$TEXT_FILE"
        echo "Security-Relevant Packages:" >> "$TEXT_FILE"
        echo "===========================" >> "$TEXT_FILE"
        
        jq -r '.packages[] | select(.name | contains("crypto") or contains("security") or contains("tls") or contains("spiffe") or contains("grpc")) | "- \(.name) \(.versionInfo)"' "$SPDX_FILE" 2>/dev/null | head -10 >> "$TEXT_FILE" || echo "Unable to extract security packages" >> "$TEXT_FILE"
    fi
else
    echo "jq not available - install for enhanced summary" >> "$TEXT_FILE"
fi

print_success "Text summary generated: $(basename "$TEXT_FILE")"

# Generate checksums for integrity verification
print_info "Generating checksums..."
cd "$SBOM_DIR"

CHECKSUM_FILE="${PROJECT_NAME}-${PROJECT_VERSION}-sbom-checksums.txt"
{
    echo "Ephemos SBOM Checksums"
    echo "======================"
    echo "Generated: $TIMESTAMP"
    echo ""
    sha256sum "$(basename "$SPDX_FILE")" 2>/dev/null || echo "Failed to generate checksum for SPDX file"
    sha256sum "$(basename "$CYCLONEDX_FILE")" 2>/dev/null || echo "Failed to generate checksum for CycloneDX file"
    sha256sum "$(basename "$TEXT_FILE")" 2>/dev/null || echo "Failed to generate checksum for text file"
} > "$CHECKSUM_FILE"

print_success "Checksums generated: $CHECKSUM_FILE"

# Display summary
echo ""
print_success "SBOM generation completed successfully!"
echo ""
print_info "Generated files in $SBOM_DIR:"
ls -la "$SBOM_DIR" | grep -E "\.(json|txt)$" | while read -r line; do
    echo "  $line"
done

echo ""
print_info "Next steps:"
echo "  • Validate SBOMs: make sbom-validate"
echo "  • View summary: cat sbom/$(basename "$TEXT_FILE")"
echo "  • Upload to compliance systems or vulnerability scanners"
echo "  • Include in CI/CD artifacts for supply chain security"

print_success "SBOM generation workflow completed!"