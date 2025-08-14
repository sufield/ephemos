# GitHub Workflows Documentation

## Security Configuration Warnings

### Common Warning: "1 configuration not found"

When you see a warning like:

```
Warning: Code scanning cannot determine the alerts introduced by this pull request, 
because 1 configuration present on refs/heads/main was not found:
Actions workflow (security.yml)
❓ .github/workflows/security.yml:sast-scan
```

**This is expected behavior and not an error.** This warning occurs when:

1. **Workflow files are modified** - Even minor changes to workflow files can trigger this warning
2. **GitHub's scanning system comparison** - The system compares configurations between branches
3. **Timing synchronization** - There may be delays in GitHub's internal processing

### What this means:

- ✅ **Your security scanning is working correctly**
- ✅ **The sast-scan job will execute as configured**
- ✅ **CodeQL and Semgrep analysis will run**
- ❌ **This is NOT a configuration error**

### Resolution:

No action needed. The warning is informational and will resolve automatically after:
- The PR is merged to main
- GitHub's scanning system synchronizes
- Subsequent runs complete successfully

### Verification:

You can verify your security configuration is working by:
1. Checking that the `sast-scan` job runs successfully
2. Confirming CodeQL and Semgrep results are uploaded
3. Viewing results in the Security tab of your repository

## Workflow Overview

| Workflow | Purpose | Triggers |
|----------|---------|----------|
| `ci.yml` | Continuous integration tests | Push, PR |
| `performance.yml` | Benchmarks and memory profiling | Push, PR, schedule |
| `sast-scan.yml` | Static application security testing | Push, PR, schedule |
| `scorecard.yml` | OpenSSF Scorecard compliance | Schedule |
| `secrets-scan.yml` | Secret detection and prevention | Push, PR |

## Support

For questions about these workflows, consult:
- GitHub Actions documentation
- OpenSSF Security documentation  
- Project maintainers