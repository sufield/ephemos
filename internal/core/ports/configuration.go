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
	Service ServiceConfig `yaml:"service"`

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
	Name string `yaml:"name"`

	// Domain is the trust domain for this service.
	// Optional field that defaults to the SPIRE trust domain if not specified.
	// Must be a valid domain name format if provided.
	Domain string `yaml:"domain,omitempty"`

	// AuthorizedClients is a list of SPIFFE IDs that are authorized to connect to this service.
	// Used for server-side authorization enforcement.
	// If empty, all clients from the same trust domain are authorized.
	AuthorizedClients []string `yaml:"authorized_clients,omitempty"`

	// TrustedServers is a list of SPIFFE IDs that this service trusts as servers.
	// Used for client-side authorization when connecting to services.
	// If empty, all servers from the same trust domain are trusted.
	TrustedServers []string `yaml:"trusted_servers,omitempty"`

	// Cache contains caching configuration for certificate and trust bundle operations.
	Cache *CacheConfig `yaml:"cache,omitempty"`
}

// AgentConfig contains identity agent connection settings.
type AgentConfig struct {
	// SocketPath is the path to the identity agent's Unix domain socket.
	// Must be an absolute path to a valid Unix socket file.
	// Common default: "/run/sockets/agent.sock"
	SocketPath string `yaml:"socketPath"`
}

// CacheConfig contains caching configuration for certificate and trust bundle operations.
type CacheConfig struct {
	// TTLMinutes specifies the time-to-live for cached certificates and trust bundles in minutes.
	// Default: 30 minutes (half of typical 1-hour SPIFFE certificate lifetime).
	// Must be between 1 and 60 minutes for security and performance reasons.
	TTLMinutes int `yaml:"ttl_minutes,omitempty"`

	// ProactiveRefreshMinutes specifies when to proactively refresh certificates before expiry.
	// Default: 10 minutes before expiry.
	// Must be less than TTLMinutes and greater than 0.
	ProactiveRefreshMinutes int `yaml:"proactive_refresh_minutes,omitempty"`
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

	// Validate agent configuration if present
	if c.Agent != nil {
		if err := c.validateAgent(); err != nil {
			return fmt.Errorf("invalid agent configuration: %w", err)
		}
	}

	// Validate health configuration if present
	if c.Health != nil {
		if err := c.validateHealth(); err != nil {
			return fmt.Errorf("invalid health configuration: %w", err)
		}
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

	if err := c.validateServiceCache(); err != nil {
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

func (c *Configuration) validateServiceCache() error {
	if c.Service.Cache == nil {
		return nil // Cache config is optional
	}

	cache := c.Service.Cache

	// Validate TTL minutes
	if cache.TTLMinutes < 0 {
		return &errors.ValidationError{
			Field:   "service.cache.ttl_minutes",
			Value:   cache.TTLMinutes,
			Message: "cache TTL cannot be negative",
		}
	}

	if cache.TTLMinutes > 60 {
		return &errors.ValidationError{
			Field:   "service.cache.ttl_minutes",
			Value:   cache.TTLMinutes,
			Message: "cache TTL cannot exceed 60 minutes for security reasons",
		}
	}

	// Validate proactive refresh minutes
	if cache.ProactiveRefreshMinutes < 0 {
		return &errors.ValidationError{
			Field:   "service.cache.proactive_refresh_minutes",
			Value:   cache.ProactiveRefreshMinutes,
			Message: "proactive refresh time cannot be negative",
		}
	}

	if cache.ProactiveRefreshMinutes >= cache.TTLMinutes && cache.TTLMinutes > 0 {
		return &errors.ValidationError{
			Field:   "service.cache.proactive_refresh_minutes",
			Value:   cache.ProactiveRefreshMinutes,
			Message: "proactive refresh time must be less than cache TTL",
		}
	}

	return nil
}

func (c *Configuration) validateAgent() error {
	if strings.TrimSpace(c.Agent.SocketPath) == "" {
		return &errors.ValidationError{
			Field:   "agent.socketPath",
			Value:   c.Agent.SocketPath,
			Message: "agent socket path is required when agent config is provided",
		}
	}

	// Validate that socket path is absolute
	socketPath := strings.TrimSpace(c.Agent.SocketPath)
	if !strings.HasPrefix(socketPath, "/") {
		return &errors.ValidationError{
			Field:   "agent.socketPath",
			Value:   c.Agent.SocketPath,
			Message: "agent socket path must be an absolute path",
		}
	}

	return nil
}

func (c *Configuration) validateHealth() error {
	if c.Health == nil {
		return nil // Health config is optional
	}

	health := c.Health

	// Validate timeout
	if health.Timeout < 0 {
		return &errors.ValidationError{
			Field:   "health.timeout",
			Value:   health.Timeout,
			Message: "health check timeout cannot be negative",
		}
	}

	// Validate interval
	if health.Interval < 0 {
		return &errors.ValidationError{
			Field:   "health.interval",
			Value:   health.Interval,
			Message: "health check interval cannot be negative",
		}
	}

	// Validate server configuration if present
	if health.Server != nil {
		if err := c.validateSpireServerHealth(health.Server); err != nil {
			return err
		}
	}

	// Validate agent configuration if present
	if health.Agent != nil {
		if err := c.validateSpireAgentHealth(health.Agent); err != nil {
			return err
		}
	}

	return nil
}

func (c *Configuration) validateSpireServerHealth(server *SpireServerHealthConfig) error {
	if strings.TrimSpace(server.Address) == "" {
		return &errors.ValidationError{
			Field:   "health.server.address",
			Value:   server.Address,
			Message: "SPIRE server health address cannot be empty",
		}
	}

	// Validate paths
	if server.LivePath != "" && !strings.HasPrefix(server.LivePath, "/") {
		return &errors.ValidationError{
			Field:   "health.server.live_path",
			Value:   server.LivePath,
			Message: "SPIRE server live path must start with '/'",
		}
	}

	if server.ReadyPath != "" && !strings.HasPrefix(server.ReadyPath, "/") {
		return &errors.ValidationError{
			Field:   "health.server.ready_path",
			Value:   server.ReadyPath,
			Message: "SPIRE server ready path must start with '/'",
		}
	}

	return nil
}

func (c *Configuration) validateSpireAgentHealth(agent *SpireAgentHealthConfig) error {
	if strings.TrimSpace(agent.Address) == "" {
		return &errors.ValidationError{
			Field:   "health.agent.address",
			Value:   agent.Address,
			Message: "SPIRE agent health address cannot be empty",
		}
	}

	// Validate paths
	if agent.LivePath != "" && !strings.HasPrefix(agent.LivePath, "/") {
		return &errors.ValidationError{
			Field:   "health.agent.live_path",
			Value:   agent.LivePath,
			Message: "SPIRE agent live path must start with '/'",
		}
	}

	if agent.ReadyPath != "" && !strings.HasPrefix(agent.ReadyPath, "/") {
		return &errors.ValidationError{
			Field:   "health.agent.ready_path",
			Value:   agent.ReadyPath,
			Message: "SPIRE agent ready path must start with '/'",
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

	// Cache Configuration
	if cacheTTL := os.Getenv(EnvCacheTTLMinutes); cacheTTL != "" {
		if ttlMinutes, err := strconv.Atoi(cacheTTL); err == nil && ttlMinutes > 0 {
			if config.Service.Cache == nil {
				config.Service.Cache = &CacheConfig{}
			}
			config.Service.Cache.TTLMinutes = ttlMinutes
		}
	}

	if cacheRefresh := os.Getenv(EnvCacheRefreshMinutes); cacheRefresh != "" {
		if refreshMinutes, err := strconv.Atoi(cacheRefresh); err == nil && refreshMinutes > 0 {
			if config.Service.Cache == nil {
				config.Service.Cache = &CacheConfig{}
			}
			config.Service.Cache.ProactiveRefreshMinutes = refreshMinutes
		}
	}

	// Agent Configuration
	agentSocket := os.Getenv(EnvAgentSocket)
	if agentSocket == "" {
		agentSocket = "/run/sockets/agent.sock" // Default socket path
	}

	config.Agent = &AgentConfig{
		SocketPath: agentSocket,
	}

	// Validate the configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("environment configuration validation failed: %w", err)
	}

	// NOTE: Production validation is NOT run here - it should only be run when explicitly requested
	// via IsProductionReady() method

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

	// Override agent socket path if set via environment
	if agentSocket := os.Getenv(EnvAgentSocket); agentSocket != "" {
		if c.Agent == nil {
			c.Agent = &AgentConfig{}
		}
		c.Agent.SocketPath = agentSocket
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
	if err := validateProductionServiceName(config.Service.Name); err != nil {
		validationErrors = append(validationErrors, err)
	}

	// Check agent socket path security
	if err := validateSocketPath(config.Agent.SocketPath); err != nil {
		validationErrors = append(validationErrors, err)
	}

	// Check for insecure certificate validation setting
	if strings.ToLower(os.Getenv(EnvInsecureSkipVerify)) == "true" {
		validationErrors = append(validationErrors, errors.ErrInsecureSkipVerify)
	}

	// Check for debug environment variables that shouldn't be enabled in production
	if debugEnabled := os.Getenv(EnvDebugEnabled); debugEnabled == "true" {
		validationErrors = append(validationErrors, errors.ErrDebugEnabled)
	}

	// Check for verbose logging
	if logLevel := strings.ToLower(os.Getenv(EnvLogLevel)); logLevel == "debug" || logLevel == "trace" {
		validationErrors = append(validationErrors, errors.ErrVerboseLogging)
	}

	// Check authorized clients for wildcards
	for _, client := range config.Service.AuthorizedClients {
		if strings.Contains(client, "*") {
			validationErrors = append(validationErrors, fmt.Errorf("%w: %s", errors.ErrWildcardClients, client))
		}
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
	return strings.ToLower(os.Getenv(EnvInsecureSkipVerify)) == "true"
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

// GetBoolEnv returns a boolean environment variable value with a default.
func GetBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
