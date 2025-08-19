// Package interceptors provides gRPC interceptors for propagating identity and metadata in microservices,
// including request IDs, call chains, and original callers. It supports cycle detection and depth limits
// to prevent abuse in distributed systems.
//
// The package implements both unary and streaming interceptors for client and server-side gRPC operations,
// with support for SPIFFE/SPIRE-based identity propagation, custom header forwarding, and comprehensive
// observability through structured logging.
//
// Key features:
//   - Request ID generation and propagation
//   - Call chain building with cycle and depth detection
//   - Original caller tracking across service boundaries
//   - Custom header forwarding via gRPC metadata
//   - Comprehensive error handling with typed errors
//   - Dependency injection for testability (clock, ID generator, logger)
//
// Example usage:
//
//	clientInterceptor := NewIdentityPropagationInterceptor(
//		myIdentityProvider,
//		WithPropagateOriginalCaller(true),
//		WithPropagateCallChain(true),
//		WithMaxCallChainDepth(10),
//	)
//	serverInterceptor := NewIdentityPropagationServerInterceptor(nil, nil)
//
//	// Use with gRPC client
//	conn, err := grpc.Dial(address,
//		grpc.WithUnaryInterceptor(clientInterceptor.UnaryClientInterceptor()),
//		grpc.WithStreamInterceptor(clientInterceptor.StreamClientInterceptor()),
//	)
//
//	// Use with gRPC server
//	server := grpc.NewServer(
//		grpc.UnaryInterceptor(serverInterceptor.UnaryServerInterceptor()),
//		grpc.StreamInterceptor(serverInterceptor.StreamServerInterceptor()),
//	)
package interceptors

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/sufield/ephemos/internal/core/ports"
)

// Default configuration constants.
const (
	defaultMaxCallChainDepth = 10
	chainSep                 = " -> "
)

// Typed errors for identity propagation.
var (
	ErrDepthLimitExceeded = status.Error(codes.PermissionDenied, "call chain depth limit exceeded")
	ErrCircularCall       = status.Error(codes.PermissionDenied, "circular call detected")
)

// Clock provides time.Now functionality for dependency injection.
type Clock func() time.Time

// IDGen provides request ID generation for dependency injection.
type IDGen func() string

// MetricsCollector provides observability hooks for interceptor operations.
// Implementations can integrate with Prometheus, OpenTelemetry, or other monitoring systems.
type MetricsCollector interface {
	// RecordPropagationSuccess records successful identity propagation
	RecordPropagationSuccess(method string, requestID string)

	// RecordPropagationFailure records failed identity propagation
	RecordPropagationFailure(method string, reason string, err error)

	// RecordExtractionSuccess records successful identity extraction
	RecordExtractionSuccess(method string, requestID string)

	// RecordCallChainDepth records call chain depth for monitoring
	RecordCallChainDepth(depth int)

	// RecordCircularCallDetected records detection of circular calls
	RecordCircularCallDetected(identity string)
}

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

// IdentityPropagationInterceptor handles identity propagation for outgoing gRPC calls.
type IdentityPropagationInterceptor struct {
	// Direct capability injection (instead of config struct)
	identityProvider ports.IdentityProvider // Direct injection
	metricsCollector MetricsCollector       // Direct injection
	logger           *slog.Logger           // Direct injection
	clock            Clock                  // Direct injection
	idGen            IDGen                  // Direct injection

	// Configuration flags
	propagateOriginalCaller bool
	propagateCallChain      bool
	maxCallChainDepth       int
	customHeaders           []string
}

// InterceptorOption is a functional option for configuring the interceptor.
type InterceptorOption func(*IdentityPropagationInterceptor)

// WithLogger sets the logger for the interceptor.
func WithLogger(logger *slog.Logger) InterceptorOption {
	return func(i *IdentityPropagationInterceptor) {
		i.logger = logger
	}
}

// WithMetricsCollector sets the metrics collector for the interceptor.
func WithMetricsCollector(collector MetricsCollector) InterceptorOption {
	return func(i *IdentityPropagationInterceptor) {
		i.metricsCollector = collector
	}
}

// WithClock sets the clock function for the interceptor.
func WithClock(clock Clock) InterceptorOption {
	return func(i *IdentityPropagationInterceptor) {
		i.clock = clock
	}
}

// WithIDGenerator sets the ID generator function for the interceptor.
func WithIDGenerator(idGen IDGen) InterceptorOption {
	return func(i *IdentityPropagationInterceptor) {
		i.idGen = idGen
	}
}

// WithPropagateOriginalCaller enables original caller propagation.
func WithPropagateOriginalCaller(enabled bool) InterceptorOption {
	return func(i *IdentityPropagationInterceptor) {
		i.propagateOriginalCaller = enabled
	}
}

// WithPropagateCallChain enables call chain propagation.
func WithPropagateCallChain(enabled bool) InterceptorOption {
	return func(i *IdentityPropagationInterceptor) {
		i.propagateCallChain = enabled
	}
}

// WithMaxCallChainDepth sets the maximum call chain depth.
func WithMaxCallChainDepth(depth int) InterceptorOption {
	return func(i *IdentityPropagationInterceptor) {
		i.maxCallChainDepth = depth
	}
}

// WithCustomHeaders sets custom headers to propagate.
func WithCustomHeaders(headers []string) InterceptorOption {
	return func(i *IdentityPropagationInterceptor) {
		i.customHeaders = headers
	}
}

// NewIdentityPropagationInterceptor creates a new identity propagation interceptor with direct capability injection.
func NewIdentityPropagationInterceptor(identityProvider ports.IdentityProvider, opts ...InterceptorOption) *IdentityPropagationInterceptor {
	i := &IdentityPropagationInterceptor{
		identityProvider:        identityProvider,
		logger:                  slog.Default(),
		clock:                   time.Now,
		idGen:                   defaultIDGen,
		maxCallChainDepth:       defaultMaxCallChainDepth,
		propagateOriginalCaller: true,
		propagateCallChain:      true,
		customHeaders:           []string{},
	}

	// Apply options
	for _, opt := range opts {
		opt(i)
	}

	return i
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
			if i.metricsCollector != nil {
				i.metricsCollector.RecordPropagationFailure(method, "propagation_error", err)
			}
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
			i.logger.Error("Failed to propagate identity in stream",
				"method", method,
				"error", err)
			if i.metricsCollector != nil {
				i.metricsCollector.RecordPropagationFailure(method, "stream_propagation_error", err)
			}
			return nil, err
		}

		// Create the stream with propagated context
		return streamer(propagatedCtx, desc, cc, method, opts...)
	}
}

// propagateIdentity adds identity metadata to the outgoing context.
func (i *IdentityPropagationInterceptor) propagateIdentity(ctx context.Context, method string) (context.Context, error) {
	// Get current service identity
	identity, err := i.identityProvider.GetServiceIdentity()
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "get service identity: %v", err)
	}

	// Create metadata for identity propagation
	md := metadata.MD{}

	// Add current service identity
	md.Set(MetadataKeyServiceName, identity.Name())
	md.Set(MetadataKeyTrustDomain, identity.Domain())
	md.Set(MetadataKeyTimestamp, fmt.Sprintf("%d", i.clock().UnixMilli()))

	// Generate or extract request ID
	requestID := i.getOrGenerateRequestID(ctx)
	md.Set(MetadataKeyRequestID, requestID)

	// Handle original caller propagation
	if i.propagateOriginalCaller {
		originalCaller := i.getOriginalCaller(ctx, identity.URI())
		md.Set(MetadataKeyOriginalCaller, originalCaller)
	}

	// Handle call chain propagation
	if i.propagateCallChain {
		callChain, err := i.buildCallChain(ctx, identity.URI())
		if err != nil {
			return nil, err
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
		"service", identity.Name(),
		"request_id", requestID)

	// Record metrics if collector is available
	if i.metricsCollector != nil {
		i.metricsCollector.RecordPropagationSuccess(method, requestID)
	}

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

// buildCallChain creates or extends the call chain with enhanced error handling and performance.
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

	// Parse existing chain with delimiter constant
	callChain = strings.Split(existingChain[0], chainSep)

	// Validate chain depth limit with enhanced error wrapping
	if len(callChain) >= i.maxCallChainDepth {
		if i.metricsCollector != nil {
			i.metricsCollector.RecordPropagationFailure("depth_limit", "max_depth_exceeded", ErrDepthLimitExceeded)
		}
		return "", fmt.Errorf("chain length %d exceeds max %d: %w",
			len(callChain), i.maxCallChainDepth, ErrDepthLimitExceeded)
	}

	// Check for circular calls with enhanced error wrapping
	if err := i.validateNoCycle(callChain, currentIdentity); err != nil {
		if i.metricsCollector != nil {
			i.metricsCollector.RecordCircularCallDetected(currentIdentity)
		}
		return "", fmt.Errorf("circular call detected for identity %s: %w", currentIdentity, err)
	}

	// Add current service to the chain
	callChain = append(callChain, currentIdentity)

	// Use strings.Builder for better performance with large chains
	if len(callChain) > 5 { // Use Builder for longer chains
		var builder strings.Builder
		for i, service := range callChain {
			if i > 0 {
				builder.WriteString(chainSep)
			}
			builder.WriteString(service)
		}
		return builder.String(), nil
	}

	// Use simple join for shorter chains
	return strings.Join(callChain, chainSep), nil
}

// validateNoCycle validates no circular calls with enhanced error information.
func (i *IdentityPropagationInterceptor) validateNoCycle(callChain []string, currentIdentity string) error {
	for i, service := range callChain {
		if service == currentIdentity {
			return fmt.Errorf("service %s already exists at position %d in chain: %w",
				currentIdentity, i, ErrCircularCall)
		}
	}
	return nil
}

// propagateCustomHeaders copies specified custom headers from incoming to outgoing metadata.
func (i *IdentityPropagationInterceptor) propagateCustomHeaders(ctx context.Context, outgoingMD metadata.MD) {
	if len(i.customHeaders) == 0 {
		return
	}

	incomingMD, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return // No incoming metadata to propagate
	}

	for _, header := range i.customHeaders {
		normalizedHeader := strings.ToLower(header)
		if values := incomingMD.Get(normalizedHeader); len(values) > 0 {
			outgoingMD.Set(normalizedHeader, values...)
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

	// Generate new request ID using injected generator
	return i.idGen()
}

// IdentityPropagationServerInterceptor provides server-side identity extraction from propagated metadata.
type IdentityPropagationServerInterceptor struct {
	logger  *slog.Logger
	metrics MetricsCollector
}

// NewIdentityPropagationServerInterceptor creates a server interceptor for identity extraction.
func NewIdentityPropagationServerInterceptor(logger *slog.Logger, metrics MetricsCollector) *IdentityPropagationServerInterceptor {
	if logger == nil {
		logger = slog.Default()
	}

	return &IdentityPropagationServerInterceptor{
		logger:  logger,
		metrics: metrics,
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

		// Wrap the server stream with the enriched context
		wrappedStream := &wrappedServerStream{
			ServerStream: ss,
			ctx:          enrichedCtx,
		}

		return handler(srv, wrappedStream)
	}
}

// extractIdentityMetadata extracts identity metadata from incoming context and enriches it.
func (i *IdentityPropagationServerInterceptor) extractIdentityMetadata(ctx context.Context, method string) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	// Create unified identity struct
	identity := &PropagatedIdentity{}

	// Extract propagated identity information
	if originalCaller := md.Get(MetadataKeyOriginalCaller); len(originalCaller) > 0 {
		identity.OriginalCaller = originalCaller[0]
	}

	if callChain := md.Get(MetadataKeyCallChain); len(callChain) > 0 {
		identity.CallChain = callChain[0]
	}

	if trustDomain := md.Get(MetadataKeyTrustDomain); len(trustDomain) > 0 {
		identity.CallerTrustDomain = trustDomain[0]
	}

	if serviceName := md.Get(MetadataKeyServiceName); len(serviceName) > 0 {
		identity.CallerService = serviceName[0]
	}

	if requestID := md.Get(MetadataKeyRequestID); len(requestID) > 0 {
		identity.RequestID = requestID[0]
	}

	// Parse timestamp if available
	if timestamp := md.Get(MetadataKeyTimestamp); len(timestamp) > 0 {
		if ts, err := parseTimestamp(timestamp[0]); err == nil {
			identity.Timestamp = ts
		}
	}

	// Store unified identity in context
	enrichedCtx := context.WithValue(ctx, propagatedIdentityKey, identity)

	if identity.RequestID != "" {
		i.logger.Debug("Identity metadata extracted",
			"method", method,
			"request_id", identity.RequestID,
			"caller_service", identity.CallerService,
			"call_chain", identity.CallChain)

		// Record metrics if collector is available
		if i.metrics != nil {
			i.metrics.RecordExtractionSuccess(method, identity.RequestID)
			if identity.CallChain != "" {
				chainDepth := len(strings.Split(identity.CallChain, chainSep))
				i.metrics.RecordCallChainDepth(chainDepth)
			}
		}
	}

	return enrichedCtx
}

// parseTimestamp parses a timestamp string to Unix milliseconds.
func parseTimestamp(timestampStr string) (int64, error) {
	var timestamp int64
	_, err := fmt.Sscanf(timestampStr, "%d", &timestamp)
	return timestamp, err
}

// defaultIDGen generates a new request ID using UUID v4.
// This replaces the custom UUID implementation for better maintainability.
func defaultIDGen() string {
	return "req-" + uuid.New().String()
}

// Identity propagation helper functions

// Context key type for safe context values.
type contextKey string

// Context keys.
const (
	propagatedIdentityKey contextKey = "propagated-identity"
)

// PropagatedIdentity contains all identity information extracted from metadata.
// This unified struct reduces context key sprawl and improves type safety.
type PropagatedIdentity struct {
	// OriginalCaller is the first service in the call chain
	OriginalCaller string `json:"original_caller,omitempty"`

	// CallChain is the complete chain of services in the call path
	CallChain string `json:"call_chain,omitempty"`

	// CallerTrustDomain is the trust domain of the immediate caller
	CallerTrustDomain string `json:"caller_trust_domain,omitempty"`

	// CallerService is the name of the immediate caller service
	CallerService string `json:"caller_service,omitempty"`

	// RequestID is the unique identifier for this request
	RequestID string `json:"request_id,omitempty"`

	// Timestamp is the Unix millisecond timestamp when the request was initiated
	Timestamp int64 `json:"timestamp,omitempty"`
}

// GetPropagatedIdentity extracts the complete propagated identity from the context.
func GetPropagatedIdentity(ctx context.Context) (*PropagatedIdentity, bool) {
	identity, ok := ctx.Value(propagatedIdentityKey).(*PropagatedIdentity)
	return identity, ok
}

// wrappedServerStream wraps a gRPC ServerStream with an enriched context.
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

// Context returns the enriched context for the wrapped stream.
func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
