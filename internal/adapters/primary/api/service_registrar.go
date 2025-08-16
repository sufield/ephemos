package api

import "google.golang.org/grpc"

// ServiceRegistrar aligns with gRPC's own registration interface.
// This allows working with real servers, wrapped servers, and test doubles.
type ServiceRegistrar interface {
	Register(s grpc.ServiceRegistrar)
}

// GRPCRegisterFunc is a function adapter implementing ServiceRegistrar.
// This provides a clean, idiomatic way to register services without extra structs.
type GRPCRegisterFunc func(grpc.ServiceRegistrar)

// Register calls the underlying function if both function and server are non-nil.
// This prevents silent no-ops and panics from nil servers.
func (f GRPCRegisterFunc) Register(s grpc.ServiceRegistrar) {
	if f == nil || s == nil {
		return
	}
	f(s)
}

// NewGRPCServiceRegistrar creates a ServiceRegistrar that enforces non-nil behavior.
// If fn is nil, returns a no-op registrar instead of allowing silent failures.
func NewGRPCServiceRegistrar(fn func(grpc.ServiceRegistrar)) ServiceRegistrar {
	if fn == nil {
		return GRPCRegisterFunc(func(grpc.ServiceRegistrar) {})
	}
	return GRPCRegisterFunc(fn)
}

// Compile-time check: *grpc.Server satisfies grpc.ServiceRegistrar.
var _ grpc.ServiceRegistrar = (*grpc.Server)(nil)
