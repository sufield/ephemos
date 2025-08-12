#!/bin/bash

# Security scanning script for local development
# This script runs various security checks on the codebase

set -e

echo "🔐 Running Security Scans for Ephemos..."
echo "========================================"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track if any issues are found
ISSUES_FOUND=0

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 1. Check for secrets/credentials
echo -e "\n${YELLOW}1. Checking for hardcoded secrets...${NC}"
if command_exists gitleaks; then
    if gitleaks detect --source . --verbose; then
        echo -e "${GREEN}✓ No secrets detected${NC}"
    else
        echo -e "${RED}✗ Potential secrets found!${NC}"
        ISSUES_FOUND=1
    fi
else
    echo "  ⚠️  gitleaks not installed. Install with: brew install gitleaks"
    # Fallback to basic grep patterns
    echo "  Running basic secret detection..."
    if grep -r -E "(api[_-]?key|apikey|secret|password|passwd|pwd|token|private[_-]?key)" \
        --exclude-dir=vendor --exclude-dir=.git --exclude="*.md" --exclude="*_test.go" .; then
        echo -e "${RED}✗ Potential secrets found!${NC}"
        ISSUES_FOUND=1
    else
        echo -e "${GREEN}✓ No obvious secrets detected${NC}"
    fi
fi

# 2. Run gosec for Go security issues
echo -e "\n${YELLOW}2. Running Go security analyzer (gosec)...${NC}"
if command_exists gosec; then
    if gosec -fmt json -out gosec-report.json ./... 2>/dev/null; then
        echo -e "${GREEN}✓ No security issues found${NC}"
        rm -f gosec-report.json
    else
        echo -e "${RED}✗ Security issues found! Check gosec-report.json${NC}"
        ISSUES_FOUND=1
    fi
else
    echo "  ⚠️  gosec not installed. Install with: go install github.com/securego/gosec/v2/cmd/gosec@latest"
fi

# 3. Check for vulnerable dependencies
echo -e "\n${YELLOW}3. Checking for vulnerable dependencies...${NC}"
if command_exists nancy; then
    go list -json -deps ./... | nancy sleuth
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ No vulnerable dependencies found${NC}"
    else
        echo -e "${RED}✗ Vulnerable dependencies detected!${NC}"
        ISSUES_FOUND=1
    fi
else
    echo "  Using go mod audit instead..."
    if go list -m all | xargs go mod why 2>/dev/null | grep -q "CVE"; then
        echo -e "${RED}✗ Potential vulnerable dependencies${NC}"
        ISSUES_FOUND=1
    else
        echo -e "${GREEN}✓ No known vulnerabilities in go.mod${NC}"
    fi
fi

# 4. Run govulncheck for known vulnerabilities
echo -e "\n${YELLOW}4. Checking for known Go vulnerabilities...${NC}"
if command_exists govulncheck; then
    if govulncheck ./...; then
        echo -e "${GREEN}✓ No known vulnerabilities${NC}"
    else
        echo -e "${RED}✗ Vulnerabilities found!${NC}"
        ISSUES_FOUND=1
    fi
else
    echo "  ⚠️  govulncheck not installed. Install with: go install golang.org/x/vuln/cmd/govulncheck@latest"
fi

# 5. Check file permissions for sensitive files
echo -e "\n${YELLOW}5. Checking file permissions...${NC}"
SENSITIVE_FILES=$(find . -type f \( -name "*.key" -o -name "*.pem" -o -name "*.p12" \) 2>/dev/null)
if [ -n "$SENSITIVE_FILES" ]; then
    echo "  Found sensitive files. Checking permissions..."
    for file in $SENSITIVE_FILES; do
        PERMS=$(stat -c %a "$file" 2>/dev/null || stat -f %A "$file" 2>/dev/null)
        if [ "$PERMS" -gt "600" ]; then
            echo -e "${RED}✗ File $file has loose permissions: $PERMS${NC}"
            ISSUES_FOUND=1
        fi
    done
    if [ $ISSUES_FOUND -eq 0 ]; then
        echo -e "${GREEN}✓ Sensitive files have appropriate permissions${NC}"
    fi
else
    echo -e "${GREEN}✓ No sensitive files found${NC}"
fi

# 6. Check for insecure code patterns
echo -e "\n${YELLOW}6. Checking for insecure code patterns...${NC}"
INSECURE_PATTERNS=(
    "http://"
    "md5.Sum"
    "sha1.Sum"
    "rand.Seed(time.Now"
    "tls.Config{InsecureSkipVerify: true"
    "fmt.Sprintf.*SELECT.*FROM"
    "exec.Command"
)

PATTERN_FOUND=0
for pattern in "${INSECURE_PATTERNS[@]}"; do
    if grep -r "$pattern" --include="*.go" --exclude-dir=vendor --exclude-dir=.git --exclude="*_test.go" . >/dev/null 2>&1; then
        echo -e "${YELLOW}  ⚠️  Found potentially insecure pattern: $pattern${NC}"
        PATTERN_FOUND=1
    fi
done

if [ $PATTERN_FOUND -eq 0 ]; then
    echo -e "${GREEN}✓ No insecure patterns detected${NC}"
else
    echo -e "${YELLOW}  Review the patterns above for security implications${NC}"
fi

# Summary
echo -e "\n========================================"
if [ $ISSUES_FOUND -eq 0 ]; then
    echo -e "${GREEN}✅ Security scan completed successfully!${NC}"
    echo "No critical security issues found."
else
    echo -e "${RED}⚠️  Security scan found issues!${NC}"
    echo "Please review and fix the issues above before committing."
    exit 1
fi

echo -e "\n💡 Tip: Install missing tools for more comprehensive scanning:"
echo "  - gitleaks: brew install gitleaks"
echo "  - gosec: go install github.com/securego/gosec/v2/cmd/gosec@latest"
echo "  - nancy: go install github.com/sonatype-nexus-community/nancy@latest"
echo "  - govulncheck: go install golang.org/x/vuln/cmd/govulncheck@latest"