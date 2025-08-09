// Package api provides high-level client and server APIs for secure SPIFFE-based communication.
package api

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/adapters/secondary/transport"
	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
	"google.golang.org/grpc"
)

// IdentityClient provides a high-level API for connecting to SPIFFE-secured services.
type IdentityClient struct {
	identityService *services.IdentityService
	connection      services.ClientConnection
	mu              sync.Mutex
}

// NewIdentityClient creates a new IdentityClient with the given configuration file path.
func NewIdentityClient(ctx context.Context, configPath string) (*IdentityClient, error) {
	configProvider := config.NewFileProvider()

	var cfg *ports.Configuration
	var err error

	if configPath != "" {
		cfg, err = configProvider.LoadConfiguration(ctx, configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration: %w", err)
		}
	} else {
		cfg = configProvider.GetDefaultConfiguration(ctx)
		if cfg == nil {
			return nil, &errors.ValidationError{
				Field:   "configuration",
				Value:   nil,
				Message: "no configuration provided and no default configuration available",
			}
		}
	}

	spiffeProvider, err := spiffe.NewProvider(cfg.SPIFFE)
	if err != nil {
		return nil, fmt.Errorf("failed to create SPIFFE provider: %w", err)
	}

	transportProvider := transport.NewGRPCProvider(spiffeProvider)

	identityService, err := services.NewIdentityService(
		spiffeProvider,
		transportProvider,
		cfg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service: %w", err)
	}

	return &IdentityClient{
		identityService: identityService,
	}, nil
}

// Connect establishes a secure connection to a remote service using SPIFFE identities.
func (c *IdentityClient) Connect(ctx context.Context, serviceName, address string) (*ClientConnection, error) {
	// Input validation
	if ctx == nil {
		return nil, &errors.ValidationError{
			Field:   "context",
			Value:   nil,
			Message: "context cannot be nil",
		}
	}

	if strings.TrimSpace(serviceName) == "" {
		return nil, &errors.ValidationError{
			Field:   "serviceName",
			Value:   serviceName,
			Message: "service name cannot be empty or whitespace",
		}
	}

	if strings.TrimSpace(address) == "" {
		return nil, &errors.ValidationError{
			Field:   "address",
			Value:   address,
			Message: "address cannot be empty or whitespace",
		}
	}

	// Validate address format (host:port)
	if _, _, err := net.SplitHostPort(address); err != nil {
		return nil, &errors.ValidationError{
			Field:   "address",
			Value:   address,
			Message: "address must be in format 'host:port'",
		}
	}

	serviceName = strings.TrimSpace(serviceName)
	address = strings.TrimSpace(address)

	// Thread-safe connection initialization
	c.mu.Lock()
	if c.connection == nil {
		conn, err := c.identityService.CreateClientIdentity(ctx)
		if err != nil {
			c.mu.Unlock()
			return nil, fmt.Errorf("failed to create client identity: %w", err)
		}
		c.connection = conn
	}
	c.mu.Unlock()

	grpcConn, err := c.connection.Connect(ctx, serviceName, address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service %s at %s: %w", serviceName, address, err)
	}

	return &ClientConnection{conn: grpcConn}, nil
}

// Close cleans up the client resources and closes any connections.
func (c *IdentityClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connection != nil {
		// Close the underlying connection if it supports closing
		// This is a safe no-op if the connection doesn't need explicit cleanup
		c.connection = nil
	}

	return nil
}

// ClientConnection represents a secure client connection to a remote service.
type ClientConnection struct {
	conn *grpc.ClientConn
}

// Close terminates the client connection and cleans up resources.
func (c *ClientConnection) Close() error {
	if c.conn == nil {
		return nil // Safe to call Close on nil connection
	}
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("failed to close client connection: %w", err)
	}
	return nil
}

// GetClientConnection returns the underlying gRPC client connection.
func (c *ClientConnection) GetClientConnection() *grpc.ClientConn {
	return c.conn
}
