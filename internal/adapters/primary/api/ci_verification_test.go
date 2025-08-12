package api_test

import (
	"testing"
)

// TestCIVerification_InternalTest is an intentionally failing test to verify CI/CD runs internal tests
func TestCIVerification_InternalTest(t *testing.T) {
	t.Error("🔥 INTENTIONAL FAILURE: This test verifies that CI/CD runs tests in internal/adapters/primary/api")
}

// TestCIVerification_CompilationCheck verifies the CI catches compilation issues in test files
func TestCIVerification_CompilationCheck(t *testing.T) {
	// Intentional compilation error
	nonExistentFunction()
	t.Error("🔥 This should never run due to compilation error")
}