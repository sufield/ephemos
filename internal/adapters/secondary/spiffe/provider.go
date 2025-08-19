// Package spiffe provides SPIFFE identity management and X.509 certificate handling.
package spiffe

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Provider provides SPIFFE identities using the new adapter architecture.
// This is a compatibility layer that delegates to the specialized adapters.
type Provider struct {
	x509SourceProvider   *X509SourceProvider
	identityAdapter *IdentityDocumentAdapter
	bundleAdapter   *SpiffeBundleAdapter
	tlsAdapter      *TLSAdapter
}

// NewProvider creates a provider using the new adapter architecture.
func NewProvider(config *ports.AgentConfig) (*Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("agent configuration must be provided - no fallback patterns allowed")
	}
	if config.SocketPath.IsEmpty() {
		return nil, fmt.Errorf("SPIFFE socket path must be explicitly configured in agent config - no fallback patterns allowed")
	}
	socketPath := config.SocketPath

	logger := slog.Default()

	// Create shared X509 source provider
	x509SourceProvider := NewX509SourceProvider(socketPath, logger)

	// Create specialized adapters with shared X509 source provider
	identityAdapter, err := NewIdentityDocumentAdapter(IdentityDocumentAdapterConfig{
		X509SourceProvider: x509SourceProvider,
		Logger:        logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create identity adapter: %w", err)
	}

	bundleAdapter, err := NewSpiffeBundleAdapter(SpiffeBundleAdapterConfig{
		X509SourceProvider: x509SourceProvider,
		Logger:        logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create bundle adapter: %w", err)
	}

	tlsAdapter, err := NewTLSAdapter(TLSAdapterConfig{
		X509SourceProvider: x509SourceProvider,
		Logger:        logger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS adapter: %w", err)
	}

	return &Provider{
		x509SourceProvider:   x509SourceProvider,
		identityAdapter: identityAdapter,
		bundleAdapter:   bundleAdapter,
		tlsAdapter:      tlsAdapter,
	}, nil
}

// GetServiceIdentity fetches identity using the identity adapter.
func (p *Provider) GetServiceIdentity() (spiffeid.ID, error) {
	ctx := context.Background() // Context managed at adapter layer
	return p.identityAdapter.GetServiceIdentity(ctx)
}

// GetCertificate fetches cert using the identity adapter.
func (p *Provider) GetCertificate() (*domain.Certificate, error) {
	ctx := context.Background() // Context managed at adapter layer
	return p.identityAdapter.GetCertificate(ctx)
}

// GetTrustBundle fetches bundle using the bundle adapter.
func (p *Provider) GetTrustBundle() (*x509bundle.Bundle, error) {
	ctx := context.Background() // Context managed at adapter layer
	return p.bundleAdapter.GetTrustBundle(ctx)
}

// GetSVID fetches the complete SVID using the identity adapter.
func (p *Provider) GetSVID() (*x509svid.SVID, error) {
	ctx := context.Background() // Context managed at adapter layer
	return p.identityAdapter.GetSVID(ctx)
}

// GetTLSConfig gets TLS config using the TLS adapter.
func (p *Provider) GetTLSConfig(ctx context.Context) (tlsconfig.Authorizer, error) {
	return p.tlsAdapter.GetTLSAuthorizer(nil) // Use default policy
}

// GetX509Source returns source from X509 source provider.
// Note: This exposes internal implementation and should be avoided in new code.
func (p *Provider) GetX509Source() *workloadapi.X509Source {
	// For backward compatibility, we need to access the internal source
	// This is not ideal but may be needed by existing code
	if p.x509SourceProvider != nil && p.x509SourceProvider.IsInitialized() {
		// Try to get the source without creating it if it doesn't exist
		ctx := context.Background()
		source, _ := p.x509SourceProvider.GetOrCreateSource(ctx)
		return source
	}
	return nil
}

// GetSocketPath returns path from X509 source provider.
func (p *Provider) GetSocketPath() string {
	if p.x509SourceProvider != nil {
		return p.x509SourceProvider.socketPath.Value()
	}
	return ""
}

// Close closes the provider and all its adapters.
func (p *Provider) Close() error {
	// Close all adapters first
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

	// Close shared X509 source provider last
	if p.x509SourceProvider != nil {
		if err := p.x509SourceProvider.Close(); err != nil {
			return fmt.Errorf("failed to close X509 source provider: %w", err)
		}
	}

	return nil
}
