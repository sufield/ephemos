package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/sufield/ephemos/internal/core/ports"
)

// Default configuration constants.
const defaultMaxCallChainDepth = 10

// RequestIDContextKey is the context key for storing request ID information.
type RequestIDContextKey struct{}

const (
	// MetadataKeyOriginalCaller is the metadata key for original caller identity.
	MetadataKeyOriginalCaller = "x-ephemos-original-caller"
	// MetadataKeyCallChain is the metadata key for the call chain.
	MetadataKeyCallChain = "x-ephemos-call-chain"
	// MetadataKeyTrustDomain is the metadata key for trust domain.
	MetadataKeyTrustDomain = "x-ephemos-trust-domain"
	// MetadataKeyServiceName is the metadata key for service name.
	MetadataKeyServiceName = "x-ephemos-service-name"
	// MetadataKeyRequestID is the metadata key for request ID.
	MetadataKeyRequestID = "x-ephemos-request-id"
	// MetadataKeyTimestamp is the metadata key for timestamp.
	MetadataKeyTimestamp = "x-ephemos-timestamp"
)

// IdentityPropagationConfig configures identity propagation behavior.
type IdentityPropagationConfig struct {
	// IdentityProvider to get current service identity
	IdentityProvider ports.IdentityProvider

	// PropagateOriginalCaller forwards the original caller identity
	PropagateOriginalCaller bool

	// PropagateCallChain builds and forwards the call chain
	PropagateCallChain bool

	// MaxCallChainDepth limits the depth of call chain to prevent loops
	MaxCallChainDepth int

	// CustomHeaders are additional headers to propagate
	CustomHeaders []string

	// Logger for propagation events
	Logger *slog.Logger
}

// IdentityPropagationInterceptor handles identity propagation for outgoing gRPC calls.
type IdentityPropagationInterceptor struct {
	config *IdentityPropagationConfig
	logger *slog.Logger
}

// NewIdentityPropagationInterceptor creates a new identity propagation interceptor.
func NewIdentityPropagationInterceptor(config *IdentityPropagationConfig) *IdentityPropagationInterceptor {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}

	// Set default max call chain depth
	if config.MaxCallChainDepth == 0 {
		config.MaxCallChainDepth = 10
	}

	return &IdentityPropagationInterceptor{
		config: config,
		logger: logger,
	}
}

// UnaryClientInterceptor returns a gRPC unary client interceptor for identity propagation.
func (i *IdentityPropagationInterceptor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// Create context with propagated identity metadata
		propagatedCtx, err := i.propagateIdentity(ctx, method)
		if err != nil {
			i.logger.Error("Failed to propagate identity",
				"method", method,
				"error", err)
			return err
		}

		// Make the call with propagated context
		return invoker(propagatedCtx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor returns a gRPC stream client interceptor for identity propagation.
func (i *IdentityPropagationInterceptor) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		// Create context with propagated identity metadata
		propagatedCtx, err := i.propagateIdentity(ctx, method)
		if err != nil {
			i.logger.Error("Failed to propagate identity for stream",
				"method", method,
				"error", err)
			return nil, err
		}

		// Create the stream with propagated context
		return streamer(propagatedCtx, desc, cc, method, opts...)
	}
}

// propagateIdentity adds identity metadata to the outgoing context.
func (i *IdentityPropagationInterceptor) propagateIdentity(ctx context.Context, method string) (context.Context, error) {
	// Get current service identity
	identity, err := i.config.IdentityProvider.GetServiceIdentity()
	if err != nil {
		return nil, fmt.Errorf("failed to get service identity: %w", err)
	}

	// Create metadata for identity propagation
	md := metadata.MD{}

	// Add current service identity
	md.Set(MetadataKeyServiceName, identity.Name)
	md.Set(MetadataKeyTrustDomain, identity.Domain)
	md.Set(MetadataKeyTimestamp, fmt.Sprintf("%d", time.Now().Unix()))

	// Generate or extract request ID
	requestID := i.getOrGenerateRequestID(ctx)
	md.Set(MetadataKeyRequestID, requestID)

	// Handle original caller propagation
	if i.config.PropagateOriginalCaller {
		originalCaller := i.getOriginalCaller(ctx, identity.URI)
		md.Set(MetadataKeyOriginalCaller, originalCaller)
	}

	// Handle call chain propagation
	if i.config.PropagateCallChain {
		callChain, err := i.buildCallChain(ctx, identity.URI)
		if err != nil {
			return nil, fmt.Errorf("failed to build call chain: %w", err)
		}
		if callChain != "" {
			md.Set(MetadataKeyCallChain, callChain)
		}
	}

	// Propagate custom headers
	i.propagateCustomHeaders(ctx, md)

	// Merge with existing metadata
	existingMD, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		md = metadata.Join(existingMD, md)
	}

	i.logger.Debug("Identity propagated",
		"method", method,
		"service", identity.Name,
		"request_id", requestID)

	return metadata.NewOutgoingContext(ctx, md), nil
}

// getOriginalCaller determines the original caller in the chain.
func (i *IdentityPropagationInterceptor) getOriginalCaller(ctx context.Context, currentIdentity string) string {
	// Check if we already have an original caller in incoming metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if originalCaller := md.Get(MetadataKeyOriginalCaller); len(originalCaller) > 0 {
			return originalCaller[0] // Preserve the original caller
		}
	}

	// If no original caller, we are the original caller
	return currentIdentity
}

// buildCallChain creates or extends the call chain.
func (i *IdentityPropagationInterceptor) buildCallChain(ctx context.Context, currentIdentity string) (string, error) {
	var callChain []string

	// Extract existing call chain using guard clauses
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		// No incoming metadata - start new chain
		return currentIdentity, nil
	}

	existingChain := md.Get(MetadataKeyCallChain)
	if len(existingChain) == 0 {
		// No existing chain - start new chain
		return currentIdentity, nil
	}

	// Parse existing chain
	callChain = strings.Split(existingChain[0], " -> ")

	// Validate chain depth limit
	if len(callChain) >= i.config.MaxCallChainDepth {
		return "", fmt.Errorf("call chain depth limit exceeded (%d)", i.config.MaxCallChainDepth)
	}

	// Check for circular calls
	if err := i.validateNoCycle(callChain, currentIdentity); err != nil {
		return "", err
	}

	// Add current service to the chain
	callChain = append(callChain, currentIdentity)

	return strings.Join(callChain, " -> "), nil
}

// Helper function to validate no circular calls (reduces complexity).
func (i *IdentityPropagationInterceptor) validateNoCycle(callChain []string, currentIdentity string) error {
	for _, service := range callChain {
		if service == currentIdentity {
			return fmt.Errorf("circular call detected: %s already in chain", currentIdentity)
		}
	}
	return nil
}

// propagateCustomHeaders copies specified custom headers from incoming to outgoing metadata.
func (i *IdentityPropagationInterceptor) propagateCustomHeaders(ctx context.Context, outgoingMD metadata.MD) {
	if len(i.config.CustomHeaders) == 0 {
		return
	}

	incomingMD, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return // No incoming metadata to propagate
	}

	for _, header := range i.config.CustomHeaders {
		if values := incomingMD.Get(header); len(values) > 0 {
			outgoingMD.Set(header, values...)
		}
	}
}

// getOrGenerateRequestID gets an existing request ID or generates a new one.
func (i *IdentityPropagationInterceptor) getOrGenerateRequestID(ctx context.Context) string {
	// Check for existing request ID in incoming metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if requestID := md.Get(MetadataKeyRequestID); len(requestID) > 0 {
			return requestID[0]
		}
	}

	// Check for request ID in context value (set by application)
	if requestID, ok := ctx.Value(RequestIDContextKey{}).(string); ok {
		return requestID
	}

	// Generate new request ID
	return generateRequestID()
}

// generateRequestID creates a new unique request ID.
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// IdentityPropagationServerInterceptor provides server-side identity extraction from propagated metadata.
type IdentityPropagationServerInterceptor struct {
	logger *slog.Logger
}

// NewIdentityPropagationServerInterceptor creates a server interceptor for identity extraction.
func NewIdentityPropagationServerInterceptor(logger *slog.Logger) *IdentityPropagationServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}

	return &IdentityPropagationServerInterceptor{
		logger: logger,
	}
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for identity extraction.
func (i *IdentityPropagationServerInterceptor) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// Extract and add identity information to context
		enrichedCtx := i.extractIdentityMetadata(ctx, info.FullMethod)

		return handler(enrichedCtx, req)
	}
}

// StreamServerInterceptor returns a gRPC stream server interceptor for identity extraction.
func (i *IdentityPropagationServerInterceptor) StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		// Extract and add identity information to context
		enrichedCtx := i.extractIdentityMetadata(ss.Context(), info.FullMethod)

		// Wrap stream with enriched context
		wrappedStream := &enrichedServerStream{
			ServerStream: ss,
			ctx:          enrichedCtx,
		}

		return handler(srv, wrappedStream)
	}
}

// enrichedServerStream wraps a grpc.ServerStream with an enriched context.
type enrichedServerStream struct {
	grpc.ServerStream
	ctx context.Context //nolint:containedctx // Required for gRPC ServerStream interface
}

// Context returns the enriched context.
func (s *enrichedServerStream) Context() context.Context {
	return s.ctx
}

// extractIdentityMetadata extracts identity metadata from incoming context and enriches it.
func (i *IdentityPropagationServerInterceptor) extractIdentityMetadata(ctx context.Context, method string) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	enrichedCtx := ctx

	// Use predefined context keys

	// Extract propagated identity information
	if originalCaller := md.Get(MetadataKeyOriginalCaller); len(originalCaller) > 0 {
		enrichedCtx = context.WithValue(enrichedCtx, originalCallerKey, originalCaller[0])
	}

	if callChain := md.Get(MetadataKeyCallChain); len(callChain) > 0 {
		enrichedCtx = context.WithValue(enrichedCtx, callChainKey, callChain[0])
	}

	if trustDomain := md.Get(MetadataKeyTrustDomain); len(trustDomain) > 0 {
		enrichedCtx = context.WithValue(enrichedCtx, callerTrustDomainKey, trustDomain[0])
	}

	if serviceName := md.Get(MetadataKeyServiceName); len(serviceName) > 0 {
		enrichedCtx = context.WithValue(enrichedCtx, callerServiceKey, serviceName[0])
	}

	if requestID := md.Get(MetadataKeyRequestID); len(requestID) > 0 {
		enrichedCtx = context.WithValue(enrichedCtx, requestIDKey, requestID[0])

		i.logger.Debug("Identity metadata extracted",
			"method", method,
			"request_id", requestID[0],
			"caller_service", getValueFromContext(enrichedCtx, "caller-service"),
			"call_chain", getValueFromContext(enrichedCtx, "call-chain"))
	}

	return enrichedCtx
}

// Helper function to safely get string values from context.
func getValueFromContext(ctx context.Context, key string) string {
	if value, ok := ctx.Value(key).(string); ok {
		return value
	}
	return ""
}

// Identity propagation helper functions

// Context key type for safe context values.
type contextKey string

// Context keys.
const (
	originalCallerKey    contextKey = "original-caller"
	callChainKey         contextKey = "call-chain"
	callerTrustDomainKey contextKey = "caller-trust-domain"
	callerServiceKey     contextKey = "caller-service"
	requestIDKey         contextKey = "request-id"
)

// GetOriginalCaller extracts the original caller from the context.
func GetOriginalCaller(ctx context.Context) (string, bool) {
	originalCaller, ok := ctx.Value(originalCallerKey).(string)
	return originalCaller, ok
}

// GetCallChain extracts the call chain from the context.
func GetCallChain(ctx context.Context) (string, bool) {
	callChain, ok := ctx.Value(callChainKey).(string)
	return callChain, ok
}

// GetCallerService extracts the immediate caller service from the context.
func GetCallerService(ctx context.Context) (string, bool) {
	callerService, ok := ctx.Value(callerServiceKey).(string)
	return callerService, ok
}

// GetCallerTrustDomain extracts the caller's trust domain from the context.
func GetCallerTrustDomain(ctx context.Context) (string, bool) {
	trustDomain, ok := ctx.Value(callerTrustDomainKey).(string)
	return trustDomain, ok
}

// GetRequestID extracts the request ID from the context.
func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey).(string)
	return requestID, ok
}

// DefaultIdentityPropagationConfig returns a default identity propagation configuration.
func DefaultIdentityPropagationConfig(identityProvider ports.IdentityProvider) *IdentityPropagationConfig {
	return &IdentityPropagationConfig{
		IdentityProvider:        identityProvider,
		PropagateOriginalCaller: true,
		PropagateCallChain:      true,
		MaxCallChainDepth:       defaultMaxCallChainDepth,
		CustomHeaders:           []string{},
		Logger:                  slog.Default(),
	}
}
