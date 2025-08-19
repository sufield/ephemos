// Example usage of the ephemos Chi middleware for SPIFFE identity authentication
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/sufield/ephemos/pkg/ephemos"
	chimiddleware "github.com/sufield/ephemos/contrib/middleware/chi"
)

func main() {
	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Create Chi router
	r := chi.NewRouter()

	// Add built-in middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	// Configure ephemos identity middleware
	identityConfig := &chimiddleware.IdentityConfig{
		ConfigPath:        "/etc/ephemos/config.yaml", // Path to ephemos config
		RequireClientCert: true,                       // Require client certs for proper SPIFFE mTLS
		TrustDomains:      []string{"example.org"},    // Only allow example.org trust domain
		Logger:            logger,
	}

	// Public routes (no authentication required)
	r.Get("/health", healthHandler)
	r.Get("/public", publicHandler)

	// Protected routes (require SPIFFE identity)
	r.Route("/api", func(r chi.Router) {
		// Apply identity middleware to all /api routes
		r.Use(chimiddleware.IdentityMiddleware(identityConfig))
		
		// This route accepts any authenticated service
		r.Get("/info", authenticatedHandler)

		// Admin routes (require specific services)
		r.Route("/admin", func(r chi.Router) {
			r.Use(chimiddleware.RequireService("admin-service", "operator-service"))
			r.Get("/users", adminHandler)
			r.Post("/config", configHandler)
		})

		// Service-specific routes example
		r.Route("/data", func(r chi.Router) {
			r.Use(chimiddleware.RequireService("data-service"))
			r.Get("/info", dataHandler)
		})
	})

	// Strict authentication routes (always require client certificates)
	r.Route("/secure", func(r chi.Router) {
		strictConfig := *identityConfig
		strictConfig.RequireClientCert = true
		
		r.Use(chimiddleware.IdentityMiddleware(&strictConfig))
		r.Use(chimiddleware.RequireIdentity)
		r.Get("/sensitive", sensitiveHandler)
	})

	// Create SPIFFE mTLS TLS configuration using go-spiffe SDK
	tlsConfig, err := createTLSConfig()
	if err != nil {
		logger.Error("Failed to create TLS config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Start server
	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
		// Configure TLS for SPIFFE certificate support
		TLSConfig: tlsConfig,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Server shutdown error", slog.String("error", err.Error()))
		}
	}()

	logger.Info("Starting server", slog.String("addr", server.Addr))

	// Start HTTPS server with SPIFFE mTLS (certificates managed by go-spiffe SDK)
	if err := server.ListenAndServeTLS("", ""); err != http.ErrServerClosed {
		logger.Error("Server error", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info("Server stopped")
}

// Health check handler (no authentication required)
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "healthy", "timestamp": "%s"}`, time.Now().Format(time.RFC3339))
}

// Public handler (no authentication required)
func publicHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "This is a public endpoint accessible without authentication")
}

// Authenticated handler (requires valid SPIFFE identity)
func authenticatedHandler(w http.ResponseWriter, r *http.Request) {
	identity := chimiddleware.IdentityFromContext(r.Context())
	if identity == nil {
		// This shouldn't happen if middleware is working correctly
		http.Error(w, "No identity found", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"message": "Authenticated successfully",
		"service": "%s",
		"spiffe_id": "%s",
		"trust_domain": "%s"
	}`, identity.Name, identity.ID, identity.Domain)
}

// Admin handler (requires specific service identity)
func adminHandler(w http.ResponseWriter, r *http.Request) {
	identity := chimiddleware.IdentityFromContext(r.Context())
	
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"message": "Admin access granted",
		"service": "%s",
		"action": "list_users"
	}`, identity.Name)
}

// Config handler (admin endpoint)
func configHandler(w http.ResponseWriter, r *http.Request) {
	identity := chimiddleware.IdentityFromContext(r.Context())
	
	slog.Info("Config update requested",
		slog.String("service", identity.Name),
		slog.String("spiffe_id", identity.ID))

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"message": "Config updated by %s"}`, identity.Name)
}

// Data handler (data service only)
func dataHandler(w http.ResponseWriter, r *http.Request) {
	identity := chimiddleware.IdentityFromContext(r.Context())
	
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"message": "Data access granted",
		"service": "%s",
		"timestamp": "%s"
	}`, identity.Name, time.Now().Format(time.RFC3339))
}

// Sensitive data handler (strict authentication required)
func sensitiveHandler(w http.ResponseWriter, r *http.Request) {
	identity := chimiddleware.IdentityFromContext(r.Context())
	
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{
		"message": "Access to sensitive data granted",
		"service": "%s",
		"clearance_level": "top_secret"
	}`, identity.Name)
}

// createTLSConfig creates a SPIFFE mTLS configuration for the server using go-spiffe SDK
func createTLSConfig() (*tls.Config, error) {
	// Create ephemos identity service for SPIFFE TLS
	identityService, err := ephemos.NewFromEnvironment()
	if err != nil {
		return nil, fmt.Errorf("failed to create identity service: %w", err)
	}

	// Create SPIFFE mTLS server configuration using go-spiffe SDK
	// Using AuthorizeAny() for authentication only - authorization is out of scope
	tlsConfig, err := ephemos.NewServerTLSConfig(identityService, ephemos.AuthorizeAny())
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS config: %w", err)
	}

	return tlsConfig, nil
}