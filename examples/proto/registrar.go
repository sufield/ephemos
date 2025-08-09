package proto

import (
	"google.golang.org/grpc"
)

// EchoServiceRegistrar is an example registrar for the Echo service.
// This demonstrates how developers should implement service registration
// when using the Ephemos library for identity-based authentication.
//
// Developers should create similar registrars for their own services
// by implementing the ServiceRegistrar interface.
type EchoServiceRegistrar struct {
	server EchoServiceServer
}

// NewEchoServiceRegistrar creates a registrar for the Echo service.
// The server parameter should implement the EchoServiceServer interface
// with the actual business logic for the service.
//
// Example:
//
//	server := &MyEchoServer{} // implements EchoServiceServer
//	registrar := proto.NewEchoServiceRegistrar(server)
//	err := ephemosServer.RegisterService(ctx, registrar)
func NewEchoServiceRegistrar(server EchoServiceServer) *EchoServiceRegistrar {
	return &EchoServiceRegistrar{
		server: server,
	}
}

// Register registers the Echo service with the provided gRPC server.
// This method is called by the Ephemos framework when setting up
// the identity-aware gRPC server.
//
// This implements the ServiceRegistrar interface required by Ephemos.
func (r *EchoServiceRegistrar) Register(grpcServer *grpc.Server) {
	RegisterEchoServiceServer(grpcServer, r.server)
}
