#!/bin/bash
# Secure security tools installation script
# Installs security scanning tools with verification

set -euo pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly TEMP_DIR="$(mktemp -d)"
readonly INSTALL_DIR="/usr/local/bin"

# Cleanup on exit
trap 'rm -rf "$TEMP_DIR"' EXIT

echo "🛠️ Installing security scanning tools..."
echo "Temporary directory: $TEMP_DIR"

# Function to verify tool installation
verify_tool() {
    local tool="$1"
    if command -v "$tool" >/dev/null 2>&1; then
        echo "✅ $tool installed successfully"
        "$tool" --version 2>/dev/null || echo "  Version info not available"
    else
        echo "❌ $tool installation failed" >&2
        return 1
    fi
}

# Install gitleaks
echo ""
echo "Installing gitleaks..."
if command -v gitleaks >/dev/null 2>&1; then
    echo "✅ gitleaks already installed"
    gitleaks version || echo "  Version info not available"
else
    echo "Downloading gitleaks..."
    readonly GITLEAKS_VERSION="v8.28.0"  # Pin to specific version
    readonly SYSTEM="$(uname -s)"
    readonly ARCH="$(uname -m)"
    readonly GITLEAKS_URL="https://github.com/gitleaks/gitleaks/releases/download/${GITLEAKS_VERSION}/gitleaks_${SYSTEM}_${ARCH}.tar.gz"
    
    if command -v wget >/dev/null 2>&1; then
        wget -q -O "$TEMP_DIR/gitleaks.tar.gz" "$GITLEAKS_URL"
    elif command -v curl >/dev/null 2>&1; then
        curl -sSL -o "$TEMP_DIR/gitleaks.tar.gz" "$GITLEAKS_URL"
    else
        echo "❌ Neither wget nor curl available" >&2
        exit 1
    fi
    
    # Extract and install
    tar -xzf "$TEMP_DIR/gitleaks.tar.gz" -C "$TEMP_DIR"
    sudo install -m 755 "$TEMP_DIR/gitleaks" "$INSTALL_DIR/gitleaks"
    verify_tool gitleaks
fi

# Install git-secrets
echo ""
echo "Installing git-secrets..."
if command -v git-secrets >/dev/null 2>&1; then
    echo "✅ git-secrets already installed"
else
    echo "Cloning git-secrets repository..."
    git clone --depth 1 https://github.com/awslabs/git-secrets.git "$TEMP_DIR/git-secrets"
    
    cd "$TEMP_DIR/git-secrets"
    sudo make install PREFIX=/usr/local
    cd - >/dev/null
    
    verify_tool git-secrets
fi

# Install TruffleHog
echo ""
echo "Installing TruffleHog..."
if command -v trufflehog >/dev/null 2>&1; then
    echo "✅ TruffleHog already installed"
    trufflehog --version || echo "  Version info not available"
else
    echo "Downloading TruffleHog..."
    readonly TRUFFLEHOG_VERSION="v3.90.4"  # Pin to specific version
    readonly SYSTEM_LOWER="$(uname -s | tr '[:upper:]' '[:lower:]')"
    readonly ARCH_MAPPED="$(uname -m | sed 's/x86_64/amd64/g' | sed 's/aarch64/arm64/g')"
    readonly TRUFFLEHOG_URL="https://github.com/trufflesecurity/trufflehog/releases/download/${TRUFFLEHOG_VERSION}/trufflehog_${TRUFFLEHOG_VERSION}_${SYSTEM_LOWER}_${ARCH_MAPPED}.tar.gz"
    
    echo "Downloading from: $TRUFFLEHOG_URL"
    
    if command -v wget >/dev/null 2>&1; then
        wget -q -O "$TEMP_DIR/trufflehog.tar.gz" "$TRUFFLEHOG_URL"
    elif command -v curl >/dev/null 2>&1; then
        curl -sSL -o "$TEMP_DIR/trufflehog.tar.gz" "$TRUFFLEHOG_URL"
    else
        echo "❌ Neither wget nor curl available" >&2
        exit 1
    fi
    
    # Extract and install
    tar -xzf "$TEMP_DIR/trufflehog.tar.gz" -C "$TEMP_DIR"
    sudo install -m 755 "$TEMP_DIR/trufflehog" "$INSTALL_DIR/trufflehog"
    verify_tool trufflehog
fi

# Install trivy
echo ""
echo "Installing trivy..."
if command -v trivy >/dev/null 2>&1; then
    echo "✅ trivy already installed"
    trivy --version || echo "  Version info not available"
else
    echo "Installing trivy via official installer..."
    readonly TRIVY_INSTALLER="$TEMP_DIR/install-trivy.sh"
    
    if command -v curl >/dev/null 2>&1; then
        curl -sSL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh > "$TRIVY_INSTALLER"
        
        # Verify installer is reasonable (basic check)
        if [[ ! -s "$TRIVY_INSTALLER" ]]; then
            echo "❌ Failed to download trivy installer" >&2
            exit 1
        fi
        
        chmod +x "$TRIVY_INSTALLER"
        sudo "$TRIVY_INSTALLER" -b "$INSTALL_DIR"
        verify_tool trivy
    else
        echo "❌ curl not available for trivy installation" >&2
        exit 1
    fi
fi

echo ""
echo "🔒 Security tools installation completed!"
echo ""
echo "Installed tools:"
echo "  - gitleaks: Secret detection"
echo "  - git-secrets: AWS secret detection"
echo "  - trufflehog: Advanced secret detection"
echo "  - trivy: Vulnerability scanning"
echo ""
echo "Next steps:"
echo "  1. Run: make security-hooks"
echo "  2. Run: make security-scan"