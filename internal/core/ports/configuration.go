// Package ports defines interfaces for core services and domain boundaries.
package ports

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sufield/ephemos/internal/core/errors"
)

// YAML tag constants to avoid hardcoding.
const (
	ServiceYAMLTag           = "service"
	SPIFFEYAMLTag            = "spiffe"
	AuthorizedClientsYAMLTag = "authorized_clients"
	TrustedServersYAMLTag    = "trusted_servers"
	NameYAMLTag              = "name"
	DomainYAMLTag            = "domain"
	SocketPathYAMLTag        = "socket_path"
)

// Configuration represents the complete configuration for Ephemos services.
// It contains all necessary settings for service identity, SPIFFE integration,
// and authorization policies.
type Configuration struct {
	// Service contains the core service identification settings.
	// This is required and must include at least a service name.
	Service ServiceConfig `yaml:"service"`

	// SPIFFE contains optional SPIFFE/SPIRE integration settings.
	// If nil, default SPIFFE settings will be used.
	SPIFFE *SPIFFEConfig `yaml:"spiffe,omitempty"`

	// AuthorizedClients lists SPIFFE IDs that are allowed to connect to this service.
	// Each entry must be a valid SPIFFE ID (e.g., "spiffe://example.org/client-service").
	// Empty list means no client authorization is enforced.
	AuthorizedClients []string `yaml:"authorizedClients,omitempty"`

	// TrustedServers lists SPIFFE IDs of servers this client trusts to connect to.
	// Each entry must be a valid SPIFFE ID (e.g., "spiffe://example.org/server-service").
	// Empty list means all servers are trusted (not recommended for production).
	TrustedServers []string `yaml:"trustedServers,omitempty"`
}

// ServiceConfig contains the core service identification settings.
type ServiceConfig struct {
	// Name is the unique identifier for this service.
	// Required field, must be non-empty and contain only valid service name characters.
	// Used for SPIFFE ID generation and service discovery.
	Name string `yaml:"name"`

	// Domain is the trust domain for this service.
	// Optional field that defaults to the SPIRE trust domain if not specified.
	// Must be a valid domain name format if provided.
	Domain string `yaml:"domain,omitempty"`
}

// SPIFFEConfig contains SPIFFE/SPIRE integration settings.
type SPIFFEConfig struct {
	// SocketPath is the path to the SPIRE agent's Unix domain socket.
	// Must be an absolute path to a valid Unix socket file.
	// Common default: "/tmp/spire-agent/public/api.sock"
	SocketPath string `yaml:"socketPath"`
}

// Validate checks if the configuration is valid and returns any validation errors.
// This method ensures all required fields are present and properly formatted.
func (c *Configuration) Validate() error {
	if c == nil {
		return &errors.ValidationError{
			Field:   "configuration",
			Value:   nil,
			Message: "configuration cannot be nil",
		}
	}

	// Validate service configuration
	if err := c.validateService(); err != nil {
		return fmt.Errorf("invalid service configuration: %w", err)
	}

	// Validate SPIFFE configuration if present
	if c.SPIFFE != nil {
		if err := c.validateSPIFFE(); err != nil {
			return fmt.Errorf("invalid SPIFFE configuration: %w", err)
		}
	}

	// Validate authorized clients
	if err := c.validateSPIFFEIDs(c.AuthorizedClients, "authorized_clients"); err != nil {
		return fmt.Errorf("invalid authorized clients: %w", err)
	}

	// Validate trusted servers
	if err := c.validateSPIFFEIDs(c.TrustedServers, "trusted_servers"); err != nil {
		return fmt.Errorf("invalid trusted servers: %w", err)
	}

	return nil
}

func (c *Configuration) validateService() error {
	if err := c.validateServiceName(); err != nil {
		return err
	}

	if err := c.validateServiceDomain(); err != nil {
		return err
	}

	return nil
}

//nolint:cyclop // Validation function has inherent complexity from multiple checks
func (c *Configuration) validateServiceName() error {
	if strings.TrimSpace(c.Service.Name) == "" {
		return &errors.ValidationError{
			Field:   "service.name",
			Value:   c.Service.Name,
			Message: "service name is required and cannot be empty",
		}
	}

	// Validate service name format (alphanumeric, hyphens, underscores)
	name := strings.TrimSpace(c.Service.Name)
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == '-' || char == '_') {
			return &errors.ValidationError{
				Field:   "service.name",
				Value:   c.Service.Name,
				Message: "service name must contain only alphanumeric characters, hyphens, and underscores",
			}
		}
	}

	return nil
}

func (c *Configuration) validateServiceDomain() error {
	// Validate domain format if provided
	if c.Service.Domain != "" {
		domain := strings.TrimSpace(c.Service.Domain)
		if domain == "" {
			return &errors.ValidationError{
				Field:   "service.domain",
				Value:   c.Service.Domain,
				Message: "service domain cannot be whitespace only",
			}
		}
		// Basic domain validation - contains dots and valid characters
		if !strings.Contains(domain, ".") {
			return &errors.ValidationError{
				Field:   "service.domain",
				Value:   c.Service.Domain,
				Message: "service domain must be a valid domain name",
			}
		}
	}

	return nil
}

func (c *Configuration) validateSPIFFE() error {
	if strings.TrimSpace(c.SPIFFE.SocketPath) == "" {
		return &errors.ValidationError{
			Field:   "spiffe.socket_path",
			Value:   c.SPIFFE.SocketPath,
			Message: "SPIFFE socket path is required when SPIFFE config is provided",
		}
	}

	// Validate that socket path is absolute
	socketPath := strings.TrimSpace(c.SPIFFE.SocketPath)
	if !strings.HasPrefix(socketPath, "/") {
		return &errors.ValidationError{
			Field:   "spiffe.socket_path",
			Value:   c.SPIFFE.SocketPath,
			Message: "SPIFFE socket path must be an absolute path",
		}
	}

	return nil
}

func (c *Configuration) validateSPIFFEIDs(ids []string, fieldName string) error {
	for i, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return &errors.ValidationError{
				Field:   fmt.Sprintf("%s[%d]", fieldName, i),
				Value:   ids[i],
				Message: "SPIFFE ID cannot be empty or whitespace",
			}
		}

		// Validate SPIFFE ID format
		if !strings.HasPrefix(id, "spiffe://") {
			return &errors.ValidationError{
				Field:   fmt.Sprintf("%s[%d]", fieldName, i),
				Value:   ids[i],
				Message: "SPIFFE ID must start with 'spiffe://' (e.g., 'spiffe://example.org/service')",
			}
		}

		// Basic structure validation - must have trust domain and path
		parts := strings.SplitN(id[9:], "/", 2) // Remove "spiffe://" prefix
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return &errors.ValidationError{
				Field:   fmt.Sprintf("%s[%d]", fieldName, i),
				Value:   ids[i],
				Message: "SPIFFE ID must have format 'spiffe://trust-domain/path' (e.g., 'spiffe://example.org/service')",
			}
		}
	}

	return nil
}

// ConfigurationProvider defines the interface for loading and providing configurations.
type ConfigurationProvider interface {
	// LoadConfiguration loads configuration from the specified file path.
	// Returns an error if the path is empty or invalid, or if loading fails.
	LoadConfiguration(ctx context.Context, path string) (*Configuration, error)

	// GetDefaultConfiguration returns a configuration with sensible defaults.
	// The context can be used for cancellation during default value computation.
	GetDefaultConfiguration(ctx context.Context) *Configuration
}

// Environment variable names for configuration
const (
	EnvServiceName          = "EPHEMOS_SERVICE_NAME"
	EnvTrustDomain         = "EPHEMOS_TRUST_DOMAIN"
	EnvSPIFFESocket        = "EPHEMOS_SPIFFE_SOCKET"
	EnvAuthorizedClients   = "EPHEMOS_AUTHORIZED_CLIENTS"
	EnvTrustedServers      = "EPHEMOS_TRUSTED_SERVERS"
	EnvRequireAuth         = "EPHEMOS_REQUIRE_AUTHENTICATION"
	EnvLogLevel            = "EPHEMOS_LOG_LEVEL"
	EnvBindAddress         = "EPHEMOS_BIND_ADDRESS"
	EnvTLSMinVersion       = "EPHEMOS_TLS_MIN_VERSION"
	EnvDebugEnabled        = "EPHEMOS_DEBUG_ENABLED"
)

// LoadFromEnvironment creates a configuration from environment variables.
// This is the most secure way to configure Ephemos in production.
func LoadFromEnvironment() (*Configuration, error) {
	config := &Configuration{}

	// Required: Service Name
	serviceName := os.Getenv(EnvServiceName)
	if serviceName == "" {
		return nil, &errors.ValidationError{
			Field:   EnvServiceName,
			Value:   "",
			Message: "service name is required via environment variable",
		}
	}

	// Optional: Trust Domain
	trustDomain := os.Getenv(EnvTrustDomain)
	if trustDomain == "" {
		trustDomain = "default.local" // Secure default, not example.org
	}

	config.Service = ServiceConfig{
		Name:   serviceName,
		Domain: trustDomain,
	}

	// SPIFFE Configuration
	spiffeSocket := os.Getenv(EnvSPIFFESocket)
	if spiffeSocket == "" {
		spiffeSocket = "/tmp/spire-agent/public/api.sock" // Default socket path
	}

	config.SPIFFE = &SPIFFEConfig{
		SocketPath: spiffeSocket,
	}

	// Parse comma-separated authorized clients
	if authorizedClients := os.Getenv(EnvAuthorizedClients); authorizedClients != "" {
		config.AuthorizedClients = parseCommaSeparatedList(authorizedClients)
	}

	// Parse comma-separated trusted servers
	if trustedServers := os.Getenv(EnvTrustedServers); trustedServers != "" {
		config.TrustedServers = parseCommaSeparatedList(trustedServers)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("environment configuration validation failed: %w", err)
	}

	// Additional production security validation
	if err := validateProductionSecurity(config); err != nil {
		return nil, fmt.Errorf("production security validation failed: %w", err)
	}

	return config, nil
}

// MergeWithEnvironment merges file-based configuration with environment variables.
// Environment variables take precedence over file values.
func (c *Configuration) MergeWithEnvironment() error {
	// Override service name if set via environment
	if serviceName := os.Getenv(EnvServiceName); serviceName != "" {
		c.Service.Name = serviceName
	}

	// Override trust domain if set via environment
	if trustDomain := os.Getenv(EnvTrustDomain); trustDomain != "" {
		c.Service.Domain = trustDomain
	}

	// Override SPIFFE socket path if set via environment
	if spiffeSocket := os.Getenv(EnvSPIFFESocket); spiffeSocket != "" {
		if c.SPIFFE == nil {
			c.SPIFFE = &SPIFFEConfig{}
		}
		c.SPIFFE.SocketPath = spiffeSocket
	}

	// Override authorized clients if set via environment
	if authorizedClients := os.Getenv(EnvAuthorizedClients); authorizedClients != "" {
		c.AuthorizedClients = parseCommaSeparatedList(authorizedClients)
	}

	// Override trusted servers if set via environment
	if trustedServers := os.Getenv(EnvTrustedServers); trustedServers != "" {
		c.TrustedServers = parseCommaSeparatedList(trustedServers)
	}

	return c.Validate()
}

// parseCommaSeparatedList parses a comma-separated string into a slice,
// trimming whitespace and filtering empty values.
func parseCommaSeparatedList(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// validateProductionSecurity performs additional security validation for production environments.
func validateProductionSecurity(config *Configuration) error {
	var errors []string

	// Check for demo/development values that should not be used in production
	if strings.Contains(config.Service.Domain, "example.org") {
		errors = append(errors, "trust domain contains 'example.org' - not suitable for production")
	}

	if strings.Contains(config.Service.Domain, "localhost") {
		errors = append(errors, "trust domain contains 'localhost' - not suitable for production")
	}

	if strings.Contains(config.Service.Domain, "example.com") {
		errors = append(errors, "trust domain contains 'example.com' - not suitable for production")
	}

	// Validate service name doesn't contain demo values
	if strings.Contains(config.Service.Name, "example") || strings.Contains(config.Service.Name, "demo") {
		errors = append(errors, "service name contains demo/example values - not suitable for production")
	}

	// Check SPIFFE socket path security
	socketPath := config.SPIFFE.SocketPath
	if !strings.HasPrefix(socketPath, "/run/") && !strings.HasPrefix(socketPath, "/var/run/") && !strings.HasPrefix(socketPath, "/tmp/") {
		errors = append(errors, "SPIFFE socket should be in a secure directory (/run, /var/run, or /tmp)")
	}

	// Warn about overly permissive authorization (but don't fail)
	for _, client := range config.AuthorizedClients {
		if strings.Contains(client, "*") {
			errors = append(errors, fmt.Sprintf("authorized client contains wildcard: %s - consider more specific authorization", client))
		}
	}

	// Check for debug environment variables that shouldn't be enabled in production
	if debugEnabled := os.Getenv(EnvDebugEnabled); debugEnabled == "true" {
		errors = append(errors, "debug mode is enabled via EPHEMOS_DEBUG_ENABLED - should be disabled in production")
	}

	if len(errors) > 0 {
		return fmt.Errorf("production security issues: %s", strings.Join(errors, "; "))
	}

	return nil
}

// IsProductionReady checks if the configuration is suitable for production use.
func (c *Configuration) IsProductionReady() error {
	return validateProductionSecurity(c)
}

// GetBoolEnv returns a boolean environment variable value with a default.
func GetBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
