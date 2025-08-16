// Package factory provides factories for creating SPIFFE/SPIRE-backed implementations
// of the core ports. This package hides the complexity of adapter creation and wiring
// from the public API.
package factory

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/sufield/ephemos/internal/adapters/primary/api"
	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/core/ports"
)

// SPIFFEDialer creates a new SPIFFE/SPIRE-backed Dialer implementation.
// The configuration must be valid and contain the necessary SPIFFE settings.
func SPIFFEDialer(ctx context.Context, cfg *ports.Configuration) (ports.Dialer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Create identity provider
	identityProvider, err := createIdentityProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity provider: %w", err)
	}
	
	// Create the internal client adapter using the proper constructor
	// Note: This is the only place where we directly depend on the adapter
	internalClient, err := api.NewClient(identityProvider, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create SPIFFE dialer: %w", err)
	}
	
	return &spiffeDialerAdapter{client: internalClient}, nil
}

// SPIFFEServer creates a new SPIFFE/SPIRE-backed AuthenticatedServer implementation.
// The configuration must be valid and contain the necessary SPIFFE settings.
func SPIFFEServer(ctx context.Context, cfg *ports.Configuration) (ports.AuthenticatedServer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	// Create identity provider
	identityProvider, err := createIdentityProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity provider: %w", err)
	}
	
	// Create configuration provider
	configProvider := config.NewFileProvider()
	
	// Create the internal server adapter using the proper constructor
	// Note: This is the only place where we directly depend on the adapter
	internalServer, err := api.NewServer(identityProvider, configProvider, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create SPIFFE server: %w", err)
	}
	
	return &spiffeServerAdapter{server: internalServer}, nil
}

// createIdentityProvider creates a SPIFFE identity provider from configuration
func createIdentityProvider(cfg *ports.Configuration) (ports.IdentityProvider, error) {
	// Create identity provider
	identityProvider, err := spiffe.NewProvider(cfg.Agent)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity provider: %w", err)
	}
	
	return identityProvider, nil
}

// spiffeDialerAdapter adapts the internal API client to the Dialer port
type spiffeDialerAdapter struct {
	client *api.Client
}

func (d *spiffeDialerAdapter) Connect(ctx context.Context, serviceName, address string) (ports.Conn, error) {
	internalConn, err := d.client.Connect(ctx, serviceName, address)
	if err != nil {
		return nil, err
	}
	
	return &spiffeConnAdapter{conn: internalConn}, nil
}

func (d *spiffeDialerAdapter) Close() error {
	return d.client.Close()
}

// spiffeConnAdapter adapts the internal API connection to the Conn port
type spiffeConnAdapter struct {
	conn *api.ClientConnection
}

func (c *spiffeConnAdapter) HTTPClient() (*http.Client, error) {
	return c.conn.HTTPClient()
}

func (c *spiffeConnAdapter) Close() error {
	return c.conn.Close()
}

// spiffeServerAdapter adapts the internal API server to the AuthenticatedServer
type spiffeServerAdapter struct {
	server *api.Server
}

func (s *spiffeServerAdapter) Serve(ctx context.Context, lis net.Listener) error {
	return s.server.Serve(ctx, lis)
}

func (s *spiffeServerAdapter) Close() error {
	return s.server.Close()
}

func (s *spiffeServerAdapter) Addr() net.Addr {
	// The API server doesn't expose an Addr() method directly
	// Return nil for now - this could be enhanced by storing the listener address
	return nil
}