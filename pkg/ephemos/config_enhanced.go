// Package ephemos provides flexible configuration with YAML, environment variables, and pure-code options.
package ephemos

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// ConfigSource represents different configuration sources.
type ConfigSource int

const (
	// ConfigSourceYAML uses YAML file as primary source with env var overrides.
	ConfigSourceYAML ConfigSource = iota
	// ConfigSourceEnvOnly uses only environment variables.
	ConfigSourceEnvOnly
	// ConfigSourcePureCode uses programmatic configuration.
	ConfigSourcePureCode
)

// ConfigBuilder provides a fluent interface for building configurations.
type ConfigBuilder struct {
	config    *Configuration
	source    ConfigSource
	yamlPath  string
	envPrefix string
}

// NewConfigBuilder creates a new configuration builder.
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		config:    &Configuration{},
		source:    ConfigSourceYAML, // Default to YAML with env overrides
		envPrefix: "EPHEMOS",
	}
}

// WithSource sets the configuration source strategy.
func (b *ConfigBuilder) WithSource(source ConfigSource) *ConfigBuilder {
	b.source = source
	return b
}

// WithYAMLFile sets the YAML file path (for YAML source).
func (b *ConfigBuilder) WithYAMLFile(path string) *ConfigBuilder {
	b.yamlPath = path
	return b
}

// WithEnvPrefix sets the environment variable prefix (default: EPHEMOS).
func (b *ConfigBuilder) WithEnvPrefix(prefix string) *ConfigBuilder {
	b.envPrefix = prefix
	return b
}

// WithServiceName sets the service name programmatically.
func (b *ConfigBuilder) WithServiceName(name string) *ConfigBuilder {
	b.config.Service.Name = name
	return b
}

// WithServiceDomain sets the service domain programmatically.
func (b *ConfigBuilder) WithServiceDomain(domain string) *ConfigBuilder {
	b.config.Service.Domain = domain
	return b
}

// WithSPIFFESocket sets the SPIFFE socket path programmatically.
func (b *ConfigBuilder) WithSPIFFESocket(socketPath string) *ConfigBuilder {
	if b.config.SPIFFE == nil {
		b.config.SPIFFE = &SPIFFEConfig{}
	}
	b.config.SPIFFE.SocketPath = socketPath
	return b
}

// WithTransport sets the transport configuration programmatically.
func (b *ConfigBuilder) WithTransport(transportType, address string) *ConfigBuilder {
	b.config.Transport.Type = transportType
	b.config.Transport.Address = address
	return b
}

// WithAuthorizedClients sets authorized clients programmatically.
func (b *ConfigBuilder) WithAuthorizedClients(clients []string) *ConfigBuilder {
	b.config.AuthorizedClients = clients
	return b
}

// WithTrustedServers sets trusted servers programmatically.
func (b *ConfigBuilder) WithTrustedServers(servers []string) *ConfigBuilder {
	b.config.TrustedServers = servers
	return b
}

// Build creates the final configuration based on the specified source.
func (b *ConfigBuilder) Build(ctx context.Context) (*Configuration, error) {
	switch b.source {
	case ConfigSourceYAML:
		return b.buildFromYAML(ctx)
	case ConfigSourceEnvOnly:
		return b.buildFromEnvOnly(ctx)
	case ConfigSourcePureCode:
		return b.buildFromPureCode(ctx)
	default:
		return nil, fmt.Errorf("unsupported config source: %d", b.source)
	}
}

// buildFromYAML builds configuration from YAML with environment variable overrides.
func (b *ConfigBuilder) buildFromYAML(ctx context.Context) (*Configuration, error) {
	// Start with YAML configuration by loading from file and converting to public type
	publicConfig, err := LoadConfigFromYAML(ctx, b.yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load YAML config: %w", err)
	}

	// Apply environment variable overrides
	config, err := b.applyEnvOverrides(publicConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to apply env overrides: %w", err)
	}

	// Apply any programmatic overrides from builder
	b.applyBuilderOverrides(config)

	// Validate final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("final config validation failed: %w", err)
	}

	return config, nil
}

// buildFromEnvOnly builds configuration entirely from environment variables.
func (b *ConfigBuilder) buildFromEnvOnly(_ context.Context) (*Configuration, error) {
	// Start with default configuration
	config := GetDefaultConfiguration()

	// Apply environment variables
	config, err := b.applyEnvOverrides(config)
	if err != nil {
		return nil, fmt.Errorf("failed to build from env vars: %w", err)
	}

	// Apply any programmatic overrides from builder
	b.applyBuilderOverrides(config)

	// Validate final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("env-only config validation failed: %w", err)
	}

	return config, nil
}

// buildFromPureCode builds configuration entirely from programmatic settings.
func (b *ConfigBuilder) buildFromPureCode(_ context.Context) (*Configuration, error) {
	// Start with default configuration
	defaultConfig := GetDefaultConfiguration()

	// Use builder config as base, falling back to defaults for missing values
	config := &Configuration{
		Service: ServiceConfig{
			Name:   b.getValueOrDefault(b.config.Service.Name, defaultConfig.Service.Name),
			Domain: b.getValueOrDefault(b.config.Service.Domain, defaultConfig.Service.Domain),
		},
		Transport: TransportConfig{
			Type:    b.getValueOrDefault(b.config.Transport.Type, defaultConfig.Transport.Type),
			Address: b.getValueOrDefault(b.config.Transport.Address, defaultConfig.Transport.Address),
		},
		AuthorizedClients: b.getSliceOrDefault(b.config.AuthorizedClients, defaultConfig.AuthorizedClients),
		TrustedServers:    b.getSliceOrDefault(b.config.TrustedServers, defaultConfig.TrustedServers),
	}

	// Handle SPIFFE config
	if b.config.SPIFFE != nil || defaultConfig.SPIFFE != nil {
		config.SPIFFE = &SPIFFEConfig{}
		if b.config.SPIFFE != nil {
			config.SPIFFE.SocketPath = b.getValueOrDefault(b.config.SPIFFE.SocketPath, "")
		}
		if config.SPIFFE.SocketPath == "" && defaultConfig.SPIFFE != nil {
			config.SPIFFE.SocketPath = defaultConfig.SPIFFE.SocketPath
		}
	}

	// Validate final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("pure-code config validation failed: %w", err)
	}

	return config, nil
}

// applyEnvOverrides applies environment variable overrides to configuration.
func (b *ConfigBuilder) applyEnvOverrides(config *Configuration) (*Configuration, error) {
	envConfig := *config // Copy the config

	// Service configuration overrides
	if val := os.Getenv(b.envPrefix + "_SERVICE_NAME"); val != "" {
		envConfig.Service.Name = val
	}
	if val := os.Getenv(b.envPrefix + "_SERVICE_DOMAIN"); val != "" {
		envConfig.Service.Domain = val
	}

	// SPIFFE configuration overrides
	if val := os.Getenv(b.envPrefix + "_SPIFFE_SOCKET"); val != "" {
		if envConfig.SPIFFE == nil {
			envConfig.SPIFFE = &SPIFFEConfig{}
		}
		envConfig.SPIFFE.SocketPath = val
	}

	// Transport configuration overrides
	if val := os.Getenv(b.envPrefix + "_TRANSPORT_TYPE"); val != "" {
		envConfig.Transport.Type = val
	}
	if val := os.Getenv(b.envPrefix + "_TRANSPORT_ADDRESS"); val != "" {
		envConfig.Transport.Address = val
	}

	// Authorized clients override (comma-separated list)
	if val := os.Getenv(b.envPrefix + "_AUTHORIZED_CLIENTS"); val != "" {
		envConfig.AuthorizedClients = strings.Split(val, ",")
		// Trim whitespace from each client
		for i, client := range envConfig.AuthorizedClients {
			envConfig.AuthorizedClients[i] = strings.TrimSpace(client)
		}
	}

	// Trusted servers override (comma-separated list)
	if val := os.Getenv(b.envPrefix + "_TRUSTED_SERVERS"); val != "" {
		envConfig.TrustedServers = strings.Split(val, ",")
		// Trim whitespace from each server
		for i, server := range envConfig.TrustedServers {
			envConfig.TrustedServers[i] = strings.TrimSpace(server)
		}
	}

	return &envConfig, nil
}

// applyBuilderOverrides applies programmatic overrides from the builder.
func (b *ConfigBuilder) applyBuilderOverrides(config *Configuration) {
	// Only override if builder has non-zero values
	if b.config.Service.Name != "" {
		config.Service.Name = b.config.Service.Name
	}
	if b.config.Service.Domain != "" {
		config.Service.Domain = b.config.Service.Domain
	}
	if b.config.Transport.Type != "" {
		config.Transport.Type = b.config.Transport.Type
	}
	if b.config.Transport.Address != "" {
		config.Transport.Address = b.config.Transport.Address
	}
	if b.config.SPIFFE != nil && b.config.SPIFFE.SocketPath != "" {
		if config.SPIFFE == nil {
			config.SPIFFE = &SPIFFEConfig{}
		}
		config.SPIFFE.SocketPath = b.config.SPIFFE.SocketPath
	}
	if len(b.config.AuthorizedClients) > 0 {
		config.AuthorizedClients = b.config.AuthorizedClients
	}
	if len(b.config.TrustedServers) > 0 {
		config.TrustedServers = b.config.TrustedServers
	}
}

// getValueOrDefault returns value if non-empty, otherwise returns default.
func (b *ConfigBuilder) getValueOrDefault(value, defaultValue string) string {
	if value != "" {
		return value
	}
	return defaultValue
}

// getSliceOrDefault returns slice if non-empty, otherwise returns default.
func (b *ConfigBuilder) getSliceOrDefault(value, defaultValue []string) []string {
	if len(value) > 0 {
		return value
	}
	return defaultValue
}

// LoadConfigFlexible provides a convenient function for flexible configuration loading.
// This maintains backward compatibility while enabling new configuration patterns.
func LoadConfigFlexible(ctx context.Context, options ...ConfigOption) (*Configuration, error) {
	builder := NewConfigBuilder()

	// Apply options
	for _, opt := range options {
		if err := opt(builder); err != nil {
			return nil, fmt.Errorf("failed to apply config option: %w", err)
		}
	}

	return builder.Build(ctx)
}

// ConfigOption represents a configuration option function.
type ConfigOption func(*ConfigBuilder) error

// WithYAMLSource creates a config option for YAML-based configuration.
func WithYAMLSource(yamlPath string) ConfigOption {
	return func(b *ConfigBuilder) error {
		b.WithSource(ConfigSourceYAML).WithYAMLFile(yamlPath)
		return nil
	}
}

// WithEnvSource creates a config option for environment-only configuration.
func WithEnvSource(envPrefix string) ConfigOption {
	return func(b *ConfigBuilder) error {
		b.WithSource(ConfigSourceEnvOnly).WithEnvPrefix(envPrefix)
		return nil
	}
}

// WithPureCodeSource creates a config option for pure-code configuration.
func WithPureCodeSource() ConfigOption {
	return func(b *ConfigBuilder) error {
		b.WithSource(ConfigSourcePureCode)
		return nil
	}
}

// WithService creates a config option for service configuration.
func WithService(name, domain string) ConfigOption {
	return func(b *ConfigBuilder) error {
		b.WithServiceName(name).WithServiceDomain(domain)
		return nil
	}
}

// WithTransportOption creates a config option for transport configuration.
func WithTransportOption(transportType, address string) ConfigOption {
	return func(b *ConfigBuilder) error {
		b.WithTransport(transportType, address)
		return nil
	}
}
