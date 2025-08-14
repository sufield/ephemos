// Package ephemos provides the workload server implementation.
package ephemos

import (
	"context"
	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// workloadServerImpl is the concrete implementation of WorkloadServer.
type workloadServerImpl struct {
	grpcServer      *grpc.Server
	spiffeProvider  SPIFFEProvider
	config          *Configuration
	mu              sync.Mutex
	isRunning       bool
	serviceRegistry map[string]ServiceRegistrar
}

// NewWorkloadServer creates a new workload server.
func NewWorkloadServer(ctx context.Context, config *Configuration, provider SPIFFEProvider) (WorkloadServer, error) {
	if config == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}

	server := &workloadServerImpl{
		config:          config,
		spiffeProvider:  provider,
		serviceRegistry: make(map[string]ServiceRegistrar),
	}

	// Create gRPC server with TLS if SPIFFE provider is available
	var opts []grpc.ServerOption
	if provider != nil {
		tlsConfig, err := provider.GetTLSConfig(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get TLS config: %w", err)
		}
		if tlsConfig != nil {
			creds := credentials.NewTLS(tlsConfig)
			opts = append(opts, grpc.Creds(creds))
		}
	}

	server.grpcServer = grpc.NewServer(opts...)
	return server, nil
}

// Start starts the server on the given listener.
func (s *workloadServerImpl) Start(ctx context.Context, listener net.Listener) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	s.isRunning = true
	s.mu.Unlock()

	// Start serving in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.grpcServer.Serve(listener)
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		s.Stop()
		return ctx.Err()
	case err := <-errCh:
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
		return err
	}
}

// Stop gracefully stops the server.
func (s *workloadServerImpl) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	s.grpcServer.GracefulStop()
	s.isRunning = false
	return nil
}

// RegisterService registers a service with the server.
func (s *workloadServerImpl) RegisterService(registrar ServiceRegistrar) error {
	if registrar == nil {
		return fmt.Errorf("service registrar cannot be nil")
	}

	// Register with the gRPC server
	registrar.Register(s.grpcServer)

	// Store in registry for tracking
	s.mu.Lock()
	s.serviceRegistry[fmt.Sprintf("%T", registrar)] = registrar
	s.mu.Unlock()

	return nil
}


// Close implements the ShutdownableServer interface for graceful shutdown.
func (s *workloadServerImpl) Close() error {
	return s.Stop()
}
