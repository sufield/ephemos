// Time Server Example - Backend developer using Ephemos library
//
// A simple time service that returns current time in requested timezone.
// Shows how Ephemos adds identity authentication to normal HTTP services.
//
// BUSINESS LOGIC: Timezone conversion service
// SECURITY: Only authenticated clients (with SPIFFE certs) can access
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

type TimeRequest struct {
	Timezone string `json:"timezone"`
}

type TimeResponse struct {
	Timezone    string `json:"timezone"`
	CurrentTime string `json:"current_time"`
	Timestamp   int64  `json:"timestamp"`
	Service     string `json:"service"`
}

// Your normal business logic - return current time in requested timezone
func timeHandler(w http.ResponseWriter, r *http.Request) {
	// Client has already been authenticated by Ephemos before this runs
	
	if r.Method != http.MethodPost {
		http.Error(w, "Use POST with timezone in JSON body", http.StatusMethodNotAllowed)
		return
	}

	var req TimeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON. Expected: {\"timezone\": \"America/New_York\"}", http.StatusBadRequest)
		return
	}

	// Load the requested timezone
	location, err := time.LoadLocation(req.Timezone)
	if err != nil {
		http.Error(w, "Invalid timezone. Examples: UTC, America/New_York, Europe/London", http.StatusBadRequest)
		return
	}

	// Get current time in that timezone
	now := time.Now().In(location)
	
	response := TimeResponse{
		Timezone:    req.Timezone,
		CurrentTime: now.Format("2006-01-02 15:04:05 MST"),
		Timestamp:   now.Unix(),
		Service:     "time-server",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	slog.Info("Served time request", 
		"timezone", req.Timezone, 
		"time", response.CurrentTime)
}

// Health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":  "healthy",
		"service": "time-server",
		"uptime":  time.Now().Format(time.RFC3339),
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
	mux.HandleFunc("/time", timeHandler)        // Main business logic
	mux.HandleFunc("/health", healthHandler)    // Health checks

	// 2. Add identity authentication with Ephemos
	server, err := ephemos.NewIdentityServer(ctx, "")
	if err != nil {
		slog.Error("Failed to create identity server", "error", err)
		os.Exit(1)
	}

	// 3. Start server - only authenticated clients can access your time service
	go func() {
		slog.Info("Starting time server with identity authentication", "addr", ":8080")
		slog.Info("📍 Endpoints:")
		slog.Info("  POST /time   - Get time in timezone (requires authentication)")
		slog.Info("  GET  /health - Health check (requires authentication)")
		
		if err := server.ServeHTTP(ctx, ":8080", mux); err != nil {
			slog.Error("Server error", "error", err)
		}
	}()

	slog.Info("✅ Time server ready - only clients with valid SPIFFE certificates can connect")

	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	slog.Info("Shutting down time server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("Server shutdown error", "error", err)
	}

	slog.Info("Time server stopped")
}