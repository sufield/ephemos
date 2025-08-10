package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sufield/ephemos/examples/proto"
	"github.com/sufield/ephemos/internal/adapters/logging"
	"github.com/sufield/ephemos/pkg/ephemos"
)

func main() {
	// Setup secure structured logging with debug level for troubleshooting
	baseHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	secureLogger := logging.NewSecureLogger(baseHandler)
	slog.SetDefault(secureLogger)

	ctx := context.Background()

	// Create identity-aware client
	client, err := ephemos.NewIdentityClient(ctx, "")
	if err != nil {
		slog.Error("Failed to create identity client", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := client.Close(); err != nil {
			slog.Warn("Failed to close client", "error", err)
		}
	}()

	// Connect to the echo server
	conn, err := client.Connect(ctx, "echo-server", "localhost:50052")
	if err != nil {
		slog.Error("Failed to connect to echo server", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			slog.Warn("Failed to close connection", "error", err)
		}
	}()

	// Create echo service client using the generic wrapper
	echoClient, err := proto.NewEchoClient(conn.GetClientConnection())
	if err != nil {
		slog.Error("Failed to create echo client", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := echoClient.Close(); err != nil {
			slog.Warn("Failed to close echo client", "error", err)
		}
	}()

	slog.Info("Connected to echo server", "address", "localhost:50052")

	// Make echo requests
	for i := 0; i < 5; i++ {
		// Create request context with timeout
		reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

		message := fmt.Sprintf("Hello from echo-client! Request #%d", i+1)
		resp, err := echoClient.Echo(reqCtx, message)
		if err != nil {
			slog.Error("Echo request failed", "request", i+1, "error", err)
		} else {
			slog.Info("Echo response received",
				"request", i+1,
				"message", resp.Message,
				"from", resp.From)
		}

		cancel() // Clean up the timeout context
		time.Sleep(2 * time.Second)
	}

	slog.Info("Echo client completed successfully")
}
