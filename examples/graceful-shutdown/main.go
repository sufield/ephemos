// Package main demonstrates graceful shutdown with SVID watcher cleanup.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	
	"github.com/sufield/ephemos/pkg/ephemos"
	pb "github.com/sufield/ephemos/proto"
)

// EchoServer implements the Echo service
type EchoServer struct {
	pb.UnimplementedEchoServiceServer
	requestCount int
}

func (s *EchoServer) Echo(ctx context.Context, req *pb.EchoRequest) (*pb.EchoResponse, error) {
	s.requestCount++
	slog.Info("Processing echo request", 
		"message", req.Message,
		"request_count", s.requestCount)
	
	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)
	
	return &pb.EchoResponse{
		Message: fmt.Sprintf("Echo: %s", req.Message),
	}, nil
}

func main() {
	// Set up structured logging
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(handler))
	
	// Parse config path from environment or use default
	configPath := os.Getenv("EPHEMOS_CONFIG")
	if configPath == "" {
		configPath = "config/echo-server.yaml"
	}
	
	ctx := context.Background()
	
	// Create shutdown configuration
	shutdownConfig := &ephemos.ShutdownConfig{
		GracePeriod:  30 * time.Second,
		DrainTimeout: 20 * time.Second,
		ForceTimeout: 45 * time.Second,
		OnShutdownStart: func() {
			slog.Info("🛑 Graceful shutdown initiated",
				"grace_period", "30s",
				"drain_timeout", "20s")
		},
		OnShutdownComplete: func(err error) {
			if err != nil {
				slog.Error("⚠️  Shutdown completed with errors", "error", err)
			} else {
				slog.Info("✅ Graceful shutdown completed successfully")
			}
		},
	}
	
	// Create server options
	serverOpts := &ephemos.ServerOptions{
		ConfigPath:           configPath,
		ShutdownConfig:       shutdownConfig,
		EnableSignalHandling: true, // Automatically handle SIGINT/SIGTERM
		PreShutdownHook: func(ctx context.Context) error {
			slog.Info("📋 Pre-shutdown: Saving state and metrics")
			// Here you could save application state, flush metrics, etc.
			return nil
		},
		PostShutdownHook: func(err error) {
			slog.Info("📊 Post-shutdown: Final cleanup")
			// Here you could perform final cleanup, close databases, etc.
		},
	}
	
	// Create enhanced server with graceful shutdown
	slog.Info("🚀 Creating enhanced identity server with graceful shutdown")
	server, err := ephemos.NewEnhancedIdentityServer(ctx, serverOpts)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	
	// Register cleanup for any additional resources
	server.RegisterCleanupFunc(func() error {
		slog.Info("🧹 Running custom cleanup: closing database connections")
		// Close database connections, flush caches, etc.
		time.Sleep(500 * time.Millisecond) // Simulate cleanup work
		return nil
	})
	
	// Register another cleanup function
	server.RegisterCleanupFunc(func() error {
		slog.Info("💾 Running custom cleanup: saving application state")
		// Save application state, write checkpoints, etc.
		time.Sleep(300 * time.Millisecond) // Simulate state saving
		return nil
	})
	
	// Create service registrar
	echoService := &EchoServer{}
	registrar := ephemos.NewServiceRegistrar(func(s *grpc.Server) {
		pb.RegisterEchoServiceServer(s, echoService)
	})
	
	// Register the service
	if err := server.RegisterService(ctx, registrar); err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}
	
	// Create listener
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}
	
	slog.Info("📡 Server starting",
		"address", listener.Addr().String(),
		"service", "echo-server")
	
	// Demonstrate deadline-based serving (optional)
	if deadline := os.Getenv("SERVER_DEADLINE"); deadline != "" {
		duration, err := time.ParseDuration(deadline)
		if err == nil {
			deadlineTime := time.Now().Add(duration)
			slog.Info("⏰ Server will run until deadline",
				"deadline", deadlineTime.Format(time.RFC3339))
			
			if err := server.ServeWithDeadline(ctx, listener, deadlineTime); err != nil {
				slog.Error("Server error", "error", err)
			}
			return
		}
	}
	
	// Normal serving with signal-based shutdown
	slog.Info("👂 Server listening (press Ctrl+C for graceful shutdown)")
	if err := server.Serve(ctx, listener); err != nil {
		slog.Error("Server error", "error", err)
		os.Exit(1)
	}
}

// Example usage:
// 1. Run the server:
//    EPHEMOS_CONFIG=config/echo-server.yaml go run examples/graceful-shutdown/main.go
//
// 2. Send requests to the server:
//    grpcurl -plaintext -d '{"message": "Hello"}' localhost:50051 ephemos.EchoService/Echo
//
// 3. Trigger graceful shutdown:
//    - Press Ctrl+C (SIGINT)
//    - Or send SIGTERM: kill -TERM <pid>
//
// 4. Observe the graceful shutdown process:
//    - Server stops accepting new connections
//    - Existing requests complete
//    - SVID watchers are closed
//    - Connection pools are drained
//    - Custom cleanup functions run
//    - All resources are released cleanly
//
// 5. Run with deadline (optional):
//    SERVER_DEADLINE=5m go run examples/graceful-shutdown/main.go
//    # Server will run for 5 minutes then shutdown gracefully