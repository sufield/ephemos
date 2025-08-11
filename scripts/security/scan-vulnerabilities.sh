#!/bin/bash
# Vulnerability scanning script for Ephemos
# Scans for known vulnerabilities in dependencies

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🔍 Scanning for vulnerabilities..."
echo "Project root: $PROJECT_ROOT"

# Change to project directory safely
cd "$PROJECT_ROOT"

# Go vulnerability checking
echo ""
echo "Running Go vulnerability scan..."
if command -v govulncheck >/dev/null 2>&1; then
    if govulncheck ./...; then
        echo "✅ govulncheck: No vulnerabilities found"
    else
        echo "❌ govulncheck found vulnerabilities" >&2
        echo "Run 'go get -u' to update vulnerable dependencies" >&2
    fi
else
    echo "❌ govulncheck not installed" >&2
    echo "Install with: go install golang.org/x/vuln/cmd/govulncheck@latest" >&2
fi

echo ""

# Trivy vulnerability scanning
echo "Running Trivy vulnerability scan..."
if command -v trivy >/dev/null 2>&1; then
    echo "Scanning filesystem for vulnerabilities..."
    if trivy fs --exit-code 1 --severity HIGH,CRITICAL .; then
        echo "✅ Trivy: No high/critical vulnerabilities found"
    else
        echo "⚠️  Trivy found high/critical vulnerabilities" >&2
        # Don't exit - show summary but continue
    fi
    
    echo ""
    echo "Scanning for misconfigurations..."
    if trivy config --exit-code 0 .; then
        echo "✅ Trivy config scan completed"
    else
        echo "⚠️  Trivy found configuration issues" >&2
    fi
else
    echo "❌ Trivy not installed" >&2
    echo "Install with: make security-tools" >&2
fi

echo ""

# Docker security scanning (if Dockerfile exists)
if [[ -f Dockerfile ]]; then
    echo "Running Docker security scan..."
    if command -v docker >/dev/null 2>&1; then
        # Build image for scanning
        docker build -t ephemos-security-scan . >/dev/null 2>&1
        
        if command -v trivy >/dev/null 2>&1; then
            trivy image --exit-code 0 ephemos-security-scan
            echo "✅ Docker image security scan completed"
        else
            echo "⚠️  Cannot scan Docker image - trivy not available" >&2
        fi
        
        # Clean up
        docker rmi ephemos-security-scan >/dev/null 2>&1 || true
    else
        echo "⚠️  Docker not available for image scanning" >&2
    fi
else
    echo "No Dockerfile found - skipping Docker security scan"
fi

echo ""
echo "🔍 Vulnerability scanning completed!"
echo ""
echo "If vulnerabilities were found:"
echo "  1. Update dependencies: go get -u"
echo "  2. Check for security patches"
echo "  3. Review Trivy output for specific fixes"
echo "  4. Consider pinning secure versions"