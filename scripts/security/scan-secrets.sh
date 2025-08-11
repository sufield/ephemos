#!/bin/bash
# Secure secrets scanning script for Ephemos
# Scans for secrets and sensitive data without shell injection risks

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

echo "🔍 Scanning for secrets and sensitive data..."
echo "Project root: $PROJECT_ROOT"

# Change to project directory safely
cd "$PROJECT_ROOT"

# Gitleaks scan with error handling
echo "Running gitleaks scan..."
if command -v gitleaks >/dev/null 2>&1; then
    if gitleaks detect --source . --no-git --verbose; then
        echo "✅ Gitleaks: No secrets found"
    else
        echo "⚠️  Gitleaks found potential secrets" >&2
        # Don't exit - continue with other scans
    fi
else
    echo "❌ gitleaks not installed" >&2
    echo "Install with: make security-tools" >&2
fi

echo ""

# Git-secrets scan with error handling  
echo "Running git-secrets scan..."
if command -v git-secrets >/dev/null 2>&1; then
    if git-secrets --scan --recursive .; then
        echo "✅ Git-secrets: No secrets found"
    else
        echo "⚠️  git-secrets found potential secrets" >&2
        # Don't exit - continue with other scans
    fi
else
    echo "❌ git-secrets not installed" >&2
    echo "Install with: make security-tools" >&2
fi

echo ""

# Manual config file audit
echo "Running manual config file audit..."
if command -v rg >/dev/null 2>&1; then
    echo "Checking for potential secrets in config files:"
    if rg -i "(password|secret|key|token|credential|api[_-]?key|private[_-]?key)" config/ 2>/dev/null; then
        echo "⚠️  Found potential secrets in config files" >&2
    else
        echo "✅ No obvious secrets found in config/"
    fi
    
    echo "Checking for hardcoded production values:"
    if rg -i "(prod|production|staging)" config/*.yaml 2>/dev/null; then
        echo "⚠️  Found hardcoded production values" >&2
    else
        echo "✅ No hardcoded production values found"
    fi
    
    echo "Checking for real domains (not example.org):"
    if rg -v "example\.org" config/*.yaml 2>/dev/null | rg "[a-zA-Z0-9.-]+\.(com|net|org|io)" >/dev/null 2>&1; then
        echo "⚠️  Found real domains in config files" >&2
        rg -v "example\.org" config/*.yaml 2>/dev/null | rg "[a-zA-Z0-9.-]+\.(com|net|org|io)" || true
    else
        echo "✅ Only example domains found"
    fi
else
    echo "❌ ripgrep (rg) not installed - skipping manual audit" >&2
fi

echo ""
echo "🔍 Secret scanning completed!"
echo ""
echo "If secrets were found:"
echo "  1. Remove them immediately: git rm <file> && git commit"
echo "  2. Rotate any exposed credentials"
echo "  3. Review .gitignore and .gitleaks.toml"
echo "  4. Consider using environment variables instead"