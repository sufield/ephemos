#!/bin/bash
# Combined security scan script - runs all security checks

set -euo pipefail

echo "🔒 Running comprehensive security scans..."

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Track overall status
OVERALL_STATUS=0

# Run secret scanning
echo "📝 Running secret scan..."
if "$SCRIPT_DIR/scan-secrets.sh"; then
    echo "✅ Secret scan passed"
else
    echo "❌ Secret scan failed"
    OVERALL_STATUS=1
fi

# Run vulnerability scanning  
echo "🔍 Running vulnerability scan..."
if "$SCRIPT_DIR/scan-vulnerabilities.sh"; then
    echo "✅ Vulnerability scan passed"
else
    echo "❌ Vulnerability scan failed"
    OVERALL_STATUS=1
fi

# Generate and validate SBOM
echo "📋 Generating and validating SBOM..."
if "$SCRIPT_DIR/generate-sbom.sh" && "$SCRIPT_DIR/validate-sbom.sh"; then
    echo "✅ SBOM generation and validation passed"
else
    echo "❌ SBOM generation or validation failed"
    OVERALL_STATUS=1
fi

# Final status
if [ $OVERALL_STATUS -eq 0 ]; then
    echo "🎉 All security scans passed!"
else
    echo "💥 Some security scans failed!"
fi

exit $OVERALL_STATUS