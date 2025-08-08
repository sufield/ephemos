package api

import "google.golang.org/grpc"

// ServiceRegistrar is a generic interface for registering gRPC services
type ServiceRegistrar interface {
	Register(grpcServer *grpc.Server)
}