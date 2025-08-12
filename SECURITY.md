# Security Policy

## Supported Versions

Currently, we provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| < latest| :x:                |

## Reporting a Vulnerability

We take the security of Ephemos seriously. If you have discovered a security vulnerability in our project, we appreciate your help in disclosing it to us responsibly.

### How to Report

Please report security vulnerabilities by emailing the maintainers directly or by opening a private security advisory on GitHub:

1. Go to the "Security" tab of this repository
2. Click on "Report a vulnerability"
3. Fill out the form with details about the vulnerability

### What to Include

When reporting a vulnerability, please include:

- A description of the vulnerability and its potential impact
- Steps to reproduce the issue
- Affected versions
- Any possible mitigations you've identified

### Response Timeline

- We will acknowledge receipt of your report within 48 hours
- We will provide an initial assessment within 7 days
- We aim to release patches for critical vulnerabilities within 30 days

### Security Scanning

This project uses automated security scanning through:

- **CodeQL**: Static Application Security Testing (SAST) for code vulnerabilities
- **Dependabot**: Automated dependency vulnerability scanning
- **GitHub Security Advisories**: Tracking known vulnerabilities

The CodeQL analysis runs on:
- Every push to main/master/develop branches
- Every pull request
- Weekly scheduled scans (Sundays at midnight UTC)

### Best Practices for Contributors

When contributing to this project, please:

1. Never commit secrets, API keys, or credentials
2. Use environment variables for sensitive configuration
3. Follow secure coding practices for Go
4. Keep dependencies up to date
5. Run security checks locally before submitting PRs

### Security Features in Ephemos

Ephemos includes several built-in security features:

- **mTLS Support**: Mutual TLS authentication via SPIFFE/SPIRE
- **Authorization**: Fine-grained access control for services
- **Secure Defaults**: Production-ready security configurations
- **Audit Logging**: Comprehensive logging for security events

## Acknowledgments

We thank the security researchers and contributors who help keep Ephemos secure. Responsible disclosure is greatly appreciated.

## Contact

For sensitive security matters that should not be discussed publicly, please contact the maintainers directly through GitHub's private security advisory feature.