// Package domain provides validation using go-playground/validator/v10 with SPIFFE-specific custom validators.
package domain

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

// Validator wraps go-playground/validator with SPIFFE-specific custom validators.
type Validator struct {
	validator *validator.Validate
}

// NewValidator creates a new validation instance with custom SPIFFE validators.
func NewValidator() *Validator {
	validate := validator.New()

	// Register custom SPIFFE validators
	_ = validate.RegisterValidation("spiffe_id", validateSPIFFEIDCustom)
	_ = validate.RegisterValidation("domain", validateDomainCustom)
	_ = validate.RegisterValidation("duration", validateDurationCustom)
	_ = validate.RegisterValidation("file_exists", validateFileExistsCustom)
	_ = validate.RegisterValidation("dir_exists", validateDirExistsCustom)
	_ = validate.RegisterValidation("abs_path", validateAbsolutePathCustom)
	_ = validate.RegisterValidation("port", validatePortCustom)
	_ = validate.RegisterValidation("ip", validateIPCustom)
	_ = validate.RegisterValidation("service_name", validateServiceNameCustom)

	return &Validator{
		validator: validate,
	}
}

// Validate validates a struct using go-playground/validator with SPIFFE extensions.
func (v *Validator) Validate(s interface{}) error {
	return v.validator.Struct(s)
}

// ValidateVar validates a single variable using the specified tag.
func (v *Validator) ValidateVar(field interface{}, tag string) error {
	return v.validator.Var(field, tag)
}

// SPIFFE ID custom validator that uses go-spiffe/v2 library for proper validation.
func validateSPIFFEIDCustom(fl validator.FieldLevel) bool {
	spiffeID := fl.Field().String()
	if spiffeID == "" {
		return true // Empty values handled by 'required' tag
	}

	// Use go-spiffe/v2 official validation
	_, err := spiffeid.FromString(spiffeID)
	return err == nil
}

// Domain custom validator for trust domains and hostnames.
func validateDomainCustom(fl validator.FieldLevel) bool {
	domain := strings.TrimSpace(fl.Field().String())
	if domain == "" {
		return true // Empty domains handled by 'required' tag
	}

	// Basic domain validation - must contain dots
	if !strings.Contains(domain, ".") {
		return false
	}

	// Validate domain format using regex
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	return domainRegex.MatchString(domain)
}

// Duration custom validator for Go duration strings.
func validateDurationCustom(fl validator.FieldLevel) bool {
	duration := fl.Field().String()
	if duration == "" {
		return true // Empty durations handled by 'required' tag
	}

	_, err := time.ParseDuration(duration)
	return err == nil
}

// File exists custom validator.
func validateFileExistsCustom(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return true // Empty paths handled by 'required' tag
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return !info.IsDir()
}

// Directory exists custom validator.
func validateDirExistsCustom(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return true // Empty paths handled by 'required' tag
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.IsDir()
}

// Absolute path custom validator with security checks.
func validateAbsolutePathCustom(fl validator.FieldLevel) bool {
	path := fl.Field().String()
	if path == "" {
		return true // Empty paths handled by 'required' tag
	}

	// Check for null bytes (security risk)
	if strings.Contains(path, "\x00") {
		return false
	}

	// Check for control characters including newlines (security risk)
	for _, r := range path {
		if r < 32 || r == 127 { // ASCII control characters
			return false
		}
	}

	// Check if path is absolute
	if !filepath.IsAbs(path) {
		return false
	}

	// Additional security checks - reject paths with unsafe components
	cleanPath := filepath.Clean(path)
	return cleanPath == path
}

// Custom port validator that handles both string and int formats.
func validatePortCustom(fl validator.FieldLevel) bool {
	field := fl.Field()
	var port int
	var err error

	switch field.Kind() {
	case reflect.String:
		portStr := strings.TrimPrefix(field.String(), ":")
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return false
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		port = int(field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		port = int(field.Uint())
	default:
		return false
	}

	return port >= 1 && port <= 65535
}

// IP address validator using Go's net package.
func validateIPCustom(fl validator.FieldLevel) bool {
	if fl.Field().String() == "" {
		return true // Empty IPs handled by 'required' tag
	}

	ip := net.ParseIP(fl.Field().String())
	return ip != nil
}

// Service name custom validator for SPIFFE service names.
// Allows alphanumeric characters, hyphens, and underscores.
func validateServiceNameCustom(fl validator.FieldLevel) bool {
	serviceName := strings.TrimSpace(fl.Field().String())
	if serviceName == "" {
		return true // Empty values handled by 'required' tag
	}

	// Validate service name format (alphanumeric, hyphens, underscores)
	for _, char := range serviceName {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_') {
			return false
		}
	}
	return true
}

// ValidationError wraps go-playground validator errors with additional context.
type ValidationError struct {
	Field   string      `json:"field"`
	Tag     string      `json:"tag"`
	Value   interface{} `json:"value"`
	Message string      `json:"message"`
}

// Error implements the error interface.
func (ve *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", ve.Field, ve.Message)
}

// ConvertValidationErrors converts go-playground validation errors to our custom format.
func ConvertValidationErrors(err error) []ValidationError {
	var errors []ValidationError

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for _, validationErr := range validationErrors {
			errors = append(errors, ValidationError{
				Field:   validationErr.Field(),
				Tag:     validationErr.Tag(),
				Value:   validationErr.Value(),
				Message: getCustomErrorMessage(validationErr),
			})
		}
	}

	return errors
}

// getCustomErrorMessage provides human-readable error messages for validation failures.
func getCustomErrorMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "field is required"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters/elements long", fe.Param())
	case "email":
		return "must be a valid email address"
	case "url":
		return "must be a valid URL"
	case "spiffe_id":
		return "must be a valid SPIFFE ID (e.g., spiffe://example.org/service)"
	case "domain":
		return "must be a valid domain name (e.g., example.org)"
	case "duration":
		return "must be a valid duration (e.g., 10s, 5m, 1h)"
	case "file_exists":
		return "file must exist and be a regular file"
	case "dir_exists":
		return "directory must exist and be a directory"
	case "abs_path":
		return "must be an absolute path without unsafe components"
	case "ip":
		return "must be a valid IP address"
	case "port":
		return "must be a valid port number (1-65535)"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", fe.Param())
	case "alphanum":
		return "must contain only alphanumeric characters"
	case "alpha":
		return "must contain only alphabetic characters"
	case "numeric":
		return "must contain only numeric characters"
	case "service_name":
		return "must contain only alphanumeric characters, hyphens, and underscores"
	default:
		return fmt.Sprintf("validation failed for tag '%s'", fe.Tag())
	}
}

// GlobalValidator is the global validator instance for convenience.
var GlobalValidator = NewValidator()

// ValidateStruct is a convenience function using the global validator.
func ValidateStruct(s interface{}) error {
	return GlobalValidator.Validate(s)
}