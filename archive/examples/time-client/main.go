// Time Client Example - Backend developer using Ephemos library
//
// A client that requests current time in different timezones from time-server.
// Shows how Ephemos adds identity authentication to normal HTTP clients.
//
// BUSINESS LOGIC: Request time in different timezones
// SECURITY: Automatically presents SPIFFE certificate for authentication
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

type TimeRequest struct {
	Timezone string `json:"timezone"`
}

type TimeResponse struct {
	Timezone    string `json:"timezone"`
	CurrentTime string `json:"current_time"`
	Timestamp   int64  `json:"timestamp"`
	Service     string `json:"service"`
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	ctx := context.Background()

	// 1. Create identity-aware client - handles SPIFFE certificates automatically
	client, err := ephemos.NewIdentityClient(ctx, "")
	if err != nil {
		slog.Error("Failed to create identity client", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// 2. Create HTTP client with automatic authentication
	httpClient, err := client.NewHTTPClient("time-server", "localhost:8080")
	if err != nil {
		slog.Error("Failed to create HTTP client", "error", err)
		os.Exit(1)
	}

	slog.Info("🕐 Time client started - requesting times in different timezones")

	// 3. Request times in different timezones
	timezones := []string{
		"UTC",
		"America/New_York",
		"America/Los_Angeles", 
		"Europe/London",
		"Europe/Paris",
		"Asia/Tokyo",
		"Australia/Sydney",
	}

	for _, tz := range timezones {
		// Create request
		request := TimeRequest{
			Timezone: tz,
		}

		jsonData, err := json.Marshal(request)
		if err != nil {
			slog.Error("Failed to marshal request", "timezone", tz, "error", err)
			continue
		}

		// Make authenticated request - identity happens automatically
		resp, err := httpClient.Post(
			"http://localhost:8080/time",
			"application/json",
			bytes.NewReader(jsonData))

		if err != nil {
			slog.Error("Request failed", "timezone", tz, "error", err)
			continue
		}

		// Read response
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			slog.Error("Failed to read response", "timezone", tz, "error", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			slog.Error("Server error", "timezone", tz, "status", resp.Status, "body", string(body))
			continue
		}

		// Parse response
		var timeResp TimeResponse
		if err := json.Unmarshal(body, &timeResp); err != nil {
			slog.Error("Failed to parse response", "timezone", tz, "error", err)
			continue
		}

		// Display result
		slog.Info("⏰ Time received",
			"timezone", timeResp.Timezone,
			"time", timeResp.CurrentTime,
			"timestamp", timeResp.Timestamp)

		// Small delay between requests
		time.Sleep(300 * time.Millisecond)
	}

	// Check server health
	slog.Info("🏥 Checking server health...")
	resp, err := httpClient.Get("http://localhost:8080/health")
	if err != nil {
		slog.Error("Health check failed", "error", err)
	} else {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		if resp.StatusCode == http.StatusOK {
			slog.Info("✅ Server is healthy", "response", string(body))
		} else {
			slog.Warn("Server health issue", "status", resp.Status, "body", string(body))
		}
	}

	slog.Info("✅ All time requests completed successfully")
}