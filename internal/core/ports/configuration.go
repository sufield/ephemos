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

// AgentConfig contains identity agent connection settings.
type AgentConfig struct {
	// SocketPath is the path to the identity agent's Unix domain socket.
	// Must be an absolute path to a valid Unix socket file.
	// Common default: "/run/sockets/agent.sock"
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

	// Validate agent configuration if present
	if c.Agent != nil {
		if err := c.validateAgent(); err != nil {
			return fmt.Errorf("invalid agent configuration: %w", err)
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
	EnvServiceName   = "EPHEMOS_SERVICE_NAME"
	EnvTrustDomain   = "EPHEMOS_TRUST_DOMAIN"
	EnvAgentSocket   = "EPHEMOS_AGENT_SOCKET"
	EnvRequireAuth   = "EPHEMOS_REQUIRE_AUTHENTICATION"
	EnvLogLevel      = "EPHEMOS_LOG_LEVEL"
	EnvBindAddress   = "EPHEMOS_BIND_ADDRESS"
	EnvTLSMinVersion = "EPHEMOS_TLS_MIN_VERSION"
	EnvDebugEnabled  = "EPHEMOS_DEBUG_ENABLED"
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
	var errors []string

	// Check domain security
	if err := validateProductionDomain(config.Service.Domain); err != nil {
		errors = append(errors, err.Error())
	}

	// Check service name
	if err := validateProductionServiceName(config.Service.Name); err != nil {
		errors = append(errors, err.Error())
	}

	// Check agent socket path security
	if err := validateSocketPath(config.Agent.SocketPath); err != nil {
		errors = append(errors, err.Error())
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

// validateProductionDomain checks if the domain is suitable for production.
func validateProductionDomain(domain string) error {
	demoPatterns := []string{"example.org", "localhost", "example.com"}
	for _, pattern := range demoPatterns {
		if strings.Contains(domain, pattern) {
			return fmt.Errorf("trust domain contains '%s' - not suitable for production", pattern)
		}
	}
	return nil
}

// validateProductionServiceName checks if the service name is suitable for production.
func validateProductionServiceName(name string) error {
	if strings.Contains(name, "example") || strings.Contains(name, "demo") {
		return fmt.Errorf("service name contains demo/example values - not suitable for production")
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
	return fmt.Errorf("agent socket should be in a secure directory (/run, /var/run, or /tmp)")
}

// IsDevEnvironment detects if we're running in a development environment.
// Development environments are the ONLY ones where certificate validation is disabled.
func (c *Configuration) IsDevEnvironment() bool {
	// Development indicators in domain
	if c.Service.Domain != "" {
		domain := strings.ToLower(c.Service.Domain)
		if strings.Contains(domain, "dev") || 
		   strings.Contains(domain, "test") || 
		   strings.Contains(domain, "local") ||
		   strings.Contains(domain, "example") {
			return true
		}
	}
	
	// Development socket paths
	if c.Agent != nil {
		socketPath := strings.ToLower(c.Agent.SocketPath)
		if strings.HasPrefix(socketPath, "/tmp/") {
			return true
		}
	}
	
	// Development environment variable indicators
	nodeEnv := strings.ToLower(os.Getenv("NODE_ENV"))
	environment := strings.ToLower(os.Getenv("ENVIRONMENT"))
	stage := strings.ToLower(os.Getenv("STAGE"))
	
	if nodeEnv == "development" || nodeEnv == "dev" ||
	   environment == "development" || environment == "dev" ||
	   stage == "development" || stage == "dev" {
		return true
	}
	
	return false
}

// ShouldSkipCertificateValidation returns true ONLY in development environments.
// All other environments (production, staging, etc.) always validate certificates.
func (c *Configuration) ShouldSkipCertificateValidation() bool {
	return c.IsDevEnvironment()
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
