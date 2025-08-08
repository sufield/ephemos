package proto

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
)

// Client provides a generic wrapper for any gRPC service client.
// This demonstrates how developers can create reusable client wrappers
// for their services when using the Ephemos library.
type Client[T any] struct {
	client T
	conn   *grpc.ClientConn
}

// NewClient creates a new Client for the given gRPC connection and client factory.
// The factory function should create a service-specific client from the connection.
//
// Example:
//	client, err := NewClient(conn, NewEchoServiceClient)
//	if err != nil {
//		return err
//	}
//	defer client.Close()
func NewClient[T any](conn *grpc.ClientConn, factory func(grpc.ClientConnInterface) T) (*Client[T], error) {
	if conn == nil {
		return nil, fmt.Errorf("gRPC connection cannot be nil")
	}
	if factory == nil {
		return nil, fmt.Errorf("client factory function cannot be nil")
	}
	
	return &Client[T]{
		client: factory(conn),
		conn:   conn,
	}, nil
}

// Close closes the underlying gRPC connection.
// Should be called when the client is no longer needed.
func (c *Client[T]) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Client returns the service-specific gRPC client.
// Use this to access the generated service methods.
func (c *Client[T]) Client() T {
	return c.client
}

// EchoClient is an example wrapper for the EchoServiceClient.
// This shows developers how to create service-specific clients
// that provide additional validation and convenience methods.
type EchoClient struct {
	*Client[EchoServiceClient]
}

// NewEchoClient creates an EchoClient using the generic Client wrapper.
// This demonstrates the recommended pattern for service-specific clients.
//
// Example:
//	echoClient, err := proto.NewEchoClient(conn)
//	if err != nil {
//		return err
//	}
//	defer echoClient.Close()
//	
//	resp, err := echoClient.Echo(ctx, "Hello, World!")
func NewEchoClient(conn *grpc.ClientConn) (*EchoClient, error) {
	client, err := NewClient(conn, NewEchoServiceClient)
	if err != nil {
		return nil, fmt.Errorf("failed to create echo client: %w", err)
	}
	
	return &EchoClient{Client: client}, nil
}

// Echo calls the Echo method on the EchoServiceClient with validation.
// The context supports cancellation and timeouts for the underlying gRPC call.
//
// Parameters:
//   - ctx: Context for cancellation, timeouts, and request metadata
//   - message: The message to echo back from the server
//
// Returns the server's echo response or an error if the call fails.
func (c *EchoClient) Echo(ctx context.Context, message string) (*EchoResponse, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	
	if c.Client == nil || c.client == nil {
		return nil, fmt.Errorf("echo client not properly initialized")
	}
	
	// Trim whitespace but allow empty messages for echo functionality
	message = strings.TrimSpace(message)
	
	req := &EchoRequest{Message: message}
	resp, err := c.client.Echo(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Echo service: %w", err)
	}
	
	return resp, nil
}