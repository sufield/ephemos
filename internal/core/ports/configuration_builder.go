// Package ports provides configuration builder for clean configuration construction.
package ports

import (
	"fmt"

	"github.com/sufield/ephemos/internal/core/domain"
)

// ConfigurationBuilder provides a fluent interface for building configurations.
// This builder pattern improves configuration construction ergonomics and validation.
type ConfigurationBuilder struct {
	config *Configuration
}

// NewConfigurationBuilder creates a new configuration builder.
func NewConfigurationBuilder() *ConfigurationBuilder {
	return &ConfigurationBuilder{
		config: &Configuration{
			Service: ServiceConfig{},
			Agent:   &AgentConfig{},
		},
	}
}

// WithService sets the service name and domain.
func (b *ConfigurationBuilder) WithService(name string, trustDomain string) *ConfigurationBuilder {
	b.config.Service.Name = domain.NewServiceNameUnsafe(name)
	b.config.Service.Domain = trustDomain
	return b
}

// WithServiceName sets just the service name.
func (b *ConfigurationBuilder) WithServiceName(name string) *ConfigurationBuilder {
	b.config.Service.Name = domain.NewServiceNameUnsafe(name)
	return b
}

// WithTrustDomain sets just the trust domain.
func (b *ConfigurationBuilder) WithTrustDomain(domain string) *ConfigurationBuilder {
	b.config.Service.Domain = domain
	return b
}

// WithCacheTTL sets the cache TTL in minutes.
func (b *ConfigurationBuilder) WithCacheTTL(minutes int) *ConfigurationBuilder {
	// Extract intermediate to avoid deep selector chains
	service := &b.config.Service
	if service.Cache == nil {
		service.Cache = &CacheConfig{}
	}
	service.Cache.TTLMinutes = minutes
	return b
}

// WithCacheRefresh sets the proactive refresh time in minutes.
func (b *ConfigurationBuilder) WithCacheRefresh(minutes int) *ConfigurationBuilder {
	// Extract intermediate to avoid deep selector chains
	service := &b.config.Service
	if service.Cache == nil {
		service.Cache = &CacheConfig{}
	}
	service.Cache.ProactiveRefreshMinutes = minutes
	return b
}

// WithAgentSocket sets the agent socket path.
func (b *ConfigurationBuilder) WithAgentSocket(socketPath string) *ConfigurationBuilder {
	if b.config.Agent == nil {
		b.config.Agent = &AgentConfig{}
	}
	b.config.Agent.SocketPath = domain.NewSocketPathUnsafe(socketPath)
	return b
}

// WithAuthorizedClients sets the authorized client SPIFFE IDs.
func (b *ConfigurationBuilder) WithAuthorizedClients(clients []string) *ConfigurationBuilder {
	b.config.Service.AuthorizedClients = clients
	return b
}

// WithTrustedServers sets the trusted server SPIFFE IDs.
func (b *ConfigurationBuilder) WithTrustedServers(servers []string) *ConfigurationBuilder {
	b.config.Service.TrustedServers = servers
	return b
}

// Note: InsecureSkipVerify is controlled via environment variables
// and is not part of the standard configuration structure for security reasons.

// Build constructs and validates the final configuration.
func (b *ConfigurationBuilder) Build() (*Configuration, error) {
	// Validate required fields
	// Extract intermediate to avoid deep selector chains
	service := &b.config.Service
	if service.Name.Value() == "" {
		return nil, fmt.Errorf("service name is required")
	}
	
	if b.config.Service.Domain == "" {
		return nil, fmt.Errorf("trust domain is required")
	}

	// Perform full configuration validation
	if err := b.config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return b.config, nil
}

// BuildUnsafe constructs the configuration without validation (for testing).
func (b *ConfigurationBuilder) BuildUnsafe() *Configuration {
	return b.config
}