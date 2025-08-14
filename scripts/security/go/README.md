# Go Security Scanner

A comprehensive security scanner for the Ephemos project, implemented in Go to replace the original bash scripts with better error handling, structured output, and improved maintainability.

## Features

- **Secrets Scanning**: Detects hardcoded credentials and sensitive data
- **Vulnerability Detection**: Scans for known vulnerabilities in dependencies
- **SBOM Generation**: Creates Software Bill of Materials for supply chain security
- **Structured Output**: Detailed results with timing and warnings
- **CLI Interface**: Cobra-based CLI with subcommands and verbose mode

## Usage

### Run All Scans
```bash
# Using wrapper script (recommended)
scripts/security/run-security-scanner.sh

# With options
scripts/security/run-security-scanner.sh --verbose --continue-on-error

# Using Bazel
bazel run //scripts/security:security_scanner

# Direct Go execution  
go run scripts/security/go/main.go
```

### Individual Scans
```bash
# Secrets scanning only
scripts/security/run-security-scanner.sh secrets --verbose

# Vulnerability scanning only  
scripts/security/run-security-scanner.sh vulnerabilities --verbose

# SBOM generation only
scripts/security/run-security-scanner.sh sbom --verbose
```

## Security Tools Integration

The scanner integrates with these security tools:

- **gitleaks**: Secret detection in Git repositories
- **TruffleHog**: Verified secrets scanning
- **govulncheck**: Go vulnerability database scanning
- **Trivy**: Container and filesystem vulnerability scanning
- **Syft**: SBOM generation (SPDX and CycloneDX formats)

## Installation Requirements

Install the required security tools:

```bash
# Install Go-based tools
go install github.com/gitleaks/gitleaks/v8@latest
go install github.com/trufflesecurity/trufflehog/v3@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/anchore/syft/cmd/syft@latest

# Install Trivy (varies by OS)
# See: https://aquasecurity.github.io/trivy/latest/getting-started/installation/
```

## Output Format

The scanner provides structured output with:
- Scan names and pass/fail status
- Execution time for each scan
- Detailed results and error messages
- Warning messages for missing tools
- Summary of overall results

Example output:
```
🔒 Running comprehensive security scans...
📝 Running secrets scans...
🔍 Running vulnerability scans...
📋 Running SBOM generation and validation...

============================================================
📊 SECURITY SCAN SUMMARY
============================================================
Secrets Scanning               ✅ PASSED (0.15s)
    Details: ✅ Custom patterns: No hardcoded credentials found

Vulnerability Scanning         ✅ PASSED (2.34s)  
    Details: ✅ govulncheck: No vulnerabilities found

SBOM Generation & Validation   ✅ PASSED (5.67s)
    Details: ✅ SBOM generated successfully

🎉 All security scans passed!
```

## Advantages over Bash Scripts

1. **Better Error Handling**: Structured error handling with timeouts
2. **Consistent Interface**: Unified CLI across all security tools  
3. **Improved Maintainability**: Easier to extend and modify
4. **Structured Output**: Machine-readable results for CI integration
5. **Cross-platform**: Works on Windows, macOS, and Linux
6. **Type Safety**: Compile-time error checking
7. **Reduced Injection Risks**: No shell command injection vulnerabilities

## CI/CD Integration

The scanner is designed for CI/CD environments:

- Exit codes: 0 for success, 1 for failure
- Timeout management for all external tools
- Graceful handling of missing tools
- Detailed logging for debugging
- Option to continue on errors for reporting

## Development

To modify the scanner:

1. Edit `scripts/security/go/main.go`
2. Update BUILD.bazel if adding dependencies
3. Test with: `go run scripts/security/go/main.go`
4. Build with Bazel: `bazel build //scripts/security:security_scanner`

## Migration from Bash Scripts

This Go implementation replaces:

- `security-scan-all.sh` → `security-scanner` (main command)
- `scan-secrets.sh` → `security-scanner secrets` 
- `scan-vulnerabilities.sh` → `security-scanner vulnerabilities`
- `generate-sbom.sh` + `validate-sbom.sh` → `security-scanner sbom`

The bash scripts are still available for compatibility but the Go version is recommended for new usage.