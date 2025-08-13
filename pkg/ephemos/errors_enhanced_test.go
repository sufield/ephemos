package ephemos

import (
	"errors"
	"strings"
	"testing"

	"github.com/joomcode/errorx"
)

func TestEnhancedErrors(t *testing.T) {
	t.Run("Enhanced Validation Error Creation", func(t *testing.T) {
		err := NewEnhancedValidationError("service.name", "invalid-name", "service name contains invalid characters")

		// Test error message
		expectedMsg := "service name contains invalid characters"
		if !strings.Contains(err.Error(), expectedMsg) {
			t.Errorf("Expected error message to contain %q, got %q", expectedMsg, err.Error())
		}

		// Test classification
		if !IsEnhancedValidationError(err) {
			t.Error("Expected error to be classified as enhanced validation error")
		}

		// Test properties
		field := GetEnhancedErrorField(err)
		if field != "service.name" {
			t.Errorf("Expected field %q, got %q", "service.name", field)
		}

		value := GetEnhancedErrorValue(err)
		if value != "invalid-name" {
			t.Errorf("Expected value %q, got %v", "invalid-name", value)
		}

		code := GetEnhancedErrorCode(err)
		if code != "VALIDATION_FAILED" {
			t.Errorf("Expected code %q, got %q", "VALIDATION_FAILED", code)
		}
	})

	t.Run("Enhanced Configuration Error", func(t *testing.T) {
		err := NewEnhancedConfigError("/path/to/config.yaml", "unable to parse YAML file")

		if !IsEnhancedConfigurationError(err) {
			t.Error("Expected error to be classified as enhanced configuration error")
		}

		file := GetEnhancedErrorFile(err)
		if file != "/path/to/config.yaml" {
			t.Errorf("Expected file %q, got %q", "/path/to/config.yaml", file)
		}

		code := GetEnhancedErrorCode(err)
		if code != "CONFIG_ERROR" {
			t.Errorf("Expected code %q, got %q", "CONFIG_ERROR", code)
		}
	})

	t.Run("Enhanced System Error", func(t *testing.T) {
		err := NewEnhancedSystemError("echo-server", "failed to connect to SPIRE agent")

		if !IsEnhancedSystemError(err) {
			t.Error("Expected error to be classified as enhanced system error")
		}

		service := GetEnhancedErrorService(err)
		if service != "echo-server" {
			t.Errorf("Expected service %q, got %q", "echo-server", service)
		}

		code := GetEnhancedErrorCode(err)
		if code != "SYSTEM_ERROR" {
			t.Errorf("Expected code %q, got %q", "SYSTEM_ERROR", code)
		}
	})

	t.Run("Enhanced Domain Error", func(t *testing.T) {
		err := NewEnhancedDomainError("CreateServerIdentity", "invalid service configuration")

		if !IsEnhancedDomainError(err) {
			t.Error("Expected error to be classified as enhanced domain error")
		}

		operation := GetEnhancedErrorOperation(err)
		if operation != "CreateServerIdentity" {
			t.Errorf("Expected operation %q, got %q", "CreateServerIdentity", operation)
		}

		code := GetEnhancedErrorCode(err)
		if code != "DOMAIN_ERROR" {
			t.Errorf("Expected code %q, got %q", "DOMAIN_ERROR", code)
		}
	})

	t.Run("Timeout and Temporary Errors", func(t *testing.T) {
		timeoutErr := NewTimeoutError("database_query", "query timed out after 30 seconds")
		tempErr := NewTemporaryError("api-gateway", "service temporarily unavailable")

		// Test timeout error
		if !IsTimeoutError(timeoutErr) {
			t.Error("Expected error to be classified as timeout error")
		}

		if !IsEnhancedSystemError(timeoutErr) {
			t.Error("Expected timeout error to also be a system error")
		}

		operation := GetEnhancedErrorOperation(timeoutErr)
		if operation != "database_query" {
			t.Errorf("Expected operation %q, got %q", "database_query", operation)
		}

		// Test temporary error
		if !IsTemporaryError(tempErr) {
			t.Error("Expected error to be classified as temporary error")
		}

		service := GetEnhancedErrorService(tempErr)
		if service != "api-gateway" {
			t.Errorf("Expected service %q, got %q", "api-gateway", service)
		}
	})

	t.Run("Predefined Enhanced Errors", func(t *testing.T) {
		testCases := []struct {
			name         string
			err          error
			expectedCode string
			namespace    func(error) bool
		}{
			{
				name:         "ErrEnhancedInvalidServiceName",
				err:          ErrEnhancedInvalidServiceName,
				expectedCode: "INVALID_SERVICE_NAME",
				namespace:    IsEnhancedDomainError,
			},
			{
				name:         "ErrEnhancedSPIFFERegistration",
				err:          ErrEnhancedSPIFFERegistration,
				expectedCode: "SPIFFE_REGISTRATION_FAILED",
				namespace:    IsEnhancedDomainError,
			},
			{
				name:         "ErrEnhancedConnectionFailed",
				err:          ErrEnhancedConnectionFailed,
				expectedCode: "CONNECTION_FAILED",
				namespace:    IsEnhancedSystemError,
			},
			{
				name:         "ErrEnhancedConfigFileNotFound",
				err:          ErrEnhancedConfigFileNotFound,
				expectedCode: "CONFIG_FILE_NOT_FOUND",
				namespace:    IsEnhancedConfigurationError,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				if !tc.namespace(tc.err) {
					t.Errorf("Expected %s to belong to expected namespace", tc.name)
				}

				code := GetEnhancedErrorCode(tc.err)
				if code != tc.expectedCode {
					t.Errorf("Expected code %q, got %q", tc.expectedCode, code)
				}
			})
		}
	})

	t.Run("Error Wrapping and Unwrapping", func(t *testing.T) {
		originalErr := errors.New("original system error")
		wrappedErr := WrapWithEnhancedContext(originalErr, EnhancedSystemError, "system operation failed")

		if !IsEnhancedSystemError(wrappedErr) {
			t.Error("Expected wrapped error to be classified as enhanced system error")
		}

		// Test unwrapping
		if !errors.Is(wrappedErr, originalErr) {
			t.Error("Expected wrapped error to be unwrappable to original error")
		}

		unwrapped := errors.Unwrap(wrappedErr)
		if unwrapped == nil {
			t.Error("Expected wrapped error to have unwrappable content")
		}
	})

	t.Run("Error Decoration", func(t *testing.T) {
		baseErr := EnhancedValidationError.New("base validation error")
		decoratedErr := DecorateError(baseErr, "additional context information")

		if !IsEnhancedValidationError(decoratedErr) {
			t.Error("Expected decorated error to maintain validation classification")
		}

		decoratedStr := decoratedErr.Error()
		if !strings.Contains(decoratedStr, "base validation error") {
			t.Error("Expected decorated error to contain original message")
		}
		if !strings.Contains(decoratedStr, "additional context") {
			t.Error("Expected decorated error to contain additional context")
		}

		// Test decorating non-errorx error
		regularErr := errors.New("regular error")
		decoratedRegularErr := DecorateError(regularErr, "enhanced context")

		if !IsEnhancedSystemError(decoratedRegularErr) {
			t.Error("Expected decorated regular error to be wrapped as enhanced system error")
		}
	})

	t.Run("Stack Trace Information", func(t *testing.T) {
		err := EnhancedValidationError.New("test error with stack trace")

		// Test that stack trace is available
		stackTrace := GetStackTrace(err)
		if stackTrace == "" {
			t.Error("Expected non-empty stack trace")
		}

		// Stack trace should contain function name
		if !strings.Contains(stackTrace, "TestEnhancedErrors") {
			t.Error("Expected stack trace to contain test function name")
		}
	})

	t.Run("Multiple Properties", func(t *testing.T) {
		err := FieldValidationError.New("complex validation error").
			WithProperty(PropertyField, "user.email").
			WithProperty(PropertyValue, "invalid@").
			WithProperty(PropertyReason, "missing domain").
			WithProperty(PropertyCode, "EMAIL_VALIDATION_FAILED")

		// Test all properties
		field := GetEnhancedErrorField(err)
		if field != "user.email" {
			t.Errorf("Expected field %q, got %q", "user.email", field)
		}

		value := GetEnhancedErrorValue(err)
		if value != "invalid@" {
			t.Errorf("Expected value %q, got %v", "invalid@", value)
		}

		reason, _ := errorx.ExtractProperty(err, PropertyReason)
		if reason != "missing domain" {
			t.Errorf("Expected reason %q, got %v", "missing domain", reason)
		}

		code := GetEnhancedErrorCode(err)
		if code != "EMAIL_VALIDATION_FAILED" {
			t.Errorf("Expected code %q, got %q", "EMAIL_VALIDATION_FAILED", code)
		}
	})

	t.Run("Namespace Hierarchy", func(t *testing.T) {
		validationErr := EnhancedValidationError.New("validation test")
		configErr := EnhancedConfigurationError.New("config test")
		domainErr := EnhancedDomainError.New("domain test")
		systemErr := EnhancedSystemError.New("system test")

		// Test namespace classification using our helper functions
		if !IsEnhancedValidationError(validationErr) {
			t.Error("Expected validation error to be classified as validation error")
		}

		if !IsEnhancedConfigurationError(configErr) {
			t.Error("Expected config error to be classified as configuration error")
		}

		if !IsEnhancedDomainError(domainErr) {
			t.Error("Expected domain error to be classified as domain error")
		}

		if !IsEnhancedSystemError(systemErr) {
			t.Error("Expected system error to be classified as system error")
		}

		// Test cross-namespace exclusion
		if IsEnhancedValidationError(configErr) {
			t.Error("Expected config error NOT to be classified as validation error")
		}

		if IsEnhancedConfigurationError(systemErr) {
			t.Error("Expected system error NOT to be classified as configuration error")
		}
	})

	t.Run("Error Type Specificity", func(t *testing.T) {
		fieldValidationErr := FieldValidationError.New("field validation error")
		collectionErr := CollectionValidationError.New("collection error")

		// Both should be classified under the validation namespace
		if !IsEnhancedValidationError(fieldValidationErr) {
			t.Error("Expected field validation error to be classified as validation error")
		}

		if !IsEnhancedValidationError(collectionErr) {
			t.Error("Expected collection error to be classified as validation error")
		}

		// But they should be different specific types
		if errorx.IsOfType(collectionErr, FieldValidationError) {
			t.Error("Expected collection error NOT to be a field validation error")
		}

		if errorx.IsOfType(fieldValidationErr, CollectionValidationError) {
			t.Error("Expected field validation error NOT to be a collection error")
		}
	})
}

func TestEnhancedErrorCompatibility(t *testing.T) {
	t.Run("Compatibility with Standard Error Interface", func(t *testing.T) {
		err := EnhancedValidationError.New("standard error interface test")

		// Test that it implements error interface
		var standardErr error = err
		expectedMsg := "standard error interface test"
		if !strings.Contains(standardErr.Error(), expectedMsg) {
			t.Errorf("Expected error message to contain %q", expectedMsg)
		}
	})

	t.Run("Compatibility with errors.Is and errors.As", func(t *testing.T) {
		originalErr := errors.New("original")
		wrappedErr := WrapWithEnhancedContext(originalErr, EnhancedSystemError, "wrapped")

		// Test errors.Is
		if !errors.Is(wrappedErr, originalErr) {
			t.Error("Expected errors.Is to work with enhanced wrapped errors")
		}

		// Test errors.As
		var errorxErr *errorx.Error
		if !errors.As(wrappedErr, &errorxErr) {
			t.Error("Expected errors.As to work with enhanced errors")
		}
	})

	t.Run("Nil Error Handling", func(t *testing.T) {
		// Test that our helper functions handle nil properly
		if IsEnhancedValidationError(nil) {
			t.Error("Expected IsEnhancedValidationError to return false for nil")
		}

		if IsTimeoutError(nil) {
			t.Error("Expected IsTimeoutError to return false for nil")
		}

		if GetEnhancedErrorField(nil) != "" {
			t.Error("Expected GetEnhancedErrorField to return empty string for nil")
		}

		if GetEnhancedErrorValue(nil) != nil {
			t.Error("Expected GetEnhancedErrorValue to return nil for nil")
		}

		// Test wrapping nil
		if WrapWithEnhancedContext(nil, EnhancedSystemError, "context") != nil {
			t.Error("Expected WrapWithEnhancedContext to return nil for nil input")
		}

		if DecorateError(nil, "context") != nil {
			t.Error("Expected DecorateError to return nil for nil input")
		}
	})
}

// Benchmark enhanced errors vs standard errors
func BenchmarkEnhancedErrorCreation(b *testing.B) {
	b.Run("Enhanced ValidationError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := NewEnhancedValidationError("field", "value", "message")
			_ = err
		}
	})

	b.Run("Enhanced SystemError", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := NewEnhancedSystemError("service", "message")
			_ = err
		}
	})

	b.Run("Standard fmt.Errorf", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := errors.New("message")
			_ = err
		}
	})

	b.Run("Enhanced with stack trace", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := EnhancedValidationError.New("message with stack trace")
			_ = err
		}
	})

	b.Run("Error Classification", func(b *testing.B) {
		err := NewEnhancedValidationError("field", "value", "message")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = IsEnhancedValidationError(err)
		}
	})

	b.Run("Property Extraction", func(b *testing.B) {
		err := NewEnhancedValidationError("field", "value", "message")
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = GetEnhancedErrorField(err)
		}
	})
}
