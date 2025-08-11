package interceptors

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/sufield/ephemos/internal/core/domain"
	"github.com/sufield/ephemos/internal/core/ports"
)

const testTrustDomain = "example.org"

type identityTestContextKey string

// Mock identity provider for testing.
type mockIdentityProvider struct {
	identity *domain.ServiceIdentity
	err      error
}

func (m *mockIdentityProvider) GetServiceIdentity() (*domain.ServiceIdentity, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.identity, nil
}

func (m *mockIdentityProvider) GetCertificate() (*domain.Certificate, error) {
	return nil, errors.New("not implemented")
}

func (m *mockIdentityProvider) GetTrustBundle() (*domain.TrustBundle, error) {
	return nil, errors.New("not implemented")
}

func (m *mockIdentityProvider) Close() error {
	return nil
}

func TestNewIdentityPropagationInterceptor(t *testing.T) {
	provider := &mockIdentityProvider{
		identity: domain.NewServiceIdentity("test-service", testTrustDomain),
	}

	config := &IdentityPropagationConfig{
		IdentityProvider: provider,
		Logger:           slog.Default(),
	}

	interceptor := NewIdentityPropagationInterceptor(config)

	if interceptor == nil {
		t.Fatal("NewIdentityPropagationInterceptor returned nil")
	}
	if interceptor.config != config {
		t.Error("Config not properly set")
	}
	if interceptor.logger == nil {
		t.Error("Logger not set")
	}
	if interceptor.config.MaxCallChainDepth != 10 {
		t.Errorf("Expected default MaxCallChainDepth 10, got: %d", interceptor.config.MaxCallChainDepth)
	}
}

func TestNewIdentityPropagationInterceptor_WithNilLogger(t *testing.T) {
	provider := &mockIdentityProvider{}
	config := &IdentityPropagationConfig{
		IdentityProvider: provider,
		Logger:           nil,
	}

	interceptor := NewIdentityPropagationInterceptor(config)

	if interceptor.logger == nil {
		t.Error("Logger should be set to default when nil provided")
	}
}

func TestIdentityPropagationInterceptor_UnaryClientInterceptor_Success(t *testing.T) {
	provider := &mockIdentityProvider{
		identity: domain.NewServiceIdentity("test-service", testTrustDomain),
	}

	config := &IdentityPropagationConfig{
		IdentityProvider:        provider,
		PropagateOriginalCaller: true,
		PropagateCallChain:      true,
		Logger:                  slog.Default(),
	}

	interceptor := NewIdentityPropagationInterceptor(config)

	var capturedMetadata metadata.MD
	invoker := func(ctx context.Context, _ string, _, _ interface{}, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		// Capture the outgoing metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			capturedMetadata = md
		}
		return nil
	}

	err := interceptor.UnaryClientInterceptor()(
		t.Context(), "/test.Service/TestMethod",
		"request", "reply", nil, invoker)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify metadata was added
	if capturedMetadata == nil {
		t.Fatal("No metadata found in outgoing context")
	}

	// Check service name
	if values := capturedMetadata.Get(MetadataKeyServiceName); len(values) == 0 || values[0] != "test-service" {
		t.Errorf("Expected service name 'test-service', got: %v", values)
	}

	// Check trust domain
	if values := capturedMetadata.Get(MetadataKeyTrustDomain); len(values) == 0 || values[0] != testTrustDomain {
		t.Errorf("Expected trust domain '%s', got: %v", testTrustDomain, values)
	}

	// Check request ID was generated
	if values := capturedMetadata.Get(MetadataKeyRequestID); len(values) == 0 {
		t.Error("Expected request ID to be generated")
	}

	// Check timestamp
	if values := capturedMetadata.Get(MetadataKeyTimestamp); len(values) == 0 {
		t.Error("Expected timestamp to be set")
	}
}

func TestIdentityPropagationInterceptor_UnaryClientInterceptor_ProviderError(t *testing.T) {
	provider := &mockIdentityProvider{
		err: errors.New("provider error"),
	}

	config := &IdentityPropagationConfig{
		IdentityProvider: provider,
		Logger:           slog.Default(),
	}

	interceptor := NewIdentityPropagationInterceptor(config)

	invoker := func(_ context.Context, _ string, _, _ interface{}, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
		return nil
	}

	err := interceptor.UnaryClientInterceptor()(
		t.Context(), "/test.Service/TestMethod",
		"request", "reply", nil, invoker)

	if err == nil {
		t.Error("Expected error from identity provider")
	}
}

func TestIdentityPropagationInterceptor_PropagateOriginalCaller(t *testing.T) {
	provider := &mockIdentityProvider{
		identity: domain.NewServiceIdentity("current-service", testTrustDomain),
	}

	config := &IdentityPropagationConfig{
		IdentityProvider:        provider,
		PropagateOriginalCaller: true,
		Logger:                  slog.Default(),
	}

	interceptor := NewIdentityPropagationInterceptor(config)

	t.Run("new_call_chain", func(t *testing.T) {
		ctx := t.Context()
		result, err := interceptor.propagateIdentity(ctx, "/test.Service/TestMethod")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		md, ok := metadata.FromOutgoingContext(result)
		if !ok {
			t.Fatal("No outgoing metadata found")
		}

		// Should set original caller to current service
		if values := md.Get(MetadataKeyOriginalCaller); len(values) == 0 || values[0] != provider.identity.URI {
			t.Errorf("Expected original caller '%s', got: %v", provider.identity.URI, values)
		}
	})

	t.Run("existing_original_caller", func(t *testing.T) {
		// Create context with existing original caller
		incomingMD := metadata.New(map[string]string{
			MetadataKeyOriginalCaller: "spiffe://example.org/original-caller",
		})
		ctx := metadata.NewIncomingContext(t.Context(), incomingMD)

		result, err := interceptor.propagateIdentity(ctx, "/test.Service/TestMethod")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		md, ok := metadata.FromOutgoingContext(result)
		if !ok {
			t.Fatal("No outgoing metadata found")
		}

		// Should preserve existing original caller
		if values := md.Get(MetadataKeyOriginalCaller); len(values) == 0 || values[0] != "spiffe://example.org/original-caller" {
			t.Errorf("Expected preserved original caller, got: %v", values)
		}
	})
}

func TestIdentityPropagationInterceptor_PropagateCallChain(t *testing.T) {
	tests := []struct {
		name          string
		incomingChain string
		maxDepth      int
		expectChain   string
		expectError   bool
	}{
		{
			name:          "new_call_chain",
			incomingChain: "",
			maxDepth:      3,
			expectChain:   "spiffe://example.org/current-service",
		},
		{
			name:          "extend_existing_call_chain",
			incomingChain: "spiffe://example.org/service1 -> spiffe://example.org/service2",
			maxDepth:      3,
			expectChain:   "spiffe://example.org/service1 -> spiffe://example.org/service2 -> spiffe://example.org/current-service",
		},
		{
			name:          "call_chain_depth_limit",
			incomingChain: "service1 -> service2 -> service3",
			maxDepth:      3,
			expectError:   true,
		},
		{
			name:          "circular_call_detection",
			incomingChain: "service1 -> spiffe://example.org/current-service",
			maxDepth:      3,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runCallChainPropagationTest(t, tt)
		})
	}
}

// Helper function to run call chain propagation test (reduces main function complexity).
func runCallChainPropagationTest(t *testing.T, tt struct {
	name          string
	incomingChain string
	maxDepth      int
	expectChain   string
	expectError   bool
},
) {
	t.Helper()

	provider := &mockIdentityProvider{
		identity: domain.NewServiceIdentity("current-service", testTrustDomain),
	}

	config := &IdentityPropagationConfig{
		IdentityProvider:   provider,
		PropagateCallChain: true,
		MaxCallChainDepth:  tt.maxDepth,
		Logger:             slog.Default(),
	}

	interceptor := NewIdentityPropagationInterceptor(config)
	ctx := setupCallChainContext(t.Context(), tt.incomingChain)

	result, err := interceptor.propagateIdentity(ctx, "/test.Service/TestMethod")

	if tt.expectError {
		if err == nil {
			t.Error("Expected error but got none")
		}
		return
	}

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	validateCallChainResult(result, t, tt.expectChain)
}

// Helper function to setup call chain context (reduces complexity).
func setupCallChainContext(ctx context.Context, incomingChain string) context.Context {
	if incomingChain == "" {
		return ctx
	}
	incomingMD := metadata.New(map[string]string{
		MetadataKeyCallChain: incomingChain,
	})
	return metadata.NewIncomingContext(ctx, incomingMD)
}

// Helper function to validate call chain result (reduces complexity).
func validateCallChainResult(result context.Context, t *testing.T, expectChain string) {
	t.Helper()

	md, ok := metadata.FromOutgoingContext(result)
	if !ok {
		t.Fatal("No outgoing metadata found")
	}

	values := md.Get(MetadataKeyCallChain)
	if len(values) == 0 {
		t.Error("Expected call chain in outgoing metadata")
		return
	}

	if values[0] != expectChain {
		t.Errorf("Expected call chain '%s', got: %s", expectChain, values[0])
	}
}

func TestIdentityPropagationInterceptor_PropagateCustomHeaders(t *testing.T) {
	provider := &mockIdentityProvider{
		identity: domain.NewServiceIdentity("current-service", testTrustDomain),
	}

	config := &IdentityPropagationConfig{
		IdentityProvider: provider,
		CustomHeaders:    []string{"x-custom-header", "x-trace-id"},
		Logger:           slog.Default(),
	}

	interceptor := NewIdentityPropagationInterceptor(config)

	// Create context with incoming custom headers
	incomingMD := metadata.New(map[string]string{
		"x-custom-header": "custom-value",
		"x-trace-id":      "trace-123",
		"x-other-header":  "not-propagated",
	})
	ctx := metadata.NewIncomingContext(t.Context(), incomingMD)

	result, err := interceptor.propagateIdentity(ctx, "/test.Service/TestMethod")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	md, ok := metadata.FromOutgoingContext(result)
	if !ok {
		t.Fatal("No outgoing metadata found")
	}

	// Check custom headers were propagated
	if values := md.Get("x-custom-header"); len(values) == 0 || values[0] != "custom-value" {
		t.Errorf("Expected x-custom-header 'custom-value', got: %v", values)
	}
	if values := md.Get("x-trace-id"); len(values) == 0 || values[0] != "trace-123" {
		t.Errorf("Expected x-trace-id 'trace-123', got: %v", values)
	}

	// Check non-configured header was not propagated
	if values := md.Get("x-other-header"); len(values) > 0 {
		t.Errorf("Expected x-other-header not to be propagated, got: %v", values)
	}
}

func TestNewIdentityPropagationServerInterceptor(t *testing.T) {
	interceptor := NewIdentityPropagationServerInterceptor(slog.Default())

	if interceptor == nil {
		t.Fatal("NewIdentityPropagationServerInterceptor returned nil")
	}
	if interceptor.logger == nil {
		t.Error("Logger not set")
	}
}

func TestNewIdentityPropagationServerInterceptor_WithNilLogger(t *testing.T) {
	interceptor := NewIdentityPropagationServerInterceptor(nil)

	if interceptor.logger == nil {
		t.Error("Logger should be set to default when nil provided")
	}
}

// serverInterceptorTestCase defines test cases for server interceptor tests.
type serverInterceptorTestCase struct {
	name                   string
	incomingMD             metadata.MD
	expectedOriginalCaller string
	expectedCallChain      string
	expectedRequestID      string
	expectedCallerService  string
	expectedTrustDomain    string
	expectError            bool
}

func TestIdentityPropagationServerInterceptor_UnaryServerInterceptor(t *testing.T) {
	tests := []serverInterceptorTestCase{
		{
			name: "full_metadata_propagation",
			incomingMD: metadata.New(map[string]string{
				MetadataKeyOriginalCaller: "spiffe://example.org/original-caller",
				MetadataKeyCallChain:      "service1 -> service2",
				MetadataKeyRequestID:      "req-123",
				MetadataKeyServiceName:    "caller-service",
				MetadataKeyTrustDomain:    testTrustDomain,
			}),
			expectedOriginalCaller: "spiffe://example.org/original-caller",
			expectedCallChain:      "service1 -> service2",
			expectedRequestID:      "req-123",
			expectedCallerService:  "caller-service",
			expectedTrustDomain:    testTrustDomain,
		},
		{
			name: "partial_metadata_propagation",
			incomingMD: metadata.New(map[string]string{
				MetadataKeyRequestID:   "req-456",
				MetadataKeyServiceName: "partial-service",
			}),
			expectedRequestID:     "req-456",
			expectedCallerService: "partial-service",
		},
		{
			name:       "no_metadata",
			incomingMD: metadata.New(map[string]string{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runServerInterceptorTest(t, &tt)
		})
	}
}

// Helper function to run individual server interceptor test (reduces main function complexity).
func runServerInterceptorTest(t *testing.T, tt *serverInterceptorTestCase) {
	t.Helper()
	interceptor := NewIdentityPropagationServerInterceptor(slog.Default())

	var enrichedContext context.Context
	handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
		enrichedContext = ctx
		return defaultResultCode, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}

	ctx := metadata.NewIncomingContext(t.Context(), tt.incomingMD)
	result, err := interceptor.UnaryServerInterceptor()(ctx, "request", info, handler)

	if tt.expectError {
		if err == nil {
			t.Error("Expected error but got none")
		}
		return
	}

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != defaultResultCode {
		t.Errorf("Expected 'success', got: %v", result)
	}

	// Validate propagated metadata using helper
	validatePropagatedMetadata(enrichedContext, t, tt)
}

// contextValidationRule defines a validation rule for context values.
type contextValidationRule struct {
	name        string
	expectedVal string
	getterFunc  func(context.Context) (string, bool)
}

// Helper function to validate propagated metadata (reduces complexity).
func validatePropagatedMetadata(ctx context.Context, t *testing.T, expected *serverInterceptorTestCase) {
	t.Helper()

	// Define validation rules to reduce cyclomatic complexity
	rules := []contextValidationRule{
		{
			name:        "original caller",
			expectedVal: expected.expectedOriginalCaller,
			getterFunc:  GetOriginalCaller,
		},
		{
			name:        "call chain",
			expectedVal: expected.expectedCallChain,
			getterFunc:  GetCallChain,
		},
		{
			name:        "request ID",
			expectedVal: expected.expectedRequestID,
			getterFunc:  GetRequestID,
		},
		{
			name:        "caller service",
			expectedVal: expected.expectedCallerService,
			getterFunc:  GetCallerService,
		},
		{
			name:        "trust domain",
			expectedVal: expected.expectedTrustDomain,
			getterFunc:  GetCallerTrustDomain,
		},
	}

	// Validate each rule
	for _, rule := range rules {
		validateContextRule(ctx, t, rule)
	}
}

// Helper function to validate individual context rule (reduces complexity).
func validateContextRule(ctx context.Context, t *testing.T, rule contextValidationRule) {
	t.Helper()

	// Skip validation if no expected value
	if rule.expectedVal == "" {
		return
	}

	actualVal, ok := rule.getterFunc(ctx)
	if !ok || actualVal != rule.expectedVal {
		t.Errorf("Expected %s '%s', got: %s (ok: %v)", rule.name, rule.expectedVal, actualVal, ok)
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := generateRequestID()
	id2 := generateRequestID()

	if id1 == "" {
		t.Error("Expected non-empty request ID")
	}
	if id1 == id2 {
		t.Error("Expected unique request IDs")
	}
	if id1[0:4] != "req-" {
		t.Errorf("Expected request ID to start with 'req-', got: %s", id1)
	}
}

func TestGetOrGenerateRequestID(t *testing.T) {
	provider := &mockIdentityProvider{}
	config := &IdentityPropagationConfig{
		IdentityProvider: provider,
		Logger:           slog.Default(),
	}
	interceptor := NewIdentityPropagationInterceptor(config)

	t.Run("from_incoming_metadata", func(t *testing.T) {
		incomingMD := metadata.New(map[string]string{
			MetadataKeyRequestID: "existing-req-id",
		})
		ctx := metadata.NewIncomingContext(t.Context(), incomingMD)

		requestID := interceptor.getOrGenerateRequestID(ctx)

		if requestID != "existing-req-id" {
			t.Errorf("Expected 'existing-req-id', got: %s", requestID)
		}
	})

	t.Run("from_context_value", func(t *testing.T) {
		ctx := context.WithValue(t.Context(), identityTestContextKey("request-id"), "context-req-id")

		requestID := interceptor.getOrGenerateRequestID(ctx)

		if requestID != "context-req-id" {
			t.Errorf("Expected 'context-req-id', got: %s", requestID)
		}
	})

	t.Run("generate_new", func(t *testing.T) {
		ctx := t.Context()

		requestID := interceptor.getOrGenerateRequestID(ctx)

		if requestID == "" {
			t.Error("Expected generated request ID")
		}
		if requestID[0:4] != "req-" {
			t.Errorf("Expected generated ID to start with 'req-', got: %s", requestID)
		}
	})
}

func TestDefaultIdentityPropagationConfig(t *testing.T) {
	provider := &mockIdentityProvider{}
	config := DefaultIdentityPropagationConfig(ports.IdentityProvider(provider))

	if config.IdentityProvider != provider {
		t.Error("Identity provider not set correctly")
	}
	if !config.PropagateOriginalCaller {
		t.Error("Expected PropagateOriginalCaller to be true")
	}
	if !config.PropagateCallChain {
		t.Error("Expected PropagateCallChain to be true")
	}
	if config.MaxCallChainDepth != 10 {
		t.Errorf("Expected MaxCallChainDepth 10, got: %d", config.MaxCallChainDepth)
	}
	if len(config.CustomHeaders) != 0 {
		t.Error("Expected empty CustomHeaders by default")
	}
	if config.Logger == nil {
		t.Error("Expected default logger")
	}
}

func TestContextHelperFunctions(t *testing.T) {
	type contextTestCase struct {
		name      string
		setupCtx  func(ctx context.Context) context.Context
		expectVal string
		expectOK  bool
	}

	testGroups := []struct {
		helperName string
		cases      []contextTestCase
		testFunc   func(ctx context.Context) (string, bool)
	}{
		{
			helperName: "GetOriginalCaller",
			testFunc:   GetOriginalCaller,
			cases: []contextTestCase{
				{
					name: "present",
					setupCtx: func(ctx context.Context) context.Context {
						return context.WithValue(ctx, originalCallerKey, "test-caller")
					},
					expectVal: "test-caller",
					expectOK:  true,
				},
				{
					name:      "absent",
					setupCtx:  func(ctx context.Context) context.Context { return ctx },
					expectVal: "",
					expectOK:  false,
				},
			},
		},
		{
			helperName: "GetCallChain",
			testFunc:   GetCallChain,
			cases: []contextTestCase{
				{
					name: "present",
					setupCtx: func(ctx context.Context) context.Context {
						return context.WithValue(ctx, callChainKey, "service1 -> service2")
					},
					expectVal: "service1 -> service2",
					expectOK:  true,
				},
				{
					name:      "absent",
					setupCtx:  func(ctx context.Context) context.Context { return ctx },
					expectVal: "",
					expectOK:  false,
				},
			},
		},
		{
			helperName: "GetCallerService",
			testFunc:   GetCallerService,
			cases: []contextTestCase{
				{
					name: "present",
					setupCtx: func(ctx context.Context) context.Context {
						return context.WithValue(ctx, callerServiceKey, "caller-service")
					},
					expectVal: "caller-service",
					expectOK:  true,
				},
				{
					name:      "absent",
					setupCtx:  func(ctx context.Context) context.Context { return ctx },
					expectVal: "",
					expectOK:  false,
				},
			},
		},
		{
			helperName: "GetCallerTrustDomain",
			testFunc:   GetCallerTrustDomain,
			cases: []contextTestCase{
				{
					name: "present",
					setupCtx: func(ctx context.Context) context.Context {
						return context.WithValue(ctx, callerTrustDomainKey, testTrustDomain)
					},
					expectVal: testTrustDomain,
					expectOK:  true,
				},
				{
					name:      "absent",
					setupCtx:  func(ctx context.Context) context.Context { return ctx },
					expectVal: "",
					expectOK:  false,
				},
			},
		},
		{
			helperName: "GetRequestID",
			testFunc:   GetRequestID,
			cases: []contextTestCase{
				{
					name:      "present",
					setupCtx:  func(ctx context.Context) context.Context { return context.WithValue(ctx, requestIDKey, "req-123") },
					expectVal: "req-123",
					expectOK:  true,
				},
				{
					name:      "absent",
					setupCtx:  func(ctx context.Context) context.Context { return ctx },
					expectVal: "",
					expectOK:  false,
				},
			},
		},
	}

	for _, group := range testGroups {
		t.Run(group.helperName, func(t *testing.T) {
			for _, tc := range group.cases {
				t.Run(tc.name, func(t *testing.T) {
					runContextHelperTest(t, tc, group.testFunc)
				})
			}
		})
	}
}

// Helper function to run individual context helper test (reduces main function complexity).
func runContextHelperTest(t *testing.T, tc struct {
	name      string
	setupCtx  func(ctx context.Context) context.Context
	expectVal string
	expectOK  bool
}, testFunc func(ctx context.Context) (string, bool),
) {
	t.Helper()

	ctx := tc.setupCtx(t.Context())
	actualVal, actualOK := testFunc(ctx)

	if actualOK != tc.expectOK {
		t.Errorf("Expected ok=%v, got ok=%v", tc.expectOK, actualOK)
		return
	}

	if actualVal != tc.expectVal {
		t.Errorf("Expected value='%s', got value='%s'", tc.expectVal, actualVal)
	}
}
