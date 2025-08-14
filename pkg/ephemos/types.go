// Package ephemos provides the public API types for the Ephemos service mesh library.
package ephemos

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
)

// Configuration represents the complete configuration for Ephemos services.
// This is the public API version of the internal Configuration type.
type Configuration struct {
	// Service contains the core service identification settings.
	Service ServiceConfig `yaml:"service"`

	// SPIFFE contains optional SPIFFE/SPIRE integration settings.
	SPIFFE *SPIFFEConfig `yaml:"spiffe,omitempty"`

	// AuthorizedClients lists SPIFFE IDs that are allowed to connect to this service.
	AuthorizedClients []string `yaml:"authorizedClients,omitempty" validate:"spiffe_id"`

	// TrustedServers lists SPIFFE IDs of servers this client trusts to connect to.
	TrustedServers []string `yaml:"trustedServers,omitempty" validate:"spiffe_id"`

	// Transport contains the transport layer configuration.
	Transport TransportConfig `yaml:"transport,omitempty"`
}

// ServiceConfig contains the core service identification settings.
type ServiceConfig struct {
	// Name is the unique identifier for this service.
	Name string `yaml:"name" validate:"required,min=1,max=100,regex=^[a-zA-Z0-9_-]+$" default:"ephemos-service"`

	// Domain is the trust domain for this service.
	Domain string `yaml:"domain,omitempty" validate:"domain" default:"default.local"`
}

// SPIFFEConfig contains SPIFFE/SPIRE integration settings.
type SPIFFEConfig struct {
	// SocketPath is the path to the SPIRE agent's Unix domain socket.
	SocketPath string `yaml:"socketPath" validate:"required,abs_path" default:"/tmp/spire-agent/public/api.sock"`
}

// TransportConfig contains transport layer configuration.
type TransportConfig struct {
	// Type specifies the transport protocol to use.
	Type string `yaml:"type,omitempty" validate:"oneof=grpc|http|tcp" default:"grpc"`

	// Address specifies the network address to bind to.
	Address string `yaml:"address,omitempty" validate:"regex=^(:[0-9]+|[^:]+:[0-9]+)$" default:":50051"`

	// TLS contains TLS configuration for the transport.
	TLS *TLSConfig `yaml:"tls,omitempty"`
}

// TLSConfig contains TLS/SSL configuration settings.
type TLSConfig struct {
	// Enabled determines whether TLS is enabled.
	Enabled bool `yaml:"enabled,omitempty" default:"true"`

	// CertFile is the path to the TLS certificate file.
	CertFile string `yaml:"certFile,omitempty" validate:"file_exists"`

	// KeyFile is the path to the TLS private key file.
	KeyFile string `yaml:"keyFile,omitempty" validate:"file_exists"`

	// UseSPIFFE determines whether to use SPIFFE X.509 certificates for TLS.
	UseSPIFFE bool `yaml:"useSpiffe,omitempty" default:"true"`
}

// Validate checks if the configuration is valid and returns any validation errors.
// This method uses the new struct tag-based validation engine for comprehensive
// validation with aggregated error reporting and automatic default value setting.
func (c *Configuration) Validate() error {
	if c == nil {
		return &ValidationError{
			Field:   "configuration",
			Value:   nil,
			Message: "configuration cannot be nil",
		}
	}

	// Use the new validation engine for comprehensive validation
	// Note: ValidateStruct is defined in validation.go in the same package
	return ValidateStruct(c)
}

// ValidateAndSetDefaults validates the configuration and sets default values.
// This is a convenience method that combines validation and default setting.
func (c *Configuration) ValidateAndSetDefaults() error {
	return ValidateStruct(c)
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
// This function now uses the new validation engine with automatic default setting.
func LoadFromEnvironment() (*Configuration, error) {
	config := &Configuration{}

	// Set values from environment variables if present
	if serviceName := os.Getenv(EnvServiceName); serviceName != "" {
		config.Service.Name = serviceName
	}

	if trustDomain := os.Getenv(EnvTrustDomain); trustDomain != "" {
		config.Service.Domain = trustDomain
	}

	// SPIFFE Configuration
	if spiffeSocket := os.Getenv(EnvSPIFFESocket); spiffeSocket != "" {
		if config.SPIFFE == nil {
			config.SPIFFE = &SPIFFEConfig{}
		}
		config.SPIFFE.SocketPath = spiffeSocket
	}

	// Parse comma-separated authorized clients
	if authorizedClients := os.Getenv(EnvAuthorizedClients); authorizedClients != "" {
		// Note: parseCommaSeparatedList is defined in config_enhanced.go in the same package
		config.AuthorizedClients = ParseCommaSeparatedList(authorizedClients)
	}

	// Parse comma-separated trusted servers
	if trustedServers := os.Getenv(EnvTrustedServers); trustedServers != "" {
		// Note: parseCommaSeparatedList is defined in config_enhanced.go in the same package
		config.TrustedServers = ParseCommaSeparatedList(trustedServers)
	}

	// Validate and set defaults using the new validation engine
	if err := config.ValidateAndSetDefaults(); err != nil {
		return nil, fmt.Errorf("environment configuration validation failed: %w", err)
	}

	return config, nil
}

// MergeWithEnvironment merges file-based configuration with environment variables.
// This function now uses the new validation engine with automatic default setting.
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
		// Note: parseCommaSeparatedList is defined in config_enhanced.go in the same package
		c.AuthorizedClients = ParseCommaSeparatedList(authorizedClients)
	}

	// Override trusted servers if set via environment
	if trustedServers := os.Getenv(EnvTrustedServers); trustedServers != "" {
		// Note: parseCommaSeparatedList is defined in config_enhanced.go in the same package
		c.TrustedServers = ParseCommaSeparatedList(trustedServers)
	}

	// Validate and set defaults using the new validation engine
	return c.ValidateAndSetDefaults()
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
// This function now uses the new validation engine to automatically set defaults.
func GetDefaultConfiguration() *Configuration {
	config := &Configuration{}

	// Use the validation engine to set all defaults automatically
	if err := config.ValidateAndSetDefaults(); err != nil {
		// This should never happen with an empty config, but handle gracefully
		// Fall back to manual defaults if validation fails
		return &Configuration{
			Service: ServiceConfig{
				Name:   "ephemos-service",
				Domain: "default.local",
			},
			SPIFFE: &SPIFFEConfig{
				SocketPath: "/tmp/spire-agent/public/api.sock",
			},
			Transport: TransportConfig{
				Type:    "grpc",
				Address: ":50051",
			},
			AuthorizedClients: []string{},
			TrustedServers:    []string{},
		}
	}

	return config
}

// LoadConfigFromYAML loads configuration from a YAML file and returns public Configuration type.
// This function now uses the new validation engine with automatic default setting.
func LoadConfigFromYAML(_ context.Context, _ string) (*Configuration, error) {
	// For now, we'll use environment variables to load config since we're transitioning away from internal types
	// This is a temporary bridge function during the refactor
	envConfig, err := LoadFromEnvironment()
	if err != nil {
		// If env loading fails, return default config
		// We're intentionally ignoring the error and returning defaults
		config := GetDefaultConfiguration()
		return config, nil //nolint:nilerr // Intentionally ignoring error and returning defaults
	}
	return envConfig, nil
}

// Common domain service interfaces that users can implement.
// These use plain Go types and are completely transport-agnostic.

// EchoService demonstrates a simple request-response service.
type EchoService interface {
	Echo(ctx context.Context, message string) (string, error)
	Ping(ctx context.Context) error
}

// FileService demonstrates binary data handling.
type FileService interface {
	Upload(ctx context.Context, filename string, data io.Reader) error
	Download(ctx context.Context, filename string) (io.Reader, error)
	List(ctx context.Context, prefix string) ([]string, error)
}
