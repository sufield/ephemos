// Package ephemos provides SPIFFE provider implementation.
package ephemos

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"
	"sync"
)

// spiffeProviderImpl is the concrete implementation of SPIFFEProvider.
type spiffeProviderImpl struct {
	config        *SPIFFEConfig
	identity      ServiceIdentity
	tlsConfig     *tls.Config
	mu            sync.RWMutex
	isInitialized bool
}

// serviceIdentityImpl implements ServiceIdentity.
type serviceIdentityImpl struct {
	name     string
	domain   string
	spiffeID string
}

// NewSPIFFEProvider creates a new SPIFFE provider.
func NewSPIFFEProvider(config *SPIFFEConfig) (SPIFFEProvider, error) {
	if config == nil {
		// Use default config
		config = &SPIFFEConfig{
			SocketPath: "/tmp/spire-agent/public/api.sock",
		}
	}

	provider := &spiffeProviderImpl{
		config: config,
	}

	// Initialize with a default identity for now
	// In production, this would connect to SPIRE agent
	provider.identity = &serviceIdentityImpl{
		name:     "ephemos-service",
		domain:   "ephemos.local",
		spiffeID: "spiffe://ephemos.local/ephemos-service",
	}

	// Create a basic TLS config
	// In production, this would use SPIFFE certificates
	provider.tlsConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
	}

	provider.isInitialized = true
	return provider, nil
}

// GetTLSConfig returns TLS configuration for the given context.
func (p *spiffeProviderImpl) GetTLSConfig(ctx context.Context) (*tls.Config, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.isInitialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	// In production, this would fetch fresh certificates from SPIRE
	return p.tlsConfig.Clone(), nil
}

// GetServiceIdentity returns the service's SPIFFE identity.
func (p *spiffeProviderImpl) GetServiceIdentity() (ServiceIdentity, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.isInitialized {
		return nil, fmt.Errorf("provider not initialized")
	}

	return p.identity, nil
}

// Close releases resources.
func (p *spiffeProviderImpl) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.isInitialized = false
	// In production, this would close connection to SPIRE agent
	return nil
}

// GetName returns the service name.
func (i *serviceIdentityImpl) GetName() string {
	return i.name
}

// GetDomain returns the trust domain.
func (i *serviceIdentityImpl) GetDomain() string {
	return i.domain
}

// GetSPIFFEID returns the full SPIFFE ID.
func (i *serviceIdentityImpl) GetSPIFFEID() string {
	return i.spiffeID
}

// Validate checks if the identity is valid.
func (i *serviceIdentityImpl) Validate() error {
	if i.name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if i.domain == "" {
		return fmt.Errorf("trust domain cannot be empty")
	}
	if !strings.HasPrefix(i.spiffeID, "spiffe://") {
		return fmt.Errorf("invalid SPIFFE ID format")
	}
	return nil
}
