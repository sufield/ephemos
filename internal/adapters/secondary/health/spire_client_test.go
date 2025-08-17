package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sufield/ephemos/internal/core/ports"
)

func TestNewSpireHealthClient(t *testing.T) {
	tests := []struct {
		name      string
		component string
		config    *ports.HealthConfig
		wantErr   bool
	}{
		{
			name:      "valid server config",
			component: "spire-server",
			config: &ports.HealthConfig{
				Timeout: 5 * time.Second,
				Server: &ports.SpireServerHealthConfig{
					Address: "localhost:8080",
				},
			},
			wantErr: false,
		},
		{
			name:      "valid agent config",
			component: "spire-agent",
			config: &ports.HealthConfig{
				Timeout: 5 * time.Second,
				Agent: &ports.SpireAgentHealthConfig{
					Address: "localhost:8081",
				},
			},
			wantErr: false,
		},
		{
			name:      "nil config",
			component: "spire-server",
			config:    nil,
			wantErr:   true,
		},
		{
			name:      "empty component name",
			component: "",
			config: &ports.HealthConfig{
				Timeout: 5 * time.Second,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewSpireHealthClient(tt.component, tt.config)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.component, client.GetComponentName())
			}
		})
	}
}

func TestSpireHealthClient_CheckLiveness(t *testing.T) {
	tests := []struct {
		name           string
		component      string
		serverResponse int
		responseBody   string
		expectStatus   ports.HealthStatus
		expectError    bool
	}{
		{
			name:           "server healthy",
			component:      "spire-server",
			serverResponse: http.StatusOK,
			responseBody:   "alive",
			expectStatus:   ports.HealthStatusHealthy,
			expectError:    false,
		},
		{
			name:           "server unhealthy",
			component:      "spire-server",
			serverResponse: http.StatusServiceUnavailable,
			responseBody:   "not ready",
			expectStatus:   ports.HealthStatusUnhealthy,
			expectError:    false,
		},
		{
			name:           "server unexpected status",
			component:      "spire-server",
			serverResponse: http.StatusInternalServerError,
			responseBody:   "error",
			expectStatus:   ports.HealthStatusUnknown,
			expectError:    false,
		},
		{
			name:           "agent healthy",
			component:      "spire-agent",
			serverResponse: http.StatusOK,
			responseBody:   "alive",
			expectStatus:   ports.HealthStatusHealthy,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/live", r.URL.Path)
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "ephemos-health-checker/1.0", r.Header.Get("User-Agent"))
				
				w.WriteHeader(tt.serverResponse)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			// Create client
			config := &ports.HealthConfig{
				Timeout: 5 * time.Second,
			}

			switch tt.component {
			case "spire-server":
				config.Server = &ports.SpireServerHealthConfig{
					Address:  server.URL[7:], // Remove "http://" prefix
					LivePath: "/live",
				}
			case "spire-agent":
				config.Agent = &ports.SpireAgentHealthConfig{
					Address:  server.URL[7:], // Remove "http://" prefix
					LivePath: "/live",
				}
			}

			client, err := NewSpireHealthClient(tt.component, config)
			require.NoError(t, err)

			// Perform liveness check
			ctx := context.Background()
			result, err := client.CheckLiveness(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectStatus, result.Status)
				assert.Equal(t, tt.component, result.Component)
				assert.NotZero(t, result.CheckedAt)
				assert.NotNil(t, result.Details)
			}
		})
	}
}

func TestSpireHealthClient_CheckReadiness(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ready", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	}))
	defer server.Close()

	config := &ports.HealthConfig{
		Timeout: 5 * time.Second,
		Server: &ports.SpireServerHealthConfig{
			Address:   server.URL[7:], // Remove "http://" prefix
			ReadyPath: "/ready",
		},
	}

	client, err := NewSpireHealthClient("spire-server", config)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := client.CheckReadiness(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ports.HealthStatusHealthy, result.Status)
	assert.Equal(t, "spire-server", result.Component)
}

func TestSpireHealthClient_CheckHealth(t *testing.T) {
	liveCallCount := 0
	readyCallCount := 0

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/live":
			liveCallCount++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("alive"))
		case "/ready":
			readyCallCount++
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &ports.HealthConfig{
		Timeout: 5 * time.Second,
		Server: &ports.SpireServerHealthConfig{
			Address:   server.URL[7:], // Remove "http://" prefix
			LivePath:  "/live",
			ReadyPath: "/ready",
		},
	}

	client, err := NewSpireHealthClient("spire-server", config)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := client.CheckHealth(ctx)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ports.HealthStatusHealthy, result.Status)
	assert.Equal(t, "spire-server", result.Component)
	assert.Contains(t, result.Message, "healthy and ready")
	
	// Verify both endpoints were called
	assert.Equal(t, 1, liveCallCount)
	assert.Equal(t, 1, readyCallCount)
	
	// Check details
	assert.Contains(t, result.Details, "liveness_status")
	assert.Contains(t, result.Details, "readiness_status")
	assert.Equal(t, "healthy", result.Details["liveness_status"])
	assert.Equal(t, "healthy", result.Details["readiness_status"])
}

func TestSpireHealthClient_UpdateConfig(t *testing.T) {
	config := &ports.HealthConfig{
		Timeout: 5 * time.Second,
		Server: &ports.SpireServerHealthConfig{
			Address: "localhost:8080",
		},
	}

	client, err := NewSpireHealthClient("spire-server", config)
	require.NoError(t, err)

	// Update config
	newConfig := &ports.HealthConfig{
		Timeout: 10 * time.Second,
		Server: &ports.SpireServerHealthConfig{
			Address: "localhost:9090",
		},
	}

	err = client.UpdateConfig(newConfig)
	assert.NoError(t, err)

	// Verify timeout was updated
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
}

func TestSpireHealthClient_InvalidComponent(t *testing.T) {
	config := &ports.HealthConfig{
		Timeout: 5 * time.Second,
	}

	client, err := NewSpireHealthClient("invalid-component", config)
	require.NoError(t, err)

	ctx := context.Background()
	
	// Should fail because no server or agent config
	_, err = client.CheckLiveness(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported component type")
}

func TestSpireHealthClient_Timeout(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &ports.HealthConfig{
		Timeout: 50 * time.Millisecond, // Shorter than server delay
		Server: &ports.SpireServerHealthConfig{
			Address:  server.URL[7:],
			LivePath: "/live",
		},
	}

	client, err := NewSpireHealthClient("spire-server", config)
	require.NoError(t, err)

	ctx := context.Background()
	result, err := client.CheckLiveness(ctx)

	// Should get timeout error
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, ports.HealthStatusUnhealthy, result.Status)
}

func TestSpireHealthClient_CustomPaths(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/custom/health/live":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("custom alive"))
		case "/custom/health/ready":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("custom ready"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &ports.HealthConfig{
		Timeout: 5 * time.Second,
		Server: &ports.SpireServerHealthConfig{
			Address:   server.URL[7:],
			LivePath:  "/custom/health/live",
			ReadyPath: "/custom/health/ready",
		},
	}

	client, err := NewSpireHealthClient("spire-server", config)
	require.NoError(t, err)

	ctx := context.Background()
	
	// Test custom liveness path
	result, err := client.CheckLiveness(ctx)
	assert.NoError(t, err)
	assert.Equal(t, ports.HealthStatusHealthy, result.Status)
	
	// Test custom readiness path
	result, err = client.CheckReadiness(ctx)
	assert.NoError(t, err)
	assert.Equal(t, ports.HealthStatusHealthy, result.Status)
}

func TestSpireHealthClient_HTTPS(t *testing.T) {
	// Create HTTPS test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("secure"))
	}))
	defer server.Close()

	config := &ports.HealthConfig{
		Timeout: 5 * time.Second,
		Server: &ports.SpireServerHealthConfig{
			Address:  server.URL[8:], // Remove "https://" prefix
			LivePath: "/live",
			UseHTTPS: true,
		},
	}

	client, err := NewSpireHealthClient("spire-server", config)
	require.NoError(t, err)

	// Note: This test will fail with certificate errors in real scenarios
	// but demonstrates the HTTPS URL construction
	ctx := context.Background()
	result, err := client.CheckLiveness(ctx)

	// We expect an error due to self-signed certificate, but the URL should be correct
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.Details["url"], "https://")
}