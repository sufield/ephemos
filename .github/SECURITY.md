# Security Policy

## Supported Versions

We take security seriously and provide security updates for the following versions of Ephemos:

| Version | Supported          |
| ------- | ------------------ |
| 1.x.x   | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

We appreciate responsible disclosure of security vulnerabilities. If you discover a security issue, please follow these steps:

### 🔒 Private Reporting (Preferred)

1. **Email**: Send details to security@sufield.com
2. **Subject**: Include "[SECURITY] Ephemos Vulnerability Report"
3. **Include**: 
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact assessment
   - Your contact information

### 📋 What to Include

Please provide as much information as possible:

- **Vulnerability Type**: Authentication bypass, injection, etc.
- **Affected Components**: SPIFFE integration, gRPC interceptors, configuration handling
- **Severity Assessment**: Your evaluation of the impact
- **Reproduction Steps**: Clear instructions to reproduce
- **Proof of Concept**: If applicable (non-destructive only)
- **Suggested Fix**: If you have recommendations

### ⏱️ Response Timeline

We are committed to responding promptly:

- **Acknowledgment**: Within 48 hours
- **Initial Assessment**: Within 5 business days  
- **Resolution Timeline**: Based on severity assessment
- **Public Disclosure**: Coordinated with reporter

### 🎯 Severity Categories

**Critical (Response: Immediate)**
- Remote code execution
- Authentication bypass
- Service impersonation via SPIFFE
- Production credential exposure

**High (Response: 1-3 days)**
- Privilege escalation
- Data exposure
- Denial of service
- Certificate validation bypass

**Medium (Response: 1-2 weeks)**
- Information disclosure
- Configuration vulnerabilities
- Logging sensitive data

**Low (Response: 1 month)**
- Documentation issues
- Non-security configuration problems

### 🛡️ Scope

**In Scope:**
- SPIFFE/SPIRE identity handling
- gRPC interceptor security
- Configuration validation
- Authentication and authorization
- Certificate management
- Secrets handling
- Docker/container security

**Out of Scope:**
- Demo configurations (use example.org)
- Social engineering
- Physical attacks
- DoS attacks requiring excessive resources
- Issues in third-party dependencies (report to upstream)

### 🏆 Recognition

We believe in recognizing security researchers:

- **Hall of Fame**: Public recognition (with permission)
- **Attribution**: Credit in security advisories
- **Swag**: Ephemos security researcher merchandise
- **Monetary Rewards**: For critical vulnerabilities (case by case)

### 📊 Security Measures

Ephemos implements multiple security layers:

- **Identity-Based Authentication**: SPIFFE/SPIRE integration
- **mTLS Transport Security**: End-to-end encryption
- **Configuration Validation**: Production security checks
- **Secrets Scanning**: Automated secret detection
- **Dependency Management**: Automated vulnerability updates
- **Supply Chain Security**: SBOM generation and validation

### 🔍 Security Testing

We welcome security research on:

- **Authentication bypass**: Attempts to circumvent SPIFFE authentication
- **Authorization flaws**: Access control vulnerabilities
- **Configuration injection**: Malicious configuration exploitation
- **Certificate validation**: X.509 and SPIFFE certificate handling
- **Interceptor bypass**: Security interceptor circumvention

### ⚖️ Legal

This security policy operates under:

- **Safe Harbor**: Good faith security research is welcome
- **Responsible Disclosure**: Coordinated vulnerability disclosure
- **No Legal Action**: We will not pursue legal action for responsible research
- **Attribution**: We may publicly credit researchers (with permission)

### 📞 Contact Information

**Security Team**: security@sufield.com  
**PGP Key**: Available upon request  
**Response Time**: Business hours (UTC-5)

### 🔄 Updates

This security policy may be updated periodically. Check back regularly for the latest version.

---

**Thank you for helping keep Ephemos secure!**

*Last updated: August 2025*