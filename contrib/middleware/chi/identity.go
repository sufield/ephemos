// Package chi provides SPIFFE identity middleware for the Chi router framework.
// This middleware enables automatic SPIFFE certificate validation and identity
// context propagation for HTTP services using Chi.
package chi

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

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

// IdentityMiddleware creates a Chi middleware that validates SPIFFE certificates
// and injects service identity into the request context.
//
// Usage:
//   config := &chi.IdentityConfig{
//       ConfigPath: "/etc/ephemos/config.yaml",
//       RequireClientCert: true,
//   }
//   r.Use(chi.IdentityMiddleware(config))
func IdentityMiddleware(config *IdentityConfig) func(http.Handler) http.Handler {
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

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client certificate from TLS connection
			if r.TLS == nil {
				config.Logger.Warn("No TLS connection found - SPIFFE identity validation skipped")
				if config.RequireClientCert {
					http.Error(w, "TLS connection required", http.StatusBadRequest)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			// Validate client certificate if present
			var identity *ServiceIdentity
			if len(r.TLS.PeerCertificates) > 0 {
				clientIdentity, err := validateClientCertificate(r.TLS.PeerCertificates[0], config)
				if err != nil {
					config.Logger.Error("Client certificate validation failed",
						slog.String("error", err.Error()),
						slog.String("remote_addr", r.RemoteAddr))
					http.Error(w, "Invalid client certificate", http.StatusUnauthorized)
					return
				}
				identity = clientIdentity
			} else if config.RequireClientCert {
				config.Logger.Warn("Client certificate required but not provided",
					slog.String("remote_addr", r.RemoteAddr))
				http.Error(w, "Client certificate required", http.StatusUnauthorized)
				return
			}

			// Add identity to request context
			ctx := r.Context()
			if identity != nil {
				ctx = context.WithValue(ctx, IdentityContextKey{}, identity)
				config.Logger.Debug("Client identity authenticated",
					slog.String("spiffe_id", identity.ID),
					slog.String("service_name", identity.Name),
					slog.String("trust_domain", identity.Domain))
			}

			// Continue with authenticated context
			next.ServeHTTP(w, r.WithContext(ctx))
		})
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

// RequireIdentity is a helper middleware that ensures a client identity is present.
// Use this after IdentityMiddleware when you want to require authentication for specific routes.
//
// Usage:
//   r.Route("/api", func(r chi.Router) {
//       r.Use(chi.IdentityMiddleware(config))
//       r.Use(chi.RequireIdentity)
//       r.Get("/secure", handler)
//   })
func RequireIdentity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity := IdentityFromContext(r.Context())
		if identity == nil {
			http.Error(w, "Authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireService creates a middleware that ensures the client identity matches
// one of the specified service names.
//
// Usage:
//   r.Use(chi.RequireService("payment-service", "order-service"))
func RequireService(allowedServices ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity := IdentityFromContext(r.Context())
			if identity == nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
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
				http.Error(w, "Access denied", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}