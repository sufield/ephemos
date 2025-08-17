// Package transport provides factory functions for creating transport providers.
package transport

import (
	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"

	"github.com/sufield/ephemos/internal/core/ports"
)

// CreateGRPCProvider creates a rotation-capable gRPC transport provider.
// If sources are provided, the provider will support automatic SVID rotation.
func CreateGRPCProvider(config *ports.Configuration, opts ...ProviderOption) (*RotatableGRPCProvider, error) {
	provider := NewRotatableGRPCProvider(config)

	// Apply options - collect any errors
	for _, opt := range opts {
		if err := opt(provider); err != nil {
			return nil, err
		}
	}

	return provider, nil
}

// ProviderOption configures a transport provider.
type ProviderOption func(provider interface{}) error

// WithSources configures the provider with SVID and bundle sources for rotation.
func WithSources(svidSource x509svid.Source, bundleSource x509bundle.Source, authorizer tlsconfig.Authorizer) ProviderOption {
	return func(provider interface{}) error {
		if p, ok := provider.(*RotatableGRPCProvider); ok {
			return p.SetSources(svidSource, bundleSource, authorizer)
		}
		return nil
	}
}

// WithIdentityProvider creates sources from an identity provider for rotation support.
// The identity provider must implement the IdentityProvider interface.
func WithIdentityProvider(identityProvider IdentityProvider) ProviderOption {
	return func(provider interface{}) error {
		if p, ok := provider.(*RotatableGRPCProvider); ok {
			// Create source adapter from identity provider
			adapter := NewSourceAdapter(identityProvider)
			// Use the same adapter for both SVID and bundle sources
			return p.SetSources(adapter, adapter, tlsconfig.AuthorizeAny())
		}
		return nil
	}
}
