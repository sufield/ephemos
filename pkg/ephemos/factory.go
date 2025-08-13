package ephemos

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
)

// createServerWithConfig creates a server with internal dependency injection.
// This is an internal factory function that handles the complexity of dependency injection.
func createServerWithConfig(_ context.Context, config *Configuration) (Server, error) {
	if config == nil {
		return nil, &ValidationError{
			Field:   "configuration",
			Value:   nil,
			Message: "configuration cannot be nil",
		}
	}

	// For now, return a simple implementation that satisfies the interface
	// In the future, this will create proper adapters with dependency injection
	return &serverImpl{
		config: config,
	}, nil
}

// createClientWithConfig creates a client with internal dependency injection.
// This is an internal factory function that handles the complexity of dependency injection.
func createClientWithConfig(_ context.Context, config *Configuration) (Client, error) {
	if config == nil {
		return nil, &ValidationError{
			Field:   "configuration",
			Value:   nil,
			Message: "configuration cannot be nil",
		}
	}

	// For now, return a simple implementation that satisfies the interface
	// In the future, this will create proper adapters with dependency injection
	return &clientImpl{
		config: config,
	}, nil
}

// Simple implementations that will be replaced with proper adapter-based implementations

type serverImpl struct {
	config *Configuration
}

func (s *serverImpl) RegisterService(ctx context.Context, serviceRegistrar ServiceRegistrar) error {
	if serviceRegistrar == nil {
		return &ValidationError{
			Field:   "serviceRegistrar",
			Value:   nil,
			Message: "service registrar cannot be nil",
		}
	}
	// Simple implementation for now - in production this would use proper adapters
	return fmt.Errorf("server implementation not yet complete - use NewTransportServer for now")
}

func (s *serverImpl) Serve(ctx context.Context, listener net.Listener) error {
	if listener == nil {
		return &ValidationError{
			Field:   "listener",
			Value:   nil,
			Message: "listener cannot be nil",
		}
	}
	// Simple implementation for now - in production this would use proper adapters
	return fmt.Errorf("server implementation not yet complete - use NewTransportServer for now")
}

func (s *serverImpl) Close() error {
	// Simple implementation for now - no cleanup needed
	return nil
}

type clientImpl struct {
	config *Configuration
}

func (c *clientImpl) Connect(ctx context.Context, serviceName, address string) (*ClientConnection, error) {
	if serviceName == "" {
		return nil, &ValidationError{
			Field:   "serviceName",
			Value:   serviceName,
			Message: "service name cannot be empty",
		}
	}
	if address == "" {
		return nil, &ValidationError{
			Field:   "address",
			Value:   address,
			Message: "address cannot be empty",
		}
	}
	// Simple implementation for now - in production this would use proper adapters
	return nil, fmt.Errorf("client implementation not yet complete - use NewTransportServer for now")
}

func (c *clientImpl) Close() error {
	// Simple implementation for now - no cleanup needed
	return nil
}

// ClientConnection placeholder type for now
type ClientConnection struct {
	// Simple placeholder - in production would contain actual gRPC connection
	conn *grpc.ClientConn
}

func (c *ClientConnection) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *ClientConnection) GetClientConnection() *grpc.ClientConn {
	return c.conn
}