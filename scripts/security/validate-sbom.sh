#!/bin/bash
# Validate generated SBOM files for Ephemos
# Checks format validity, completeness, and security compliance

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

print_info "Starting SBOM validation for Ephemos..."
print_info "SBOM directory: $SBOM_DIR"

# Check if SBOM directory exists
if [ ! -d "$SBOM_DIR" ]; then
    print_error "SBOM directory not found: $SBOM_DIR"
    print_info "Run 'make sbom-generate' first to create SBOM files"
    exit 1
fi

# Check if jq is available for JSON validation
if ! command -v jq >/dev/null 2>&1; then
    print_warning "jq not found - JSON structure validation will be limited"
    print_info "Install jq for enhanced validation: apt-get install jq (Ubuntu/Debian) or brew install jq (macOS)"
fi

# Validation results
VALIDATION_ERRORS=0
VALIDATION_WARNINGS=0

# Function to validate file exists and is readable
validate_file_exists() {
    local file="$1"
    local description="$2"
    
    if [ ! -f "$file" ]; then
        print_error "$description not found: $(basename "$file")"
        ((VALIDATION_ERRORS++))
        return 1
    fi
    
    if [ ! -r "$file" ]; then
        print_error "$description not readable: $(basename "$file")"
        ((VALIDATION_ERRORS++))
        return 1
    fi
    
    # Check if file is empty
    if [ ! -s "$file" ]; then
        print_error "$description is empty: $(basename "$file")"
        ((VALIDATION_ERRORS++))
        return 1
    fi
    
    print_success "$description exists and is readable"
    return 0
}

# Function to validate JSON format
validate_json_format() {
    local file="$1"
    local description="$2"
    
    if ! command -v jq >/dev/null 2>&1; then
        print_warning "Skipping JSON validation for $description (jq not available)"
        return 0
    fi
    
    if jq empty "$file" >/dev/null 2>&1; then
        print_success "$description has valid JSON format"
        return 0
    else
        print_error "$description has invalid JSON format"
        ((VALIDATION_ERRORS++))
        return 1
    fi
}

# Function to validate SPDX SBOM structure
validate_spdx_structure() {
    local file="$1"
    
    if ! command -v jq >/dev/null 2>&1; then
        print_warning "Skipping SPDX structure validation (jq not available)"
        return 0
    fi
    
    print_info "Validating SPDX SBOM structure..."
    
    # Check required SPDX fields
    local required_fields=("spdxVersion" "documentName" "packages")
    local missing_fields=0
    
    for field in "${required_fields[@]}"; do
        if jq -e ".$field" "$file" >/dev/null 2>&1; then
            print_success "SPDX field present: $field"
        else
            print_error "SPDX field missing: $field"
            ((missing_fields++))
        fi
    done
    
    if [ $missing_fields -eq 0 ]; then
        print_success "SPDX structure validation passed"
        
        # Additional checks
        local package_count=$(jq '.packages | length' "$file" 2>/dev/null || echo "0")
        print_info "SPDX package count: $package_count"
        
        if [ "$package_count" -lt 5 ]; then
            print_warning "Low package count ($package_count) - may indicate incomplete scan"
            ((VALIDATION_WARNINGS++))
        fi
        
        return 0
    else
        print_error "SPDX structure validation failed ($missing_fields missing fields)"
        ((VALIDATION_ERRORS++))
        return 1
    fi
}

# Function to validate CycloneDX SBOM structure
validate_cyclonedx_structure() {
    local file="$1"
    
    if ! command -v jq >/dev/null 2>&1; then
        print_warning "Skipping CycloneDX structure validation (jq not available)"
        return 0
    fi
    
    print_info "Validating CycloneDX SBOM structure..."
    
    # Check required CycloneDX fields
    local required_fields=("bomFormat" "specVersion" "components")
    local missing_fields=0
    
    for field in "${required_fields[@]}"; do
        if jq -e ".$field" "$file" >/dev/null 2>&1; then
            print_success "CycloneDX field present: $field"
        else
            print_error "CycloneDX field missing: $field"
            ((missing_fields++))
        fi
    done
    
    if [ $missing_fields -eq 0 ]; then
        print_success "CycloneDX structure validation passed"
        
        # Additional checks
        local component_count=$(jq '.components | length' "$file" 2>/dev/null || echo "0")
        print_info "CycloneDX component count: $component_count"
        
        if [ "$component_count" -lt 5 ]; then
            print_warning "Low component count ($component_count) - may indicate incomplete scan"
            ((VALIDATION_WARNINGS++))
        fi
        
        return 0
    else
        print_error "CycloneDX structure validation failed ($missing_fields missing fields)"
        ((VALIDATION_ERRORS++))
        return 1
    fi
}

# Function to validate checksums
validate_checksums() {
    local checksum_file="$SBOM_DIR"/*-checksums.txt
    
    # Find checksum file (handle glob pattern)
    local found_checksum_file=""
    for file in $checksum_file; do
        if [ -f "$file" ]; then
            found_checksum_file="$file"
            break
        fi
    done
    
    if [ -z "$found_checksum_file" ]; then
        print_warning "Checksum file not found - skipping integrity validation"
        ((VALIDATION_WARNINGS++))
        return 0
    fi
    
    print_info "Validating file checksums..."
    
    cd "$SBOM_DIR"
    
    if sha256sum -c "$(basename "$found_checksum_file")" >/dev/null 2>&1; then
        print_success "All file checksums validated successfully"
        return 0
    else
        print_error "Checksum validation failed"
        ((VALIDATION_ERRORS++))
        return 1
    fi
}

# Function to check for security-relevant packages
validate_security_packages() {
    local spdx_file=""
    
    # Find SPDX file
    for file in "$SBOM_DIR"/*.spdx.json; do
        if [ -f "$file" ]; then
            spdx_file="$file"
            break
        fi
    done
    
    if [ -z "$spdx_file" ] || ! command -v jq >/dev/null 2>&1; then
        print_warning "Skipping security package validation"
        return 0
    fi
    
    print_info "Checking for security-relevant packages..."
    
    # Check for SPIFFE/SPIRE packages (critical for Ephemos)
    local spiffe_packages=$(jq -r '.packages[] | select(.name | contains("spiffe")) | .name' "$spdx_file" 2>/dev/null | wc -l)
    if [ "$spiffe_packages" -gt 0 ]; then
        print_success "SPIFFE packages detected: $spiffe_packages"
    else
        print_warning "No SPIFFE packages detected - may indicate incomplete scan"
        ((VALIDATION_WARNINGS++))
    fi
    
    # Check for crypto packages
    local crypto_packages=$(jq -r '.packages[] | select(.name | contains("crypto")) | .name' "$spdx_file" 2>/dev/null | wc -l)
    if [ "$crypto_packages" -gt 0 ]; then
        print_success "Crypto packages detected: $crypto_packages"
    fi
    
    # Check for gRPC packages (critical for Ephemos)
    local grpc_packages=$(jq -r '.packages[] | select(.name | contains("grpc")) | .name' "$spdx_file" 2>/dev/null | wc -l)
    if [ "$grpc_packages" -gt 0 ]; then
        print_success "gRPC packages detected: $grpc_packages"
    else
        print_warning "No gRPC packages detected - may indicate incomplete scan"
        ((VALIDATION_WARNINGS++))
    fi
}

# Main validation workflow
echo ""
print_info "=== SBOM File Validation ==="

# Find SBOM files
SPDX_FILES=("$SBOM_DIR"/*.spdx.json)
CYCLONEDX_FILES=("$SBOM_DIR"/*.cyclonedx.json)

# Validate SPDX files
for spdx_file in "${SPDX_FILES[@]}"; do
    if [ -f "$spdx_file" ]; then
        echo ""
        print_info "Validating SPDX SBOM: $(basename "$spdx_file")"
        
        validate_file_exists "$spdx_file" "SPDX SBOM"
        if [ $? -eq 0 ]; then
            validate_json_format "$spdx_file" "SPDX SBOM"
            validate_spdx_structure "$spdx_file"
        fi
    fi
done

# Validate CycloneDX files
for cyclonedx_file in "${CYCLONEDX_FILES[@]}"; do
    if [ -f "$cyclonedx_file" ]; then
        echo ""
        print_info "Validating CycloneDX SBOM: $(basename "$cyclonedx_file")"
        
        validate_file_exists "$cyclonedx_file" "CycloneDX SBOM"
        if [ $? -eq 0 ]; then
            validate_json_format "$cyclonedx_file" "CycloneDX SBOM"
            validate_cyclonedx_structure "$cyclonedx_file"
        fi
    fi
done

# Additional validations
echo ""
print_info "=== Additional Validations ==="

validate_checksums
validate_security_packages

# Summary
echo ""
print_info "=== Validation Summary ==="

if [ $VALIDATION_ERRORS -eq 0 ]; then
    print_success "SBOM validation completed successfully!"
    
    if [ $VALIDATION_WARNINGS -gt 0 ]; then
        print_warning "Found $VALIDATION_WARNINGS warnings (review recommended)"
    fi
    
    print_info "SBOMs are ready for:"
    echo "  • Supply chain security analysis"
    echo "  • Vulnerability scanning"
    echo "  • Compliance reporting"
    echo "  • CI/CD artifact storage"
    
    exit 0
else
    print_error "SBOM validation failed with $VALIDATION_ERRORS errors"
    
    if [ $VALIDATION_WARNINGS -gt 0 ]; then
        print_warning "Also found $VALIDATION_WARNINGS warnings"
    fi
    
    print_info "Fix errors and regenerate SBOMs:"
    echo "  • make sbom-generate"
    echo "  • Check Syft installation and permissions"
    echo "  • Ensure go.mod is up to date"
    
    exit 1
fi