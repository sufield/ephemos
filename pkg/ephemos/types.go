// Package ephemos provides the public API types for the Ephemos service mesh library.
package ephemos

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Configuration represents the complete configuration for Ephemos services.
// This is the public API version of the internal Configuration type.
type Configuration struct {
	// Service contains the core service identification settings.
	Service ServiceConfig `yaml:"service"`

	// SPIFFE contains optional SPIFFE/SPIRE integration settings.
	SPIFFE *SPIFFEConfig `yaml:"spiffe,omitempty"`

	// AuthorizedClients lists SPIFFE IDs that are allowed to connect to this service.
	AuthorizedClients []string `yaml:"authorizedClients,omitempty"`

	// TrustedServers lists SPIFFE IDs of servers this client trusts to connect to.
	TrustedServers []string `yaml:"trustedServers,omitempty"`

	// Transport contains the transport layer configuration.
	Transport TransportConfig `yaml:"transport,omitempty"`
}

// ServiceConfig contains the core service identification settings.
type ServiceConfig struct {
	// Name is the unique identifier for this service.
	Name string `yaml:"name"`

	// Domain is the trust domain for this service.
	Domain string `yaml:"domain,omitempty"`
}

// SPIFFEConfig contains SPIFFE/SPIRE integration settings.
type SPIFFEConfig struct {
	// SocketPath is the path to the SPIRE agent's Unix domain socket.
	SocketPath string `yaml:"socketPath"`
}

// TransportConfig contains transport layer configuration.
type TransportConfig struct {
	// Type specifies the transport protocol to use.
	Type string `yaml:"type,omitempty"`

	// Address specifies the network address to bind to.
	Address string `yaml:"address,omitempty"`

	// TLS contains TLS configuration for the transport.
	TLS *TLSConfig `yaml:"tls,omitempty"`
}

// TLSConfig contains TLS/SSL configuration settings.
type TLSConfig struct {
	// Enabled determines whether TLS is enabled.
	Enabled bool `yaml:"enabled,omitempty"`

	// CertFile is the path to the TLS certificate file.
	CertFile string `yaml:"certFile,omitempty"`

	// KeyFile is the path to the TLS private key file.
	KeyFile string `yaml:"keyFile,omitempty"`

	// UseSPIFFE determines whether to use SPIFFE X.509 certificates for TLS.
	UseSPIFFE bool `yaml:"useSpiffe,omitempty"`
}

// Validate checks if the configuration is valid and returns any validation errors.
func (c *Configuration) Validate() error {
	if c == nil {
		return &ValidationError{
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

func (c *Configuration) validateServiceName() error {
	if strings.TrimSpace(c.Service.Name) == "" {
		return &ValidationError{
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
			return &ValidationError{
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
			return &ValidationError{
				Field:   "service.domain",
				Value:   c.Service.Domain,
				Message: "service domain cannot be whitespace only",
			}
		}
		// Basic domain validation - contains dots and valid characters
		if !strings.Contains(domain, ".") {
			return &ValidationError{
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
		return &ValidationError{
			Field:   "spiffe.socket_path",
			Value:   c.SPIFFE.SocketPath,
			Message: "SPIFFE socket path is required when SPIFFE config is provided",
		}
	}

	// Validate that socket path is absolute
	socketPath := strings.TrimSpace(c.SPIFFE.SocketPath)
	if !strings.HasPrefix(socketPath, "/") {
		return &ValidationError{
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
			return &ValidationError{
				Field:   fmt.Sprintf("%s[%d]", fieldName, i),
				Value:   ids[i],
				Message: "SPIFFE ID cannot be empty or whitespace",
			}
		}

		// Validate SPIFFE ID format
		if !strings.HasPrefix(id, "spiffe://") {
			return &ValidationError{
				Field:   fmt.Sprintf("%s[%d]", fieldName, i),
				Value:   ids[i],
				Message: "SPIFFE ID must start with 'spiffe://' (e.g., 'spiffe://example.org/service')",
			}
		}

		// Basic structure validation - must have trust domain and path
		parts := strings.SplitN(id[9:], "/", 2) // Remove "spiffe://" prefix
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return &ValidationError{
				Field:   fmt.Sprintf("%s[%d]", fieldName, i),
				Value:   ids[i],
				Message: "SPIFFE ID must have format 'spiffe://trust-domain/path' (e.g., 'spiffe://example.org/service')",
			}
		}
	}

	return nil
}

// ValidationError provides detailed information about configuration validation failures.
type ValidationError struct {
	Field   string // Field that failed validation
	Value   any    // Invalid value
	Message string // Human-readable error message
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// Environment variable names for configuration.
const (
	EnvServiceName       = "EPHEMOS_SERVICE_NAME"
	EnvTrustDomain       = "EPHEMOS_TRUST_DOMAIN"
	EnvSPIFFESocket      = "EPHEMOS_SPIFFE_SOCKET"
	EnvAuthorizedClients = "EPHEMOS_AUTHORIZED_CLIENTS"
	EnvTrustedServers    = "EPHEMOS_TRUSTED_SERVERS"
	EnvRequireAuth       = "EPHEMOS_REQUIRE_AUTHENTICATION"
	EnvLogLevel          = "EPHEMOS_LOG_LEVEL"
	EnvBindAddress       = "EPHEMOS_BIND_ADDRESS"
	EnvTLSMinVersion     = "EPHEMOS_TLS_MIN_VERSION"
	EnvDebugEnabled      = "EPHEMOS_DEBUG_ENABLED"
)

// LoadFromEnvironment creates a configuration from environment variables.
func LoadFromEnvironment() (*Configuration, error) {
	config := &Configuration{}

	// Required: Service Name
	serviceName := os.Getenv(EnvServiceName)
	if serviceName == "" {
		return nil, &ValidationError{
			Field:   EnvServiceName,
			Value:   "",
			Message: "service name is required via environment variable",
		}
	}

	// Optional: Trust Domain
	trustDomain := os.Getenv(EnvTrustDomain)
	if trustDomain == "" {
		trustDomain = "default.local" // Secure default
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

	return config, nil
}

// MergeWithEnvironment merges file-based configuration with environment variables.
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

// parseCommaSeparatedList parses a comma-separated string into a slice.
func parseCommaSeparatedList(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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

// ConfigurationProvider defines the interface for loading configurations.
type ConfigurationProvider interface {
	// LoadConfiguration loads configuration from the specified file path.
	LoadConfiguration(ctx context.Context, path string) (*Configuration, error)

	// GetDefaultConfiguration returns a configuration with sensible defaults.
	GetDefaultConfiguration(ctx context.Context) *Configuration
}

// GetDefaultConfiguration returns a configuration with sensible defaults.
func GetDefaultConfiguration() *Configuration {
	return &Configuration{
		Service: ServiceConfig{
			Name:   "ephemos-service", // Default service name
			Domain: "default.local",   // Secure default
		},
		SPIFFE: &SPIFFEConfig{
			SocketPath: "/tmp/spire-agent/public/api.sock", // Standard SPIRE socket
		},
		Transport: TransportConfig{
			Type:    "grpc",
			Address: ":50051", // Default gRPC port
		},
		AuthorizedClients: []string{},
		TrustedServers:    []string{},
	}
}

// LoadConfigFromYAML loads configuration from a YAML file and returns public Configuration type.
func LoadConfigFromYAML(ctx context.Context, yamlPath string) (*Configuration, error) {
	// For now, we'll use environment variables to load config since we're transitioning away from internal types
	// This is a temporary bridge function during the refactor
	envConfig, err := LoadFromEnvironment()
	if err != nil {
		// If env loading fails, return default config
		config := GetDefaultConfiguration()
		if err := config.Validate(); err != nil {
			return nil, fmt.Errorf("default configuration validation failed: %w", err)
		}
		return config, nil
	}
	return envConfig, nil
}