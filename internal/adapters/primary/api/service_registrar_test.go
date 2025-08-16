package api_test

import (
	"testing"

	"google.golang.org/grpc"

	"github.com/sufield/ephemos/internal/adapters/primary/api"
)

func TestGRPCRegisterFunc_Register(t *testing.T) {
	t.Parallel()

	t.Run("successful registration", func(t *testing.T) {
		t.Parallel()
		called := false
		var fn api.GRPCRegisterFunc = func(s grpc.ServiceRegistrar) { 
			called = true 
		}
		
		fn.Register(&grpc.Server{})
		
		if !called {
			t.Fatal("expected registration to be called")
		}
	})

	t.Run("nil function - no panic", func(t *testing.T) {
		t.Parallel()
		var nilFn api.GRPCRegisterFunc
		
		// This should not panic
		nilFn.Register(&grpc.Server{})
	})

	t.Run("nil server - no panic", func(t *testing.T) {
		t.Parallel()
		called := false
		var fn api.GRPCRegisterFunc = func(s grpc.ServiceRegistrar) { 
			called = true 
		}
		
		// This should not panic or call the function
		fn.Register(nil)
		
		if called {
			t.Error("function should not be called with nil server")
		}
	})

	t.Run("works with grpc.ServiceRegistrar interface", func(t *testing.T) {
		t.Parallel()
		called := false
		var registrarPassed grpc.ServiceRegistrar
		
		var fn api.GRPCRegisterFunc = func(s grpc.ServiceRegistrar) { 
			called = true
			registrarPassed = s
		}
		
		server := &grpc.Server{}
		fn.Register(server)
		
		if !called {
			t.Fatal("expected registration to be called")
		}
		
		if registrarPassed != server {
			t.Error("expected the same server instance to be passed")
		}
	})
}

func TestNewGRPCServiceRegistrar(t *testing.T) {
	t.Parallel()

	t.Run("with valid function", func(t *testing.T) {
		t.Parallel()
		called := false
		fn := func(s grpc.ServiceRegistrar) { 
			called = true 
		}
		
		registrar := api.NewGRPCServiceRegistrar(fn)
		
		if registrar == nil {
			t.Fatal("expected non-nil registrar")
		}
		
		registrar.Register(&grpc.Server{})
		
		if !called {
			t.Error("expected function to be called")
		}
	})

	t.Run("with nil function returns no-op", func(t *testing.T) {
		t.Parallel()
		registrar := api.NewGRPCServiceRegistrar(nil)
		
		if registrar == nil {
			t.Fatal("expected non-nil registrar even with nil function")
		}
		
		// This should not panic - it should be a no-op
		registrar.Register(&grpc.Server{})
	})

	t.Run("no-op registrar is safe", func(t *testing.T) {
		t.Parallel()
		registrar := api.NewGRPCServiceRegistrar(nil)
		
		// Multiple calls should be safe
		registrar.Register(&grpc.Server{})
		registrar.Register(nil)
		registrar.Register(&grpc.Server{})
	})
}

func TestServiceRegistrar_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	t.Run("GRPCRegisterFunc implements ServiceRegistrar", func(t *testing.T) {
		t.Parallel()
		var _ api.ServiceRegistrar = api.GRPCRegisterFunc(func(grpc.ServiceRegistrar) {})
	})

	t.Run("NewGRPCServiceRegistrar returns ServiceRegistrar", func(t *testing.T) {
		t.Parallel()
		var _ api.ServiceRegistrar = api.NewGRPCServiceRegistrar(func(grpc.ServiceRegistrar) {})
	})

	t.Run("grpc.Server implements grpc.ServiceRegistrar", func(t *testing.T) {
		t.Parallel()
		var _ grpc.ServiceRegistrar = &grpc.Server{}
	})
}

// TestServiceRegistrar_RealUsage demonstrates how this would be used with actual gRPC services
func TestServiceRegistrar_RealUsage(t *testing.T) {
	t.Parallel()

	t.Run("typical usage pattern", func(t *testing.T) {
		t.Parallel()
		
		// Simulate how generated gRPC code would use this
		registerFunc := func(s grpc.ServiceRegistrar) {
			// In real code, this would be something like:
			// pb.RegisterMyServiceServer(s, &MyServiceImpl{})
			// For testing, we just verify the server is correct type
			if s == nil {
				t.Error("server should not be nil")
			}
		}
		
		registrar := api.NewGRPCServiceRegistrar(registerFunc)
		server := grpc.NewServer()
		
		// This should work without issues
		registrar.Register(server)
	})

	t.Run("works with wrapped servers", func(t *testing.T) {
		t.Parallel()
		
		// Create a simple wrapper that implements grpc.ServiceRegistrar
		type wrappedServer struct {
			*grpc.Server
			registerCalled bool
		}
		
		called := false
		registerFunc := func(s grpc.ServiceRegistrar) {
			called = true
			if ws, ok := s.(*wrappedServer); ok {
				ws.registerCalled = true
			}
		}
		
		registrar := api.NewGRPCServiceRegistrar(registerFunc)
		wrapper := &wrappedServer{Server: grpc.NewServer()}
		
		registrar.Register(wrapper)
		
		if !called {
			t.Error("register function should have been called")
		}
		
		if !wrapper.registerCalled {
			t.Error("wrapper should have been accessed")
		}
	})
}