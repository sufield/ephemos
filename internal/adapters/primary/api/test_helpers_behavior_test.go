//go:build test

package api

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestService_ContextCancellation(t *testing.T) {
	t.Parallel()
	
	service := NewTestService()
	
	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	
	req := &TestRequest{Input: "test"}
	resp, err := service.TestMethod(ctx, req)
	
	require.Error(t, err)
	require.Nil(t, resp)
	
	// Verify the error is a proper gRPC status error from context cancellation
	st, ok := status.FromError(err)
	require.True(t, ok, "Expected gRPC status error")
	assert.Equal(t, codes.Canceled, st.Code())
}

func TestTestService_ContextTimeout(t *testing.T) {
	t.Parallel()
	
	service := NewTestService()
	
	// Create a context that times out immediately
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	
	// Give it a moment to timeout
	time.Sleep(1 * time.Millisecond)
	
	req := &TestRequest{Input: "test"}
	resp, err := service.TestMethod(ctx, req)
	
	require.Error(t, err)
	require.Nil(t, resp)
	
	// Verify the error is a proper gRPC status error from context deadline
	st, ok := status.FromError(err)
	require.True(t, ok, "Expected gRPC status error")
	assert.Equal(t, codes.DeadlineExceeded, st.Code())
}

func TestTestService_StatusCodePreservation(t *testing.T) {
	t.Parallel()
	
	service := NewTestService()
	
	testCases := []struct {
		name string
		code codes.Code
	}{
		{"NotFound", codes.NotFound},
		{"InvalidArgument", codes.InvalidArgument},
		{"PermissionDenied", codes.PermissionDenied},
		{"Internal", codes.Internal},
		{"Unavailable", codes.Unavailable},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			
			service.SetShouldFail(true, tc.code)
			
			ctx := context.Background()
			req := &TestRequest{Input: "test"}
			resp, err := service.TestMethod(ctx, req)
			
			require.Error(t, err)
			require.Nil(t, resp)
			
			// Verify the exact status code is preserved
			st, ok := status.FromError(err)
			require.True(t, ok, "Expected gRPC status error")
			assert.Equal(t, tc.code, st.Code())
			assert.Contains(t, st.Message(), "simulated failure")
		})
	}
}

func TestTestService_SuccessfulCall(t *testing.T) {
	t.Parallel()
	
	service := NewTestService()
	
	ctx := context.Background()
	req := &TestRequest{Input: "hello"}
	resp, err := service.TestMethod(ctx, req)
	
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "processed: hello", resp.GetOutput())
	
	// Verify call tracking
	assert.Equal(t, 1, service.GetCallCount())
	assert.Equal(t, "hello", service.GetLastInput())
}

func TestTestService_Reset(t *testing.T) {
	t.Parallel()
	
	service := NewTestService()
	
	// Make some calls and set failure state
	ctx := context.Background()
	req := &TestRequest{Input: "test"}
	_, _ = service.TestMethod(ctx, req)
	service.SetShouldFail(true, codes.NotFound)
	
	// Verify state is set
	assert.Equal(t, 1, service.GetCallCount())
	assert.Equal(t, "test", service.GetLastInput())
	
	// Reset and verify clean state
	service.Reset()
	assert.Equal(t, 0, service.GetCallCount())
	assert.Equal(t, "", service.GetLastInput())
	
	// Verify failure state is reset
	resp, err := service.TestMethod(ctx, req)
	require.NoError(t, err)
	assert.Equal(t, "processed: test", resp.GetOutput())
}

func TestTestServiceRegistrar_ActualRegistration(t *testing.T) {
	t.Parallel()
	
	service := NewTestService()
	registrar := NewTestServiceRegistrar(service)
	
	// Create a real gRPC server
	server := grpc.NewServer()
	defer server.Stop()
	
	// Register the service
	registrar.Register(server)
	
	// Verify registration tracking
	assert.True(t, registrar.IsRegistered())
	assert.Equal(t, 1, registrar.GetRegisterCount())
	
	// The service should now be registered with the gRPC server
	// We can't easily test the actual registration without starting the server,
	// but we can verify our tracking is correct
}

func TestTestServiceRegistrar_Reset(t *testing.T) {
	t.Parallel()
	
	service := NewTestService()
	registrar := NewTestServiceRegistrar(service)
	
	// Register with separate servers (gRPC doesn't allow duplicate service registration)
	server1 := grpc.NewServer()
	defer server1.Stop()
	server2 := grpc.NewServer()
	defer server2.Stop()
	
	registrar.Register(server1)
	registrar.Register(server2)
	
	assert.True(t, registrar.IsRegistered())
	assert.Equal(t, 2, registrar.GetRegisterCount())
	
	// Reset and verify clean state
	registrar.Reset()
	assert.False(t, registrar.IsRegistered())
	assert.Equal(t, 0, registrar.GetRegisterCount())
}

func TestTestServiceRegistrar_NilServer(t *testing.T) {
	t.Parallel()
	
	service := NewTestService()
	registrar := NewTestServiceRegistrar(service)
	
	// Register with nil server should not panic
	registrar.Register(nil)
	
	// Should not be considered registered
	assert.False(t, registrar.IsRegistered())
	assert.Equal(t, 0, registrar.GetRegisterCount())
}