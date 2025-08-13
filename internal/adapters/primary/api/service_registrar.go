package api

import "google.golang.org/grpc"

// ServiceRegistrar is a generic interface for registering gRPC services.
type ServiceRegistrar interface {
	Register(grpcServer *grpc.Server)
}

// GRPCServiceRegistrar is a concrete implementation of ServiceRegistrar for gRPC services.
type GRPCServiceRegistrar struct {
	registerFunc func(*grpc.Server)
}

// NewGRPCServiceRegistrar creates a new gRPC service registrar with the provided registration function.
func NewGRPCServiceRegistrar(registerFunc func(*grpc.Server)) *GRPCServiceRegistrar {
	return &GRPCServiceRegistrar{
		registerFunc: registerFunc,
	}
}

// Register implements ServiceRegistrar by calling the stored registration function.
func (r *GRPCServiceRegistrar) Register(grpcServer *grpc.Server) {
	if r.registerFunc != nil {
		r.registerFunc(grpcServer)
	}
}
