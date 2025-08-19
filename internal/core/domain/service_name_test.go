package domain

import (
	"testing"
)

func TestNewServiceName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid service name",
			input:     "payment-service",
			wantError: false,
		},
		{
			name:      "valid service name with underscores",
			input:     "user_management_service",
			wantError: false,
		},
		{
			name:      "valid service name with dots",
			input:     "api.v1.service",
			wantError: false,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: true,
			errorMsg:  "service name cannot be empty or whitespace-only",
		},
		{
			name:      "whitespace only",
			input:     "   ",
			wantError: true,
			errorMsg:  "service name cannot be empty or whitespace-only",
		},
		{
			name:      "too long",
			input:     "this-is-a-very-long-service-name-that-exceeds-the-maximum-allowed-length-of-100-characters-for-service-names",
			wantError: true,
			errorMsg:  "service name too long",
		},
		{
			name:      "invalid characters",
			input:     "service@name!",
			wantError: true,
			errorMsg:  "service name contains invalid characters",
		},
		{
			name:      "starts with hyphen",
			input:     "-invalid-service",
			wantError: true,
			errorMsg:  "service name contains invalid characters",
		},
		{
			name:      "ends with hyphen",
			input:     "invalid-service-",
			wantError: true,
			errorMsg:  "service name contains invalid characters",
		},
		{
			name:      "contains example",
			input:     "example-service",
			wantError: true,
			errorMsg:  "service name cannot contain 'example'",
		},
		{
			name:      "contains test in middle without proper separation",
			input:     "mytestingservice",
			wantError: true,
			errorMsg:  "service name should not contain 'test'",
		},
		{
			name:      "valid test service ending",
			input:     "payment-service-test",
			wantError: false,
		},
		{
			name:      "valid test service starting",
			input:     "test-service",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewServiceName(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewServiceName(%q) expected error, got nil", tt.input)
					return
				}
				if tt.errorMsg != "" && !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("NewServiceName(%q) error = %v, expected to contain %q", tt.input, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("NewServiceName(%q) unexpected error: %v", tt.input, err)
					return
				}
				if result.Value() != tt.input {
					t.Errorf("NewServiceName(%q) value = %q, expected %q", tt.input, result.Value(), tt.input)
				}
			}
		})
	}
}

func TestServiceNameMethods(t *testing.T) {
	serviceName, err := NewServiceName("payment-service")
	if err != nil {
		t.Fatalf("Failed to create ServiceName: %v", err)
	}

	// Test String() method
	if serviceName.String() != "payment-service" {
		t.Errorf("String() = %q, expected %q", serviceName.String(), "payment-service")
	}

	// Test Value() method
	if serviceName.Value() != "payment-service" {
		t.Errorf("Value() = %q, expected %q", serviceName.Value(), "payment-service")
	}

	// Test Length() method
	if serviceName.Length() != 15 {
		t.Errorf("Length() = %d, expected 15", serviceName.Length())
	}

	// Test IsEmpty() method
	if serviceName.IsEmpty() {
		t.Errorf("IsEmpty() = true, expected false")
	}

	// Test ToLower() method
	lower := serviceName.ToLower()
	if lower.Value() != "payment-service" {
		t.Errorf("ToLower().Value() = %q, expected %q", lower.Value(), "payment-service")
	}

	// Test Contains() method
	if !serviceName.Contains("payment") {
		t.Errorf("Contains('payment') = false, expected true")
	}
	if serviceName.Contains("auth") {
		t.Errorf("Contains('auth') = true, expected false")
	}

	// Test HasPrefix() method
	if !serviceName.HasPrefix("payment") {
		t.Errorf("HasPrefix('payment') = false, expected true")
	}
	if serviceName.HasPrefix("auth") {
		t.Errorf("HasPrefix('auth') = true, expected false")
	}

	// Test HasSuffix() method
	if !serviceName.HasSuffix("service") {
		t.Errorf("HasSuffix('service') = false, expected true")
	}
	if serviceName.HasSuffix("payment") {
		t.Errorf("HasSuffix('payment') = true, expected false")
	}

	// Test Equals() method
	other, _ := NewServiceName("payment-service")
	if !serviceName.Equals(other) {
		t.Errorf("Equals() = false, expected true for identical service names")
	}

	different, _ := NewServiceName("auth-service")
	if serviceName.Equals(different) {
		t.Errorf("Equals() = true, expected false for different service names")
	}
}

func TestServiceNameUnsafe(t *testing.T) {
	// Test that unsafe constructor doesn't validate
	unsafe := NewServiceNameUnsafe("")
	if !unsafe.IsEmpty() {
		t.Errorf("NewServiceNameUnsafe('') should create empty service name")
	}

	unsafe2 := NewServiceNameUnsafe("invalid@name!")
	if unsafe2.Value() != "invalid@name!" {
		t.Errorf("NewServiceNameUnsafe should accept invalid characters")
	}
}

func TestServiceNameProductionValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid production service",
			input:     "payment-service",
			wantError: false,
		},
		{
			name:      "contains demo",
			input:     "demo-service",
			wantError: true,
			errorMsg:  "service name contains 'demo': not suitable for production",
		},
		{
			name:      "contains example",
			input:     "example-api",
			wantError: true,
			errorMsg:  "service name contains 'example': not suitable for production",
		},
		{
			name:      "contains localhost",
			input:     "localhost-service",
			wantError: true,
			errorMsg:  "service name contains 'localhost': not suitable for production",
		},
		{
			name:      "ends with -test",
			input:     "service-test",
			wantError: true,
			errorMsg:  "service name ends with '-test': not suitable for production",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceName := NewServiceNameUnsafe(tt.input)
			err := serviceName.IsValidForProduction()

			if tt.wantError {
				if err == nil {
					t.Errorf("IsValidForProduction() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("IsValidForProduction() error = %v, expected to contain %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("IsValidForProduction() unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || (len(s) > 0 && s[len(s)-len(substr):] == substr) ||
		(len(s) >= len(substr) && s[:len(substr)] == substr) ||
		(len(s) > len(substr) && containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
