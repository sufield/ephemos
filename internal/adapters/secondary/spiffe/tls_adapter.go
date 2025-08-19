// Package spiffe provides SPIFFE TLS configuration adapters.
package spiffe

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"

	"github.com/sufield/ephemos/internal/core/domain"
)

// TLSAdapter provides SPIFFE-based TLS configuration.
// This adapter creates TLS configurations using SPIFFE identities and trust bundles.
type TLSAdapter struct {
	x509SourceProvider *X509SourceProvider
	logger        *slog.Logger
}

// TLSAdapterConfig provides configuration for the TLS adapter.
type TLSAdapterConfig struct {
	X509SourceProvider *X509SourceProvider
	Logger        *slog.Logger
}

// NewTLSAdapter creates a new SPIFFE TLS adapter.
func NewTLSAdapter(config TLSAdapterConfig) (*TLSAdapter, error) {
	if config.X509SourceProvider == nil {
		return nil, fmt.Errorf("source manager is required")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &TLSAdapter{
		x509SourceProvider: config.X509SourceProvider,
		logger:        logger,
	}, nil
}

// CreateClientTLSConfig creates a TLS configuration for SPIFFE clients.
func (a *TLSAdapter) CreateClientTLSConfig(ctx context.Context, policy *domain.AuthenticationPolicy) (*tls.Config, error) {
	a.logger.Debug("creating client TLS config with SPIFFE")

	x509Source, err := a.x509SourceProvider.GetOrCreateSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 source: %w", err)
	}

	// Determine authorizer from policy
	authorizer, err := a.createAuthorizer(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %w", err)
	}

	// Create SPIFFE mTLS client config
	tlsConfig := tlsconfig.MTLSClientConfig(x509Source, x509Source, authorizer)

	a.logger.Debug("client TLS config created")
	return tlsConfig, nil
}

// CreateServerTLSConfig creates a TLS configuration for SPIFFE servers.
func (a *TLSAdapter) CreateServerTLSConfig(ctx context.Context, policy *domain.AuthenticationPolicy) (*tls.Config, error) {
	a.logger.Debug("creating server TLS config with SPIFFE")

	x509Source, err := a.x509SourceProvider.GetOrCreateSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 source: %w", err)
	}

	// Determine authorizer from policy
	authorizer, err := a.createAuthorizer(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer: %w", err)
	}

	// Create SPIFFE mTLS server config
	tlsConfig := tlsconfig.MTLSServerConfig(x509Source, x509Source, authorizer)

	a.logger.Debug("server TLS config created")
	return tlsConfig, nil
}

// CreateClientTLSConfigWithTarget creates a client TLS config for a specific target service.
func (a *TLSAdapter) CreateClientTLSConfigWithTarget(ctx context.Context, targetSPIFFEID string) (*tls.Config, error) {
	a.logger.Debug("creating client TLS config for specific target",
		"target", targetSPIFFEID)

	x509Source, err := a.x509SourceProvider.GetOrCreateSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 source: %w", err)
	}

	// Parse target SPIFFE ID
	id, err := spiffeid.FromString(targetSPIFFEID)
	if err != nil {
		return nil, fmt.Errorf("invalid target SPIFFE ID %q: %w", targetSPIFFEID, err)
	}

	// Create authorizer for specific ID
	authorizer := tlsconfig.AuthorizeID(id)

	// Create SPIFFE mTLS client config
	tlsConfig := tlsconfig.MTLSClientConfig(x509Source, x509Source, authorizer)

	a.logger.Debug("client TLS config created for target",
		"target", targetSPIFFEID)
	return tlsConfig, nil
}

// CreateServerTLSConfigWithAllowedClients creates a server TLS config that allows specific clients.
func (a *TLSAdapter) CreateServerTLSConfigWithAllowedClients(ctx context.Context, allowedClients []string) (*tls.Config, error) {
	a.logger.Debug("creating server TLS config with allowed clients",
		"client_count", len(allowedClients))

	x509Source, err := a.x509SourceProvider.GetOrCreateSource(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 source: %w", err)
	}

	// Parse allowed client SPIFFE IDs
	var allowedIDs []spiffeid.ID
	for i, clientStr := range allowedClients {
		id, err := spiffeid.FromString(clientStr)
		if err != nil {
			return nil, fmt.Errorf("invalid client SPIFFE ID at index %d (%q): %w", i, clientStr, err)
		}
		allowedIDs = append(allowedIDs, id)
	}

	// Create authorizer for allowed IDs
	var authorizer tlsconfig.Authorizer
	if len(allowedIDs) == 0 {
		// No specific clients - use any member of trust domain
		authorizer = tlsconfig.AuthorizeAny()
	} else if len(allowedIDs) == 1 {
		// Single client
		authorizer = tlsconfig.AuthorizeID(allowedIDs[0])
	} else {
		// Multiple clients
		authorizer = tlsconfig.AuthorizeOneOf(allowedIDs...)
	}

	// Create SPIFFE mTLS server config
	tlsConfig := tlsconfig.MTLSServerConfig(x509Source, x509Source, authorizer)

	a.logger.Debug("server TLS config created with allowed clients")
	return tlsConfig, nil
}

// GetTLSAuthorizer creates a TLS authorizer based on policy.
func (a *TLSAdapter) GetTLSAuthorizer(policy *domain.AuthenticationPolicy) (tlsconfig.Authorizer, error) {
	return a.createAuthorizer(policy)
}

// createAuthorizer creates a SPIFFE TLS authorizer from authentication policy.
func (a *TLSAdapter) createAuthorizer(policy *domain.AuthenticationPolicy) (tlsconfig.Authorizer, error) {
	if policy == nil {
		a.logger.Debug("no policy provided, using AuthorizeAny for SPIFFE identity validation")
		return tlsconfig.AuthorizeAny(), nil
	}

	// Authentication-only: authorize based on trust domain membership
	if !policy.TrustDomain.IsZero() {
		td, err := spiffeid.TrustDomainFromString(policy.TrustDomain.String())
		if err != nil {
			return nil, fmt.Errorf("invalid trust domain in policy: %w", err)
		}
		a.logger.Debug("authorizing trust domain members for authentication",
			"trust_domain", td.String())
		return tlsconfig.AuthorizeMemberOf(td), nil
	}

	// Default: validate any valid SPIFFE identity (authentication-only)
	a.logger.Debug("using SPIFFE identity validation (authentication-only)")
	return tlsconfig.AuthorizeAny(), nil
}

// Close releases resources held by the adapter.
func (a *TLSAdapter) Close() error {
	a.logger.Debug("closing SPIFFE TLS adapter")

	// Note: X509 source is managed by X509SourceProvider, not closed here

	a.logger.Debug("SPIFFE TLS adapter closed")
	return nil
}

