package ephemos

import (
	"testing"
)

// TestCIVerification_UnitTest is an intentionally failing test to verify CI/CD runs unit tests
func TestCIVerification_UnitTest(t *testing.T) {
	t.Error("🔥 INTENTIONAL FAILURE: This test verifies that CI/CD runs unit tests in pkg/ephemos")
}

// TestCIVerification_CompilationError contains a compilation error to verify CI catches it
func TestCIVerification_CompilationError(t *testing.T) {
	// This should cause a compilation error
	var undefined UndefinedType
	_ = undefined
	t.Error("🔥 This line should never be reached due to compilation error above")
}