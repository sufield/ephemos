# Security Policy

## Supported Versions

We actively support the following versions of Ephemos with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

The Ephemos team takes security seriously. If you believe you have found a security vulnerability in Ephemos, we encourage you to report it to us responsibly.

### How to Report

Send an email to **security@sufield.com** with the following information:

- **Subject**: `[SECURITY] Ephemos Vulnerability Report`
- **Description**: Detailed description of the vulnerability
- **Impact**: Assessment of potential impact and severity
- **Reproduction**: Step-by-step instructions to reproduce the issue
- **Environment**: Versions affected, operating system, etc.
- **Proof of Concept**: If available (please be responsible)

### What to Expect

1. **Acknowledgment**: We will acknowledge receipt of your report within 48 hours
2. **Initial Assessment**: We will provide an initial assessment within 5 business days
3. **Regular Updates**: We will send updates on our progress at least every 5 business days
4. **Resolution Timeline**: We aim to resolve critical security issues within 30 days

### Security Vulnerability Categories

We are particularly interested in reports concerning:

#### High Priority
- **Authentication Bypass**: Circumventing SPIFFE/mTLS authentication
- **Identity Spoofing**: Impersonating other services or identities
- **Certificate Validation**: Issues with X.509 certificate verification
- **Privilege Escalation**: Gaining unauthorized access to resources
- **Injection Attacks**: SQL, command, or other injection vulnerabilities
- **Cryptographic Issues**: Weak encryption, key management problems

#### Medium Priority
- **Information Disclosure**: Unintended exposure of sensitive data
- **Denial of Service**: Resource exhaustion or availability issues
- **gRPC Interceptor Bypass**: Circumventing security interceptors
- **Logging Vulnerabilities**: Sensitive data exposure in logs
- **Configuration Issues**: Insecure default configurations

#### Lower Priority
- **Dependency Vulnerabilities**: Issues in third-party dependencies
- **Documentation Issues**: Security-related documentation problems
- **Best Practice Violations**: Sub-optimal security practices

### Scope

#### In Scope
- Ephemos core library (`internal/`, `pkg/`, `cmd/`)
- gRPC interceptors and authentication logic
- SPIFFE/SPIRE integration components
- Configuration and deployment scripts
- Official examples and templates
- CI/CD pipelines and build processes

#### Out of Scope
- Issues in third-party dependencies (report to respective maintainers)
- Theoretical attacks without practical exploitation
- Social engineering attacks
- Physical security issues
- Issues requiring physical access to infrastructure
- Vulnerabilities in SPIFFE/SPIRE themselves (report to SPIFFE project)

### Responsible Disclosure

We request that you:

- **Do not** publicly disclose the vulnerability until we have addressed it
- **Do not** access, modify, or delete data that doesn't belong to you
- **Do not** perform actions that could negatively impact other users
- **Do not** execute denial-of-service attacks
- **Provide** sufficient information to reproduce the vulnerability
- **Use** the latest version of Ephemos when testing

### Security Response Process

1. **Triage** (1-2 business days)
   - Verify and reproduce the vulnerability
   - Assess severity using CVSS v3.1 scoring
   - Determine affected versions and components

2. **Investigation** (3-7 business days)
   - Analyze root cause and impact
   - Develop fix strategy
   - Plan coordinated disclosure timeline

3. **Development** (5-21 business days)
   - Implement and test security fix
   - Prepare security advisory
   - Coordinate with distributors if needed

4. **Release** (1-3 business days)
   - Release patched versions
   - Publish security advisory
   - Update documentation

5. **Post-Release** (ongoing)
   - Monitor for additional issues
   - Update security documentation
   - Improve security practices

### Security Advisories

Security advisories will be published:

- On our [GitHub Security Advisories page](https://github.com/sufield/ephemos/security/advisories)
- In the project README and changelog
- Through appropriate security mailing lists
- On the project website (if applicable)

### Recognition

We appreciate the security research community's efforts in keeping Ephemos secure. With your permission, we will:

- Acknowledge your contribution in the security advisory
- List you in our security researchers hall of fame (if you wish)
- Provide a reference letter for your responsible disclosure (upon request)

### Security Best Practices for Users

#### Deployment Security
- Always use the latest supported version
- Follow the [Security Architecture Guide](../docs/security/SECURITY_ARCHITECTURE.md)
- Implement the [Security Runbook](../docs/security/SECURITY_RUNBOOK.md) procedures
- Review the [Threat Model](../docs/security/THREAT_MODEL.md) for your use case

#### Configuration Security
- Use strong, unique trust domains (not `example.org`)
- Implement proper SPIRE server hardening
- Enable security logging and monitoring
- Regularly rotate SPIFFE certificates
- Follow principle of least privilege for service authorizations

#### Development Security
- Review interceptor configurations carefully
- Validate all input parameters
- Implement proper error handling without information disclosure
- Use the built-in security interceptors
- Follow secure coding practices

### Contact Information

- **Security Email**: security@sufield.com
- **General Contact**: maintainer@sufield.com
- **Project Repository**: https://github.com/sufield/ephemos
- **Security Documentation**: [docs/security/](../docs/security/)

### Legal

This security policy is provided "as is" without warranty. The Ephemos team reserves the right to modify this policy at any time. By reporting security vulnerabilities, you agree to these terms and the responsible disclosure process outlined above.

---

**Thank you for helping keep Ephemos and our users secure.**

*Last updated: August 2025*