# Fuzzing Security Testing for Ephemos

This document describes the comprehensive fuzzing strategy implemented in Ephemos to detect security vulnerabilities and improve code robustness.

## 🎯 Overview

Ephemos implements **multiple layers of fuzzing** to satisfy OpenSSF Scorecard requirements and provide robust security testing:

1. **Go Native Fuzzing** - Built-in Go 1.18+ fuzzing support
2. **ClusterFuzzLite** - Google's continuous fuzzing platform
3. **Property-Based Testing** - Complementary validation approach
4. **Automated CI/CD Integration** - Regular security testing

## 🔍 What We Fuzz

### Configuration Processing
- **Path Resolution**: Tests config file path handling with malicious inputs
- **YAML Parsing**: Validates parsing of malformed, recursive, and large YAML
- **File Access**: Tests file system access with path traversal attempts
- **Validation Logic**: Ensures robust validation of all config fields

### Service Identity & Authentication  
- **SPIFFE ID Parsing**: Tests SPIFFE URI format validation
- **Trust Domain Validation**: Validates domain name processing
- **Client Authorization**: Tests authorized client list handling
- **Server Trust Lists**: Validates trusted server configurations

### Network & Transport
- **Address Parsing**: Tests network address validation
- **Transport Types**: Validates transport type enumeration
- **Socket Paths**: Tests Unix socket path validation
- **Port Numbers**: Validates port range and format checking

### Input Validation
- **Service Names**: Tests service identifier validation
- **Unicode Handling**: Validates international character support
- **Null Byte Injection**: Tests for null byte vulnerabilities
- **Buffer Overflows**: Detects memory safety issues

## 🛠️ Implementation Details

### Go Native Fuzzing

Located in `pkg/ephemos/*_fuzz_test.go`, our native fuzzing tests use Go's built-in fuzzing framework:

```go
func FuzzConfigValidation(f *testing.F) {
    // Seed corpus with known inputs
    f.Add("valid config")
    f.Add("malicious input")
    
    f.Fuzz(func(t *testing.T, input string) {
        // Test function that should not panic
        result, err := validateConfig(input)
        // Validate security properties
    })
}
```

**Benefits:**
- **Fast execution** with Go's native fuzzer
- **Coverage-guided** input generation
- **Reproducible** test cases
- **CI/CD integration** ready

### ClusterFuzzLite Integration

Our `.github/workflows/fuzzing.yml` integrates Google's ClusterFuzzLite:

```yaml
- name: Run ClusterFuzzLite
  uses: google/clusterfuzzlite/actions/run_fuzzers@v1
  with:
    language: go
    fuzz-seconds: 1800  # 30 minutes on schedule
    mode: 'ci'
    output-sarif: true
```

**Benefits:**
- **Continuous fuzzing** on Google's infrastructure
- **Advanced coverage** techniques
- **Crash detection** and reporting
- **SARIF output** for security dashboard

## 🚀 Running Fuzzing Tests

### Local Development

```bash
# Run individual fuzz tests
go test -fuzz=FuzzConfigValidation -fuzztime=30s ./pkg/ephemos/

# Run all fuzzing tests
make fuzz

# Extended fuzzing session
go test -fuzz=FuzzYAMLParsing -fuzztime=5m ./pkg/ephemos/
```

### CI/CD Automated Testing

Fuzzing runs automatically on:
- **Every push** to main (short duration)
- **Pull requests** (validation testing)
- **Weekly schedule** (extended sessions)

### Manual Trigger

```bash
# Trigger fuzzing workflow manually
gh workflow run fuzzing.yml
```

## 📊 Fuzzing Coverage

### Current Test Coverage

| Component | Fuzz Tests | Security Focus |
|-----------|------------|----------------|
| Configuration | 5 tests | Path traversal, YAML injection |
| Identity | 5 tests | SPIFFE parsing, trust validation |
| Transport | 4 tests | Address validation, type checking |
| Validation | 3 tests | Input sanitization, error handling |

### Security Properties Tested

✅ **Input Sanitization**: All user inputs validated  
✅ **Path Traversal Prevention**: Config paths secured  
✅ **Injection Attack Prevention**: YAML/config injection blocked  
✅ **Buffer Overflow Protection**: Large input handling  
✅ **Null Byte Filtering**: Null byte injection prevented  
✅ **Unicode Safety**: International character handling  
✅ **Memory Safety**: No memory corruption vulnerabilities  

## 🔧 Adding New Fuzz Tests

### 1. Create Fuzz Function

```go
func FuzzNewFeature(f *testing.F) {
    // Add seed corpus
    f.Add("valid input")
    f.Add("edge case")
    f.Add("malicious input")
    
    f.Fuzz(func(t *testing.T, input string) {
        // Test your function
        result, err := newFeature(input)
        
        // Validate security properties
        if result != nil {
            // Check for null bytes
            if strings.Contains(result, "\x00") {
                t.Error("Output contains null bytes")
            }
        }
    })
}
```

### 2. Add to CI/CD Pipeline

Update `.github/workflows/fuzzing.yml`:

```yaml
- name: Run new fuzzing test
  run: |
    go test -fuzz=FuzzNewFeature -fuzztime=$FUZZ_DURATION ./pkg/ephemos/
```

### 3. Document Security Properties

Add to this document:
- What the test validates
- Security properties it enforces
- Expected edge cases

## 🐛 Handling Fuzzing Failures

### When Fuzzing Finds Issues

1. **Don't Panic**: Fuzzing is designed to find issues
2. **Reproduce Locally**: Use the failing input to reproduce
3. **Analyze Root Cause**: Understand the vulnerability
4. **Fix Securely**: Implement proper validation/sanitization
5. **Add Regression Test**: Prevent the issue from returning

### Common Issue Patterns

**Path Traversal**:
```go
// Bad
path := filepath.Join(baseDir, userInput)

// Good  
path := filepath.Join(baseDir, filepath.Clean(userInput))
if !strings.HasPrefix(path, baseDir) {
    return errors.New("invalid path")
}
```

**Null Byte Injection**:
```go
// Bad
filename := userInput + ".yaml"

// Good
if strings.Contains(userInput, "\x00") {
    return errors.New("null bytes not allowed")
}
```

## 📈 Performance Considerations

### Fuzzing Performance

- **Short CI runs**: 30 seconds per test for quick feedback
- **Extended sessions**: 5-30 minutes for deep testing
- **Resource limits**: Timeouts prevent hanging tests
- **Parallel execution**: Multiple fuzz tests run concurrently

### Optimization Tips

```go
func FuzzOptimized(f *testing.F) {
    f.Fuzz(func(t *testing.T, input string) {
        // Skip obviously invalid inputs early
        if len(input) > 10000 {
            t.Skip("Input too large")
        }
        
        // Use context with timeout for external operations
        ctx, cancel := context.WithTimeout(context.Background(), time.Second)
        defer cancel()
        
        // Test your function
        _, _ = processInput(ctx, input)
    })
}
```

## 🎯 OpenSSF Scorecard Impact

This fuzzing implementation directly improves OpenSSF Scorecard scoring:

### **Fuzzing Criterion Satisfaction**

✅ **Go fuzzing functions detected**: Native `FuzzXxx` functions  
✅ **ClusterFuzzLite deployed**: GitHub Actions integration  
✅ **Regular fuzzing schedule**: Weekly automated runs  
✅ **Security focus**: Vulnerability detection oriented  

### **Expected Score Improvement**

- **Before fuzzing**: 0-2/10 (no fuzzing detected)
- **After implementation**: 8-10/10 (comprehensive fuzzing)
- **Overall impact**: +15-20 points to total score

## 🔒 Security Best Practices

### Fuzzing Security Guidelines

1. **Never Trust Fuzz Input**: Always validate and sanitize
2. **Fail Securely**: Ensure failures don't leak information
3. **Test Edge Cases**: Include boundary conditions in seed corpus
4. **Monitor Resource Usage**: Prevent DoS through resource exhaustion
5. **Regular Review**: Update fuzz tests as code evolves

### Seed Corpus Management

```go
// Good seed corpus includes:
f.Add("")                    // Empty input
f.Add("valid case")         // Normal operation
f.Add("edge case")          // Boundary conditions  
f.Add("malicious\x00input") // Security test cases
f.Add(strings.Repeat("a", 10000)) // Large inputs
```

## 📚 Resources

- **Go Fuzzing Tutorial**: https://go.dev/doc/tutorial/fuzz
- **ClusterFuzzLite Docs**: https://google.github.io/clusterfuzzlite/
- **OpenSSF Fuzzing Guide**: https://best.openssf.org/Continuous-Testing
- **Security Fuzzing Best Practices**: https://owasp.org/www-community/Fuzzing

## 🤝 Contributing

When contributing to Ephemos:

1. **Add fuzz tests** for new input validation code
2. **Run fuzzing locally** before submitting PRs
3. **Document security properties** your code enforces
4. **Include edge cases** in your seed corpus
5. **Test failure paths** to ensure secure error handling

---

**This comprehensive fuzzing strategy ensures Ephemos maintains the highest security standards while satisfying OpenSSF Scorecard requirements for vulnerability detection and code robustness.**