package ephemos

import (
	"context"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
)

// createServerWithConfig creates a server with internal dependency injection.
// This is an internal factory function that handles the complexity of dependency injection.
func createServerWithConfig(ctx context.Context, config *Configuration) (Server, error) {
	if config == nil {
		return nil, &ValidationError{
			Field:   "configuration",
			Value:   nil,
			Message: "configuration cannot be nil",
		}
	}

	// Create a proper server implementation using TransportServer
	return &transportServerWrapper{
		config: config,
		ctx:    ctx,
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

// transportServerWrapper wraps TransportServer to implement the Server interface
type transportServerWrapper struct {
	config          *Configuration
	ctx             context.Context
	transportServer *TransportServer
	mu              sync.Mutex
}

func (s *transportServerWrapper) RegisterService(ctx context.Context, serviceRegistrar ServiceRegistrar) error {
	if serviceRegistrar == nil {
		return &ValidationError{
			Field:   "serviceRegistrar",
			Value:   nil,
			Message: "service registrar cannot be nil",
		}
	}
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Lazy initialize the transport server
	if s.transportServer == nil {
		var err error
		s.transportServer, err = s.createTransportServer()
		if err != nil {
			return fmt.Errorf("failed to create transport server: %w", err)
		}
	}
	
	// Mount the ServiceRegistrar using the transport server
	// The adapters know how to handle ServiceRegistrar objects properly
	return s.transportServer.mountService(serviceRegistrar)
}

func (s *transportServerWrapper) Serve(ctx context.Context, listener net.Listener) error {
	if listener == nil {
		return &ValidationError{
			Field:   "listener",
			Value:   nil,
			Message: "listener cannot be nil",
		}
	}
	
	s.mu.Lock()
	if s.transportServer == nil {
		var err error
		s.transportServer, err = s.createTransportServer()
		if err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to create transport server: %w", err)
		}
	}
	s.mu.Unlock()
	
	// Use the transport server's serving logic
	return s.transportServer.serveOnListener(ctx, listener)
}

func (s *transportServerWrapper) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.transportServer != nil {
		return s.transportServer.Close()
	}
	return nil
}

func (s *transportServerWrapper) createTransportServer() (*TransportServer, error) {
	server := &TransportServer{
		config: s.config,
	}
	
	// Initialize the appropriate transport based on config
	transportType := s.config.Transport.Type
	if transportType == "" {
		transportType = TransportTypeGRPC // default
	}
	
	switch transportType {
	case TransportTypeGRPC:
		server.grpcAdapter = NewGRPCAdapter()
		server.grpcServer = server.grpcAdapter.GetGRPCServer()
	case TransportTypeHTTP:
		server.httpAdapter = NewHTTPAdapter(s.config.Transport.Address)
		server.httpServer = server.httpAdapter.GetHTTPServer()
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", transportType)
	}
	
	return server, nil
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