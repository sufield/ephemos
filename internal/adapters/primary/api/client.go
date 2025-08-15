// Package api provides high-level client and server APIs for secure SPIFFE-based communication.
package api

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"google.golang.org/grpc"

	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// Client provides a high-level API for connecting to SPIFFE-secured services.
type Client struct {
	identityService *services.IdentityService
	domainClient    ports.Client
	mu              sync.Mutex
}

// IdentityClient creates a new identity client with injected dependencies.
// This constructor follows proper dependency injection and hexagonal architecture principles.
func IdentityClient(
	identityProvider ports.IdentityProvider,
	transportProvider ports.TransportProvider,
	cfg *ports.Configuration,
) (*Client, error) {
	if cfg == nil {
		return nil, &errors.ValidationError{
			Field:   "configuration",
			Value:   nil,
			Message: "configuration cannot be nil",
		}
	}

	if identityProvider == nil {
		return nil, &errors.ValidationError{
			Field:   "identityProvider",
			Value:   nil,
			Message: "identity provider cannot be nil",
		}
	}

	if transportProvider == nil {
		return nil, &errors.ValidationError{
			Field:   "transportProvider",
			Value:   nil,
			Message: "transport provider cannot be nil",
		}
	}

	identityService, err := services.NewIdentityService(
		identityProvider,
		transportProvider,
		cfg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service: %w", err)
	}

	return &Client{
		identityService: identityService,
	}, nil
}

// Connect establishes a secure connection to a remote service using SPIFFE identities.
func (c *Client) Connect(ctx context.Context, serviceName, address string) (*ClientConnection, error) {
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
	if c.domainClient == nil {
		client, err := c.identityService.CreateClientIdentity()
		if err != nil {
			c.mu.Unlock()
			return nil, fmt.Errorf("failed to create client identity: %w", err)
		}
		c.domainClient = client
	}
	c.mu.Unlock()

	domainConn, err := c.domainClient.Connect(serviceName, address)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to service %s at %s: %w", serviceName, address, err)
	}

	// Extract the underlying gRPC connection
	grpcConn, ok := domainConn.GetClientConnection().(*grpc.ClientConn)
	if !ok {
		return nil, fmt.Errorf("unexpected connection type from domain client")
	}

	return &ClientConnection{conn: grpcConn, domainConn: domainConn}, nil
}

// Close cleans up the client resources and closes any connections.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.domainClient != nil {
		if err := c.domainClient.Close(); err != nil {
			return fmt.Errorf("failed to close domain client: %w", err)
		}
		c.domainClient = nil
	}

	return nil
}

// ClientConnection represents a secure client connection to a remote service.
type ClientConnection struct {
	conn       *grpc.ClientConn
	domainConn ports.Connection
}

// Close terminates the client connection and cleans up resources.
func (c *ClientConnection) Close() error {
	if c.domainConn != nil {
		if err := c.domainConn.Close(); err != nil {
			return fmt.Errorf("failed to close domain connection: %w", err)
		}
	}
	return nil
}

// GetClientConnection returns the underlying gRPC client connection.
func (c *ClientConnection) GetClientConnection() *grpc.ClientConn {
	return c.conn
}
