package main

import (
	"testing"
)

// TestCIVerification_CLITest is an intentionally failing test to verify CI/CD runs CLI tests
func TestCIVerification_CLITest(t *testing.T) {
	t.Fatal("🔥 INTENTIONAL FAILURE: This test verifies that CI/CD runs tests in cmd/ephemos-cli")
}

// TestCIVerification_MainPackage verifies main package tests are executed
func TestCIVerification_MainPackage(t *testing.T) {
	expected := "should pass"
	actual := "will fail"
	
	if expected != actual {
		t.Errorf("🔥 INTENTIONAL FAILURE: Expected %q but got %q - CI/CD should catch this", expected, actual)
	}
}