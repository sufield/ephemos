# CI/CD Verification Tests

## Purpose
This document explains the intentionally failing tests added to verify CI/CD pipeline coverage.

## 🔥 IMPORTANT: These tests are INTENTIONALLY FAILING
These tests were added to verify that the CI/CD pipeline actually runs tests across all packages.
They should be **REMOVED** once CI/CD verification is complete.

## Test Files Added

### 1. `/pkg/ephemos/ci_verification_test.go`
- **Purpose**: Verify unit tests run in main package
- **Failures**:
  - Compilation error: `UndefinedType` 
  - Runtime error: Test failure message

### 2. `/internal/adapters/primary/api/ci_verification_test.go`
- **Purpose**: Verify internal package tests run
- **Failures**:
  - Compilation error: `nonExistentFunction()`
  - Runtime error: Test failure message

### 3. `/internal/integration/ci_verification_test.go`
- **Purpose**: Verify integration and vertical tests run
- **Failures**:
  - Integration test failure
  - Vertical test failure
  - Benchmark compilation check

### 4. `/cmd/ephemos-cli/ci_verification_test.go`
- **Purpose**: Verify CLI tests run
- **Failures**:
  - Fatal test failure
  - Assertion failure

### 5. `/examples/echo-server/ci_verification_test.go`
- **Purpose**: Verify example tests run
- **Failures**:
  - Fatal test failure
  - Math assertion failure

## Expected CI/CD Behavior

When these tests are present, the CI/CD should:

1. **Build Job**: Fail during "Verify test compilation" step due to compilation errors
2. **Test Job**: Fail during "Compile test files" step due to compilation errors  
3. **Test Job**: If compilation errors are fixed, fail during "Run tests" due to runtime failures

## Verification Steps

1. Push these changes to a branch
2. Create a PR to main/develop
3. Verify CI/CD pipeline fails with clear error messages
4. Check that failures occur in:
   - Build job (test compilation verification)
   - Test job (test execution)

## Cleanup

Once CI/CD is verified to be working correctly:

```bash
# Remove all verification test files
rm pkg/ephemos/ci_verification_test.go
rm internal/adapters/primary/api/ci_verification_test.go
rm internal/integration/ci_verification_test.go
rm cmd/ephemos-cli/ci_verification_test.go
rm examples/echo-server/ci_verification_test.go
rm CI_VERIFICATION_TESTS.md

# Commit the cleanup
git add -A
git commit -m "Remove CI/CD verification tests - pipeline confirmed working"
```

## CI/CD Improvements Made

1. **Added test compilation check** in Build job to catch compilation errors early
2. **Added explicit test compilation step** in Test job before running tests
3. **Enhanced error reporting** with clear messages for compilation failures

These improvements ensure that:
- Compilation errors in test files are caught
- Tests actually run across all packages
- Failures are reported clearly with package information