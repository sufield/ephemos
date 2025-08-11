# Security Policy

## Supported Versions

We actively support the following versions of Ephemos:

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report security vulnerabilities to us via:

### Preferred Method: GitHub Security Advisories
- Go to the [Security Advisories](../../security/advisories) page
- Click "Report a vulnerability"
- Fill out the form with details

### Alternative Method: Email
Send an email to: **security@[your-domain].com**

Please include the following information:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fixes

## Response Timeline

- **Initial Response**: Within 24 hours
- **Status Update**: Within 72 hours
- **Fix Timeline**: Critical issues within 7 days, others within 30 days

## Security Best Practices

When using Ephemos:

1. **Keep Dependencies Updated**: Regularly update to the latest version
2. **Secure Configuration**: Follow our security configuration guide
3. **Monitor Vulnerabilities**: Use our SBOM files for vulnerability scanning
4. **Certificate Management**: Ensure SPIRE is properly configured and monitored

## Security Features

Ephemos includes several built-in security features:

- **Zero Plaintext Secrets**: No secrets stored in code or configuration
- **Ephemeral Certificates**: Short-lived X.509 certificates (1-hour expiration)
- **Automatic Rotation**: Certificates rotate every ~30 minutes
- **mTLS Authentication**: Mutual TLS for all service communication
- **SPIFFE/SPIRE Integration**: Industry-standard identity management
- **SBOM Generation**: Complete software bill of materials for supply chain security

## Vulnerability Disclosure

We follow responsible disclosure practices:

1. Security issues are investigated and fixed privately
2. Fixes are released as soon as possible
3. CVEs are published after fixes are available
4. Credit is given to security researchers (with permission)

## Security Auditing

- Regular security audits are performed
- SAST scanning in CI/CD pipeline
- Dependency vulnerability monitoring
- Supply chain security with SBOM generation

For more information about Ephemos security architecture, see our [Security Documentation](docs/security/).