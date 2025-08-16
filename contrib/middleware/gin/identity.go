// Package gin provides SPIFFE identity middleware for the Gin framework.
// This middleware enables automatic SPIFFE certificate validation and identity
// context propagation for HTTP services using Gin.
package gin

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spiffe/go-spiffe/v2/spiffeid"

	"github.com/sufield/ephemos/pkg/ephemos"
)

// IdentityContextKey is the context key for storing service identity information
type IdentityContextKey struct{}

// ServiceIdentity represents the authenticated service identity from the client certificate
type ServiceIdentity struct {
	ID     string // SPIFFE ID (e.g., "spiffe://example.org/workload")
	Name   string // Service name extracted from SPIFFE ID
	Domain string // Trust domain from SPIFFE ID
}

// IdentityConfig configures the identity middleware
type IdentityConfig struct {
	// ConfigPath is the path to the ephemos configuration file
	ConfigPath string
	
	// RequireClientCert determines if client certificates are required
	// Default: true (mutual TLS required)
	RequireClientCert bool
	
	// TrustDomains specifies allowed trust domains. Empty means allow all.
	TrustDomains []string
	
	// Logger for middleware events
	Logger *slog.Logger
}

// IdentityMiddleware creates a Gin middleware that validates SPIFFE certificates
// and injects service identity into the request context.
//
// Usage:
//   config := &gin.IdentityConfig{
//       ConfigPath: "/etc/ephemos/config.yaml",
//       RequireClientCert: true,
//   }
//   r.Use(gin.IdentityMiddleware(config))
func IdentityMiddleware(config *IdentityConfig) gin.HandlerFunc {
	if config == nil {
		panic("IdentityConfig cannot be nil")
	}

	// Set defaults
	if config.Logger == nil {
		config.Logger = slog.Default()
	}

	// Initialize ephemos client for certificate validation
	// This validates that the ephemos configuration is accessible
	ctx := context.Background()
	_, err := ephemos.IdentityClient(ctx, config.ConfigPath)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize ephemos client: %v", err))
	}

	return func(c *gin.Context) {
		// Extract client certificate from TLS connection
		if c.Request.TLS == nil {
			config.Logger.Warn("No TLS connection found - SPIFFE identity validation skipped")
			if config.RequireClientCert {
				c.JSON(http.StatusBadRequest, gin.H{"error": "TLS connection required"})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// Validate client certificate if present
		var identity *ServiceIdentity
		if len(c.Request.TLS.PeerCertificates) > 0 {
			clientIdentity, err := validateClientCertificate(c.Request.TLS.PeerCertificates[0], config)
			if err != nil {
				config.Logger.Error("Client certificate validation failed",
					slog.String("error", err.Error()),
					slog.String("remote_addr", c.Request.RemoteAddr))
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid client certificate"})
				c.Abort()
				return
			}
			identity = clientIdentity
		} else if config.RequireClientCert {
			config.Logger.Warn("Client certificate required but not provided",
				slog.String("remote_addr", c.Request.RemoteAddr))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Client certificate required"})
			c.Abort()
			return
		}

		// Add identity to request context
		if identity != nil {
			ctx := context.WithValue(c.Request.Context(), IdentityContextKey{}, identity)
			c.Request = c.Request.WithContext(ctx)
			
			// Also set in Gin context for easy access
			c.Set("spiffe_identity", identity)
			
			config.Logger.Debug("Client identity authenticated",
				slog.String("spiffe_id", identity.ID),
				slog.String("service_name", identity.Name),
				slog.String("trust_domain", identity.Domain))
		}

		// Continue to next handler
		c.Next()
	}
}

// validateClientCertificate validates a client certificate and extracts SPIFFE identity
func validateClientCertificate(cert *x509.Certificate, config *IdentityConfig) (*ServiceIdentity, error) {
	// Extract SPIFFE IDs from certificate URI SANs
	var spiffeID string
	for _, uri := range cert.URIs {
		if uri.Scheme == "spiffe" {
			spiffeID = uri.String()
			break
		}
	}

	if spiffeID == "" {
		return nil, fmt.Errorf("no SPIFFE ID found in client certificate")
	}

	// Parse and validate SPIFFE ID format
	parsedID, err := spiffeid.FromString(spiffeID)
	if err != nil {
		return nil, fmt.Errorf("invalid SPIFFE ID format %q: %w", spiffeID, err)
	}

	// Validate trust domain if specified
	if len(config.TrustDomains) > 0 {
		allowed := false
		for _, domain := range config.TrustDomains {
			if parsedID.TrustDomain().String() == domain {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("trust domain %q not allowed", parsedID.TrustDomain().String())
		}
	}

	// Extract service name from SPIFFE ID path
	serviceName := extractServiceName(parsedID.Path())

	return &ServiceIdentity{
		ID:     spiffeID,
		Name:   serviceName,
		Domain: parsedID.TrustDomain().String(),
	}, nil
}

// extractServiceName extracts service name from SPIFFE ID path
// For example: "/workload" -> "workload", "/ns/production/sa/api" -> "api"
func extractServiceName(path string) string {
	if path == "" {
		return ""
	}

	// Remove leading slash
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}

	// Split by "/" and take the last component as service name
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return path
}

// IdentityFromContext extracts the service identity from the request context.
// Returns nil if no identity is present (e.g., when client certificates are optional).
func IdentityFromContext(ctx context.Context) *ServiceIdentity {
	if identity, ok := ctx.Value(IdentityContextKey{}).(*ServiceIdentity); ok {
		return identity
	}
	return nil
}

// IdentityFromGinContext extracts the service identity from the Gin context.
// This is a convenience method for Gin-specific access patterns.
func IdentityFromGinContext(c *gin.Context) *ServiceIdentity {
	if identity, exists := c.Get("spiffe_identity"); exists {
		if serviceIdentity, ok := identity.(*ServiceIdentity); ok {
			return serviceIdentity
		}
	}
	return nil
}

// RequireIdentity is a Gin middleware that ensures a client identity is present.
// Use this after IdentityMiddleware when you want to require authentication for specific routes.
//
// Usage:
//   authenticated := r.Group("/api")
//   authenticated.Use(gin.IdentityMiddleware(config))
//   authenticated.Use(gin.RequireIdentity)
//   authenticated.GET("/secure", handler)
func RequireIdentity(c *gin.Context) {
	identity := IdentityFromGinContext(c)
	if identity == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		c.Abort()
		return
	}
	c.Next()
}

// RequireService creates a Gin middleware that ensures the client identity matches
// one of the specified service names.
//
// Usage:
//   r.Use(gin.RequireService("payment-service", "order-service"))
func RequireService(allowedServices ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identity := IdentityFromGinContext(c)
		if identity == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// Check if service name is allowed
		allowed := false
		for _, service := range allowedServices {
			if identity.Name == service {
				allowed = true
				break
			}
		}

		if !allowed {
			slog.Warn("Service access denied",
				slog.String("service", identity.Name),
				slog.String("spiffe_id", identity.ID),
				slog.Any("allowed_services", allowedServices))
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireTrustDomain creates a Gin middleware that ensures the client identity
// belongs to one of the specified trust domains.
//
// Usage:
//   r.Use(gin.RequireTrustDomain("prod.company.com", "staging.company.com"))
func RequireTrustDomain(allowedDomains ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		identity := IdentityFromGinContext(c)
		if identity == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// Check if trust domain is allowed
		allowed := false
		for _, domain := range allowedDomains {
			if identity.Domain == domain {
				allowed = true
				break
			}
		}

		if !allowed {
			slog.Warn("Trust domain access denied",
				slog.String("trust_domain", identity.Domain),
				slog.String("spiffe_id", identity.ID),
				slog.Any("allowed_domains", allowedDomains))
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			c.Abort()
			return
		}

		c.Next()
	}
}