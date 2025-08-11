// Package ephemos provides early config validation integrated into high-level API entry points.
package ephemos

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"

	coreErrors "github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Domain-specific errors for configuration validation failures.
// These provide clear, actionable error messages without exposing internal complexity.
var (
	// ErrInvalidConfig is returned when configuration validation fails.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrConfigFileNotFound is returned when the specified config file doesn't exist.
	ErrConfigFileNotFound = errors.New("configuration file not found")

	// ErrConfigFileUnreadable is returned when config file exists but cannot be read.
	ErrConfigFileUnreadable = errors.New("configuration file unreadable")

	// ErrConfigMalformed is returned when config file has invalid YAML syntax.
	ErrConfigMalformed = errors.New("configuration file malformed")
)

// ConfigValidationError provides detailed information about configuration validation failures.
// This wraps the internal validation errors with user-friendly messages.
type ConfigValidationError struct {
	File    string // Configuration file path
	Field   string // Field that failed validation
	Value   any    // Invalid value
	Message string // Human-readable error message
	Cause   error  // Underlying error
}

func (e *ConfigValidationError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("config validation failed in %s: %s", e.File, e.Message)
	}
	return fmt.Sprintf("config validation failed: %s", e.Message)
}

func (e *ConfigValidationError) Unwrap() error {
	return e.Cause
}

// loadAndValidateConfig loads and validates configuration with comprehensive error handling.
// This is the core function that provides early validation for all high-level API entry points.
//
// EARLY VALIDATION APPROACH:
// 1. Validates config path and file accessibility before parsing
// 2. Parses YAML and provides domain-specific errors for syntax issues
// 3. Runs comprehensive validation on all configuration fields
// 4. Returns domain-specific errors that developers can handle with standard Go error handling
// 5. Prevents partial setups by failing fast with clear error messages.
//
// Error Types Returned:
// - ErrConfigFileNotFound: File doesn't exist at specified path
// - ErrConfigFileUnreadable: File exists but cannot be read (permissions, etc.)
// - ErrConfigMalformed: Invalid YAML syntax in config file
// - ErrInvalidConfig: Configuration validation failed (wrapped with specific details).
func loadAndValidateConfig(ctx context.Context, configPath string) (*ports.Configuration, error) {
	// Step 1: Resolve and validate config path
	resolvedPath, err := resolveConfigPath(configPath)
	if err != nil {
		return nil, fmt.Errorf("config path resolution failed: %w", err)
	}

	// Step 2: Check file accessibility early
	if err := validateFileAccess(resolvedPath); err != nil {
		return nil, err
	}

	// Step 3: Load and parse config file with context cancellation support
	config, err := loadConfigFile(ctx, resolvedPath)
	if err != nil {
		return nil, err
	}

	// Step 4: Comprehensive validation with domain-specific error wrapping
	if err := validateConfigComprehensive(config, resolvedPath); err != nil {
		return nil, err
	}

	return config, nil
}

// resolveConfigPath determines the actual config file path to use.
// Priority order: explicit path > EPHEMOS_CONFIG env var > default locations.
func resolveConfigPath(configPath string) (string, error) {
	// Use explicit path if provided
	if strings.TrimSpace(configPath) != "" {
		cleanPath := filepath.Clean(strings.TrimSpace(configPath))
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve config path: %w", err)
		}
		return absPath, nil
	}

	// Check EPHEMOS_CONFIG environment variable
	if envPath := os.Getenv("EPHEMOS_CONFIG"); envPath != "" {
		cleanPath := filepath.Clean(strings.TrimSpace(envPath))
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve EPHEMOS_CONFIG path: %w", err)
		}
		return absPath, nil
	}

	// Try default locations in order
	defaultPaths := []string{
		"ephemos.yaml",
		"config/ephemos.yaml",
		"configs/ephemos.yaml",
		"/etc/ephemos/ephemos.yaml",
	}

	for _, path := range defaultPaths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, err := os.Stat(absPath); err == nil {
			return absPath, nil
		}
	}

	// No config file found - return empty path to use defaults
	return "", nil
}

// validateFileAccess performs early validation of file accessibility.
func validateFileAccess(path string) error {
	if path == "" {
		// Empty path means use defaults - not an error
		return nil
	}

	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrConfigFileNotFound, path)
		}
		return fmt.Errorf("%w: %s: %w", ErrConfigFileUnreadable, path, err)
	}

	// Check if it's a file (not directory)
	if info.IsDir() {
		return fmt.Errorf("%w: %s is a directory, not a file", ErrInvalidConfig, path)
	}

	// Check if file is readable
	//nolint:gosec // G304: Path is validated above for security
	if _, err := os.Open(path); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrConfigFileUnreadable, path, err)
	}

	return nil
}

// loadConfigFile loads and parses the configuration file.
func loadConfigFile(ctx context.Context, path string) (*ports.Configuration, error) {
	// If no config file, return default configuration
	if path == "" {
		provider := &fileProviderCompat{}
		return provider.GetDefaultConfiguration(ctx), nil
	}

	// Check for context cancellation
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("config loading canceled: %w", ctx.Err())
		default:
		}
	}

	// Read file contents
	//nolint:gosec // G304: Path is validated above for security
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read %s: %w", ErrConfigFileUnreadable, path, err)
	}

	// Parse YAML with enhanced error handling
	var config ports.Configuration
	if err := yaml.Unmarshal(data, &config); err != nil {
		// Provide more user-friendly YAML parsing errors
		return nil, &ConfigValidationError{
			File:    path,
			Field:   "yaml_syntax",
			Value:   string(data),
			Message: fmt.Sprintf("invalid YAML syntax: %v", err),
			Cause:   fmt.Errorf("%w: %w", ErrConfigMalformed, err),
		}
	}

	return &config, nil
}

// validateConfigComprehensive performs comprehensive validation with better error messages.
func validateConfigComprehensive(config *ports.Configuration, configPath string) error {
	if config == nil {
		return &ConfigValidationError{
			File:    configPath,
			Field:   "configuration",
			Value:   nil,
			Message: "configuration cannot be nil",
			Cause:   ErrInvalidConfig,
		}
	}

	// Run the standard validation
	if err := config.Validate(); err != nil {
		// Wrap validation errors with better context
		return wrapValidationError(err, configPath)
	}

	return nil
}

// wrapValidationError converts internal validation errors to user-friendly ConfigValidationError.
func wrapValidationError(err error, configPath string) error {
	// Check if it's already a validation error
	var validationErr *coreErrors.ValidationError
	if errors.As(err, &validationErr) {
		return &ConfigValidationError{
			File:    configPath,
			Field:   validationErr.Field,
			Value:   validationErr.Value,
			Message: enhanceValidationMessage(validationErr),
			Cause:   fmt.Errorf("%w: %s", ErrInvalidConfig, validationErr.Message),
		}
	}

	// Handle wrapped errors
	return &ConfigValidationError{
		File:    configPath,
		Field:   "unknown",
		Value:   nil,
		Message: err.Error(),
		Cause:   fmt.Errorf("%w: %w", ErrInvalidConfig, err),
	}
}

// enhanceValidationMessage provides more user-friendly validation error messages.
func enhanceValidationMessage(err *coreErrors.ValidationError) string {
	switch err.Field {
	case "service.name":
		return fmt.Sprintf("Service name '%v' is invalid. Must be non-empty and contain only "+
			"alphanumeric characters, hyphens, and underscores.", err.Value)
	case "service.domain":
		return fmt.Sprintf("Service domain '%v' is invalid. Must be a valid domain name (e.g., 'example.org').", err.Value)
	case "spiffe.socket_path":
		return fmt.Sprintf("SPIFFE socket path '%v' is invalid. Must be an absolute path to a Unix socket file "+
			"(e.g., '/tmp/spire-agent/public/api.sock').", err.Value)
	case "transport.type":
		return fmt.Sprintf("Transport type '%v' is invalid. Must be 'grpc' or 'http'.", err.Value)
	case "transport.address":
		return fmt.Sprintf("Transport address '%v' is invalid. Must be in format ':port' or 'host:port' (e.g., ':50051').", err.Value)
	default:
		return err.Message
	}
}

// fileProviderCompat provides compatibility with the existing file provider interface.
type fileProviderCompat struct{}

func (p *fileProviderCompat) GetDefaultConfiguration(_ context.Context) *ports.Configuration {
	return &ports.Configuration{
		Service: ports.ServiceConfig{
			Name:   "ephemos-service",
			Domain: "", // Empty domain uses SPIRE trust domain
		},
		SPIFFE: &ports.SPIFFEConfig{
			SocketPath: "/tmp/spire-agent/public/api.sock",
		},
		Transport: ports.TransportConfig{
			Type:    "grpc",
			Address: ":50051",
		},
		AuthorizedClients: []string{},
		TrustedServers:    []string{},
	}
}

// IsConfigurationError checks if an error is a configuration-related error.
// This allows developers to handle config errors differently from other types of errors.
func IsConfigurationError(err error) bool {
	if err == nil {
		return false
	}

	// Check for domain-specific config errors
	var configErr *ConfigValidationError
	return errors.Is(err, ErrInvalidConfig) ||
		errors.Is(err, ErrConfigFileNotFound) ||
		errors.Is(err, ErrConfigFileUnreadable) ||
		errors.Is(err, ErrConfigMalformed) ||
		errors.As(err, &configErr)
}

// GetConfigValidationError extracts ConfigValidationError from an error chain.
// Returns nil if the error is not a configuration validation error.
func GetConfigValidationError(err error) *ConfigValidationError {
	var configErr *ConfigValidationError
	if errors.As(err, &configErr) {
		return configErr
	}
	return nil
}
