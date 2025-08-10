package config

import (
	"context"
	"fmt"
	"sync"

	"github.com/sufield/ephemos/internal/core/ports"
)

// InMemoryProvider is a real implementation of ConfigurationProvider that stores
// configurations in memory. This is useful for testing without mocks.
type InMemoryProvider struct {
	mu            sync.RWMutex
	configs       map[string]*ports.Configuration
	defaultConfig *ports.Configuration
}

// NewInMemoryProvider creates a new in-memory configuration provider.
func NewInMemoryProvider() *InMemoryProvider {
	return &InMemoryProvider{
		configs: make(map[string]*ports.Configuration),
		defaultConfig: &ports.Configuration{
			Service: ports.ServiceConfig{
				Name:   "default-service",
				Domain: "default.org",
			},
			SPIFFE: &ports.SPIFFEConfig{
				SocketPath: "/tmp/spire-agent/public/api.sock",
			},
		},
	}
}

// SetConfiguration stores a configuration for a given path.
func (p *InMemoryProvider) SetConfiguration(path string, config *ports.Configuration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.configs[path] = config
}

// SetDefaultConfiguration updates the default configuration.
func (p *InMemoryProvider) SetDefaultConfiguration(config *ports.Configuration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.defaultConfig = config
}

// LoadConfiguration retrieves a configuration by path.
func (p *InMemoryProvider) LoadConfiguration(ctx context.Context, path string) (*ports.Configuration, error) {
	// Check for context cancellation
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("context canceled: %w", ctx.Err())
		default:
		}
	}

	p.mu.RLock()
	defer p.mu.RUnlock()

	if path == "" {
		return p.defaultConfig, nil
	}

	config, ok := p.configs[path]
	if !ok {
		return nil, fmt.Errorf("configuration not found for path: %s", path)
	}

	// Return a copy to prevent mutations
	return copyConfiguration(config), nil
}

// GetDefaultConfiguration returns the default configuration.
func (p *InMemoryProvider) GetDefaultConfiguration(ctx context.Context) *ports.Configuration {
	// Check for context cancellation
	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil // Return nil on cancellation
		default:
		}
	}

	p.mu.RLock()
	defer p.mu.RUnlock()
	return copyConfiguration(p.defaultConfig)
}

// copyConfiguration creates a deep copy of a configuration.
func copyConfiguration(config *ports.Configuration) *ports.Configuration {
	if config == nil {
		return nil
	}

	cfgCopy := &ports.Configuration{
		Service: config.Service,
	}

	if config.SPIFFE != nil {
		cfgCopy.SPIFFE = &ports.SPIFFEConfig{
			SocketPath: config.SPIFFE.SocketPath,
		}
	}

	if config.AuthorizedClients != nil {
		cfgCopy.AuthorizedClients = make([]string, len(config.AuthorizedClients))
		cfgCopy.AuthorizedClients = append(cfgCopy.AuthorizedClients, config.AuthorizedClients...)
	}

	if config.TrustedServers != nil {
		cfgCopy.TrustedServers = make([]string, len(config.TrustedServers))
		cfgCopy.TrustedServers = append(cfgCopy.TrustedServers, config.TrustedServers...)
	}

	return cfgCopy
}
