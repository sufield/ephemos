package spiffe

import (
	"context"
	"fmt"
	"strings"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"
	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

type SPIFFEProvider struct {
	socketPath string
	x509Source *workloadapi.X509Source
}

func NewSPIFFEProvider(config *ports.SPIFFEConfig) (*SPIFFEProvider, error) {
	if config == nil {
		// Use default socket path when no config is provided
		return &SPIFFEProvider{
			socketPath: "/tmp/spire-agent/public/api.sock",
		}, nil
	}

	return &SPIFFEProvider{
		socketPath: config.SocketPath,
	}, nil
}

func (p *SPIFFEProvider) GetServiceIdentity(ctx context.Context) (*domain.ServiceIdentity, error) {
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

	return &domain.ServiceIdentity{
		Name:   serviceName,
		Domain: id.TrustDomain().String(),
		URI:    id.String(),
	}, nil
}

func (p *SPIFFEProvider) GetCertificate(ctx context.Context) (*domain.Certificate, error) {
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

func (p *SPIFFEProvider) GetTrustBundle(ctx context.Context) (*domain.TrustBundle, error) {
	if err := p.ensureSource(ctx); err != nil {
		return nil, err
	}

	bundle, err := p.x509Source.GetX509BundleForTrustDomain(spiffeid.RequireTrustDomainFromString("example.org"))
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle: %w", err)
	}

	return &domain.TrustBundle{
		Certificates: bundle.X509Authorities(),
	}, nil
}

func (p *SPIFFEProvider) ensureSource(ctx context.Context) error {
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

func (p *SPIFFEProvider) GetTLSConfig(ctx context.Context) (tlsconfig.Authorizer, error) {
	if err := p.ensureSource(ctx); err != nil {
		return nil, err
	}

	return tlsconfig.AuthorizeAny(), nil
}

func (p *SPIFFEProvider) GetX509Source() *workloadapi.X509Source {
	return p.x509Source
}

func (p *SPIFFEProvider) Close() error {
	if p.x509Source != nil {
		if err := p.x509Source.Close(); err != nil {
			return fmt.Errorf("failed to close X509 source: %w", err)
		}
	}
	return nil
}
