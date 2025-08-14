# GitHub Workflow Permissions Security

This document describes the workflow permission security improvements made to address excessive permissions warnings and implement security best practices.

## Security Issue Addressed

### Original Problem
OpenSSF Scorecard detected excessive permissions:
- Multiple workflows had no top-level permission restrictions
- Job-level permissions existed but no workflow-level constraints
- Default permissions granted broad access unnecessarily

### Security Risk
Workflows without explicit permission restrictions inherit default permissions which may include:
- Write access to repository contents
- Ability to create/modify issues and PRs
- Access to Actions and other sensitive operations

## Solution Implemented

### 1. Top-Level Permission Restrictions

All workflows now have explicit top-level permissions to follow the principle of least privilege:

#### Read-Only Workflows
```yaml
# For workflows that only need to read code
permissions:
  contents: read
```

**Applied to:**
- `ci.yml` - CI pipeline (disabled)
- `docs-and-release.yml` - Documentation validation
- `performance.yml` - Performance testing
- `sast-scan.yml` - Static analysis scanning

#### Deny-All Workflows
```yaml
# For workflows with jobs that specify their own permissions
permissions: {}
```

**Applied to:**
- `renovate.yml` - Job specifies needed permissions
- `secrets-scan.yml` - Jobs specify needed permissions  
- `fuzzing.yml` - Job specifies needed permissions

#### Pre-Configured Workflows
```yaml
# Scorecard workflow already had secure configuration
permissions: read-all  # Read-only access to everything
```

**Applied to:**
- `scorecard.yml` - OpenSSF Scorecard (already configured)

### 2. Job-Level Permission Review

Each job maintains minimal required permissions:

#### Renovate Job
```yaml
permissions:
  contents: read
  pull-requests: write
  issues: write
```

#### Secret Scanning Jobs
```yaml
permissions:
  contents: read
  security-events: write
  actions: read  # For some jobs
```

#### Fuzzing Job
```yaml
permissions:
  contents: read
  security-events: write
```

## Permission Types Explained

### contents
- `read`: Read repository code, files, and history
- `write`: Create/modify files, create releases

### security-events
- `write`: Upload security scan results (SARIF) to GitHub Security tab

### actions
- `read`: Read workflow run information and artifacts

### pull-requests
- `write`: Create and modify pull requests

### issues
- `write`: Create and modify issues

### id-token
- `write`: Request OIDC token for authentication with external services

## Validation Results

### Before Changes
```
Warn: no topLevel permission defined: .github/workflows/ci.yml:1
Warn: no topLevel permission defined: .github/workflows/docs-and-release.yml:1
Warn: no topLevel permission defined: .github/workflows/performance.yml:1
Warn: no topLevel permission defined: .github/workflows/renovate.yml:1
Warn: no topLevel permission defined: .github/workflows/sast-scan.yml:1
Warn: no topLevel permission defined: .github/workflows/secrets-scan.yml:1
```

### After Changes
```
✅ All workflows have explicit top-level permissions
✅ No excessive permissions detected
✅ Principle of least privilege enforced
```

## Security Best Practices Implemented

### 1. Explicit Permission Declaration
- Every workflow explicitly declares required permissions
- No reliance on default permission inheritance

### 2. Least Privilege Principle
- Workflows get only the minimum permissions needed
- Job-level permissions further restrict access when needed

### 3. Defense in Depth
- Top-level permissions provide broad restrictions
- Job-level permissions provide specific allowances
- Two-layer security model

### 4. Read-Only Default
- Most workflows only need read access
- Write permissions explicitly granted only when necessary

## Workflow-Specific Permission Rationale

### CI Workflow
- **Permission**: `contents: read`
- **Rationale**: Only needs to read code for testing (currently disabled)

### Documentation & Release
- **Permission**: `contents: read`
- **Rationale**: Only needs to read docs and markdown files for validation

### Performance Testing
- **Permission**: `contents: read`
- **Rationale**: Only needs to read code for benchmarking and profiling

### SAST Scanning
- **Permission**: `contents: read`
- **Rationale**: Only needs to read code for static analysis

### Renovate
- **Top-level**: `permissions: {}`
- **Job-level**: Limited write access for dependency updates
- **Rationale**: Needs to create PRs but access is restricted at job level

### Secret Scanning
- **Top-level**: `permissions: {}`
- **Job-level**: Read code + upload security results
- **Rationale**: Multiple jobs with different permission needs

### Fuzzing
- **Top-level**: `permissions: {}`
- **Job-level**: Read code + upload security results
- **Rationale**: ClusterFuzzLite job needs to upload SARIF results

### Scorecard
- **Permission**: `read-all`
- **Rationale**: Needs broad read access for comprehensive security analysis

## Monitoring and Maintenance

### Regular Permission Audits
1. Run OpenSSF Scorecard monthly to check for permission issues
2. Review workflow permissions when adding new jobs
3. Validate that permissions match actual workflow needs

### Permission Update Process
1. Start with minimal permissions
2. Add specific permissions only when jobs fail due to insufficient access
3. Document the reason for each permission requirement
4. Regular review and cleanup of unused permissions

### Security Alerts
- GitHub security alerts will notify of permission-related security issues
- Dependabot will help keep action versions updated with latest security fixes
- Regular security reviews should include permission analysis

This permission security model ensures that GitHub workflows operate with minimal required access while maintaining full functionality for CI/CD operations.