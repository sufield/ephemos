package api

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/sufield/ephemos/internal/adapters/secondary/config"
	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/adapters/secondary/transport"
	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
	"github.com/sufield/ephemos/internal/core/services"
	"google.golang.org/grpc"
)

type IdentityServer struct {
	identityService *services.IdentityService
	configProvider  ports.ConfigurationProvider
	serviceName     string
	grpcServer      *grpc.Server
	mu              sync.Mutex
}

func NewIdentityServer(configPath string) (*IdentityServer, error) {
	configProvider := config.NewConfigProvider()

	var cfg *ports.Configuration
	var err error
	if configPath != "" {
		cfg, err = configProvider.LoadConfiguration(context.Background(), configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load configuration: %w", err)
		}
	} else {
		cfg = configProvider.GetDefaultConfiguration(context.Background())
		if cfg == nil {
			return nil, &errors.ValidationError{
				Field:   "configuration",
				Value:   nil,
				Message: "no configuration provided and no default configuration available",
			}
		}
	}

	spiffeProvider, err := spiffe.NewSPIFFEProvider(cfg.SPIFFE)
	if err != nil {
		return nil, fmt.Errorf("failed to create SPIFFE provider: %w", err)
	}

	transportProvider := transport.NewGRPCTransportProvider(spiffeProvider)

	identityService, err := services.NewIdentityService(
		spiffeProvider,
		transportProvider,
		cfg,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service: %w", err)
	}

	return &IdentityServer{
		identityService: identityService,
		configProvider:  configProvider,
		serviceName:     cfg.Service.Name,
	}, nil
}

func (s *IdentityServer) RegisterService(ctx context.Context, serviceRegistrar ServiceRegistrar) error {
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

	if s.grpcServer == nil {
		if err := s.initializeServer(ctx); err != nil {
			return fmt.Errorf("failed to initialize server: %w", err)
		}
	}

	serviceRegistrar.Register(s.grpcServer)
	slog.Info("Service registered successfully", "service", s.serviceName)
	return nil
}

func (s *IdentityServer) Serve(ctx context.Context, listener net.Listener) error {
	// Input validation
	if listener == nil {
		return &errors.ValidationError{
			Field:   "listener",
			Value:   listener,
			Message: "listener cannot be nil",
		}
	}

	s.mu.Lock()
	if s.grpcServer == nil {
		if err := s.initializeServer(ctx); err != nil {
			s.mu.Unlock()
			return fmt.Errorf("failed to initialize server: %w", err)
		}
	}
	s.mu.Unlock()

	slog.Info("Server ready", "service", s.serviceName, "address", listener.Addr().String())
	return s.grpcServer.Serve(listener)
}

func (s *IdentityServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.grpcServer != nil {
		s.grpcServer.GracefulStop()
		slog.Info("Server stopped gracefully", "service", s.serviceName)
	}

	return nil
}

func (s *IdentityServer) initializeServer(ctx context.Context) error {
	server, err := s.identityService.CreateServerIdentity(ctx)
	if err != nil {
		return fmt.Errorf("failed to create server identity: %w", err)
	}
	s.grpcServer = server
	slog.Info("Server identity created", "service", s.serviceName)
	return nil
}
