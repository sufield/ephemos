package integration

import (
	"testing"
)

// TestCIVerification_IntegrationTest is an intentionally failing test to verify CI/CD runs integration tests
func TestCIVerification_IntegrationTest(t *testing.T) {
	t.Error("🔥 INTENTIONAL FAILURE: This test verifies that CI/CD runs integration tests in internal/integration")
}

// TestCIVerification_VerticalTest simulates a vertical/end-to-end test failure
func TestCIVerification_VerticalTest(t *testing.T) {
	t.Error("🔥 INTENTIONAL FAILURE: This test verifies that CI/CD runs vertical integration tests")
}

// BenchmarkCIVerification ensures benchmarks are also compiled
func BenchmarkCIVerification(b *testing.B) {
	b.Error("🔥 INTENTIONAL FAILURE: This benchmark verifies that CI/CD compiles benchmark tests")
}