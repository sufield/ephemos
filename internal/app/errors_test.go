package app_test

import (
	"errors"
	"strings"
	"testing"

	coreerrors "github.com/sufield/ephemos/internal/core/errors"
)

func TestDomainError_Error(t *testing.T) {
	tests := []struct {
		name        string
		domainError *coreerrors.DomainError
		want        string
	}{
		{
			name: "simple error",
			domainError: &coreerrors.DomainError{
				Code:    "INVALID_DOMAIN",
				Message: "Domain cannot be empty",
			},
			want: "INVALID_DOMAIN: Domain cannot be empty",
		},
		{
			name: "error with wrapped error",
			domainError: &coreerrors.DomainError{
				Code:    "CONNECTION_FAILED",
				Message: "Failed to connect to SPIRE",
				Err:     errors.New("connection refused"),
			},
			want: "CONNECTION_FAILED: Failed to connect to SPIRE: connection refused",
		},
		{
			name: "empty message",
			domainError: &coreerrors.DomainError{
				Code:    "UNKNOWN",
				Message: "",
			},
			want: "UNKNOWN: ",
		},
		{
			name: "empty code",
			domainError: &coreerrors.DomainError{
				Code:    "",
				Message: "Something went wrong",
			},
			want: ": Something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.domainError.Error()
			if got != tt.want {
				t.Errorf("DomainError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDomainError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	domainErr := &coreerrors.DomainError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
		Err:     originalErr,
	}

	unwrapped := domainErr.Unwrap()
	if !errors.Is(unwrapped, originalErr) {
		t.Errorf("DomainError.Unwrap() = %v, want %v", unwrapped, originalErr)
	}

	// Test with no wrapped error
	domainErrNoWrap := &coreerrors.DomainError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
	}

	unwrapped = domainErrNoWrap.Unwrap()
	if unwrapped != nil {
		t.Errorf("DomainError.Unwrap() = %v, want nil", unwrapped)
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name            string
		validationError *coreerrors.ValidationError
		want            string
	}{
		{
			name: "string field",
			validationError: &coreerrors.ValidationError{
				Field:   "domain",
				Value:   "example.com",
				Message: "invalid domain format",
			},
			want: "validation failed for field 'domain' with value 'example.com': invalid domain format",
		},
		{
			name: "integer field",
			validationError: &coreerrors.ValidationError{
				Field:   "port",
				Value:   8080,
				Message: "port must be between 1 and 65535",
			},
			want: "validation failed for field 'port' with value '8080': port must be between 1 and 65535",
		},
		{
			name: "nil value",
			validationError: &coreerrors.ValidationError{
				Field:   "config",
				Value:   nil,
				Message: "configuration cannot be nil",
			},
			want: "validation failed for field 'config' with value '<nil>': configuration cannot be nil",
		},
		{
			name: "empty message",
			validationError: &coreerrors.ValidationError{
				Field:   "test",
				Value:   "value",
				Message: "",
			},
			want: "validation failed for field 'test' with value 'value': ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.validationError.Error()
			if got != tt.want {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewDomainError(t *testing.T) {
	code := "TEST_CODE"
	message := "Test message"
	err := errors.New("wrapped error")

	baseDomainError := &coreerrors.DomainError{
		Code:    code,
		Message: message,
	}

	resultErr := coreerrors.NewDomainError(baseDomainError, err)
	var domainErr *coreerrors.DomainError
	if !errors.As(resultErr, &domainErr) {
		t.Fatalf("coreerrors.NewDomainError() did not return a DomainError")
	}

	if domainErr.Code != code {
		t.Errorf("coreerrors.NewDomainError() code = %v, want %v", domainErr.Code, code)
	}

	if domainErr.Message != message {
		t.Errorf("coreerrors.NewDomainError() message = %v, want %v", domainErr.Message, message)
	}

	if !errors.Is(domainErr.Err, err) {
		t.Errorf("coreerrors.NewDomainError() err = %v, want %v", domainErr.Err, err)
	}
}

func TestNewValidationError(t *testing.T) {
	field := "testField"
	value := "testValue"
	message := "Test validation message"

	validationErr := coreerrors.NewValidationError(field, value, message)

	if validationErr.Field != field {
		t.Errorf("coreerrors.NewValidationError() field = %v, want %v", validationErr.Field, field)
	}

	if validationErr.Value != value {
		t.Errorf("coreerrors.NewValidationError() value = %v, want %v", validationErr.Value, value)
	}

	if validationErr.Message != message {
		t.Errorf("coreerrors.NewValidationError() message = %v, want %v", validationErr.Message, message)
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that errors can be properly wrapped and unwrapped
	originalErr := errors.New("original error")
	baseDomainError := &coreerrors.DomainError{
		Code:    "WRAPPER",
		Message: "Wrapped error",
	}
	domainErr := coreerrors.NewDomainError(baseDomainError, originalErr)

	// Test errors.Is
	if !errors.Is(domainErr, originalErr) {
		t.Error("errors.Is should recognize wrapped error")
	}

	// Test errors.As
	var targetErr *coreerrors.DomainError
	if !errors.As(domainErr, &targetErr) {
		t.Error("errors.As should extract DomainError")
	}

	if !errors.Is(targetErr.Err, originalErr) {
		t.Errorf("errors.As extracted wrong wrapped error: got %v, want %v", targetErr.Err, originalErr)
	}
}

func TestErrorTypes(t *testing.T) {
	// Test type assertions
	baseDomainError := &coreerrors.DomainError{
		Code:    "TEST",
		Message: "Test message",
	}
	domainErr := coreerrors.NewDomainError(baseDomainError, nil)
	validationErr := coreerrors.NewValidationError("field", "value", "message")

	// Test DomainError type assertion
	var testDomainErr *coreerrors.DomainError
	if !errors.As(domainErr, &testDomainErr) {
		t.Error("DomainError should be of type *coreerrors.DomainError")
	}

	// Test ValidationError type assertion via error interface
	validationErrInterface := error(validationErr)
	var testValidationErr *coreerrors.ValidationError
	if !errors.As(validationErrInterface, &testValidationErr) {
		t.Error("ValidationError should be of type *coreerrors.ValidationError")
	}

	// Cross-type assertions should fail
	var crossTestValidationErr *coreerrors.ValidationError
	if errors.As(domainErr, &crossTestValidationErr) {
		t.Error("DomainError should not be of type *coreerrors.ValidationError")
	}

	domainErrInterface := domainErr
	var crossTestValidationErr2 *coreerrors.ValidationError
	if errors.As(domainErrInterface, &crossTestValidationErr2) {
		t.Error("DomainError should not be of type *coreerrors.ValidationError when cast to error interface")
	}
}

func TestErrorChaining(t *testing.T) {
	// Test chaining multiple errors
	level1Err := errors.New("level 1 error")

	level2Base := &coreerrors.DomainError{
		Code:    "LEVEL2",
		Message: "Level 2 error",
	}
	level2Err := coreerrors.NewDomainError(level2Base, level1Err)

	level3Base := &coreerrors.DomainError{
		Code:    "LEVEL3",
		Message: "Level 3 error",
	}
	level3Err := coreerrors.NewDomainError(level3Base, level2Err)

	// Should be able to find the original error
	if !errors.Is(level3Err, level1Err) {
		t.Error("Should be able to find level 1 error in chain")
	}

	// Should be able to find intermediate error
	if !errors.Is(level3Err, level2Err) {
		t.Error("Should be able to find level 2 error in chain")
	}

	// Error message should contain all levels
	errStr := level3Err.Error()
	if !strings.Contains(errStr, "Level 3 error") {
		t.Error("Error string should contain level 3 message")
	}

	if !strings.Contains(errStr, "Level 2 error") {
		t.Error("Error string should contain level 2 message")
	}

	if !strings.Contains(errStr, "level 1 error") {
		t.Error("Error string should contain level 1 message")
	}
}

func BenchmarkDomainError_Error(b *testing.B) {
	baseDomainError := &coreerrors.DomainError{
		Code:    "BENCHMARK",
		Message: "Benchmark error message",
	}
	domainErr := coreerrors.NewDomainError(baseDomainError, errors.New("wrapped"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = domainErr.Error()
	}
}

func BenchmarkValidationError_Error(b *testing.B) {
	validationErr := coreerrors.NewValidationError("field", "value", "validation message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validationErr.Error()
	}
}

func BenchmarkNewDomainError(b *testing.B) {
	wrappedErr := errors.New("wrapped error")
	baseDomainError := &coreerrors.DomainError{
		Code:    "CODE",
		Message: "message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = coreerrors.NewDomainError(baseDomainError, wrappedErr)
	}
}

func BenchmarkNewValidationError(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = coreerrors.NewValidationError("field", "value", "message")
	}
}

func TestErrorInterface(t *testing.T) {
	// Test that our custom errors implement the error interface
	baseDomainError := &coreerrors.DomainError{
		Code:    "TEST",
		Message: "Test message",
	}
	domainErr := coreerrors.NewDomainError(baseDomainError, nil)
	// Verify it implements error interface by calling Error()
	if domainErr.Error() == "" {
		t.Error("DomainError should have non-empty error message")
	}

	validationErr := coreerrors.NewValidationError("field", "value", "message")
	// Verify it implements error interface by calling Error()
	if validationErr.Error() == "" {
		t.Error("ValidationError should have non-empty error message")
	}

	// Test that both can be assigned to error interface
	var err error
	err = domainErr
	_ = err // Use the variable to avoid unused variable error
	err = validationErr
	_ = err // Use the variable to avoid unused variable error
}
