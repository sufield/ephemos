// HTTP Server Example - Lightweight HTTP over mTLS with X.509 SVIDs
//
// Simple abstraction over go-spiffe HTTP example:
// https://github.com/spiffe/go-spiffe/tree/main/examples/spiffe-http
//
// Shows how to create an HTTP server with automatic SPIFFE identity authentication
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sufield/ephemos/pkg/ephemos"
)

// Your normal HTTP handlers - no changes needed
func statusHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "my-service",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx := context.Background()

	// Create your HTTP handlers as normal
	mux := http.NewServeMux()
	mux.HandleFunc("/status", statusHandler)

	// Start HTTP server with automatic mTLS using X.509 SVIDs
	server, err := ephemos.NewHTTPServer(ctx, ":8080", mux)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	go func() {
		slog.Info("Starting HTTP server with SPIFFE mTLS", "addr", ":8080")
		if err := server.ListenAndServe(); err != nil {
			slog.Error("Server error", "error", err)
		}
	}()

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	slog.Info("Shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Shutdown error", "error", err)
	}
}