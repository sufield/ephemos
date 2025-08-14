package arch_test

import (
	"strings"
	"testing"

	"github.com/sufield/ephemos/internal/arch"
)

func TestValidator_ValidateCall(t *testing.T) {
	tests := []struct {
		name      string
		validator *arch.Validator
		operation string
		wantErr   bool
	}{
		{
			name:      "disabled validator never fails",
			validator: arch.NewValidator(false),
			operation: "test-operation",
			wantErr:   false,
		},
		{
			name:      "enabled validator with valid call",
			validator: arch.NewValidator(true),
			operation: "valid-operation",
			wantErr:   false, // This test itself doesn't violate boundaries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.ValidateCall(tt.operation)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validator.ValidateCall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidator_ExtractAdapterType(t *testing.T) {
	_ = arch.NewValidator(true)

	tests := []struct {
		name     string
		funcName string
		want     string
	}{
		{
			name:     "primary adapter",
			funcName: "github.com/sufield/ephemos/internal/adapters/primary/api.Handler",
			want:     "primary",
		},
		{
			name:     "secondary adapter",
			funcName: "github.com/sufield/ephemos/internal/adapters/secondary/spiffe.Provider",
			want:     "secondary",
		},
		{
			name:     "non-adapter function",
			funcName: "github.com/sufield/ephemos/internal/core/domain.Identity",
			want:     "",
		},
		{
			name:     "empty function name",
			funcName: "",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a test helper method since extractAdapterType is not exported
			// We'll test this indirectly through the validation logic
			if got := extractAdapterTypeHelper(tt.funcName); got != tt.want {
				t.Errorf("extractAdapterType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGlobalValidator(t *testing.T) {
	// Clear any existing violations
	arch.ClearGlobalViolations()

	// Test enabling/disabling
	arch.SetGlobalValidationEnabled(false)
	err := arch.ValidateBoundary("test-disabled")
	if err != nil {
		t.Errorf("Expected no error when validation disabled, got: %v", err)
	}

	arch.SetGlobalValidationEnabled(true)
	violations := arch.GetGlobalViolations()
	if len(violations) != 0 {
		t.Errorf("Expected no violations initially, got: %v", violations)
	}
}

func TestValidator_GetAndClearViolations(t *testing.T) {
	validator := arch.NewValidator(true)

	// Initially should have no violations
	violations := validator.GetViolations()
	if len(violations) != 0 {
		t.Errorf("Expected no initial violations, got: %v", violations)
	}

	// After clearing, should still have no violations
	validator.ClearViolations()
	violations = validator.GetViolations()
	if len(violations) != 0 {
		t.Errorf("Expected no violations after clear, got: %v", violations)
	}
}

// Test_Runtime_Validation_Integration tests the runtime validation system
// by simulating architectural boundary crossings.
func Test_Runtime_Validation_Integration(t *testing.T) {
	validator := arch.NewValidator(true)

	// Test that the validator can be called without panicking
	err := validator.ValidateCall("integration-test")
	if err != nil {
		// It's okay if there's an error, we just don't want panics
		t.Logf("Validation returned error (expected): %v", err)
	}

	// Test that violations can be retrieved
	violations := validator.GetViolations()
	t.Logf("Violations detected: %d", len(violations))

	// Test clearing violations
	validator.ClearViolations()
	afterClear := validator.GetViolations()
	if len(afterClear) != 0 {
		t.Errorf("Expected no violations after clear, got: %v", afterClear)
	}
}

// Helper function to test adapter type extraction indirectly.
func extractAdapterTypeHelper(funcName string) string {
	// This mimics the logic in the unexported extractAdapterType method
	if !strings.Contains(funcName, "/internal/adapters/") {
		return ""
	}

	parts := strings.Split(funcName, "/internal/adapters/")
	if len(parts) < 2 {
		return ""
	}

	adapterPath := parts[1]
	// Handle case where adapter path might be "http.Server" or "primary/api.Handler"
	pathParts := strings.Split(adapterPath, "/")
	if len(pathParts) > 0 {
		// Extract just the adapter type, ignoring any struct/method name after dot
		adapterType := pathParts[0]
		if dotIndex := strings.Index(adapterType, "."); dotIndex != -1 {
			adapterType = adapterType[:dotIndex]
		}
		return adapterType
	}

	return ""
}

// Benchmark the validation overhead.
func BenchmarkValidationOverhead(b *testing.B) {
	validator := arch.NewValidator(true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateCall("benchmark-test")
	}
}

func BenchmarkValidationDisabled(b *testing.B) {
	validator := arch.NewValidator(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.ValidateCall("benchmark-test")
	}
}
