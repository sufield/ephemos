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

// WorkloadServer provides a secure gRPC server with SPIFFE-based workload identity management.
type WorkloadServer struct {
	identityService *services.IdentityService
	configProvider  ports.ConfigurationProvider
	serviceName     string
	domainServer    ports.Server
	mu              sync.Mutex
}

// NewWorkloadServer creates a new workload server with the given configuration.
// Deprecated: Use NewWorkloadServerWithDependencies for proper dependency injection.
func NewWorkloadServer(ctx context.Context, configPath string) (*WorkloadServer, error) {
	// This is a legacy method maintained for backward compatibility.
	// For new code, use NewWorkloadServerWithDependencies instead.
	return nil, &errors.ValidationError{
		Field:   "constructor",
		Value:   "NewWorkloadServer",
		Message: "deprecated constructor - use NewWorkloadServerWithDependencies instead",
	}
}

// NewWorkloadServerWithDependencies creates a new workload server with injected dependencies.
// This constructor follows proper dependency injection and hexagonal architecture principles.
func NewWorkloadServerWithDependencies(
	identityProvider ports.IdentityProvider,
	transportProvider ports.TransportProvider,
	configProvider ports.ConfigurationProvider,
	cfg *ports.Configuration,
) (*WorkloadServer, error) {
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
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service: %w", err)
	}

	return &WorkloadServer{
		identityService: identityService,
		configProvider:  configProvider,
		serviceName:     cfg.Service.Name,
	}, nil
}

// RegisterService registers a gRPC service with the identity server.
func (s *WorkloadServer) RegisterService(ctx context.Context, serviceRegistrar ServiceRegistrar) error {
	// Input validation
	if serviceRegistrar == nil {
		return &errors.ValidationError{
			Field:   "serviceRegistrar",
			Value:   serviceRegistrar,
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

	// Adapt our ServiceRegistrar to ports.ServiceRegistrar
	portServiceRegistrar := &serviceRegistrarAdapter{registrar: serviceRegistrar}
	if err := s.domainServer.RegisterService(portServiceRegistrar); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	slog.Info("Service registered successfully", "service", s.serviceName)
	return nil
}

// Serve starts the identity server on the provided listener.
func (s *WorkloadServer) Serve(ctx context.Context, listener net.Listener) error {
	// Input validation
	if listener == nil {
		return &errors.ValidationError{
			Field:   "listener",
			Value:   listener,
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

	// Adapt net.Listener to ports.Listener
	portListener := &netListenerAdapter{listener: listener}
	if err := s.domainServer.Start(portListener); err != nil {
		return fmt.Errorf("failed to serve domain server: %w", err)
	}
	return nil
}

// Close gracefully shuts down the identity server.
func (s *WorkloadServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.domainServer != nil {
		if err := s.domainServer.Stop(); err != nil {
			return fmt.Errorf("failed to stop server: %w", err)
		}
		slog.Info("Server stopped gracefully", "service", s.serviceName)
	}

	return nil
}

func (s *WorkloadServer) initializeServer(_ context.Context) error {
	server, err := s.identityService.CreateServerIdentity()
	if err != nil {
		return fmt.Errorf("failed to create server identity: %w", err)
	}
	s.domainServer = server
	slog.Info("Server identity created", "service", s.serviceName)
	return nil
}

// serviceRegistrarAdapter adapts our API ServiceRegistrar to ports.ServiceRegistrar.
type serviceRegistrarAdapter struct {
	registrar ServiceRegistrar
}

func (a *serviceRegistrarAdapter) Register(server interface{}) {
	if grpcServer, ok := server.(*grpc.Server); ok {
		a.registrar.Register(grpcServer)
	}
}

// netListenerAdapter adapts net.Listener to ports.Listener.
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
