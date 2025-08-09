package test

import (
	"testing"

	"github.com/sufield/ephemos/examples/proto"
	"google.golang.org/grpc"
)

func TestNewEchoServiceRegistrar(t *testing.T) {
	tests := []struct {
		name   string
		server proto.EchoServiceServer
		want   bool
	}{
		{
			name:   "valid server",
			server: &MockEchoServer{},
			want:   true,
		},
		{
			name:   "nil server",
			server: nil,
			want:   true, // Constructor allows nil but registrar should handle it
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrar := proto.NewEchoServiceRegistrar(tt.server)
			if (registrar != nil) != tt.want {
				t.Errorf("NewEchoServiceRegistrar() = %v, want %v", registrar != nil, tt.want)
			}
		})
	}
}

func TestEchoServiceRegistrar_Register(t *testing.T) {
	// Create a mock gRPC server for testing
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()

	// Create registrar with mock server
	mockServer := &MockEchoServer{}
	registrar := proto.NewEchoServiceRegistrar(mockServer)

	// This should not panic or return error
	registrar.Register(grpcServer)

	// Test with nil server in registrar
	nilRegistrar := proto.NewEchoServiceRegistrar(nil)

	// This test checks that Register doesn't panic with nil server
	// The actual gRPC registration may fail, but the registrar should handle it gracefully
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Register() with nil server panicked: %v", r)
		}
	}()
	nilRegistrar.Register(grpcServer)
}

func TestEchoServiceRegistrar_Integration(t *testing.T) {
	// Integration test showing how the registrar would be used
	grpcServer := grpc.NewServer()
	defer grpcServer.Stop()

	// Create and register echo service
	echoServer := &MockEchoServer{}
	registrar := proto.NewEchoServiceRegistrar(echoServer)
	registrar.Register(grpcServer)

	// Verify the service info was registered
	serviceInfo := grpcServer.GetServiceInfo()
	if _, exists := serviceInfo["ephemos.echo.EchoService"]; !exists {
		t.Error("EchoService was not registered with gRPC server")
	}

	// Check that the Echo method is available
	if service, exists := serviceInfo["ephemos.echo.EchoService"]; exists {
		found := false
		for _, method := range service.Methods {
			if method.Name == "Echo" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Echo method was not registered")
		}
	}
}

// Additional test to verify registrar interface compliance
func TestServiceRegistrarInterface(t *testing.T) {
	// This test ensures that EchoServiceRegistrar implements the expected interface
	var registrar interface{} = proto.NewEchoServiceRegistrar(&MockEchoServer{})

	// Check if it has the Register method with correct signature
	if r, ok := registrar.(interface{ Register(*grpc.Server) }); !ok {
		t.Error("EchoServiceRegistrar does not implement Register(*grpc.Server) method")
	} else {
		// Verify we can call Register without panic
		grpcServer := grpc.NewServer()
		defer grpcServer.Stop()

		defer func() {
			if recover() != nil {
				t.Error("Register method panicked")
			}
		}()
		r.Register(grpcServer)
	}
}

func BenchmarkEchoServiceRegistrar_Register(b *testing.B) {
	mockServer := &MockEchoServer{}
	registrar := proto.NewEchoServiceRegistrar(mockServer)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		grpcServer := grpc.NewServer()
		registrar.Register(grpcServer)
		grpcServer.Stop()
	}
}
