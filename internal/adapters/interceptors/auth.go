// Package interceptors provides built-in gRPC interceptors for authentication and identity propagation.
package interceptors

import (
	"context"
	"crypto/x509"
	"fmt"
	"log/slog"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// IdentityContextKey is the context key for storing identity information.
type IdentityContextKey struct{}

// AuthConfig contains configuration for authentication interceptors.
type AuthConfig struct {
	// RequireAuthentication determines if authentication is mandatory
	RequireAuthentication bool

	// AllowedServices contains SPIFFE IDs that are allowed to call this service
	AllowedServices []string

	// DenyAllowed services (blacklist mode) - if true, AllowedServices acts as a blacklist
	DenyMode bool

	// RequiredClaims are custom claims that must be present in the identity
	RequiredClaims map[string]string

	// SkipMethods are method names that bypass authentication (format: /service.Service/Method)
	SkipMethods []string

	// Logger for authentication events
	Logger *slog.Logger
}

// AuthenticatedIdentity represents an authenticated client identity.
type AuthenticatedIdentity struct {
	// SPIFFE ID of the authenticated client
	SPIFFEID string

	// X.509 certificate used for authentication
	Certificate *x509.Certificate

	// Trust domain extracted from SPIFFE ID
	TrustDomain string

	// Service name extracted from SPIFFE ID
	ServiceName string

	// Workload path extracted from SPIFFE ID
	WorkloadPath string

	// Additional claims from the certificate
	Claims map[string]string

	// Authentication timestamp
	AuthTime int64
}

// AuthInterceptor provides authentication and authorization for gRPC services.
type AuthInterceptor struct {
	config *AuthConfig
	logger *slog.Logger
}

// NewAuthInterceptor creates a new authentication interceptor with the given configuration.
func NewAuthInterceptor(config *AuthConfig) *AuthInterceptor {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &AuthInterceptor{
		config: config,
		logger: logger,
	}
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for authentication.
func (a *AuthInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Check if method should skip authentication
		if a.shouldSkipMethod(info.FullMethod) {
			a.logger.Debug("Skipping authentication for method", "method", info.FullMethod)
			return handler(ctx, req)
		}

		// Perform authentication
		authenticatedCtx, err := a.authenticateRequest(ctx, info.FullMethod)
		if err != nil {
			a.logger.Warn("Authentication failed",
				"method", info.FullMethod,
				"error", err)
			return nil, err
		}

		// Call the handler with authenticated context
		return handler(authenticatedCtx, req)
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for authentication.
func (a *AuthInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Check if method should skip authentication
		if a.shouldSkipMethod(info.FullMethod) {
			a.logger.Debug("Skipping authentication for stream method", "method", info.FullMethod)
			return handler(srv, ss)
		}

		// Perform authentication
		authenticatedCtx, err := a.authenticateRequest(ss.Context(), info.FullMethod)
		if err != nil {
			a.logger.Warn("Stream authentication failed",
				"method", info.FullMethod,
				"error", err)
			return err
		}

		// Wrap the server stream with authenticated context
		wrappedStream := &authenticatedServerStream{
			ServerStream: ss,
			ctx:          authenticatedCtx,
		}

		return handler(srv, wrappedStream)
	}
}

// authenticatedServerStream wraps a grpc.ServerStream with an authenticated context.
type authenticatedServerStream struct {
	grpc.ServerStream
	ctx context.Context //nolint:containedctx // Required for gRPC ServerStream interface
}

// Context returns the authenticated context.
func (s *authenticatedServerStream) Context() context.Context {
	return s.ctx
}

// authenticateRequest performs the actual authentication logic.
func (a *AuthInterceptor) authenticateRequest(ctx context.Context, method string) (context.Context, error) {
	// Extract peer information
	peer, ok := peer.FromContext(ctx)
	if !ok {
		if a.config.RequireAuthentication {
			return nil, fmt.Errorf("no peer information available: %w", status.Error(codes.Unauthenticated, "no peer information available"))
		}
		return ctx, nil
	}

	// Extract TLS certificate information
	tlsInfo, ok := peer.AuthInfo.(credentials.TLSInfo)
	if !ok {
		if a.config.RequireAuthentication {
			return nil, fmt.Errorf("TLS authentication required: %w", status.Error(codes.Unauthenticated, "TLS authentication required"))
		}
		return ctx, nil
	}

	// Verify we have peer certificates
	if len(tlsInfo.State.PeerCertificates) == 0 {
		if a.config.RequireAuthentication {
			return nil, fmt.Errorf("no peer certificates provided: %w", status.Error(codes.Unauthenticated, "no peer certificates provided"))
		}
		return ctx, nil
	}

	// Extract identity from certificate
	clientCert := tlsInfo.State.PeerCertificates[0]
	identity, err := a.extractIdentityFromCertificate(clientCert)
	if err != nil {
		a.logger.Error("Failed to extract identity from certificate", "error", err)
		return nil, fmt.Errorf("invalid certificate identity: %w", status.Error(codes.Unauthenticated, "invalid certificate identity"))
	}

	// Perform authorization checks
	if err := a.authorizeIdentity(identity, method); err != nil {
		return nil, err
	}

	// Add identity to context
	authenticatedCtx := context.WithValue(ctx, IdentityContextKey{}, identity)

	a.logger.Info("Client authenticated successfully",
		"spiffe_id", identity.SPIFFEID,
		"service", identity.ServiceName,
		"method", method)

	return authenticatedCtx, nil
}

// extractIdentityFromCertificate extracts SPIFFE identity from an X.509 certificate.
func (a *AuthInterceptor) extractIdentityFromCertificate(cert *x509.Certificate) (*AuthenticatedIdentity, error) {
	// Look for SPIFFE ID in Subject Alternative Names (URI SAN)
	var spiffeID string
	for _, uri := range cert.URIs {
		if uri.Scheme == "spiffe" {
			spiffeID = uri.String()
			break
		}
	}

	if spiffeID == "" {
		return nil, fmt.Errorf("no SPIFFE ID found in certificate")
	}

	// Parse SPIFFE ID components
	identity, err := a.parseSpiffeID(spiffeID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SPIFFE ID %s: %w", spiffeID, err)
	}

	// Set certificate and authentication time
	identity.Certificate = cert
	identity.AuthTime = cert.NotBefore.Unix()

	return identity, nil
}

// parseSpiffeID parses a SPIFFE ID into its components.
func (a *AuthInterceptor) parseSpiffeID(spiffeID string) (*AuthenticatedIdentity, error) {
	// SPIFFE ID format: spiffe://trust-domain/path/to/workload
	if !strings.HasPrefix(spiffeID, "spiffe://") {
		return nil, fmt.Errorf("invalid SPIFFE ID format: %s", spiffeID)
	}

	// Remove spiffe:// prefix
	idPath := strings.TrimPrefix(spiffeID, "spiffe://")

	// Split trust domain and path
	parts := strings.SplitN(idPath, "/", 2)
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid SPIFFE ID structure: %s", spiffeID)
	}

	trustDomain := parts[0]
	if trustDomain == "" {
		return nil, fmt.Errorf("empty trust domain in SPIFFE ID: %s", spiffeID)
	}

	var workloadPath, serviceName string

	if len(parts) > 1 {
		workloadPath = "/" + parts[1]

		// Extract service name from path (last component)
		pathParts := strings.Split(parts[1], "/")
		if len(pathParts) > 0 {
			serviceName = pathParts[len(pathParts)-1]
		}
	}

	return &AuthenticatedIdentity{
		SPIFFEID:     spiffeID,
		TrustDomain:  trustDomain,
		ServiceName:  serviceName,
		WorkloadPath: workloadPath,
		Claims:       make(map[string]string),
	}, nil
}

// authorizeIdentity performs authorization checks on the authenticated identity.
func (a *AuthInterceptor) authorizeIdentity(identity *AuthenticatedIdentity, _ string) error {
	// Check allowed/denied services with early returns (guard clauses)
	if len(a.config.AllowedServices) == 0 {
		// No service restrictions - proceed to claims check
		return a.validateRequiredClaims(identity)
	}

	isInList := a.isServiceInList(identity.SPIFFEID, a.config.AllowedServices)

	// Blacklist mode - deny if in list
	if a.config.DenyMode && isInList {
		return status.Errorf(codes.PermissionDenied,
			"service %s is denied access", identity.SPIFFEID)
	}

	// Whitelist mode - allow only if in list
	if !a.config.DenyMode && !isInList {
		return status.Errorf(codes.PermissionDenied,
			"service %s is not authorized", identity.SPIFFEID)
	}

	// Service authorization passed - check claims
	return a.validateRequiredClaims(identity)
}

// Helper function to validate required claims (reduces complexity).
func (a *AuthInterceptor) validateRequiredClaims(identity *AuthenticatedIdentity) error {
	for claimKey, requiredValue := range a.config.RequiredClaims {
		actualValue, exists := identity.Claims[claimKey]
		if !hasRequiredClaimValue(exists, actualValue, requiredValue) {
			return status.Errorf(codes.PermissionDenied,
				"missing or invalid claim: %s", claimKey)
		}
	}
	return nil
}

// hasRequiredClaimValue checks if a claim exists and matches the required value.
func hasRequiredClaimValue(exists bool, actualValue, requiredValue string) bool {
	return exists && actualValue == requiredValue
}

// isServiceInList checks if a SPIFFE ID matches any pattern in the service list.
func (a *AuthInterceptor) isServiceInList(spiffeID string, services []string) bool {
	for _, service := range services {
		if a.matchesPattern(spiffeID, service) {
			return true
		}
	}
	return false
}

// matchesPattern checks if a SPIFFE ID matches a pattern (supports wildcards).
func (a *AuthInterceptor) matchesPattern(spiffeID, pattern string) bool {
	// Exact match
	if spiffeID == pattern {
		return true
	}

	// Wildcard support
	if strings.Contains(pattern, "*") {
		// Simple prefix/suffix wildcard matching
		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			return strings.HasSuffix(spiffeID, suffix)
		}
		if strings.HasSuffix(pattern, "*") {
			prefix := strings.TrimSuffix(pattern, "*")
			return strings.HasPrefix(spiffeID, prefix)
		}
	}

	return false
}

// shouldSkipMethod checks if a method should skip authentication.
func (a *AuthInterceptor) shouldSkipMethod(method string) bool {
	for _, skipMethod := range a.config.SkipMethods {
		if method == skipMethod {
			return true
		}
	}
	return false
}

// GetIdentityFromContext extracts the authenticated identity from a gRPC context.
func GetIdentityFromContext(ctx context.Context) (*AuthenticatedIdentity, bool) {
	identity, ok := ctx.Value(IdentityContextKey{}).(*AuthenticatedIdentity)
	return identity, ok
}

// RequireIdentity extracts the authenticated identity from context and returns an error if not found.
func RequireIdentity(ctx context.Context) (*AuthenticatedIdentity, error) {
	identity, ok := GetIdentityFromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no authenticated identity found: %w", status.Error(codes.Unauthenticated, "no authenticated identity found"))
	}
	return identity, nil
}

// DefaultAuthConfig returns a default authentication configuration.
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		RequireAuthentication: true,
		AllowedServices:       []string{}, // Allow all by default
		DenyMode:              false,
		RequiredClaims:        make(map[string]string),
		SkipMethods:           []string{},
		Logger:                slog.Default(),
	}
}

// NewAllowListAuthConfig creates an auth config that only allows specific services.
func NewAllowListAuthConfig(allowedServices []string) *AuthConfig {
	config := DefaultAuthConfig()
	config.AllowedServices = allowedServices
	config.DenyMode = false
	return config
}

// NewDenyListAuthConfig creates an auth config that denies specific services.
func NewDenyListAuthConfig(deniedServices []string) *AuthConfig {
	config := DefaultAuthConfig()
	config.AllowedServices = deniedServices
	config.DenyMode = true
	return config
}
