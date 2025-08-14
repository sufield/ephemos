package test

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/sufield/ephemos/examples/proto"
)

const bufSize = 1024 * 1024

var lis *bufconn.Listener

// TestEchoServer is a real implementation of EchoServiceServer for testing.
// It provides configurable behavior without using mocks.
type TestEchoServer struct {
	proto.UnimplementedEchoServiceServer
	callCount   int
	lastMessage string
}

func (s *TestEchoServer) Echo(ctx context.Context, req *proto.EchoRequest) (*proto.EchoResponse, error) {
	s.callCount++
	s.lastMessage = req.Message

	// Simulate different behaviors based on input
	if req.Message == "error" {
		return nil, status.Error(codes.Internal, "simulated server error")
	}
	if req.Message == "timeout" {
		time.Sleep(2 * time.Second)
		return nil, status.Error(codes.DeadlineExceeded, "simulated timeout")
	}

	return &proto.EchoResponse{
		Message: req.Message,
		From:    "test-echo-server",
	}, nil
}

func init() {
	lis = bufconn.Listen(bufSize)
	s := grpc.NewServer()
	proto.RegisterEchoServiceServer(s, &TestEchoServer{})
	go func() {
		if err := s.Serve(lis); err != nil {
			panic(err)
		}
	}()
}

func bufDialer(context.Context, string) (net.Conn, error) {
	return lis.Dial()
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		conn    *grpc.ClientConn
		factory func(grpc.ClientConnInterface) proto.EchoServiceClient
		wantErr bool
	}{
		{
			name:    "nil connection",
			conn:    nil,
			factory: proto.NewEchoServiceClient,
			wantErr: true,
		},
		{
			name:    "nil factory",
			conn:    &grpc.ClientConn{}, // This is just for testing - normally would be a real connection
			factory: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := proto.NewClient(tt.conn, tt.factory)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewEchoClient(t *testing.T) {
	tests := []struct {
		name    string
		conn    *grpc.ClientConn
		wantErr bool
	}{
		{
			name:    "nil connection",
			conn:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := proto.NewEchoClient(tt.conn)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEchoClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEchoClient_Echo(t *testing.T) {
	// Create a connection using the buffer
	ctx := context.Background()
	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	client, err := proto.NewEchoClient(conn)
	if err != nil {
		t.Fatalf("Failed to create echo client: %v", err)
	}
	defer client.Close()

	tests := []struct {
		name    string
		ctx     context.Context
		message string
		wantErr bool
	}{
		{
			name:    "nil context",
			ctx:     nil,
			message: "test",
			wantErr: true,
		},
		{
			name:    "successful echo",
			ctx:     ctx,
			message: "hello world",
			wantErr: false,
		},
		{
			name:    "empty message",
			ctx:     ctx,
			message: "",
			wantErr: false,
		},
		{
			name:    "whitespace message",
			ctx:     ctx,
			message: "   ",
			wantErr: false,
		},
		{
			name:    "server error",
			ctx:     ctx,
			message: "error",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := client.Echo(tt.ctx, tt.message)
			if (err != nil) != tt.wantErr {
				t.Errorf("Echo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if resp == nil {
					t.Error("Echo() returned nil response")
					return
				}
				expectedMessage := tt.message
				if tt.message == "   " {
					expectedMessage = "" // Trimmed whitespace
				}
				if resp.Message != expectedMessage {
					t.Errorf("Echo() response message = %v, want %v", resp.Message, expectedMessage)
				}
				if resp.From != "test-echo-server" {
					t.Errorf("Echo() response from = %v, want test-echo-server", resp.From)
				}
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	// Test with nil connection
	client := &proto.Client[proto.EchoServiceClient]{}
	if err := client.Close(); err != nil {
		t.Errorf("Close() with nil connection returned error: %v", err)
	}
}

func TestEchoClient_Integration(t *testing.T) {
	// Integration test that demonstrates the full flow
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(bufDialer), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	echoClient, err := proto.NewEchoClient(conn)
	if err != nil {
		t.Fatalf("Failed to create echo client: %v", err)
	}
	defer echoClient.Close()

	// Test multiple requests
	messages := []string{"Hello", "World", "Testing", "Echo", "Service"}
	for _, msg := range messages {
		resp, err := echoClient.Echo(ctx, msg)
		if err != nil {
			t.Errorf("Echo(%q) failed: %v", msg, err)
			continue
		}
		if resp.Message != msg {
			t.Errorf("Echo(%q) returned %q, expected %q", msg, resp.Message, msg)
		}
	}
}
