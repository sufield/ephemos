# Software Bill of Materials (SBOM) Generation

Ephemos provides comprehensive Software Bill of Materials (SBOM) generation for supply chain security, compliance, and vulnerability management. This document outlines the SBOM generation capabilities, usage, and integration with security workflows.

## 📋 Overview

SBOM generation in Ephemos uses **Syft** (by Anchore) to create industry-standard software bills of materials that catalog all dependencies and components in the project. This enables:

- **Supply chain security analysis**
- **Vulnerability scanning and management**  
- **License compliance verification**
- **Dependency tracking and auditing**
- **Regulatory compliance reporting**

## 🛠️ SBOM Generation Methods

### 1. Manual Generation (Development)

Generate SBOMs locally during development:

```bash
# Generate all SBOM formats
make sbom-generate

# Validate generated SBOMs
make sbom-validate

# Complete SBOM workflow
make sbom-all
```

### 2. Automated Generation (CI/CD)

SBOMs are automatically generated in CI/CD pipelines:

- **Security Workflow**: Daily/triggered SBOM generation with vulnerability scanning
- **Release Workflow**: Comprehensive SBOMs included in every release
- **Pull Request Reviews**: SBOM validation and change detection

### 3. Script-Based Generation

Use the underlying scripts directly:

```bash
# Generate SBOMs
./scripts/security/generate-sbom.sh

# Validate SBOMs
./scripts/security/validate-sbom.sh
```

## 📁 Generated Files

The SBOM generation process creates multiple files in the `sbom/` directory:

### Core SBOM Files

1. **`ephemos-{version}-sbom.spdx.json`**
   - SPDX 2.3 format (industry standard)
   - Comprehensive dependency catalog
   - License information included
   - Perfect for compliance tools

2. **`ephemos-{version}-sbom.cyclonedx.json`**
   - CycloneDX format (security-focused)
   - Vulnerability correlation support
   - Component relationships mapped
   - Ideal for security scanners

3. **`ephemos-{version}-sbom-summary.txt`**
   - Human-readable summary
   - Component count statistics
   - Security-relevant package highlights
   - Quick overview for developers

### Integrity Files

4. **`ephemos-{version}-sbom-checksums.txt`**
   - SHA256 checksums for all SBOM files
   - Integrity verification support
   - Tamper detection capability

## 🔍 SBOM Validation

The validation process ensures SBOM quality and completeness:

### Validation Checks

1. **File Existence & Readability**
   - Verifies all SBOM files are present
   - Checks file permissions and accessibility
   - Ensures files are not empty

2. **JSON Structure Validation**
   - Validates JSON syntax and structure
   - Verifies required fields are present
   - Checks format compliance

3. **Content Validation**
   - Minimum component count verification
   - Security package detection
   - Critical dependency verification (SPIFFE, gRPC, crypto)

4. **Checksum Integrity**
   - SHA256 checksum verification
   - Tamper detection
   - File integrity assurance

### Validation Results

```bash
✅ SBOM validation completed successfully!
📋 SBOMs are ready for:
  • Supply chain security analysis
  • Vulnerability scanning
  • Compliance reporting
  • CI/CD artifact storage
```

## 🔐 Security Integration

### Vulnerability Scanning

SBOMs integrate with multiple vulnerability scanners:

```bash
# OSV Scanner (Google)
osv-scanner --sbom sbom/ephemos-sbom.spdx.json

# Grype (Anchore)
grype sbom:sbom/ephemos-sbom.spdx.json

# Trivy (Aqua Security)
trivy sbom sbom/ephemos-sbom.spdx.json
```

### CI/CD Security Pipeline

The security workflow automatically:

1. **Generates** comprehensive SBOMs using Syft
2. **Validates** SBOM structure and content
3. **Scans** for vulnerabilities using multiple tools
4. **Reports** findings in PR comments and artifacts
5. **Uploads** SBOMs as CI artifacts for compliance

### Compliance Reporting

SBOMs support various compliance frameworks:

- **NTIA Minimum Elements** - SBOM contains required metadata
- **Executive Order 14028** - Supply chain security compliance
- **ISO/IEC 5962** - SPDX standard compliance
- **NIST SP 800-161** - Supply chain risk management

## 📊 Component Analysis

### Security-Relevant Components

The SBOM generation process highlights security-critical components:

#### SPIFFE/SPIRE Components
```
- github.com/spiffe/go-spiffe/v2 v2.5.0
- github.com/spiffe/spire-api-sdk v1.9.0
```

#### Cryptographic Components
```
- golang.org/x/crypto v0.41.0
- github.com/go-jose/go-jose/v4 v4.0.5
```

#### Network Security Components
```
- google.golang.org/grpc v1.74.2
- golang.org/x/net v0.43.0
```

### License Analysis

SBOM includes license information for compliance:

```json
{
  "name": "github.com/spiffe/go-spiffe/v2",
  "licenseConcluded": "Apache-2.0",
  "licenseDeclared": "Apache-2.0"
}
```

## 🚀 Usage Examples

### Development Workflow

```bash
# Daily development SBOM generation
make sbom-generate

# Check SBOM quality
make sbom-validate

# Review security packages
cat sbom/ephemos-dev-sbom-summary.txt
```

### Release Workflow

```bash
# Release builds automatically include:
# - Versioned SBOM files
# - Vulnerability scan reports
# - Compliance documentation
# - Integrity checksums

git tag v1.0.0
git push origin v1.0.0  # Triggers release with SBOM
```

### Compliance Workflow

```bash
# Extract license information
jq '.packages[].licenseConcluded' sbom/ephemos-sbom.spdx.json | sort | uniq

# Generate dependency report
jq -r '.packages[] | "\(.name) \(.versionInfo) \(.licenseConcluded)"' \
  sbom/ephemos-sbom.spdx.json > dependency-report.txt

# Verify checksums
cd sbom && sha256sum -c ephemos-*-checksums.txt
```

## 🔧 Configuration

### Syft Configuration

SBOMs are generated with optimal settings for Go projects:

- **Source Analysis**: Scans `go.mod` and build artifacts
- **Transitive Dependencies**: Includes indirect dependencies
- **Multiple Formats**: Both SPDX and CycloneDX output
- **Rich Metadata**: Version, license, and vulnerability data

### CI/CD Configuration

SBOM generation is configured in `.github/workflows/security.yml`:

- **Triggers**: Daily, on dependency changes, and releases
- **Artifacts**: 90-day retention for compliance
- **Integration**: Automatic vulnerability scanning
- **Reporting**: PR comments with SBOM status

## 📈 Benefits

### For Security Teams

- **Vulnerability Management**: Complete dependency visibility
- **Risk Assessment**: Identify high-risk components
- **Incident Response**: Rapid impact analysis for CVEs
- **Compliance**: Automated regulatory reporting

### for Development Teams

- **Dependency Tracking**: Clear view of all dependencies
- **License Compliance**: Automated license verification
- **Supply Chain**: Transparent component sourcing
- **Quality Assurance**: SBOM validation in CI/CD

### For Operations Teams

- **Deployment Security**: Runtime component verification
- **Change Management**: SBOM diff analysis
- **Audit Trail**: Complete component history
- **Monitoring**: Continuous security posture tracking

## 🚨 Troubleshooting

### Common Issues

#### SBOM Generation Fails
```bash
# Check Syft installation
syft --version

# Verify project structure
ls -la go.mod

# Check build status
make build
```

#### Empty SBOM Files
```bash
# Ensure dependencies are downloaded
go mod download

# Check for build artifacts
ls -la bin/

# Verify Go module structure
go mod verify
```

#### Validation Errors
```bash
# Check file permissions
ls -la sbom/

# Validate JSON manually
jq empty sbom/*.json

# Review validation logs
./scripts/security/validate-sbom.sh
```

### Getting Help

- **Documentation**: Check this file and security runbooks
- **Logs**: Review CI/CD workflow logs for detailed errors
- **Scripts**: Run validation scripts with debug output
- **Community**: File issues with SBOM generation problems

## 📚 Related Documentation

- [Security Architecture](SECURITY_ARCHITECTURE.md) - Overall security design
- [CI/CD Security](CI_CD_SECURITY.md) - Pipeline security configuration  
- [Threat Model](THREAT_MODEL.md) - Security threat analysis
- [Security Runbook](SECURITY_RUNBOOK.md) - Operational security procedures

---

**Note**: SBOM generation is automatically configured and requires no additional setup. The system uses industry-standard tools and formats to ensure maximum compatibility with security and compliance tools.