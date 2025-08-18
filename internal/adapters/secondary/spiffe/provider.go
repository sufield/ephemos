// Package spiffe provides SPIFFE identity management and X.509 certificate handling.
package spiffe

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Provider provides SPIFFE identities using the new adapter architecture.
// This is a compatibility layer that delegates to the specialized adapters.
type Provider struct {
	identityAdapter *IdentityDocumentAdapter
	bundleAdapter   *SpiffeBundleAdapter
	tlsAdapter      *TLSAdapter
}

// NewProvider creates a provider using the new adapter architecture.
func NewProvider(config *ports.AgentConfig) (*Provider, error) {
	var socketPath string
	if config != nil {
		socketPath = config.SocketPath
	} else {
		// For nil config, use default path without unix:// prefix to match tests
		socketPath = "/tmp/spire-agent/public/api.sock"
	}
	
	logger := slog.Default()
	
	// Create specialized adapters
	identityAdapter, err := NewIdentityDocumentAdapter(IdentityDocumentAdapterConfig{
		SocketPath: socketPath,
		Logger:     logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create identity adapter: %w", err)
	}
	
	bundleAdapter, err := NewSpiffeBundleAdapter(SpiffeBundleAdapterConfig{
		SocketPath: socketPath,
		Logger:     logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bundle adapter: %w", err)
	}
	
	tlsAdapter, err := NewTLSAdapter(TLSAdapterConfig{
		SocketPath: socketPath,
		Logger:     logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS adapter: %w", err)
	}
	
	return &Provider{
		identityAdapter: identityAdapter,
		bundleAdapter:   bundleAdapter,
		tlsAdapter:      tlsAdapter,
	}, nil
}

// GetServiceIdentity fetches identity using the identity adapter.
func (p *Provider) GetServiceIdentity() (*domain.ServiceIdentity, error) {
	ctx := context.Background() // Context managed at adapter layer
	return p.identityAdapter.GetServiceIdentity(ctx)
}

// GetCertificate fetches cert using the identity adapter.
func (p *Provider) GetCertificate() (*domain.Certificate, error) {
	ctx := context.Background() // Context managed at adapter layer
	return p.identityAdapter.GetCertificate(ctx)
}

// GetTrustBundle fetches bundle using the bundle adapter.
func (p *Provider) GetTrustBundle() (*domain.TrustBundle, error) {
	ctx := context.Background() // Context managed at adapter layer
	return p.bundleAdapter.GetTrustBundle(ctx)
}

// GetIdentityDocument fetches the complete identity document using the identity adapter.
func (p *Provider) GetIdentityDocument() (*domain.IdentityDocument, error) {
	ctx := context.Background() // Context managed at adapter layer
	return p.identityAdapter.GetIdentityDocument(ctx)
}


// GetTLSConfig gets TLS config using the TLS adapter.
func (p *Provider) GetTLSConfig(ctx context.Context) (tlsconfig.Authorizer, error) {
	return p.tlsAdapter.GetTLSAuthorizer(nil) // Use default policy
}

// GetX509Source returns source from identity adapter.
// Note: This exposes internal implementation and should be avoided in new code.
func (p *Provider) GetX509Source() *workloadapi.X509Source {
	// For backward compatibility, we need to access the internal source
	// This is not ideal but may be needed by existing code
	if p.identityAdapter != nil {
		// Access internal source from adapter - this is a compatibility hack
		return p.identityAdapter.x509Source
	}
	return nil
}

// GetSocketPath returns path from identity adapter.
func (p *Provider) GetSocketPath() string {
	if p.identityAdapter != nil {
		return p.identityAdapter.socketPath
	}
	return ""
}

// Close closes the provider and all its adapters.
func (p *Provider) Close() error {
	// Close all adapters
	if p.identityAdapter != nil {
		if err := p.identityAdapter.Close(); err != nil {
			return fmt.Errorf("failed to close identity adapter: %w", err)
		}
	}
	
	if p.bundleAdapter != nil {
		if err := p.bundleAdapter.Close(); err != nil {
			return fmt.Errorf("failed to close bundle adapter: %w", err)
		}
	}
	
	if p.tlsAdapter != nil {
		if err := p.tlsAdapter.Close(); err != nil {
			return fmt.Errorf("failed to close TLS adapter: %w", err)
		}
	}
	
	return nil
}
