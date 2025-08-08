package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestDomainError_Error(t *testing.T) {
	tests := []struct {
		name        string
		domainError *DomainError
		want        string
	}{
		{
			name: "simple error",
			domainError: &DomainError{
				Code:    "INVALID_DOMAIN",
				Message: "Domain cannot be empty",
			},
			want: "INVALID_DOMAIN: Domain cannot be empty",
		},
		{
			name: "error with wrapped error",
			domainError: &DomainError{
				Code:    "CONNECTION_FAILED",
				Message: "Failed to connect to SPIRE",
				Err:     errors.New("connection refused"),
			},
			want: "CONNECTION_FAILED: Failed to connect to SPIRE: connection refused",
		},
		{
			name: "empty message",
			domainError: &DomainError{
				Code:    "UNKNOWN",
				Message: "",
			},
			want: "UNKNOWN: ",
		},
		{
			name: "empty code",
			domainError: &DomainError{
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
	domainErr := &DomainError{
		Code:    "TEST_ERROR",
		Message: "Test error message",
		Err:     originalErr,
	}

	unwrapped := domainErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("DomainError.Unwrap() = %v, want %v", unwrapped, originalErr)
	}

	// Test with no wrapped error
	domainErrNoWrap := &DomainError{
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
		validationError *ValidationError
		want            string
	}{
		{
			name: "string field",
			validationError: &ValidationError{
				Field:   "domain",
				Value:   "example.com",
				Message: "invalid domain format",
			},
			want: "validation failed for field 'domain' with value 'example.com': invalid domain format",
		},
		{
			name: "integer field",
			validationError: &ValidationError{
				Field:   "port",
				Value:   8080,
				Message: "port must be between 1 and 65535",
			},
			want: "validation failed for field 'port' with value '8080': port must be between 1 and 65535",
		},
		{
			name: "nil value",
			validationError: &ValidationError{
				Field:   "config",
				Value:   nil,
				Message: "configuration cannot be nil",
			},
			want: "validation failed for field 'config' with value '<nil>': configuration cannot be nil",
		},
		{
			name: "empty message",
			validationError: &ValidationError{
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

	baseDomainError := &DomainError{
		Code:    code,
		Message: message,
	}
	
	resultErr := NewDomainError(baseDomainError, err)
	domainErr := resultErr.(*DomainError)

	if domainErr.Code != code {
		t.Errorf("NewDomainError() code = %v, want %v", domainErr.Code, code)
	}

	if domainErr.Message != message {
		t.Errorf("NewDomainError() message = %v, want %v", domainErr.Message, message)
	}

	if domainErr.Err != err {
		t.Errorf("NewDomainError() err = %v, want %v", domainErr.Err, err)
	}
}

func TestNewValidationError(t *testing.T) {
	field := "testField"
	value := "testValue"
	message := "Test validation message"

	validationErr := NewValidationError(field, value, message)

	if validationErr.Field != field {
		t.Errorf("NewValidationError() field = %v, want %v", validationErr.Field, field)
	}

	if validationErr.Value != value {
		t.Errorf("NewValidationError() value = %v, want %v", validationErr.Value, value)
	}

	if validationErr.Message != message {
		t.Errorf("NewValidationError() message = %v, want %v", validationErr.Message, message)
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test that errors can be properly wrapped and unwrapped
	originalErr := errors.New("original error")
	baseDomainError := &DomainError{
		Code:    "WRAPPER",
		Message: "Wrapped error",
	}
	domainErr := NewDomainError(baseDomainError, originalErr)

	// Test errors.Is
	if !errors.Is(domainErr, originalErr) {
		t.Error("errors.Is should recognize wrapped error")
	}

	// Test errors.As
	var targetErr *DomainError
	if !errors.As(domainErr, &targetErr) {
		t.Error("errors.As should extract DomainError")
	}

	if targetErr.Err != originalErr {
		t.Errorf("errors.As extracted wrong wrapped error: got %v, want %v", targetErr.Err, originalErr)
	}
}

func TestErrorTypes(t *testing.T) {
	// Test type assertions
	baseDomainError := &DomainError{
		Code:    "TEST",
		Message: "Test message",
	}
	domainErr := NewDomainError(baseDomainError, nil)
	validationErr := NewValidationError("field", "value", "message")

	// Test DomainError type assertion
	if _, ok := domainErr.(*DomainError); !ok {
		t.Error("DomainError should be of type *DomainError")
	}

	// Test ValidationError type assertion via error interface
	var validationErrInterface error = validationErr
	if _, ok := validationErrInterface.(*ValidationError); !ok {
		t.Error("ValidationError should be of type *ValidationError")
	}

	// Cross-type assertions should fail
	if _, ok := domainErr.(*ValidationError); ok {
		t.Error("DomainError should not be of type *ValidationError")
	}

	var domainErrInterface error = domainErr
	if _, ok := domainErrInterface.(*ValidationError); ok {
		t.Error("DomainError should not be of type *ValidationError when cast to error interface")
	}
}

func TestErrorChaining(t *testing.T) {
	// Test chaining multiple errors
	level1Err := errors.New("level 1 error")
	
	level2Base := &DomainError{
		Code:    "LEVEL2",
		Message: "Level 2 error",
	}
	level2Err := NewDomainError(level2Base, level1Err)
	
	level3Base := &DomainError{
		Code:    "LEVEL3", 
		Message: "Level 3 error",
	}
	level3Err := NewDomainError(level3Base, level2Err)

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
	baseDomainError := &DomainError{
		Code:    "BENCHMARK",
		Message: "Benchmark error message",
	}
	domainErr := NewDomainError(baseDomainError, errors.New("wrapped"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = domainErr.Error()
	}
}

func BenchmarkValidationError_Error(b *testing.B) {
	validationErr := NewValidationError("field", "value", "validation message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validationErr.Error()
	}
}

func BenchmarkNewDomainError(b *testing.B) {
	wrappedErr := errors.New("wrapped error")
	baseDomainError := &DomainError{
		Code:    "CODE",
		Message: "message",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewDomainError(baseDomainError, wrappedErr)
	}
}

func BenchmarkNewValidationError(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewValidationError("field", "value", "message")
	}
}

func TestErrorInterface(t *testing.T) {
	// Test that our custom errors implement the error interface
	var err error

	baseDomainError := &DomainError{
		Code:    "TEST",
		Message: "Test message",
	}
	err = NewDomainError(baseDomainError, nil)
	if err == nil {
		t.Error("DomainError should implement error interface")
	}

	err = NewValidationError("field", "value", "message")
	if err == nil {
		t.Error("ValidationError should implement error interface")
	}
}