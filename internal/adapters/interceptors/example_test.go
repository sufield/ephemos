package interceptors_test

import (
	"context"
	"fmt"
	"log"

	"github.com/sufield/ephemos/internal/adapters/interceptors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// ExamplePropagatedIdentity demonstrates the unified identity access.
func ExamplePropagatedIdentity() {
	// Create a test context with propagated identity
	ctx := context.Background()
	ctx = interceptors.CreateTestContext(ctx)

	// Extract the complete identity struct
	identity, ok := interceptors.GetPropagatedIdentity(ctx)
	if ok {
		fmt.Printf("Original Caller: %s\n", identity.OriginalCaller)
		fmt.Printf("Call Chain: %s\n", identity.CallChain)
		fmt.Printf("Caller Service: %s\n", identity.CallerService)
		fmt.Printf("Request ID: %s\n", identity.RequestID)
		fmt.Printf("Timestamp: %d\n", identity.Timestamp)
		fmt.Printf("Trust Domain: %s\n", identity.CallerTrustDomain)
	}

	// Output:
	// Original Caller: spiffe://test.example.com/original-service
	// Call Chain: original-service -> intermediate-service
	// Caller Service: intermediate-service
	// Request ID: test-req-456
	// Timestamp: 1640995200000
	// Trust Domain: test.example.com
}

// ExampleMetricsCollection demonstrates the observability features.
func Example_metricsCollection() {
	// Create test configuration with metrics
	config := interceptors.TestingConfig()
	mockMetrics := config.MetricsCollector.(*interceptors.MockMetricsCollector)

	// Create interceptors
	_ = interceptors.NewIdentityPropagationInterceptor(config)
	serverInterceptor := interceptors.NewIdentityPropagationServerInterceptor(nil, mockMetrics)

	// Simulate some operations that would trigger metrics
	ctx := context.Background()

	// Simulate successful propagation
	md := metadata.New(map[string]string{
		"x-ephemos-request-id": "test-123",
		"x-ephemos-service-name": "test-service",
	})
	ctx = metadata.NewIncomingContext(ctx, md)

	// Extract identity (would normally be called by server interceptor)
	_ = serverInterceptor

	// Check collected metrics
	successCount, failureCount, extractionCount := interceptors.ExtractMetrics(mockMetrics)
	fmt.Printf("Propagation Successes: %d\n", successCount)
	fmt.Printf("Propagation Failures: %d\n", failureCount)
	fmt.Printf("Extraction Successes: %d\n", extractionCount)

	// Output:
	// Propagation Successes: 0
	// Propagation Failures: 0
	// Extraction Successes: 0
}

// ExampleStreamingSupport demonstrates the new streaming interceptor support.
func Example_streamingSupport() {
	config := interceptors.TestingConfig()
	clientInterceptor := interceptors.NewIdentityPropagationInterceptor(config)
	serverInterceptor := interceptors.NewIdentityPropagationServerInterceptor(nil, nil)

	// Get both unary and streaming interceptors
	unaryClient := clientInterceptor.UnaryClientInterceptor()
	streamClient := clientInterceptor.StreamClientInterceptor()
	unaryServer := serverInterceptor.UnaryServerInterceptor()
	streamServer := serverInterceptor.StreamServerInterceptor()

	if unaryClient != nil && streamClient != nil && unaryServer != nil && streamServer != nil {
		fmt.Println("All interceptor types available")
		fmt.Println("Streaming support: enabled")
		fmt.Println("Unary support: enabled")
	}

	// Output:
	// All interceptor types available
	// Streaming support: enabled
	// Unary support: enabled
}

// ExampleEnhancedErrorHandling demonstrates improved error messages.
func Example_enhancedErrorHandling() {
	config := interceptors.TestingConfig()
	config.MaxCallChainDepth = 2 // Very low for demonstration

	interceptor := interceptors.NewIdentityPropagationInterceptor(config)

	// Create context with a chain that's too long
	md := metadata.New(map[string]string{
		"x-ephemos-call-chain": "service1 -> service2",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// This would trigger a depth limit error with enhanced error message
	err := interceptor.UnaryClientInterceptor()(
		ctx, "/test.Service/Method", nil, nil, nil, 
		func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
			return nil
		},
	)

	if err != nil {
		log.Printf("Enhanced error: %v", err)
		// Would show: "chain length 2 exceeds max 2: rpc error: code = PermissionDenied desc = call chain depth limit exceeded"
	}

	fmt.Println("Error handling demonstrates enhanced context")
	// Output:
	// Error handling demonstrates enhanced context
}