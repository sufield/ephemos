//go:build ignore
// Package main demonstrates the new transport-agnostic API for Ephemos.
// This example shows how to create a service that can run over gRPC or HTTP
// without changing any code - just configuration.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/sufield/ephemos/pkg/ephemos"
)

// EchoServiceImpl implements the domain EchoService interface.
// Note: This implementation has no knowledge of transport protocols.
// It works with plain Go types - no gRPC, no HTTP, no protobuf.
type EchoServiceImpl struct {
	name string
}

// Echo implements the echo functionality using plain Go types.
func (e *EchoServiceImpl) Echo(ctx context.Context, message string) (string, error) {
	return fmt.Sprintf("[%s] Echo: %s", e.name, message), nil
}

// Ping implements a simple health check.
func (e *EchoServiceImpl) Ping(ctx context.Context) error {
	log.Printf("[%s] Ping received", e.name)
	return nil
}

// FileServiceImpl demonstrates binary data handling.
type FileServiceImpl struct{}

func (f *FileServiceImpl) Upload(ctx context.Context, filename string, data io.Reader) error {
	log.Printf("Upload request for file: %s", filename)
	// In a real implementation, this would save the file
	return nil
}

func (f *FileServiceImpl) Download(ctx context.Context, filename string) (io.Reader, error) {
	log.Printf("Download request for file: %s", filename)
	// In a real implementation, this would return the file content
	return strings.NewReader(fmt.Sprintf("Content of %s", filename)), nil
}

func (f *FileServiceImpl) List(ctx context.Context, prefix string) ([]string, error) {
	log.Printf("List request with prefix: %s", prefix)
	// In a real implementation, this would list actual files
	return []string{
		fmt.Sprintf("%sfile1.txt", prefix),
		fmt.Sprintf("%sfile2.txt", prefix),
		fmt.Sprintf("%sfile3.txt", prefix),
	}, nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a transport-agnostic server.
	// The transport is determined by configuration, not code.
	configPath := os.Getenv("EPHEMOS_CONFIG")
	if configPath == "" {
		configPath = "config/transport-demo.yaml"
	}

	server, err := ephemos.NewTransportServer(ctx, configPath)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	// Mount services using the generic Mount[T] API.
	// The same code works for gRPC, HTTP, or any future transport.

	// Mount an echo service
	echoService := &EchoServiceImpl{name: "transport-demo"}
	if err := ephemos.Mount[ephemos.EchoService](server, echoService); err != nil {
		log.Fatalf("Failed to mount echo service: %v", err)
	}
	log.Println("✅ Mounted EchoService")

	// Mount a file service
	fileService := &FileServiceImpl{}
	if err := ephemos.Mount[ephemos.FileService](server, fileService); err != nil {
		log.Fatalf("Failed to mount file service: %v", err)
	}
	log.Println("✅ Mounted FileService")

	// Start the server.
	// This will use gRPC, HTTP, or another transport based on configuration.
	log.Printf("🚀 Starting transport-agnostic server...")

	go func() {
		if err := server.ListenAndServe(ctx); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	log.Println("✨ Server is running! Check your config to see which transport is active.")
	log.Println("   - gRPC: Test with grpc_cli or a gRPC client")
	log.Println("   - HTTP: Test with curl or any HTTP client")
	log.Println("   - Press Ctrl+C to stop")

	// Wait for shutdown signal
	<-ctx.Done()
	log.Println("🛑 Shutting down server...")
}
