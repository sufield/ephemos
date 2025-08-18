// Package spiffe provides SPIFFE identity management and X.509 certificate handling.
package spiffe

import (
	"context"
	"crypto/x509"
	"fmt"
	"strings"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

// Provider provides SPIFFE identities.
type Provider struct {
	socketPath string
	x509Source *workloadapi.X509Source
}

// NewProvider creates a provider.
func NewProvider(config *ports.AgentConfig) (*Provider, error) {
	if config == nil {
		// Use default socket path when no config is provided
		return &Provider{
			socketPath: "/run/sockets/agent.sock",
		}, nil
	}

	return &Provider{
		socketPath: config.SocketPath,
	}, nil
}

// GetServiceIdentity fetches identity.
func (p *Provider) GetServiceIdentity() (*domain.ServiceIdentity, error) {
	ctx := context.Background() // Context managed at adapter layer
	if err := p.ensureSource(ctx); err != nil {
		return nil, err
	}

	svid, err := p.x509Source.GetX509SVID()
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 SVID: %w", err)
	}

	id := svid.ID
	serviceName := "unknown"
	path := id.Path()
	if path != "" && len(path) > 1 {
		// Remove leading slash and split by remaining slashes
		pathSegments := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(pathSegments) > 0 && pathSegments[0] != "" {
			serviceName = pathSegments[0]
		}
	}

	return domain.NewServiceIdentity(serviceName, id.TrustDomain().String()), nil
}

// GetCertificate fetches cert.
func (p *Provider) GetCertificate() (*domain.Certificate, error) {
	ctx := context.Background() // Context managed at adapter layer
	if err := p.ensureSource(ctx); err != nil {
		return nil, err
	}

	svid, err := p.x509Source.GetX509SVID()
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 SVID: %w", err)
	}

	return &domain.Certificate{
		Cert:       svid.Certificates[0],
		PrivateKey: svid.PrivateKey,
		Chain:      svid.Certificates,
	}, nil
}

// GetTrustBundle fetches bundle.
func (p *Provider) GetTrustBundle() (*domain.TrustBundle, error) {
	ctx := context.Background() // Context managed at adapter layer
	if err := p.ensureSource(ctx); err != nil {
		return nil, err
	}

	bundle, err := p.x509Source.GetX509BundleForTrustDomain(spiffeid.RequireTrustDomainFromString("example.org"))
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle: %w", err)
	}

	return domain.NewTrustBundle(bundle.X509Authorities())
}

// GetIdentityDocument fetches the complete identity document with certificate and private key.
func (p *Provider) GetIdentityDocument() (*domain.IdentityDocument, error) {
	ctx := context.Background() // Context managed at adapter layer
	if err := p.ensureSource(ctx); err != nil {
		return nil, err
	}

	svid, err := p.x509Source.GetX509SVID()
	if err != nil {
		return nil, fmt.Errorf("failed to get X509 SVID: %w", err)
	}

	// Get the trust bundle for validation
	bundle, err := p.x509Source.GetX509BundleForTrustDomain(svid.ID.TrustDomain())
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle for validation: %w", err)
	}

	// Use the first CA certificate from the bundle for validation
	caCerts := bundle.X509Authorities()
	var caCert *x509.Certificate
	if len(caCerts) > 0 {
		caCert = caCerts[0]
	}

	// Create identity document from SVID
	return domain.NewIdentityDocument(svid.Certificates, svid.PrivateKey, caCert)
}

func (p *Provider) ensureSource(ctx context.Context) error {
	if p.x509Source != nil {
		return nil
	}

	source, err := workloadapi.NewX509Source(
		ctx,
		workloadapi.WithClientOptions(
			workloadapi.WithAddr("unix://"+p.socketPath),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create X509 source: %w", err)
	}

	p.x509Source = source
	return nil
}

// GetTLSConfig gets TLS config.
func (p *Provider) GetTLSConfig(ctx context.Context) (tlsconfig.Authorizer, error) {
	if err := p.ensureSource(ctx); err != nil {
		return nil, err
	}

	return tlsconfig.AuthorizeAny(), nil
}

// GetX509Source returns source.
func (p *Provider) GetX509Source() *workloadapi.X509Source {
	return p.x509Source
}

// GetSocketPath returns path.
func (p *Provider) GetSocketPath() string {
	return p.socketPath
}

// Close closes the provider.
func (p *Provider) Close() error {
	if p.x509Source != nil {
		if err := p.x509Source.Close(); err != nil {
			return fmt.Errorf("failed to close X509 source: %w", err)
		}
	}
	return nil
}
