// Example usage of the ephemos Gin middleware for SPIFFE identity authentication
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

	"github.com/gin-gonic/gin"

	"github.com/sufield/ephemos/pkg/ephemos"
	ginmiddleware "github.com/sufield/ephemos/contrib/middleware/gin"
)

func main() {
	// Configure structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Set Gin to release mode for production-like behavior
	gin.SetMode(gin.ReleaseMode)

	// Create Gin router
	r := gin.New()

	// Add recovery middleware
	r.Use(gin.Recovery())

	// Add request logging middleware
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("%s - [%s] \"%s %s %s %d %s \"%s\" %s\"\n",
			param.ClientIP,
			param.TimeStamp.Format(time.RFC1123),
			param.Method,
			param.Path,
			param.Request.Proto,
			param.StatusCode,
			param.Latency,
			param.Request.UserAgent(),
			param.ErrorMessage,
		)
	}))

	// Configure SPIFFE identity middleware
	identityConfig := &ginmiddleware.IdentityConfig{
		ConfigPath:        getConfigPath(),
		RequireClientCert: true, // Enable mTLS for proper SPIFFE authentication
		TrustDomains:      []string{"example.org", "prod.company.com"},
		Logger:            logger,
	}

	// Add identity middleware to all routes
	r.Use(ginmiddleware.IdentityMiddleware(identityConfig))

	// Public endpoints (no authentication required)
	r.GET("/", func(c *gin.Context) {
		identity := ginmiddleware.IdentityFromGinContext(c)
		if identity != nil {
			c.JSON(http.StatusOK, gin.H{
				"message":      "Welcome to Ephemos Gin Example",
				"authenticated": true,
				"identity":     identity,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message":      "Welcome to Ephemos Gin Example",
				"authenticated": false,
			})
		}
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Authenticated endpoints - require valid SPIFFE identity
	authenticated := r.Group("/api")
	authenticated.Use(ginmiddleware.RequireIdentity)
	{
		authenticated.GET("/whoami", func(c *gin.Context) {
			identity := ginmiddleware.IdentityFromGinContext(c)
			c.JSON(http.StatusOK, gin.H{
				"spiffe_id":    identity.ID,
				"service_name": identity.Name,
				"trust_domain": identity.Domain,
				"message":      fmt.Sprintf("Hello, %s!", identity.Name),
			})
		})

		authenticated.GET("/secure", func(c *gin.Context) {
			identity := ginmiddleware.IdentityFromGinContext(c)
			c.JSON(http.StatusOK, gin.H{
				"message": "This is a secure endpoint",
				"data":    "Secret data accessible only to authenticated services",
				"caller":  identity.ID,
			})
		})
	}

	// Service-specific endpoints - require specific service identity
	paymentAPI := r.Group("/payment")
	paymentAPI.Use(ginmiddleware.RequireIdentity)
	paymentAPI.Use(ginmiddleware.RequireService("payment-service"))
	{
		paymentAPI.POST("/charge", func(c *gin.Context) {
			identity := ginmiddleware.IdentityFromGinContext(c)
			c.JSON(http.StatusOK, gin.H{
				"message": "Payment processed",
				"service": identity.Name,
				"amount":  "100.00",
			})
		})
	}

	// Trust domain restricted endpoints
	internalAPI := r.Group("/internal")
	internalAPI.Use(ginmiddleware.RequireIdentity)
	internalAPI.Use(ginmiddleware.RequireTrustDomain("prod.company.com"))
	{
		internalAPI.GET("/metrics", func(c *gin.Context) {
			identity := ginmiddleware.IdentityFromGinContext(c)
			c.JSON(http.StatusOK, gin.H{
				"metrics":      "Internal service metrics",
				"trust_domain": identity.Domain,
				"caller":       identity.ID,
			})
		})
	}

	// Create ephemos identity service for SPIFFE TLS
	identityService, err := ephemos.NewFromEnvironment()
	if err != nil {
		logger.Error("Failed to create identity service", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer identityService.Close()

	// Create SPIFFE mTLS server configuration using go-spiffe SDK
	authorizer, err := ephemos.AuthorizeMemberOf("example.org")
	if err != nil {
		logger.Error("Failed to create authorizer", slog.String("error", err.Error()))
		os.Exit(1)
	}
	
	tlsConfig, err := ephemos.NewServerTLSConfig(identityService, authorizer)
	if err != nil {
		logger.Error("Failed to create TLS config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create server with SPIFFE mTLS configuration
	server := &http.Server{
		Addr:         ":8443",
		Handler:      r,
		TLSConfig:    tlsConfig,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Also create a plain HTTP server for testing
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}

	// Start servers in goroutines
	go func() {
		logger.Info("Starting HTTPS server with TLS", slog.String("addr", server.Addr))
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTPS server failed", slog.String("error", err.Error()))
		}
	}()

	go func() {
		logger.Info("Starting HTTP server", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", slog.String("error", err.Error()))
		}
	}()

	logger.Info("Ephemos Gin example server started")
	logger.Info("Try these endpoints:")
	logger.Info("  GET  http://localhost:8080/          - Public endpoint")
	logger.Info("  GET  http://localhost:8080/health    - Health check")
	logger.Info("  GET  http://localhost:8080/api/whoami - Requires SPIFFE auth")
	logger.Info("  POST http://localhost:8080/payment/charge - Requires payment-service identity")

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown both servers
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("HTTPS server forced to shutdown", slog.String("error", err.Error()))
	}

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("HTTP server forced to shutdown", slog.String("error", err.Error()))
	}

	logger.Info("Servers exited")
}

func getConfigPath() string {
	if path := os.Getenv("EPHEMOS_CONFIG"); path != "" {
		return path
	}
	return "/etc/ephemos/config.yaml"
}