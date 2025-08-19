// Package ports defines interfaces for core services and domain boundaries.
package ports

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/errors"
)

// YAML tag constants to avoid hardcoding.
const (
	ServiceYAMLTag    = "service"
	SPIFFEYAMLTag     = "spiffe"
	NameYAMLTag       = "name"
	DomainYAMLTag     = "domain"
	SocketPathYAMLTag = "socket_path"
)

// Configuration represents the complete configuration for Ephemos services.
// It contains all necessary settings for service identity and agent connection.
type Configuration struct {
	// Service contains the core service identification settings.
	// This is required and must include at least a service name.
	Service ServiceConfig `yaml:"service" validate:"required"`

	// Agent contains the connection settings for the identity agent.
	// If nil, default agent settings will be used.
	Agent *AgentConfig `yaml:"agent,omitempty"`

	// Health contains the health monitoring configuration.
	// If nil, health monitoring is disabled.
	Health *HealthConfig `yaml:"health,omitempty"`
}

// ServiceConfig contains the core service identification settings.
type ServiceConfig struct {
	// Name is the unique identifier for this service.
	// Required field, must be non-empty and contain only valid service name characters.
	// Used for SPIFFE ID generation and service discovery.
	Name domain.ServiceName `yaml:"name" validate:"required"`

	// Domain is the trust domain for this service.
	// Optional field that defaults to the SPIRE trust domain if not specified.
	// Must be a valid domain name format if provided.
	Domain string `yaml:"domain,omitempty" validate:"omitempty,domain"`

	// Cache contains caching configuration for certificate and trust bundle operations.
	Cache *CacheConfig `yaml:"cache,omitempty"`
}

// AgentConfig contains identity agent connection settings.
type AgentConfig struct {
	// SocketPath is the path to the identity agent's Unix domain socket.
	// Must be an absolute path to a valid Unix socket file.
	// Common default: "/run/sockets/agent.sock"
	SocketPath domain.SocketPath `yaml:"socketPath" validate:"required"`
}

// CacheConfig contains caching configuration for certificate and trust bundle operations.
type CacheConfig struct {
	// TTLMinutes specifies the time-to-live for cached certificates and trust bundles in minutes.
	// Default: 30 minutes (half of typical 1-hour SPIFFE certificate lifetime).
	// Must be between 1 and 60 minutes for security and performance reasons.
	TTLMinutes int `yaml:"ttl_minutes,omitempty" validate:"omitempty,min=1,max=60"`

	// ProactiveRefreshMinutes specifies when to proactively refresh certificates before expiry.
	// Default: 10 minutes before expiry.
	// Must be less than TTLMinutes and greater than 0.
	ProactiveRefreshMinutes int `yaml:"proactive_refresh_minutes,omitempty" validate:"omitempty,min=1"`
}

// Validate checks if the configuration is valid using go-playground/validator.
// This method ensures all required fields are present and properly formatted.
func (c *Configuration) Validate() error {
	if c == nil {
		return &errors.ValidationError{
			Field:   "configuration",
			Value:   nil,
			Message: "configuration cannot be nil",
		}
	}

	// Use the new validator with SPIFFE-specific validations
	validator := domain.NewValidator()
	if err := validator.Validate(c); err != nil {
		// Convert go-playground validation errors to our custom format
		validationErrors := domain.ConvertValidationErrors(err)
		if len(validationErrors) == 1 {
			return &errors.ValidationError{
				Field:   validationErrors[0].Field,
				Value:   validationErrors[0].Value,
				Message: validationErrors[0].Message,
			}
		}
		// For multiple errors, return a combined error message
		var messages []string
		for _, vErr := range validationErrors {
			messages = append(messages, fmt.Sprintf("%s: %s", vErr.Field, vErr.Message))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(messages, "; "))
	}

	// Validate cache configuration cross-field constraints
	if c.Service.Cache != nil {
		if err := c.validateCacheConstraints(); err != nil {
			return err
		}
	}

	return nil
}

// validateCacheConstraints validates cross-field constraints for cache configuration
// that cannot be expressed with simple validation tags.
func (c *Configuration) validateCacheConstraints() error {
	cache := c.Service.Cache
	if cache.ProactiveRefreshMinutes >= cache.TTLMinutes && cache.TTLMinutes > 0 {
		return &errors.ValidationError{
			Field:   "service.cache.proactive_refresh_minutes",
			Value:   cache.ProactiveRefreshMinutes,
			Message: "proactive refresh time must be less than cache TTL",
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

// Environment variable names for configuration.
const (
	EnvServiceName         = "EPHEMOS_SERVICE_NAME"
	EnvTrustDomain         = "EPHEMOS_TRUST_DOMAIN"
	EnvAgentSocket         = "EPHEMOS_AGENT_SOCKET"
	EnvInsecureSkipVerify  = "EPHEMOS_INSECURE_SKIP_VERIFY"
	EnvRequireAuth         = "EPHEMOS_REQUIRE_AUTHENTICATION"
	EnvLogLevel            = "EPHEMOS_LOG_LEVEL"
	EnvBindAddress         = "EPHEMOS_BIND_ADDRESS"
	EnvTLSMinVersion       = "EPHEMOS_TLS_MIN_VERSION"
	EnvDebugEnabled        = "EPHEMOS_DEBUG_ENABLED"
	EnvCacheTTLMinutes     = "EPHEMOS_CACHE_TTL_MINUTES"
	EnvCacheRefreshMinutes = "EPHEMOS_CACHE_REFRESH_MINUTES"
)

// LoadFromEnvironment creates a configuration from environment variables.
// This is the most secure way to configure Ephemos in production.
func LoadFromEnvironment() (*Configuration, error) {
	v := viper.New()

	// Configure viper for environment variables
	v.SetEnvPrefix("EPHEMOS")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults
	v.SetDefault("service.domain", "default.local")
	v.SetDefault("agent.socketpath", "/run/sockets/agent.sock")
	v.SetDefault("service.cache.ttl_minutes", 30)
	v.SetDefault("service.cache.proactive_refresh_minutes", 10)

	// Required: Service Name
	if !v.IsSet("service_name") {
		return nil, &errors.ValidationError{
			Field:   EnvServiceName,
			Value:   "",
			Message: "service name is required via environment variable",
		}
	}

	// Unmarshal configuration
	var config Configuration
	if err := v.Unmarshal(&config, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
			domain.SocketPathDecodeHook(),
			domain.ServiceNameDecodeHook(),
		),
	)); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// Manual mapping for specific fields due to naming conventions
	serviceNameStr := v.GetString("service_name")
	if serviceNameStr != "" {
		serviceName, err := domain.NewServiceName(serviceNameStr)
		if err != nil {
			return nil, fmt.Errorf("invalid service name from environment: %w", err)
		}
		config.Service.Name = serviceName
	}
	config.Service.Domain = v.GetString("trust_domain")
	if config.Service.Domain == "" {
		config.Service.Domain = v.GetString("service.domain")
	}

	// Initialize cache config if needed
	if v.IsSet("cache_ttl_minutes") || v.IsSet("cache_refresh_minutes") {
		if config.Service.Cache == nil {
			config.Service.Cache = &CacheConfig{}
		}
		if v.IsSet("cache_ttl_minutes") {
			config.Service.Cache.TTLMinutes = v.GetInt("cache_ttl_minutes")
		}
		if v.IsSet("cache_refresh_minutes") {
			config.Service.Cache.ProactiveRefreshMinutes = v.GetInt("cache_refresh_minutes")
		}
	}

	// Initialize agent config if needed
	if config.Agent == nil {
		config.Agent = &AgentConfig{}
	}

	// Handle agent socket path from environment variables
	if socketPath := v.GetString("agent_socket"); socketPath != "" {
		config.Agent.SocketPath = domain.NewSocketPathUnsafe(socketPath)
	} else if socketPath := v.GetString("agent.socketpath"); socketPath != "" {
		config.Agent.SocketPath = domain.NewSocketPathUnsafe(socketPath)
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("environment configuration validation failed: %w", err)
	}

	return &config, nil
}

// MergeWithEnvironment merges file-based configuration with environment variables.
// Environment variables take precedence over file values.
func (c *Configuration) MergeWithEnvironment() error {
	v := viper.New()
	v.SetEnvPrefix("EPHEMOS")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Override service name if set via environment
	if serviceNameStr := v.GetString("service_name"); serviceNameStr != "" {
		serviceName, err := domain.NewServiceName(serviceNameStr)
		if err != nil {
			return fmt.Errorf("invalid service name from environment: %w", err)
		}
		c.Service.Name = serviceName
	}

	// Override trust domain if set via environment
	if trustDomain := v.GetString("trust_domain"); trustDomain != "" {
		c.Service.Domain = trustDomain
	}

	// Override agent socket path if set via environment
	if agentSocket := v.GetString("agent_socket"); agentSocket != "" {
		if c.Agent == nil {
			c.Agent = &AgentConfig{}
		}
		// Create SocketPath Value Object from string - fail fast on invalid input
		socketPath, err := domain.NewSocketPath(agentSocket)
		if err != nil {
			return fmt.Errorf("invalid agent socket path from environment: %w", err)
		}
		c.Agent.SocketPath = socketPath
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
	var validationErrors []error

	// Check domain security
	if err := validateProductionDomain(config.Service.Domain); err != nil {
		validationErrors = append(validationErrors, err)
	}

	// Check service name
	if err := validateProductionServiceName(config.Service.Name.Value()); err != nil {
		validationErrors = append(validationErrors, err)
	}

	// Check agent socket path security
	if config.Agent != nil {
		if err := validateSocketPath(config.Agent.SocketPath.Value()); err != nil {
			validationErrors = append(validationErrors, err)
		}
	}

	// Use viper for security checks
	v := viper.New()
	v.SetEnvPrefix("EPHEMOS")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Check for insecure certificate validation setting
	if v.GetBool("insecure_skip_verify") {
		validationErrors = append(validationErrors, errors.ErrInsecureSkipVerify)
	}

	// Check for debug environment variables that shouldn't be enabled in production
	if v.GetBool("debug_enabled") {
		validationErrors = append(validationErrors, errors.ErrDebugEnabled)
	}

	// Check for verbose logging
	if logLevel := strings.ToLower(v.GetString("log_level")); logLevel == "debug" || logLevel == "trace" {
		validationErrors = append(validationErrors, errors.ErrVerboseLogging)
	}

	if len(validationErrors) > 0 {
		return errors.NewProductionValidationError(validationErrors...)
	}

	return nil
}

// validateProductionDomain checks if the domain is suitable for production.
func validateProductionDomain(domain string) error {
	if strings.Contains(domain, "example.org") || strings.Contains(domain, "example.com") {
		return errors.ErrExampleTrustDomain
	}
	if strings.Contains(domain, "localhost") {
		return errors.ErrLocalhostTrustDomain
	}
	if strings.Contains(domain, "demo") || strings.Contains(domain, "test") {
		return errors.ErrDemoTrustDomain
	}
	return nil
}

// validateProductionServiceName checks if the service name is suitable for production.
func validateProductionServiceName(name string) error {
	if strings.Contains(name, "example") {
		return errors.ErrExampleServiceName
	}
	if strings.Contains(name, "demo") {
		return errors.ErrDemoServiceName
	}
	return nil
}

// validateSocketPath checks if the agent socket path is in a secure location.
func validateSocketPath(socketPath string) error {
	secureDirectories := []string{"/run/", "/var/run/", "/tmp/"}
	for _, dir := range secureDirectories {
		if strings.HasPrefix(socketPath, dir) {
			return nil
		}
	}
	return errors.ErrInsecureSocketPath
}

// ShouldSkipCertificateValidation follows industry best practices for explicit opt-in.
// Returns true ONLY when EPHEMOS_INSECURE_SKIP_VERIFY=true is explicitly set.
// This matches patterns used by Docker, Argo Workflows, Consul, and other successful Go projects.
func (c *Configuration) ShouldSkipCertificateValidation() bool {
	// Explicit opt-in following industry standard pattern
	// Similar to DOCKER_TLS_VERIFY, ARGO_INSECURE_SKIP_VERIFY, CONSUL_TLS_SKIP_VERIFY
	v := viper.New()
	v.SetEnvPrefix("EPHEMOS")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	return v.GetBool("insecure_skip_verify")
}

// IsInsecureModeExplicitlyEnabled checks if insecure mode was explicitly requested.
// Used for logging warnings when security is disabled.
func (c *Configuration) IsInsecureModeExplicitlyEnabled() bool {
	return c.ShouldSkipCertificateValidation()
}

// IsProductionReady checks if the configuration is suitable for production use.
func (c *Configuration) IsProductionReady() error {
	return validateProductionSecurity(c)
}

// GetBoolEnv gets a boolean value from an environment variable with a default fallback.
// Returns the default value if the environment variable is not set or cannot be parsed.
func GetBoolEnv(key string, defaultValue bool) bool {
	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault(key, defaultValue)
	return v.GetBool(key)
}
