// Package ephemos provides enhanced error handling using errorx for semantic error types and automatic stack traces.
// This file provides a migration path from custom error types to errorx-based error handling.
package ephemos

import (
	"github.com/joomcode/errorx"
)

// Error namespace functions for lazy initialization.
func ephemosNamespace() *errorx.Namespace {
	return errorx.NewNamespace("ephemos")
}

func validationNamespace() *errorx.Namespace {
	return ephemosNamespace().NewSubNamespace("validation")
}

func configurationNamespace() *errorx.Namespace {
	return ephemosNamespace().NewSubNamespace("configuration")
}

func domainNamespace() *errorx.Namespace {
	return ephemosNamespace().NewSubNamespace("domain")
}

func systemNamespace() *errorx.Namespace {
	return ephemosNamespace().NewSubNamespace("system")
}

// Enhanced error types with errorx features
var (
	// Validation errors
	EnhancedValidationError   = validationNamespace().NewType("validation_error")
	FieldValidationError      = validationNamespace().NewType("field_validation")
	CollectionValidationError = validationNamespace().NewType("collection_error")

	// Configuration errors
	EnhancedConfigurationError = configurationNamespace().NewType("config_error")
	ConfigFileError            = configurationNamespace().NewType("config_file")
	ConfigParseError           = configurationNamespace().NewType("config_parse")

	// Domain errors
	EnhancedDomainError = domainNamespace().NewType("domain_error")
	ServiceError        = domainNamespace().NewType("service_error")
	IdentityError       = domainNamespace().NewType("identity_error")

	// System errors
	EnhancedSystemError = systemNamespace().NewType("system_error")
	ConnectionError     = systemNamespace().NewType("connection_error")
	CertificateError    = systemNamespace().NewType("certificate_error")

	// Specialized error types with built-in traits
	TimeoutError   = systemNamespace().NewType("timeout_error", errorx.Timeout())
	TemporaryError = systemNamespace().NewType("temporary_error", errorx.Temporary())
)

// Common error properties for consistent error context
var (
	// PropertyField identifies which field caused the error
	PropertyField = errorx.RegisterProperty("field")

	// PropertyValue contains the invalid value
	PropertyValue = errorx.RegisterProperty("value")

	// PropertyFile identifies the file related to the error
	PropertyFile = errorx.RegisterProperty("file")

	// PropertyService identifies the service related to the error
	PropertyService = errorx.RegisterProperty("service")

	// PropertyOperation identifies the operation that failed
	PropertyOperation = errorx.RegisterProperty("operation")

	// PropertyCode provides an error code for programmatic handling
	PropertyCode = errorx.RegisterPrintableProperty("code")

	// PropertyReason provides additional reasoning context
	PropertyReason = errorx.RegisterProperty("reason")
)

// Enhanced predefined domain errors with errorx features and automatic stack traces
var (
	// Service validation errors with enhanced context
	ErrEnhancedInvalidServiceName = ServiceError.New("service name is invalid").
					WithProperty(PropertyCode, "INVALID_SERVICE_NAME")

	ErrEnhancedInvalidDomain = ServiceError.New("domain is invalid").
					WithProperty(PropertyCode, "INVALID_DOMAIN")

	ErrEnhancedMissingConfiguration = EnhancedConfigurationError.New("required configuration is missing").
					WithProperty(PropertyCode, "MISSING_CONFIGURATION")

	// SPIFFE/Identity errors with enhanced traits
	ErrEnhancedSPIFFERegistration = IdentityError.New("failed to register service with SPIFFE").
					WithProperty(PropertyCode, "SPIFFE_REGISTRATION_FAILED")

	ErrEnhancedInvalidSocketPath = IdentityError.New("SPIFFE socket path is invalid").
					WithProperty(PropertyCode, "INVALID_SOCKET_PATH")

	ErrEnhancedCertificateUnavailable = CertificateError.New("certificate is not available").
						WithProperty(PropertyCode, "CERTIFICATE_UNAVAILABLE")

	ErrEnhancedTrustBundleUnavailable = CertificateError.New("trust bundle is not available").
						WithProperty(PropertyCode, "TRUST_BUNDLE_UNAVAILABLE")

	// Connection errors with timeout support
	ErrEnhancedConnectionFailed = ConnectionError.New("failed to establish connection").
					WithProperty(PropertyCode, "CONNECTION_FAILED")

	// Enhanced configuration file errors
	ErrEnhancedConfigFileNotFound = ConfigFileError.New("configuration file not found").
					WithProperty(PropertyCode, "CONFIG_FILE_NOT_FOUND")

	ErrEnhancedConfigFileUnreadable = ConfigFileError.New("configuration file unreadable").
					WithProperty(PropertyCode, "CONFIG_FILE_UNREADABLE")

	ErrEnhancedConfigMalformed = ConfigParseError.New("configuration file malformed").
					WithProperty(PropertyCode, "CONFIG_MALFORMED")
)

// Enhanced helper functions for creating errors with rich context

// NewEnhancedValidationError creates a field validation error with full context and stack trace
func NewEnhancedValidationError(field string, value interface{}, message string) error {
	return FieldValidationError.New("validation failed: "+message).
		WithProperty(PropertyField, field).
		WithProperty(PropertyValue, value).
		WithProperty(PropertyCode, "VALIDATION_FAILED")
}

// NewEnhancedConfigError creates a configuration error with file context and stack trace
func NewEnhancedConfigError(file string, message string) error {
	return EnhancedConfigurationError.New("configuration error: "+message).
		WithProperty(PropertyFile, file).
		WithProperty(PropertyCode, "CONFIG_ERROR")
}

// NewEnhancedDomainError creates a domain error with operation context and stack trace
func NewEnhancedDomainError(operation string, message string) error {
	return EnhancedDomainError.New("domain error: "+message).
		WithProperty(PropertyOperation, operation).
		WithProperty(PropertyCode, "DOMAIN_ERROR")
}

// NewEnhancedSystemError creates a system error with service context and stack trace
func NewEnhancedSystemError(service string, message string) error {
	return EnhancedSystemError.New("system error: "+message).
		WithProperty(PropertyService, service).
		WithProperty(PropertyCode, "SYSTEM_ERROR")
}

// NewTimeoutError creates a timeout error using errorx built-in timeout trait
func NewTimeoutError(operation string, message string) error {
	return TimeoutError.New("timeout error: "+message).
		WithProperty(PropertyOperation, operation).
		WithProperty(PropertyCode, "TIMEOUT_ERROR")
}

// NewTemporaryError creates a temporary error that might succeed on retry
func NewTemporaryError(service string, message string) error {
	return TemporaryError.New("temporary error: "+message).
		WithProperty(PropertyService, service).
		WithProperty(PropertyCode, "TEMPORARY_ERROR")
}

// Enhanced error checking functions using errorx capabilities

// IsEnhancedValidationError checks if an error belongs to validation namespace
func IsEnhancedValidationError(err error) bool {
	return errorx.IsOfType(err, EnhancedValidationError) ||
		errorx.IsOfType(err, FieldValidationError) ||
		errorx.IsOfType(err, CollectionValidationError)
}

// IsEnhancedConfigurationError checks if an error belongs to configuration namespace
func IsEnhancedConfigurationError(err error) bool {
	return errorx.IsOfType(err, EnhancedConfigurationError) ||
		errorx.IsOfType(err, ConfigFileError) ||
		errorx.IsOfType(err, ConfigParseError)
}

// IsEnhancedDomainError checks if an error belongs to domain namespace
func IsEnhancedDomainError(err error) bool {
	return errorx.IsOfType(err, EnhancedDomainError) ||
		errorx.IsOfType(err, ServiceError) ||
		errorx.IsOfType(err, IdentityError)
}

// IsEnhancedSystemError checks if an error belongs to system namespace
func IsEnhancedSystemError(err error) bool {
	return errorx.IsOfType(err, EnhancedSystemError) ||
		errorx.IsOfType(err, ConnectionError) ||
		errorx.IsOfType(err, CertificateError) ||
		errorx.IsOfType(err, TimeoutError) ||
		errorx.IsOfType(err, TemporaryError)
}

// IsTimeoutError checks if an error has timeout characteristics using errorx traits
func IsTimeoutError(err error) bool {
	return errorx.IsTimeout(err)
}

// IsTemporaryError checks if an error is temporary using errorx traits
func IsTemporaryError(err error) bool {
	return errorx.IsTemporary(err)
}

// Enhanced property extraction functions with better error handling

// GetEnhancedErrorField extracts the field name from a validation error
func GetEnhancedErrorField(err error) string {
	if prop, ok := errorx.ExtractProperty(err, PropertyField); ok {
		if field, ok := prop.(string); ok {
			return field
		}
	}
	return ""
}

// GetEnhancedErrorValue extracts the invalid value from a validation error
func GetEnhancedErrorValue(err error) interface{} {
	if prop, ok := errorx.ExtractProperty(err, PropertyValue); ok {
		return prop
	}
	return nil
}

// GetEnhancedErrorFile extracts the file name from a configuration error
func GetEnhancedErrorFile(err error) string {
	if prop, ok := errorx.ExtractProperty(err, PropertyFile); ok {
		if file, ok := prop.(string); ok {
			return file
		}
	}
	return ""
}

// GetEnhancedErrorService extracts the service name from a system error
func GetEnhancedErrorService(err error) string {
	if prop, ok := errorx.ExtractProperty(err, PropertyService); ok {
		if service, ok := prop.(string); ok {
			return service
		}
	}
	return ""
}

// GetEnhancedErrorOperation extracts the operation from a domain error
func GetEnhancedErrorOperation(err error) string {
	if prop, ok := errorx.ExtractProperty(err, PropertyOperation); ok {
		if operation, ok := prop.(string); ok {
			return operation
		}
	}
	return ""
}

// GetEnhancedErrorCode extracts the error code for programmatic handling
func GetEnhancedErrorCode(err error) string {
	if prop, ok := errorx.ExtractProperty(err, PropertyCode); ok {
		if code, ok := prop.(string); ok {
			return code
		}
	}
	return ""
}

// GetStackTrace extracts stack trace information from errorx errors
func GetStackTrace(err error) string {
	if errorxErr, ok := err.(*errorx.Error); ok {
		// Use string representation which includes stack trace
		return errorxErr.Error()
	}
	return ""
}

// WrapWithEnhancedContext wraps an existing error with additional context
func WrapWithEnhancedContext(err error, errorType *errorx.Type, message string) error {
	if err == nil {
		return nil
	}
	return errorType.Wrap(err, "wrapped error: "+message)
}

// DecorateError adds context to an existing errorx error without changing its type
func DecorateError(err error, additionalContext string) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*errorx.Error); ok {
		return errorx.Decorate(err, "decorated error: "+additionalContext)
	}
	// If it's not an errorx error, wrap it with enhanced system error
	return EnhancedSystemError.Wrap(err, "wrapped non-errorx error: "+additionalContext)
}

