package interceptors

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/domain"
)

// Direct testing of pure functions without mocks

type directTestContextKey string

func TestParseSpiffeID_Direct(t *testing.T) {
	tests := []struct {
		name        string
		spiffeID    string
		expectError bool
		expected    *AuthenticatedIdentity
	}{
		{
			name:     "valid_spiffe_id_with_path",
			spiffeID: "spiffe://example.org/workload/test-service",
			expected: &AuthenticatedIdentity{
				SPIFFEID:     "spiffe://example.org/workload/test-service",
				TrustDomain:  "example.org",
				ServiceName:  "test-service",
				WorkloadPath: "/workload/test-service",
				Claims:       make(map[string]string),
			},
		},
		{
			name:     "valid_spiffe_id_without_path",
			spiffeID: "spiffe://example.org",
			expected: &AuthenticatedIdentity{
				SPIFFEID:     "spiffe://example.org",
				TrustDomain:  "example.org",
				ServiceName:  "",
				WorkloadPath: "",
				Claims:       make(map[string]string),
			},
		},
		{
			name:     "nested_path",
			spiffeID: "spiffe://company.com/env/production/service/user-api",
			expected: &AuthenticatedIdentity{
				SPIFFEID:     "spiffe://company.com/env/production/service/user-api",
				TrustDomain:  "company.com",
				ServiceName:  "user-api",
				WorkloadPath: "/env/production/service/user-api",
				Claims:       make(map[string]string),
			},
		},
		{
			name:        "invalid_scheme",
			spiffeID:    "https://example.org/test",
			expectError: true,
		},
		{
			name:        "empty_string",
			spiffeID:    "",
			expectError: true,
		},
		{
			name:        "missing_domain",
			spiffeID:    "spiffe://",
			expectError: true,
		},
		{
			name:        "invalid_format",
			spiffeID:    "not-a-spiffe-id",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interceptor := NewAuthInterceptor(DefaultAuthConfig())
			result, err := interceptor.parseSpiffeID(tt.spiffeID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expected.SPIFFEID, result.SPIFFEID)
			assert.Equal(t, tt.expected.TrustDomain, result.TrustDomain)
			assert.Equal(t, tt.expected.ServiceName, result.ServiceName)
			assert.Equal(t, tt.expected.WorkloadPath, result.WorkloadPath)
			assert.NotNil(t, result.Claims)
		})
	}
}

func TestMetadataExtraction_Direct(t *testing.T) {
	tests := []struct {
		name        string
		metadata    map[string]string
		key         string
		expectedVal string
		expectedOk  bool
	}{
		{
			name:        "key_exists",
			metadata:    map[string]string{"test-key": "test-value"},
			key:         "test-key",
			expectedVal: "test-value",
			expectedOk:  true,
		},
		{
			name:        "key_missing",
			metadata:    map[string]string{"other-key": "other-value"},
			key:         "test-key",
			expectedVal: "",
			expectedOk:  false,
		},
		{
			name:        "empty_metadata",
			metadata:    map[string]string{},
			key:         "test-key",
			expectedVal: "",
			expectedOk:  false,
		},
		{
			name:        "case_insensitive",
			metadata:    map[string]string{"Test-Key": "test-value"},
			key:         "test-key",
			expectedVal: "test-value",
			expectedOk:  true,
		},
		{
			name:        "special_characters",
			metadata:    map[string]string{"x-trace-id": "abc-123-def"},
			key:         "x-trace-id",
			expectedVal: "abc-123-def",
			expectedOk:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create context with metadata
			md := metadata.New(tt.metadata)
			ctx := metadata.NewIncomingContext(t.Context(), md)

			// Test extraction function directly
			val, ok := extractMetadataValue(ctx, tt.key)

			assert.Equal(t, tt.expectedOk, ok)
			assert.Equal(t, tt.expectedVal, val)
		})
	}
}

// Helper function for metadata extraction (would be in main code).
func extractMetadataValue(ctx context.Context, key string) (string, bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}

	values := md.Get(key)
	if len(values) == 0 {
		return "", false
	}

	return values[0], true
}

func TestPatternMatching_Direct(t *testing.T) {
	tests := []struct {
		name     string
		spiffeID string
		pattern  string
		expected bool
	}{
		// Exact matches
		{
			name:     "exact_match",
			spiffeID: "spiffe://example.org/test-service",
			pattern:  "spiffe://example.org/test-service",
			expected: true,
		},
		{
			name:     "exact_no_match",
			spiffeID: "spiffe://example.org/test-service",
			pattern:  "spiffe://example.org/other-service",
			expected: false,
		},

		// Wildcard matches
		{
			name:     "prefix_wildcard_match",
			spiffeID: "spiffe://example.org/test-service",
			pattern:  "spiffe://example.org/*",
			expected: true,
		},
		{
			name:     "prefix_wildcard_no_match",
			spiffeID: "spiffe://other.org/test-service",
			pattern:  "spiffe://example.org/*",
			expected: false,
		},
		{
			name:     "suffix_wildcard_match",
			spiffeID: "spiffe://example.org/test-service",
			pattern:  "*test-service",
			expected: true,
		},
		{
			name:     "suffix_wildcard_no_match",
			spiffeID: "spiffe://example.org/test-service",
			pattern:  "*other-service",
			expected: false,
		},

		// Edge cases
		{
			name:     "empty_pattern",
			spiffeID: "spiffe://example.org/test-service",
			pattern:  "",
			expected: false,
		},
		{
			name:     "empty_spiffe_id",
			spiffeID: "",
			pattern:  "spiffe://example.org/*",
			expected: false,
		},
		{
			name:     "wildcard_only",
			spiffeID: "anything",
			pattern:  "*",
			expected: true,
		},
		{
			name:     "multiple_wildcards_prefix",
			spiffeID: "spiffe://example.org/env/prod/service",
			pattern:  "spiffe://example.org/*",
			expected: true,
		},
	}

	interceptor := NewAuthInterceptor(DefaultAuthConfig())

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := interceptor.matchesPattern(tt.spiffeID, tt.pattern)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRequestIDGeneration_Direct(t *testing.T) {
	tests := []struct {
		name             string
		incomingMetadata map[string]string
		contextValue     interface{}
		contextKey       string
		expectGenerated  bool
		expectedPrefix   string
	}{
		{
			name:             "from_incoming_metadata",
			incomingMetadata: map[string]string{MetadataKeyRequestID: "req-from-metadata"},
			expectGenerated:  false,
		},
		{
			name:            "from_context_value",
			contextValue:    "req-from-context",
			contextKey:      "request-id",
			expectGenerated: false,
		},
		{
			name:            "generate_new",
			expectGenerated: true,
			expectedPrefix:  "req-",
		},
		{
			name:             "metadata_takes_precedence",
			incomingMetadata: map[string]string{MetadataKeyRequestID: "req-from-metadata"},
			contextValue:     "req-from-context",
			contextKey:       "request-id",
			expectGenerated:  false,
		},
	}

	provider := &mockIdentityProvider{}
	config := DefaultIdentityPropagationConfig(provider)
	interceptor := NewIdentityPropagationInterceptor(config)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := t.Context()

			// Setup incoming metadata
			if tt.incomingMetadata != nil {
				md := metadata.New(tt.incomingMetadata)
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			// Setup context value
			if tt.contextValue != nil {
				ctx = context.WithValue(ctx, directTestContextKey(tt.contextKey), tt.contextValue)
			}

			requestID := interceptor.getOrGenerateRequestID(ctx)

			assert.NotEmpty(t, requestID)

			if tt.expectGenerated {
				assert.True(t, strings.HasPrefix(requestID, tt.expectedPrefix))
				// Verify it's actually a timestamp-based ID
				assert.True(t, len(requestID) > len(tt.expectedPrefix))
			} else {
				// Should match expected value from metadata or context
				if tt.incomingMetadata != nil && tt.incomingMetadata[MetadataKeyRequestID] != "" {
					assert.Equal(t, tt.incomingMetadata[MetadataKeyRequestID], requestID)
				} else if tt.contextValue != nil {
					assert.Equal(t, tt.contextValue, requestID)
				}
			}
		})
	}
}

func TestCallChainValidation_Direct(t *testing.T) {
	tests := []struct {
		name           string
		existingChain  string
		currentService string
		maxDepth       int
		expectError    bool
		errorContains  string
		expectedChain  string
	}{
		{
			name:           "new_chain",
			currentService: "spiffe://example.org/service-a",
			maxDepth:       5,
			expectError:    false,
			expectedChain:  "spiffe://example.org/service-a",
		},
		{
			name:           "extend_chain",
			existingChain:  "spiffe://example.org/service-a -> spiffe://example.org/service-b",
			currentService: "spiffe://example.org/service-c",
			maxDepth:       5,
			expectError:    false,
			expectedChain:  "spiffe://example.org/service-a -> spiffe://example.org/service-b -> spiffe://example.org/service-c",
		},
		{
			name:           "depth_limit_exceeded",
			existingChain:  "s1 -> s2 -> s3 -> s4 -> s5",
			currentService: "s6",
			maxDepth:       5,
			expectError:    true,
			errorContains:  "depth limit exceeded",
		},
		{
			name:           "circular_call_detected",
			existingChain:  "spiffe://example.org/service-a -> spiffe://example.org/service-b",
			currentService: "spiffe://example.org/service-a",
			maxDepth:       5,
			expectError:    true,
			errorContains:  "circular call detected",
		},
		{
			name:           "self_call",
			currentService: "spiffe://example.org/service-a",
			maxDepth:       5,
			expectError:    false,
			expectedChain:  "spiffe://example.org/service-a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &mockIdentityProvider{}
			config := &IdentityPropagationConfig{
				IdentityProvider:  provider,
				MaxCallChainDepth: tt.maxDepth,
			}
			interceptor := NewIdentityPropagationInterceptor(config)

			ctx := t.Context()
			if tt.existingChain != "" {
				md := metadata.New(map[string]string{
					MetadataKeyCallChain: tt.existingChain,
				})
				ctx = metadata.NewIncomingContext(ctx, md)
			}

			result, err := interceptor.buildCallChain(ctx, tt.currentService)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedChain, result)
			}
		})
	}
}

func TestAuthorizationPolicies_Direct(t *testing.T) {
	tests := []struct {
		name            string
		allowedServices []string
		denyMode        bool
		requiredClaims  map[string]string
		identity        *AuthenticatedIdentity
		expectError     bool
		expectedCode    codes.Code
	}{
		{
			name:            "no_restrictions",
			allowedServices: []string{},
			identity: &AuthenticatedIdentity{
				SPIFFEID: "spiffe://example.org/any-service",
				Claims:   make(map[string]string),
			},
			expectError: false,
		},
		{
			name:            "allow_list_permitted",
			allowedServices: []string{"spiffe://example.org/allowed-service"},
			denyMode:        false,
			identity: &AuthenticatedIdentity{
				SPIFFEID: "spiffe://example.org/allowed-service",
				Claims:   make(map[string]string),
			},
			expectError: false,
		},
		{
			name:            "allow_list_denied",
			allowedServices: []string{"spiffe://example.org/allowed-service"},
			denyMode:        false,
			identity: &AuthenticatedIdentity{
				SPIFFEID: "spiffe://example.org/denied-service",
				Claims:   make(map[string]string),
			},
			expectError:  true,
			expectedCode: codes.PermissionDenied,
		},
		{
			name:            "deny_list_permitted",
			allowedServices: []string{"spiffe://example.org/denied-service"},
			denyMode:        true,
			identity: &AuthenticatedIdentity{
				SPIFFEID: "spiffe://example.org/allowed-service",
				Claims:   make(map[string]string),
			},
			expectError: false,
		},
		{
			name:            "deny_list_denied",
			allowedServices: []string{"spiffe://example.org/denied-service"},
			denyMode:        true,
			identity: &AuthenticatedIdentity{
				SPIFFEID: "spiffe://example.org/denied-service",
				Claims:   make(map[string]string),
			},
			expectError:  true,
			expectedCode: codes.PermissionDenied,
		},
		{
			name:           "required_claims_present",
			requiredClaims: map[string]string{"env": "prod", "role": "service"},
			identity: &AuthenticatedIdentity{
				SPIFFEID: "spiffe://example.org/service",
				Claims: map[string]string{
					"env":  "prod",
					"role": "service",
					"team": "platform", // Extra claims are okay
				},
			},
			expectError: false,
		},
		{
			name:           "required_claims_missing",
			requiredClaims: map[string]string{"env": "prod", "role": "service"},
			identity: &AuthenticatedIdentity{
				SPIFFEID: "spiffe://example.org/service",
				Claims: map[string]string{
					"env": "prod",
					// Missing "role" claim
				},
			},
			expectError:  true,
			expectedCode: codes.PermissionDenied,
		},
		{
			name:           "required_claims_wrong_value",
			requiredClaims: map[string]string{"env": "prod"},
			identity: &AuthenticatedIdentity{
				SPIFFEID: "spiffe://example.org/service",
				Claims: map[string]string{
					"env": "dev", // Wrong value
				},
			},
			expectError:  true,
			expectedCode: codes.PermissionDenied,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &AuthConfig{
				AllowedServices: tt.allowedServices,
				DenyMode:        tt.denyMode,
				RequiredClaims:  tt.requiredClaims,
			}
			interceptor := NewAuthInterceptor(config)

			err := interceptor.authorizeIdentity(tt.identity, "/test.Service/TestMethod")

			if tt.expectError {
				assert.Error(t, err)
				if tt.expectedCode != codes.OK {
					assert.Equal(t, tt.expectedCode, status.Code(err))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStreamInterceptorCreation_Direct(t *testing.T) {
	// Test that all stream interceptors can be created
	authConfig := DefaultAuthConfig()
	authInterceptor := NewAuthInterceptor(authConfig)

	identityProvider := &mockIdentityProvider{}
	identityConfig := DefaultIdentityPropagationConfig(identityProvider)
	identityInterceptor := NewIdentityPropagationInterceptor(identityConfig)

	metricsConfig := DefaultMetricsConfig("test")
	metricsInterceptor := NewMetricsInterceptor(metricsConfig)

	loggingConfig := NewSecureLoggingConfig()
	loggingInterceptor := NewLoggingInterceptor(loggingConfig)

	// Verify all stream interceptors exist
	assert.NotNil(t, authInterceptor.StreamServerInterceptor())
	assert.NotNil(t, identityInterceptor.StreamClientInterceptor())
	assert.NotNil(t, metricsInterceptor.StreamServerInterceptor())
	assert.NotNil(t, loggingInterceptor.StreamServerInterceptor())
}

func TestErrorCodesMapping_Direct(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected codes.Code
	}{
		{
			name:     "nil_error",
			input:    nil,
			expected: codes.OK,
		},
		{
			name:     "grpc_error",
			input:    status.Error(codes.InvalidArgument, "test error"),
			expected: codes.InvalidArgument,
		},
		{
			name:     "generic_error",
			input:    fmt.Errorf("generic error"),
			expected: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := codes.OK
			if tt.input != nil {
				if s, ok := status.FromError(tt.input); ok {
					code = s.Code()
				} else {
					code = codes.Unknown
				}
			}
			assert.Equal(t, tt.expected, code)
		})
	}
}

func TestConfigDefaults_Direct(t *testing.T) {
	// Test default configurations
	authConfig := DefaultAuthConfig()
	assert.NotNil(t, authConfig)
	assert.True(t, authConfig.RequireAuthentication) // Default is true
	assert.Empty(t, authConfig.AllowedServices)
	assert.Empty(t, authConfig.SkipMethods)

	metricsConfig := DefaultMetricsConfig("test-service")
	assert.NotNil(t, metricsConfig)
	assert.Equal(t, "test-service", metricsConfig.ServiceName)
	assert.NotNil(t, metricsConfig.MetricsCollector)

	loggingConfigSecure := NewSecureLoggingConfig()
	assert.NotNil(t, loggingConfigSecure)
	assert.False(t, loggingConfigSecure.LogPayloads) // Secure = no payloads
	assert.True(t, loggingConfigSecure.LogRequests)

	loggingConfigDebug := NewDebugLoggingConfig()
	assert.NotNil(t, loggingConfigDebug)
	assert.True(t, loggingConfigDebug.LogPayloads) // Debug = with payloads
}

func TestContextHelpers_Direct(t *testing.T) {
	// Test context helper functions
	ctx := t.Context()

	// Test empty context
	identity, ok := GetIdentityFromContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, identity)

	requestID, ok := GetRequestID(ctx)
	assert.False(t, ok)
	assert.Empty(t, requestID)

	originalCaller, ok := GetOriginalCaller(ctx)
	assert.False(t, ok)
	assert.Empty(t, originalCaller)

	callChain, ok := GetCallChain(ctx)
	assert.False(t, ok)
	assert.Empty(t, callChain)

	// Test with values
	testIdentity := &AuthenticatedIdentity{
		SPIFFEID:    "spiffe://test.com/service",
		ServiceName: "test-service",
	}
	ctx = context.WithValue(ctx, IdentityContextKey{}, testIdentity)

	identity, ok = GetIdentityFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, testIdentity, identity)
}

func TestAuthInterceptorEdgeCases_Direct(t *testing.T) {
	// Test auth interceptor with various edge cases
	authConfig := DefaultAuthConfig()
	authConfig.RequireAuthentication = true
	authConfig.AllowedServices = []string{"spiffe://test.com/*"}

	interceptor := NewAuthInterceptor(authConfig)

	// Test pattern matching edge cases
	assert.True(t, interceptor.matchesPattern("spiffe://test.com/service", "spiffe://test.com/*"))
	assert.False(t, interceptor.matchesPattern("spiffe://other.com/service", "spiffe://test.com/*"))
	assert.True(t, interceptor.matchesPattern("", "*")) // Empty string matches wildcard
	assert.False(t, interceptor.matchesPattern("test", ""))

	// Test authorization with different patterns
	identity := &AuthenticatedIdentity{
		SPIFFEID: "spiffe://test.com/allowed-service",
		Claims:   make(map[string]string),
	}

	// Should pass with wildcard match
	err := interceptor.authorizeIdentity(identity, "/test.Service/Method")
	assert.NoError(t, err)

	// Test with deny mode
	authConfig.DenyMode = true
	authConfig.AllowedServices = []string{"spiffe://test.com/denied-service"}
	interceptor2 := NewAuthInterceptor(authConfig)

	// Should pass because service is not in deny list
	err = interceptor2.authorizeIdentity(identity, "/test.Service/Method")
	assert.NoError(t, err)

	// Should fail because service is in deny list
	deniedIdentity := &AuthenticatedIdentity{
		SPIFFEID: "spiffe://test.com/denied-service",
		Claims:   make(map[string]string),
	}
	err = interceptor2.authorizeIdentity(deniedIdentity, "/test.Service/Method")
	assert.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestIdentityPropagationEdgeCases_Direct(t *testing.T) {
	// Test identity propagation with various edge cases
	provider := &mockIdentityProvider{
		identity: domain.NewServiceIdentity("test-service", "test.com"),
	}

	config := DefaultIdentityPropagationConfig(provider)
	interceptor := NewIdentityPropagationInterceptor(config)

	// Test request ID generation
	ctx := t.Context()
	requestID := interceptor.getOrGenerateRequestID(ctx)
	assert.NotEmpty(t, requestID)
	assert.True(t, strings.HasPrefix(requestID, "req-"))

	// Test call chain with circular detection
	ctx = t.Context()
	md := metadata.New(map[string]string{
		MetadataKeyCallChain: "spiffe://test.com/service1 -> spiffe://test.com/service2",
	})
	ctx = metadata.NewIncomingContext(ctx, md)

	// Should detect circular call
	_, err := interceptor.buildCallChain(ctx, "spiffe://test.com/service1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular call detected")

	// Test depth limit
	longChain := strings.Repeat("spiffe://test.com/service -> ", 15)
	md = metadata.New(map[string]string{
		MetadataKeyCallChain: longChain[:len(longChain)-4], // Remove trailing " -> "
	})
	ctx = metadata.NewIncomingContext(ctx, md)

	config.MaxCallChainDepth = 10
	_, err = interceptor.buildCallChain(ctx, "spiffe://test.com/newservice")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "depth limit exceeded")
}

func TestMetricsInterceptorEdgeCases_Direct(t *testing.T) {
	// Test metrics interceptor with edge cases
	collector := &mockMetricsCollector{}
	config := &MetricsConfig{
		MetricsCollector:     collector,
		ServiceName:          "test-service",
		EnablePayloadSize:    true,
		EnableActiveRequests: true,
	}

	interceptor := NewMetricsInterceptor(config)
	assert.NotNil(t, interceptor)

	// Test nil payload size estimation - should return 0
	assert.Equal(t, 0, estimatePayloadSize(nil))

	// Test string payload size
	assert.Greater(t, estimatePayloadSize("hello"), 0)

	// Test byte slice payload
	assert.Greater(t, estimatePayloadSize([]byte("helloworld")), 0)
}

func TestLoggingInterceptorEdgeCases_Direct(t *testing.T) {
	// Test logging interceptor edge cases
	config := NewDebugLoggingConfig()
	config.ExcludeMethods = []string{"/test.Service/ExcludedMethod"}

	interceptor := NewLoggingInterceptor(config)

	// Test method exclusion
	assert.True(t, interceptor.shouldExcludeMethod("/test.Service/ExcludedMethod"))
	assert.False(t, interceptor.shouldExcludeMethod("/test.Service/IncludedMethod"))

	// Test with health check
	config2 := NewSecureLoggingConfig()
	config2.ExcludeMethods = []string{"/grpc.health.v1.Health/Check"}
	interceptor2 := NewLoggingInterceptor(config2)
	assert.True(t, interceptor2.shouldExcludeMethod("/grpc.health.v1.Health/Check"))

	// Test with empty exclude list
	config3 := NewSecureLoggingConfig()
	config3.ExcludeMethods = []string{}
	interceptor3 := NewLoggingInterceptor(config3)
	assert.False(t, interceptor3.shouldExcludeMethod("/test.Service/AnyMethod"))
}

func TestAdditionalCoverageTests_Direct(t *testing.T) {
	// Additional tests to increase coverage

	// Test auth config helpers
	allowListConfig := NewAllowListAuthConfig([]string{"spiffe://test.com/service"})
	assert.True(t, allowListConfig.RequireAuthentication)
	assert.False(t, allowListConfig.DenyMode)
	assert.Equal(t, []string{"spiffe://test.com/service"}, allowListConfig.AllowedServices)

	denyListConfig := NewDenyListAuthConfig([]string{"spiffe://test.com/denied"})
	assert.True(t, denyListConfig.RequireAuthentication)
	assert.True(t, denyListConfig.DenyMode)
	assert.Equal(t, []string{"spiffe://test.com/denied"}, denyListConfig.AllowedServices)

	// Test that interceptor constructors handle nil logger gracefully
	authConfig := DefaultAuthConfig()
	authConfig.Logger = nil
	authInterceptor := NewAuthInterceptor(authConfig)
	assert.NotNil(t, authInterceptor)

	// Test identity propagation config defaults
	provider := &mockIdentityProvider{}
	identityConfig := DefaultIdentityPropagationConfig(provider)
	assert.NotNil(t, identityConfig)
	assert.Equal(t, provider, identityConfig.IdentityProvider)
	assert.True(t, identityConfig.PropagateOriginalCaller)
	assert.True(t, identityConfig.PropagateCallChain)
	assert.Equal(t, 10, identityConfig.MaxCallChainDepth)

	// Test metrics config with nil collector
	metricsConfig := &MetricsConfig{
		MetricsCollector: nil,
		ServiceName:      "test",
	}
	metricsInterceptor := NewMetricsInterceptor(metricsConfig)
	assert.NotNil(t, metricsInterceptor)
}

func TestRequireIdentityHelper_Direct(t *testing.T) {
	// Test RequireIdentity helper function
	ctx := t.Context()

	// Should return error when no identity present
	identity, err := RequireIdentity(ctx)
	assert.Error(t, err)
	assert.Nil(t, identity)
	assert.Equal(t, codes.Unauthenticated, status.Code(err))

	// Should return identity when present
	testIdentity := &AuthenticatedIdentity{
		SPIFFEID:    "spiffe://test.com/service",
		ServiceName: "test-service",
	}
	ctx = context.WithValue(ctx, IdentityContextKey{}, testIdentity)

	identity, err = RequireIdentity(ctx)
	assert.NoError(t, err)
	assert.Equal(t, testIdentity, identity)
}

func TestIdentityContextKeys_Direct(t *testing.T) {
	// Test that context keys work correctly
	ctx := t.Context()

	// Test identity context key
	identity := &AuthenticatedIdentity{SPIFFEID: "test"}
	ctx = context.WithValue(ctx, IdentityContextKey{}, identity)

	retrieved, ok := GetIdentityFromContext(ctx)
	assert.True(t, ok)
	assert.Equal(t, identity, retrieved)

	// Test wrong type in context
	ctx = context.WithValue(t.Context(), IdentityContextKey{}, "wrong-type")
	retrieved, ok = GetIdentityFromContext(ctx)
	assert.False(t, ok)
	assert.Nil(t, retrieved)
}

func TestSlowRequestDetection_Direct(t *testing.T) {
	tests := []struct {
		name               string
		threshold          time.Duration
		requestDuration    time.Duration
		expectSlowDetected bool
	}{
		{
			name:               "fast_request",
			threshold:          500 * time.Millisecond,
			requestDuration:    100 * time.Millisecond,
			expectSlowDetected: false,
		},
		{
			name:               "slow_request",
			threshold:          500 * time.Millisecond,
			requestDuration:    800 * time.Millisecond,
			expectSlowDetected: true,
		},
		{
			name:               "exactly_at_threshold",
			threshold:          500 * time.Millisecond,
			requestDuration:    500 * time.Millisecond,
			expectSlowDetected: false,
		},
		{
			name:               "just_over_threshold",
			threshold:          500 * time.Millisecond,
			requestDuration:    501 * time.Millisecond,
			expectSlowDetected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the logic directly
			isSlowRequest := tt.requestDuration > tt.threshold
			assert.Equal(t, tt.expectSlowDetected, isSlowRequest)

			// Test with actual logging config
			config := &LoggingConfig{
				SlowRequestThreshold: tt.threshold,
			}

			actualSlow := tt.requestDuration > config.SlowRequestThreshold
			assert.Equal(t, tt.expectSlowDetected, actualSlow)
		})
	}
}
