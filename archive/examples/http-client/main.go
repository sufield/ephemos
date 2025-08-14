//go:build ignore
// HTTP Client Example - Backend developer using Ephemos library
//
// This shows how a backend developer creates a client with identity-based authentication.
// The client automatically presents SPIFFE certificates when connecting to servers.
//
// SIMPLE PATTERN:
// 1. Create Ephemos client 
// 2. Use it like normal HTTP client
// 3. Identity authentication happens automatically
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/sufield/ephemos/pkg/ephemos"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx := context.Background()

	// 1. Create identity-aware client - automatically handles SPIFFE certificates
	client, err := ephemos.NewIdentityClient(ctx, "")
	if err != nil {
		slog.Error("Failed to create identity client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// 2. Create HTTP client that uses identity authentication
	httpClient, err := client.NewHTTPClient("my-service", "localhost:8080")
	if err != nil {
		slog.Error("Failed to create HTTP client", "error", err)
		os.Exit(1)
	}

	// 3. Use like normal HTTP client - identity authentication is transparent
	slog.Info("Making authenticated requests...")

	// GET request
	resp, err := httpClient.Get("http://localhost:8080/api/status")
	if err != nil {
		slog.Error("GET request failed", "error", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Failed to read response", "error", err)
		os.Exit(1)
	}

	slog.Info("GET response received", 
		"status", resp.Status,
		"body", string(body))

	// POST requests
	for i := 0; i < 3; i++ {
		requestData := map[string]interface{}{
			"message":   fmt.Sprintf("Request %d", i+1),
			"timestamp": time.Now().Unix(),
		}

		jsonData, _ := json.Marshal(requestData)

		resp, err := httpClient.Post(
			"http://localhost:8080/api/data", 
			"application/json", 
			bytes.NewReader(jsonData))
		
		if err != nil {
			slog.Error("POST request failed", "request", i+1, "error", err)
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		slog.Info("POST response received", 
			"request", i+1,
			"status", resp.Status, 
			"body", string(body))

		time.Sleep(500 * time.Millisecond)
	}

	// Health check
	resp, err = httpClient.Get("http://localhost:8080/health")
	if err != nil {
		slog.Error("Health check failed", "error", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		slog.Info("Health check", "status", resp.Status, "body", string(body))
	}

	slog.Info("All requests completed successfully")
}