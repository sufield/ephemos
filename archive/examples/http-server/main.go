// HTTP Server Example - Backend developer using Ephemos library
//
// This shows how a backend developer creates a server with identity-based authentication.
// The server accepts connections only from clients with valid SPIFFE certificates.
//
// SIMPLE PATTERN:
// 1. Create your normal HTTP handlers
// 2. Wrap with Ephemos for identity authentication  
// 3. Only authenticated clients can reach your handlers
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

// Your normal HTTP handlers - write them as usual
func statusHandler(w http.ResponseWriter, r *http.Request) {
	// By the time this runs, the client has been authenticated by Ephemos
	response := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"service":   "my-service",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func dataHandler(w http.ResponseWriter, r *http.Request) {
	// Client identity has already been verified - just handle the business logic
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := map[string]interface{}{
		"received":     request,
		"processed_at": time.Now().Unix(),
		"from":         "my-service",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx := context.Background()

	// 1. Create your HTTP service as normal
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", statusHandler)
	mux.HandleFunc("/api/data", dataHandler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 2. Create identity-aware server with Ephemos
	server, err := ephemos.NewIdentityServer(ctx, "")
	if err != nil {
		slog.Error("Failed to create identity server", "error", err)
		os.Exit(1)
	}

	// 3. Start server with identity authentication
	// Ephemos handles all SPIFFE/SPIRE complexity - you just get authenticated requests
	go func() {
		slog.Info("Starting identity-aware HTTP server", "addr", ":8080")
		
		// This wraps your HTTP handlers with identity authentication
		if err := server.ServeHTTP(ctx, ":8080", mux); err != nil {
			slog.Error("Server error", "error", err)
		}
	}()

	slog.Info("Server started - only clients with valid SPIFFE certificates can connect")

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	slog.Info("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown error", "error", err)
	}

	slog.Info("Server stopped")
}