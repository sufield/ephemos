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
	config := b.applyEnvOverrides(publicConfig)

	// Apply any programmatic overrides from builder
	b.applyBuilderOverrides(config)

	// Validate final configuration
	if err := config.Validate(); err != nil {
		return nil, wrapValidationError(err, "yaml-config")
	}

	return config, nil
}

// buildFromEnvOnly builds configuration entirely from environment variables.
func (b *ConfigBuilder) buildFromEnvOnly(_ context.Context) (*Configuration, error) {
	// Start with default configuration
	config := GetDefaultConfiguration()

	// Apply environment variables
	config = b.applyEnvOverrides(config)

	// Apply any programmatic overrides from builder
	b.applyBuilderOverrides(config)

	// Validate final configuration
	if err := config.Validate(); err != nil {
		return nil, wrapValidationError(err, "env-only-config")
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
		return nil, wrapValidationError(err, "pure-code-config")
	}

	return config, nil
}

// applyServiceEnvOverrides applies service-related environment overrides.
func (b *ConfigBuilder) applyServiceEnvOverrides(config *Configuration) {
	if val := os.Getenv(b.envPrefix + "_SERVICE_NAME"); val != "" {
		config.Service.Name = val
	}
	if val := os.Getenv(b.envPrefix + "_SERVICE_DOMAIN"); val != "" {
		config.Service.Domain = val
	}
}

// applyTransportEnvOverrides applies transport-related environment overrides.
func (b *ConfigBuilder) applyTransportEnvOverrides(config *Configuration) {
	if val := os.Getenv(b.envPrefix + "_TRANSPORT_TYPE"); val != "" {
		config.Transport.Type = val
	}
	if val := os.Getenv(b.envPrefix + "_TRANSPORT_ADDRESS"); val != "" {
		config.Transport.Address = val
	}
}

// ParseCommaSeparatedList parses and trims a comma-separated environment variable.
func ParseCommaSeparatedList(value string) []string {
	items := strings.Split(value, ",")
	for i, item := range items {
		items[i] = strings.TrimSpace(item)
	}
	return items
}

// applyEnvOverrides applies environment variable overrides to configuration.
func (b *ConfigBuilder) applyEnvOverrides(config *Configuration) *Configuration {
	envConfig := *config // Copy the config

	// Apply service overrides
	b.applyServiceEnvOverrides(&envConfig)

	// SPIFFE configuration override
	if val := os.Getenv(b.envPrefix + "_SPIFFE_SOCKET"); val != "" {
		if envConfig.SPIFFE == nil {
			envConfig.SPIFFE = &SPIFFEConfig{}
		}
		envConfig.SPIFFE.SocketPath = val
	}

	// Apply transport overrides
	b.applyTransportEnvOverrides(&envConfig)

	// Authorized clients override
	if val := os.Getenv(b.envPrefix + "_AUTHORIZED_CLIENTS"); val != "" {
		envConfig.AuthorizedClients = ParseCommaSeparatedList(val)
	}

	// Trusted servers override
	if val := os.Getenv(b.envPrefix + "_TRUSTED_SERVERS"); val != "" {
		envConfig.TrustedServers = ParseCommaSeparatedList(val)
	}

	return &envConfig
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
