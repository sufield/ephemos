# YAML Validation Fixes

This document describes the YAML syntax errors that were identified and fixed across the repository's GitHub Actions workflows and configuration files.

## Issues Identified

### 1. Heredoc Indentation Problems

**Files Affected:**
- `.github/workflows/performance.yml` (line 134)
- `.github/workflows/fuzzing.yml` (line 167)

**Problem:**
Multi-line content within heredoc blocks (`<< 'EOF'`) was not properly indented within YAML `run: |` blocks. In YAML, all content within a literal block scalar must have consistent indentation.

**Error Message:**
```
yaml: line 167: could not find expected ':'
```

**Root Cause:**
When using heredoc syntax within GitHub Actions `run: |` blocks, the heredoc content must be indented to match the YAML structure. The original code had:

```yaml
run: |
  cat > file.go << 'EOF'
package main  # ❌ Not indented - breaks YAML parsing

func main() {  # ❌ Not indented - breaks YAML parsing
    // code
}
EOF
```

**Solution:**
Properly indent all content within the heredoc to match the YAML block structure:

```yaml
run: |
  cat > file.go << 'EOF'
  package main  # ✅ Properly indented

  func main() { # ✅ Properly indented  
      // code
  }
  EOF            # ✅ EOF marker also indented
```

### 2. Validation Tooling

**Tool Used:** [yq](https://github.com/mikefarah/yq) v4.47.1
- **Why yq?** Well-maintained, actively developed YAML processor with excellent validation capabilities
- **Installation:** `go install github.com/mikefarah/yq/v4@latest`
- **Alternative considered:** yamllint (Python-based, but not available in the environment)

## Files Fixed

### Performance Workflow
**File:** `.github/workflows/performance.yml`
**Lines Fixed:** 134-179 (first heredoc block)
**Content:** Go memory profiling test code

### Fuzzing Workflow  
**File:** `.github/workflows/fuzzing.yml`
**Lines Fixed:** 
- 167-207 (config fuzz target)
- 212-258 (server fuzz target)  
- 263-313 (identity fuzz target)
**Content:** ClusterFuzzLite fuzz target source code

## Validation Results

### Before Fixes
```
❌ .github/workflows/performance.yml - line 134 syntax error
❌ .github/workflows/fuzzing.yml - line 167 syntax error
```

### After Fixes
```
✅ .github/workflows/performance.yml
✅ .github/workflows/fuzzing.yml  
✅ All 12 YAML files validated successfully
```

## Automated Prevention

### New Validation Workflow
**File:** `.github/workflows/yaml-validation.yml`

**Features:**
- Runs on every push/PR that modifies YAML files
- Validates all `.yml` and `.yaml` files using yq
- Checks for heredoc indentation issues
- Runs actionlint validation for workflow files
- Generates validation summary

**Triggers:**
```yaml
on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]
    paths:
      - '**.yml'
      - '**.yaml'
```

**Validation Steps:**
1. **yq validation**: Ensures YAML syntax correctness
2. **ActionLint**: GitHub Actions workflow-specific validation
3. **Heredoc checks**: Custom validation for indentation issues
4. **Summary reporting**: Detailed validation results

## Files Validated

### GitHub Actions Files (12 files)
- ✅ `.github/workflows/ci.yml`
- ✅ `.github/workflows/codeql.yml`
- ✅ `.github/workflows/docs-and-release.yml`
- ✅ `.github/workflows/fuzzing.yml`
- ✅ `.github/workflows/performance.yml`
- ✅ `.github/workflows/renovate.yml`
- ✅ `.github/workflows/sast-scan.yml`
- ✅ `.github/workflows/scorecard.yml`
- ✅ `.github/workflows/secrets-scan.yml`
- ✅ `.github/workflows/yaml-validation.yml`
- ✅ `.github/dependabot.yml`
- ✅ `.github/codeql/codeql-config.yml`
- ✅ `.github/actions/setup-go/action.yml`

### Configuration Files (5 files)
- ✅ `./config/templates/k8s/configmap.yaml`
- ✅ `./config/transport-http.yaml`
- ✅ `./config/ephemos.yaml`
- ✅ `./.golangci.yml`
- ✅ `./.goreleaser.yaml`

## Best Practices

### 1. Heredoc in GitHub Actions
When using heredoc within `run: |` blocks:

```yaml
# ✅ Correct way
run: |
  cat > script.sh << 'EOF'
  #!/bin/bash
  echo "This is properly indented"
  EOF

# ❌ Incorrect way  
run: |
  cat > script.sh << 'EOF'
#!/bin/bash
echo "This breaks YAML parsing"
EOF
```

### 2. YAML Validation in CI
- Always validate YAML files in CI/CD pipelines
- Use multiple tools for comprehensive coverage (yq + actionlint)
- Run validation on file changes to catch issues early

### 3. Editor Configuration
Configure your editor to:
- Show whitespace and indentation
- Validate YAML syntax on save
- Use consistent indentation (2 spaces recommended)

## Troubleshooting

### Common YAML Errors

**Error:** `could not find expected ':'`
**Cause:** Inconsistent indentation in literal blocks
**Solution:** Ensure all content within `|` blocks has consistent indentation

**Error:** `yaml: line X: mapping values are not allowed in this context`
**Cause:** Missing quotes around strings containing special characters
**Solution:** Quote strings with `:`, `{`, `}`, `[`, `]`, etc.

### Validation Commands

**Manual validation:**
```bash
# Validate single file
yq eval . .github/workflows/example.yml

# Validate all YAML files
find . -name "*.yml" -o -name "*.yaml" | \
  xargs -I {} sh -c 'echo "Checking {}" && yq eval . "{}"'
```

**ActionLint validation:**
```bash
# Install actionlint
wget https://github.com/rhysd/actionlint/releases/latest/download/actionlint_linux_amd64.tar.gz
tar xf actionlint_linux_amd64.tar.gz

# Validate workflow
./actionlint .github/workflows/example.yml
```

## Impact

### Security Benefits
- **Supply Chain Security**: Valid workflows prevent malicious YAML injection
- **CI/CD Reliability**: Prevents workflow failures due to syntax errors
- **Maintainability**: Easier to debug and modify workflows

### Development Experience  
- **Faster Feedback**: Catch errors at commit time vs CI runtime
- **Reduced Debugging**: Clear error messages from validation tools
- **Consistent Quality**: Automated checks ensure high standards

## References

- [YAML Specification](https://yaml.org/spec/1.2.2/)
- [GitHub Actions Workflow Syntax](https://docs.github.com/en/actions/using-workflows/workflow-syntax-for-github-actions)
- [yq Documentation](https://mikefarah.gitbook.io/yq/)
- [ActionLint Documentation](https://github.com/rhysd/actionlint)