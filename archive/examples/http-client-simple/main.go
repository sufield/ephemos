//go:build ignore
// HTTP Client Example - Lightweight HTTP over mTLS with X.509 SVIDs  
//
// Simple abstraction over go-spiffe HTTP example:
// https://github.com/spiffe/go-spiffe/tree/main/examples/spiffe-http
//
// Shows how to create an HTTP client with automatic SPIFFE identity authentication
package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"

	"github.com/sufield/ephemos/pkg/ephemos"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx := context.Background()

	// Create HTTP client with automatic mTLS using X.509 SVIDs
	client, err := ephemos.NewHTTPClient(ctx, "localhost:8080")
	if err != nil {
		slog.Error("Failed to create client", "error", err)
		os.Exit(1)
	}

	// Use like normal HTTP client - mTLS happens automatically
	resp, err := client.Get("http://localhost:8080/status")
	if err != nil {
		slog.Error("Request failed", "error", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read response", "error", err)
		os.Exit(1)
	}

	slog.Info("Response received", 
		"status", resp.Status,
		"body", string(body))
}