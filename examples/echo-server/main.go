// Echo Server Example - Demonstrates Identity-Based Authentication Enforcement
//
// This example shows how Ephemos automatically enforces transport-layer authentication
// using SPIFFE/SPIRE X.509 certificates WITHOUT any changes to your service code.
//
// IDENTITY AUTHENTICATION ENFORCEMENT IN ACTION:
// 1. Server obtains SPIFFE identity: spiffe://example.org/echo-server
// 2. All client connections MUST present valid SPIFFE certificates
// 3. Authentication happens at TLS handshake - BEFORE Echo() method runs
// 4. Unauthorized clients are rejected automatically by transport layer
// 5. Your Echo service code runs only for authenticated clients
//
// Authentication Flow:
//
//	Client connects → mTLS handshake → certificate verification → Echo() method runs
//	If ANY step fails, connection is rejected and Echo() is never called
//
// To see authentication enforcement:
//
//	✅ Run with valid client: successful Echo responses
//	❌ Run with invalid client: "transport: authentication handshake failed"
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

	"google.golang.org/grpc"

	"github.com/sufield/ephemos/examples/proto"
	"github.com/sufield/ephemos/internal/adapters/logging"
	"github.com/sufield/ephemos/pkg/ephemos"
)

// EchoServer implements the EchoServiceServer interface.
// This is an example implementation that developers can use as reference
// when building their own services with the Ephemos library.
type EchoServer struct {
	proto.UnimplementedEchoServiceServer
}

// Echo implements the Echo method of the EchoServiceServer interface.
//
// AUTHENTICATION ENFORCEMENT POINT:
// If this method is called, it means the client has ALREADY been authenticated!
// Ephemos performed mTLS authentication at the transport layer before this code runs.
//
// Authentication verifications that already occurred:
// ✅ Client presented valid X.509 certificate with SPIFFE ID
// ✅ Certificate was verified against SPIRE trust bundle
// ✅ Certificate is not expired (1-hour validity)
// ✅ Client SPIFFE ID is authorized in server configuration
//
// This is your clean business logic - no authentication code needed!
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

	// Setup secure structured logging with debug level for troubleshooting
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	secureLogger := logging.NewSecureLogger(baseHandler)
	slog.SetDefault(secureLogger)

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
