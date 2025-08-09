package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sufield/ephemos/examples/proto"
	"github.com/sufield/ephemos/pkg/ephemos"
	"google.golang.org/grpc"
)

// EchoServer implements the EchoServiceServer interface.
// This is an example implementation that developers can use as reference
// when building their own services with the Ephemos library.
type EchoServer struct {
	proto.UnimplementedEchoServiceServer
}

// Echo implements the Echo method of the EchoServiceServer interface.
// This is the actual business logic of the service.
func (s *EchoServer) Echo(ctx context.Context, req *proto.EchoRequest) (*proto.EchoResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	slog.Info("Processing echo request", "message", req.Message)

	return &proto.EchoResponse{
		Message: req.Message,
		From:    "echo-server",
	}, nil
}

func main() {
	// Create cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup structured logging with debug level for troubleshooting
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Get server configuration
	serverConfig, err := getServerConfig()
	if err != nil {
		slog.Error("Failed to get server config", "error", err)
		os.Exit(1)
	}

	// Create identity-aware server with context
	server, err := createIdentityServer(ctx, serverConfig)
	if err != nil {
		slog.Error("Failed to create identity server", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := server.Close(); err != nil {
			slog.Warn("Failed to close server", "error", err)
		}
	}()

	// Register service using the generic registrar (no boilerplate required)
	serviceRegistrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
		proto.RegisterEchoServiceServer(s, &EchoServer{})
	})

	if err := server.RegisterService(ctx, serviceRegistrar); err != nil {
		slog.Error("Failed to register service", "error", err)
		os.Exit(1)
	}

	// Setup listener with cleanup
	lis, err := net.Listen("tcp", serverConfig.Address)
	if err != nil {
		slog.Error("Failed to listen", "address", serverConfig.Address, "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := lis.Close(); err != nil {
			slog.Warn("Failed to close listener", "error", err)
		}
	}()

	// Setup graceful shutdown
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-shutdownChan
		slog.Info("Shutdown signal received, stopping server gracefully")
		cancel()
	}()

	slog.Info("Echo server starting", "address", serverConfig.Address, "service", "echo-server")

	// Serve with context for cancellation
	serveCtx, serveCancel := context.WithTimeout(ctx, 30*time.Second)
	defer serveCancel()

	if err := server.Serve(serveCtx, lis); err != nil {
		if ctx.Err() != nil {
			slog.Info("Server stopped gracefully")
		} else {
			slog.Error("Server failed", "error", err)
			os.Exit(1)
		}
	}
}

type ServerConfig struct {
	Address    string
	ConfigPath string
}

func getServerConfig() (*ServerConfig, error) {
	address := os.Getenv("ECHO_SERVER_ADDRESS")
	if address == "" {
		address = ":50051" // Default fallback
	}

	configPath := os.Getenv("EPHEMOS_CONFIG")

	return &ServerConfig{
		Address:    address,
		ConfigPath: configPath,
	}, nil
}

func createIdentityServer(ctx context.Context, config *ServerConfig) (ephemos.Server, error) {
	if config == nil {
		return nil, fmt.Errorf("server config cannot be nil")
	}

	server, err := ephemos.NewIdentityServer(ctx, config.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create identity server: %w", err)
	}

	return server, nil
}
