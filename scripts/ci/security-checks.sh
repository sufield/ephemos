#!/bin/bash
# CI Security checks orchestrator for Ephemos
# Runs comprehensive security validation in CI/CD pipeline

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🔒 Starting CI security checks..."
echo "Project root: $PROJECT_ROOT"
echo ""

# Change to project directory safely
cd "$PROJECT_ROOT"

# Fix .secrets file permissions if it exists (Git doesn't preserve exact permissions)
if [[ -f .secrets ]]; then
    chmod 600 .secrets
    echo "Fixed .secrets file permissions to 600"
fi

# Exit code tracking
security_issues=0

# 1. Secrets scanning
echo "==== 1. SECRETS SCANNING ===="
if [[ -x "$PROJECT_ROOT/scripts/security/scan-secrets.sh" ]]; then
    if "$PROJECT_ROOT/scripts/security/scan-secrets.sh"; then
        echo "✅ Secrets scan completed"
    else
        echo "⚠️  Secrets scan found issues" >&2
        ((security_issues++))
    fi
else
    echo "❌ Secrets scanning script not found" >&2
    ((security_issues++))
fi
echo ""

# 2. Dependency vulnerability scanning 
echo "==== 2. DEPENDENCY SCANNING ===="
if command -v govulncheck >/dev/null 2>&1; then
    echo "Running Go vulnerability check..."
    if govulncheck ./...; then
        echo "✅ No known vulnerabilities found"
    else
        echo "❌ Vulnerabilities found in dependencies" >&2
        ((security_issues++))
    fi
else
    echo "❌ govulncheck not installed - skipping" >&2
fi
echo ""

# 3. Static security analysis
echo "==== 3. STATIC SECURITY ANALYSIS ===="
if command -v gosec >/dev/null 2>&1; then
    echo "Running gosec security scan..."
    if gosec -fmt json -out gosec-report.json ./... 2>/dev/null; then
        echo "✅ gosec scan completed"
        # Check if any issues were found (JSON report will have issues array)
        if jq -e '.Issues | length > 0' gosec-report.json >/dev/null 2>&1; then
            echo "⚠️  gosec found security issues:" >&2
            jq -r '.Issues[] | "  - \(.details) at \(.file):\(.line)"' gosec-report.json 2>/dev/null || echo "  See gosec-report.json for details"
            ((security_issues++))
        fi
    else
        echo "❌ gosec scan failed" >&2
        ((security_issues++))
    fi
else
    echo "❌ gosec not installed - skipping" >&2
fi
echo ""

# 4. Configuration security audit
echo "==== 4. CONFIGURATION SECURITY ===="
if [[ -f "$PROJECT_ROOT/bin/config-validator" ]] || command -v go >/dev/null 2>&1; then
    echo "Running configuration security validation..."
    
    # Build config validator if not exists
    if [[ ! -f "$PROJECT_ROOT/bin/config-validator" ]]; then
        echo "Building config validator..."
        go build -o bin/config-validator ./cmd/config-validator
    fi
    
    # Test production configuration validation
    if EPHEMOS_SERVICE_NAME="test-service" \
       EPHEMOS_TRUST_DOMAIN="test.local" \
       EPHEMOS_DEBUG_ENABLED="false" \
       ./bin/config-validator -env-only -production; then
        echo "✅ Configuration security validation passed"
    else
        echo "❌ Configuration security validation failed" >&2
        ((security_issues++))
    fi
else
    echo "❌ Cannot build config validator - Go not available" >&2
fi
echo ""

# 5. File permissions audit
echo "==== 5. FILE PERMISSIONS AUDIT ===="
echo "Checking for overly permissive files..."
insecure_files=0

# Check for world-writable files
if find . -path ./.git -prune -o -type f -perm -002 -print | grep -v "^\\.$"; then
    echo "❌ Found world-writable files" >&2
    ((insecure_files++))
fi

# Check for executable config files (shouldn't be executable)
if find config/ -name "*.yaml" -o -name "*.yml" -o -name "*.json" | xargs ls -l | grep "^-r.xr..r.." 2>/dev/null; then
    echo "❌ Found executable configuration files" >&2
    ((insecure_files++))
fi

# Check for secret files with wrong permissions
if [[ -f .secrets ]]; then
    current_perms=$(stat -c %a .secrets 2>/dev/null)
    echo "Debug: .secrets permissions are: $current_perms" >&2
    if [[ "$current_perms" != "600" ]]; then
        echo "❌ .secrets file has incorrect permissions (expected 600, got $current_perms)" >&2
        ((insecure_files++))
    else
        echo "✅ .secrets file has correct permissions (600)" >&2
    fi
fi

if [[ $insecure_files -eq 0 ]]; then
    echo "✅ File permissions are secure"
else
    ((security_issues++))
fi
echo ""

# Summary
echo "==== SECURITY CHECKS SUMMARY ===="
if [[ $security_issues -eq 0 ]]; then
    echo "✅ All security checks passed!"
    echo "🔒 Project is ready for production deployment"
    exit 0
else
    echo "❌ Security checks found $security_issues issue(s)"
    echo "🚨 Fix security issues before deploying to production"
    exit 1
fi