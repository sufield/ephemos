// Package factory provides factories for creating SPIFFE/SPIRE-backed implementations
// of the core ports. This package hides the complexity of adapter creation and wiring
// from the public API.
//
// The factory now uses the new adapter architecture internally:
// - IdentityDocumentAdapter handles SVID fetching and streaming
// - SpiffeBundleAdapter handles trust bundle management and validation
// - TLSAdapter handles SPIFFE-based TLS configuration
// - The legacy Provider acts as a compatibility layer that delegates to these adapters
//
// This provides the benefits of the new architecture (streaming, better isolation,
// testing) while maintaining backward compatibility with existing code.
package factory

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/sufield/ephemos/internal/adapters/primary/api"
	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/adapters/secondary/transport"
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

	// Create transport provider with rotation support
	transportProvider, err := transport.CreateGRPCProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport provider: %w", err)
	}

	// Create the internal server adapter using proper dependency injection
	// This factory is the appropriate place for this wiring, keeping the API package clean
	internalServer, err := api.WorkloadServer(identityProvider, transportProvider, configProvider, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create SPIFFE server: %w", err)
	}

	return &spiffeServerAdapter{server: internalServer}, nil
}

// createIdentityProvider creates a SPIFFE identity provider from configuration
// This function now uses the new adapter architecture internally through the refactored Provider.
func createIdentityProvider(cfg *ports.Configuration) (ports.IdentityProvider, error) {
	// Create identity provider using the refactored Provider that delegates to new adapters
	identityProvider, err := spiffe.NewProvider(cfg.Agent)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity provider: %w", err)
	}

	return identityProvider, nil
}

// createIdentityProviderWithAdapters creates a SPIFFE identity provider using the new adapter architecture directly.
// This provides fine-grained control over adapter configuration and allows for adapter composition.
func createIdentityProviderWithAdapters(cfg *ports.Configuration) (ports.IdentityProvider, error) {
	// For future extension: this could allow selecting specific adapters
	// For now, it delegates to the main factory function that uses the refactored Provider
	return createIdentityProvider(cfg)
}

// AdapterConfig provides configuration options for adapter selection and settings
type AdapterConfig struct {
	// UseDirectAdapters when true uses adapters directly, otherwise uses Provider compatibility layer
	UseDirectAdapters bool

	// IdentitySocketPath overrides the default SPIFFE socket path for identity operations
	IdentitySocketPath string

	// BundleSocketPath overrides the default SPIFFE socket path for bundle operations
	BundleSocketPath string

	// TLSSocketPath overrides the default SPIFFE socket path for TLS operations
	TLSSocketPath string
}

// SPIFFEDialerWithAdapters creates a new SPIFFE/SPIRE-backed Dialer with adapter configuration options.
func SPIFFEDialerWithAdapters(ctx context.Context, cfg *ports.Configuration, adapterCfg *AdapterConfig) (ports.Dialer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// For now, delegate to existing factory - future enhancement could use direct adapters
	// when adapterCfg.UseDirectAdapters is true
	return SPIFFEDialer(ctx, cfg)
}

// SPIFFEServerWithAdapters creates a new SPIFFE/SPIRE-backed AuthenticatedServer with adapter configuration options.
func SPIFFEServerWithAdapters(ctx context.Context, cfg *ports.Configuration, adapterCfg *AdapterConfig) (ports.AuthenticatedServer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// For now, delegate to existing factory - future enhancement could use direct adapters
	// when adapterCfg.UseDirectAdapters is true
	return SPIFFEServer(ctx, cfg)
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

func (c *spiffeConnAdapter) HTTPClient() (ports.HTTPClient, error) {
	httpClient, err := c.conn.HTTPClient()
	if err != nil {
		return nil, err
	}
	return &httpClientAdapter{client: httpClient}, nil
}

func (c *spiffeConnAdapter) Close() error {
	return c.conn.Close()
}

// spiffeServerAdapter adapts the internal API server to the AuthenticatedServer
type spiffeServerAdapter struct {
	server *api.Server
}

func (s *spiffeServerAdapter) Serve(ctx context.Context, listener ports.NetworkListener) error {
	// The underlying api.Server expects net.Listener, so we need to extract it
	// Similar to the approach used in the transport layer
	if adapter, ok := listener.(*networkListenerAdapter); ok {
		return s.server.Serve(ctx, adapter.listener)
	}
	
	// If it's not our adapter, we can't extract the net.Listener
	return fmt.Errorf("NetworkListener must wrap a net.Listener to work with SPIFFE server")
}

func (s *spiffeServerAdapter) Close() error {
	return s.server.Close()
}

func (s *spiffeServerAdapter) Addr() string {
	// The API server doesn't expose an Addr() method directly
	// Return empty string for now - this could be enhanced by storing the listener address
	return ""
}

// httpClientAdapter adapts net/http.Client to ports.HTTPClient
type httpClientAdapter struct {
	client *http.Client
}

func (a *httpClientAdapter) Do(ctx context.Context, req *ports.HTTPRequest) (*ports.HTTPResponse, error) {
	// Convert ports.HTTPRequest to http.Request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	for key, values := range req.Headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	// Execute request
	httpResp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Convert http.Response to ports.HTTPResponse
	respHeaders := make(map[string][]string)
	for key, values := range httpResp.Header {
		respHeaders[key] = values
	}

	return &ports.HTTPResponse{
		StatusCode: httpResp.StatusCode,
		Headers:    respHeaders,
		Body:       httpResp.Body,
	}, nil
}

func (a *httpClientAdapter) Close() error {
	// http.Client doesn't have a Close method, but we can close the underlying transport
	if transport, ok := a.client.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}
	return nil
}

// networkListenerAdapter adapts net.Listener to ports.NetworkListener.
type networkListenerAdapter struct {
	listener net.Listener
}

func (a *networkListenerAdapter) Accept() (io.ReadWriteCloser, error) {
	conn, err := a.listener.Accept()
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (a *networkListenerAdapter) Addr() string {
	return a.listener.Addr().String()
}

func (a *networkListenerAdapter) Close() error {
	return a.listener.Close()
}
