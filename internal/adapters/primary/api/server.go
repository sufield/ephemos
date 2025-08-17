package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"google.golang.org/grpc"

	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
)

// Server provides a secure gRPC server with SPIFFE-based workload identity management.
type Server struct {
	identityService *services.IdentityService
	configProvider  ports.ConfigurationProvider
	serviceName     string
	domainServer    ports.ServerPort
	mu              sync.Mutex
}

// Note: Factory functions that create secondary adapters have been moved to avoid cross-adapter imports.
// Use WorkloadServer with dependency injection instead.

// WorkloadServer creates a new workload server with injected dependencies.
// This constructor follows proper dependency injection and hexagonal architecture principles.
func WorkloadServer(
	identityProvider ports.IdentityProvider,
	transportProvider ports.TransportProvider,
	configProvider ports.ConfigurationProvider,
	cfg *ports.Configuration,
) (*Server, error) {
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

	if configProvider == nil {
		return nil, &errors.ValidationError{
			Field:   "configProvider",
			Value:   nil,
			Message: "configuration provider cannot be nil",
		}
	}

	identityService, err := services.NewIdentityService(
		identityProvider,
		transportProvider,
		cfg,
		nil, // default validator
		nil, // no-op metrics
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service: %w", err)
	}

	return &Server{
		identityService: identityService,
		configProvider:  configProvider,
		serviceName:     cfg.Service.Name,
	}, nil
}

// RegisterService registers a gRPC service with the identity server.
func (s *Server) RegisterService(ctx context.Context, serviceRegistrar ServiceRegistrar) error {
	if ctx == nil {
		return &errors.ValidationError{
			Field:   "context",
			Message: "context cannot be nil",
		}
	}
	
	if serviceRegistrar == nil {
		return &errors.ValidationError{
			Field:   "serviceRegistrar",
			Message: "service registrar cannot be nil",
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.domainServer == nil {
		if err := s.initializeServer(ctx); err != nil {
			return fmt.Errorf("failed to initialize server: %w", err)
		}
	}

	// Adapt our ServiceRegistrar to ports.ServiceRegistrarPort
	portServiceRegistrar := &serviceRegistrarAdapter{registrar: serviceRegistrar}
	if err := s.domainServer.RegisterService(portServiceRegistrar); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	slog.Info("Service registered successfully", "service", s.serviceName)
	return nil
}

// Serve starts the identity server on the provided listener.
func (s *Server) Serve(ctx context.Context, listener net.Listener) error {
	if listener == nil {
		return &errors.ValidationError{
			Field:   "listener",
			Message: "listener cannot be nil",
		}
	}
	s.mu.Lock()
	if s.domainServer == nil {
		if err := s.initializeServer(ctx); err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to initialize server: %w", err)
		}
	}
	s.mu.Unlock()

	slog.Info("Server ready", "service", s.serviceName, "address", listener.Addr().String())

	portListener := &netListenerAdapter{listener: listener}
	errCh := make(chan error, 1)

	go func() { errCh <- s.domainServer.Start(portListener) }()

	select {
	case <-ctx.Done():
		_ = s.domainServer.Stop() // best-effort
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

// Close gracefully shuts down the identity server.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.domainServer != nil {
		if err := s.domainServer.Stop(); err != nil {
			return fmt.Errorf("failed to stop server: %w", err)
		}
		s.domainServer = nil
		slog.Info("Server stopped gracefully", "service", s.serviceName)
	}
	return nil
}

func (s *Server) initializeServer(_ context.Context) error {
	server, err := s.identityService.CreateServerIdentity()
	if err != nil {
		return fmt.Errorf("failed to create server identity: %w", err)
	}
	s.domainServer = server
	slog.Info("Server identity created", "service", s.serviceName)
	return nil
}

// serviceRegistrarAdapter adapts our API ServiceRegistrar to ports.ServiceRegistrarPort.
type serviceRegistrarAdapter struct {
	registrar ServiceRegistrar
}

func (a *serviceRegistrarAdapter) Register(server interface{}) {
	if grpcServer, ok := server.(grpc.ServiceRegistrar); ok {
		a.registrar.Register(grpcServer)
	}
}

// netListenerAdapter adapts net.Listener to ports.ListenerPort.
type netListenerAdapter struct {
	listener net.Listener
}

// Accept waits for and returns the next connection.
func (l *netListenerAdapter) Accept() (interface{}, error) {
	return l.listener.Accept()
}

// Close closes the listener.
func (l *netListenerAdapter) Close() error {
	return l.listener.Close()
}

// Addr returns the listener's network address.
func (l *netListenerAdapter) Addr() string {
	return l.listener.Addr().String()
}
