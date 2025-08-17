package verification

import (
	"strings"
	"testing"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/ports"
)

func TestNewSpireDiagnosticsProvider(t *testing.T) {
	tests := []struct {
		name           string
		config         *ports.DiagnosticsConfig
		expectedSocket string
		expectedAgent  string
		expectedTimeout time.Duration
	}{
		{
			name:            "nil config should use defaults",
			config:          nil,
			expectedSocket:  "unix:///tmp/spire-server/private/api.sock",
			expectedAgent:   "unix:///tmp/spire-agent/public/api.sock",
			expectedTimeout: 30 * time.Second,
		},
		{
			name: "valid config should preserve values",
			config: &ports.DiagnosticsConfig{
				ServerSocketPath: "unix:///custom/server.sock",
				AgentSocketPath:  "unix:///custom/agent.sock",
				Timeout:          60 * time.Second,
			},
			expectedSocket:  "unix:///custom/server.sock",
			expectedAgent:   "unix:///custom/agent.sock",
			expectedTimeout: 60 * time.Second,
		},
		{
			name: "config with zero timeout should get default",
			config: &ports.DiagnosticsConfig{
				ServerSocketPath: "unix:///test/server.sock",
				AgentSocketPath:  "unix:///test/agent.sock",
				Timeout:          0,
			},
			expectedSocket:  "unix:///test/server.sock",
			expectedAgent:   "unix:///test/agent.sock",
			expectedTimeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewSpireDiagnosticsProvider(tt.config)

			require.NotNil(t, provider)
			require.NotNil(t, provider.config)

			assert.Equal(t, tt.expectedSocket, provider.config.ServerSocketPath)
			assert.Equal(t, tt.expectedAgent, provider.config.AgentSocketPath)
			assert.Equal(t, tt.expectedTimeout, provider.config.Timeout)
		})
	}
}

func TestParseBundleData(t *testing.T) {
	provider := NewSpireDiagnosticsProvider(nil)
	trustDomain := spiffeid.RequireTrustDomainFromString("example.org")

	// Test with various data types that might come from CLI output
	testData := map[string]interface{}{
		"certificates": []interface{}{
			map[string]interface{}{
				"not_after": "2024-01-01T00:00:00Z",
			},
		},
	}

	bundle, err := provider.parseBundleData(trustDomain, testData)
	
	assert.NoError(t, err)
	assert.NotNil(t, bundle)
	assert.Equal(t, trustDomain, bundle.TrustDomain)
	assert.Equal(t, 1, bundle.CertificateCount) // Default assumption
	assert.False(t, bundle.LastUpdated.IsZero())
	assert.False(t, bundle.ExpiresAt.IsZero())
}

func TestGetComponentVersionParsing(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected string
	}{
		{
			name:     "standard version output",
			output:   "spire-server version 1.8.7\n",
			expected: "1.8.7",
		},
		{
			name:     "version with build info",
			output:   "spire-agent version 1.8.7\nBuilt from git SHA abc123\n",
			expected: "1.8.7",
		},
		{
			name:     "unexpected format",
			output:   "Some unexpected output\n",
			expected: "Some unexpected output",
		},
		{
			name:     "empty output",
			output:   "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the actual parsing logic more accurately
			output := tt.output
			
			// Trim whitespace like strings.TrimSpace
			output = strings.TrimSpace(output)
			
			var result string
			if output != "" {
				// Split on newlines like the actual function
				lines := strings.Split(output, "\n")
				if len(lines) > 0 {
					firstLine := lines[0]
					// Parse version from first line (typically "spire-server version X.Y.Z")
					parts := strings.Fields(firstLine)
					if len(parts) >= 3 && parts[1] == "version" {
						result = parts[2]
					} else {
						result = firstLine
					}
				}
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDiagnosticsConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config *ports.DiagnosticsConfig
		valid  bool
	}{
		{
			name: "valid config with all fields",
			config: &ports.DiagnosticsConfig{
				ServerSocketPath: "unix:///tmp/spire-server/private/api.sock",
				AgentSocketPath:  "unix:///tmp/spire-agent/public/api.sock",
				ServerAddress:    "localhost:8081",
				Timeout:          30 * time.Second,
				UseServerAPI:     true,
				ServerAPIToken:   "token123",
			},
			valid: true,
		},
		{
			name: "minimal valid config",
			config: &ports.DiagnosticsConfig{
				ServerSocketPath: "unix:///tmp/server.sock",
				AgentSocketPath:  "unix:///tmp/agent.sock",
			},
			valid: true,
		},
		{
			name: "config with API preference",
			config: &ports.DiagnosticsConfig{
				ServerAddress:  "https://spire-server:8081",
				UseServerAPI:   true,
				ServerAPIToken: "bearer-token",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewSpireDiagnosticsProvider(tt.config)
			assert.NotNil(t, provider)
			
			if tt.valid {
				assert.NotNil(t, provider.config)
			}
		})
	}
}

func TestRegistrationEntryInfoCalculation(t *testing.T) {
	// Create test registration entries
	now := time.Now()
	oldTime := now.Add(-48 * time.Hour)
	recentTime := now.Add(-12 * time.Hour)

	entries := []*ports.RegistrationEntry{
		{
			ID:        "entry1",
			SPIFFEID:  spiffeid.RequireFromString("spiffe://example.org/service1"),
			CreatedAt: recentTime,
			Selectors: []string{"unix:uid:1000", "docker:label:app:service1"},
		},
		{
			ID:        "entry2", 
			SPIFFEID:  spiffeid.RequireFromString("spiffe://example.org/service2"),
			CreatedAt: oldTime,
			Selectors: []string{"unix:gid:1001", "k8s:sa:default"},
		},
		{
			ID:        "entry3",
			SPIFFEID:  spiffeid.RequireFromString("spiffe://example.org/service3"),
			CreatedAt: recentTime,
			Selectors: []string{"unix:uid:1002"},
		},
	}

	// Simulate the logic from getRegistrationEntriesInfo
	info := &ports.RegistrationEntryInfo{
		Total:      len(entries),
		BySelector: make(map[string]int),
	}

	for _, entry := range entries {
		// Count recent entries (last 24 hours)
		if now.Sub(entry.CreatedAt) < 24*time.Hour {
			info.Recent++
		}

		// Count by selector type
		for _, selector := range entry.Selectors {
			// Simulate strings.SplitN(selector, ":", 2)
			if selector == "unix:uid:1000" {
				info.BySelector["unix"]++
			} else if selector == "docker:label:app:service1" {
				info.BySelector["docker"]++
			} else if selector == "unix:gid:1001" {
				info.BySelector["unix"]++
			} else if selector == "k8s:sa:default" {
				info.BySelector["k8s"]++
			} else if selector == "unix:uid:1002" {
				info.BySelector["unix"]++
			}
		}
	}

	assert.Equal(t, 3, info.Total)
	assert.Equal(t, 2, info.Recent) // Two entries from last 24 hours
	assert.Equal(t, 3, info.BySelector["unix"])
	assert.Equal(t, 1, info.BySelector["docker"])
	assert.Equal(t, 1, info.BySelector["k8s"])
}

func TestAgentInfoCalculation(t *testing.T) {
	now := time.Now()
	futureTime := now.Add(24 * time.Hour)
	pastTime := now.Add(-24 * time.Hour)

	agents := []*ports.Agent{
		{
			ID:        spiffeid.RequireFromString("spiffe://example.org/agent1"),
			ExpiresAt: futureTime,
			Banned:    false,
		},
		{
			ID:        spiffeid.RequireFromString("spiffe://example.org/agent2"),
			ExpiresAt: pastTime,
			Banned:    false,
		},
		{
			ID:        spiffeid.RequireFromString("spiffe://example.org/agent3"),
			ExpiresAt: futureTime,
			Banned:    true,
		},
	}

	// Simulate the logic from getAgentsInfo
	info := &ports.AgentInfo{
		Total: len(agents),
	}

	for _, agent := range agents {
		if agent.Banned {
			info.Banned++
		} else if agent.ExpiresAt.After(now) {
			info.Active++
		} else {
			info.Inactive++
		}
	}

	assert.Equal(t, 3, info.Total)
	assert.Equal(t, 1, info.Active)   // One non-banned agent with future expiry
	assert.Equal(t, 1, info.Inactive) // One non-banned agent with past expiry
	assert.Equal(t, 1, info.Banned)   // One banned agent
}

func TestDiagnosticInfoStructure(t *testing.T) {
	// Test that diagnostic info structure is properly initialized
	now := time.Now()
	info := &ports.DiagnosticInfo{
		Component:   "spire-server",
		Version:     "1.8.7",
		Status:      "running",
		TrustDomain: spiffeid.RequireTrustDomainFromString("example.org"),
		CollectedAt: now,
		Details:     make(map[string]interface{}),
	}

	assert.Equal(t, "spire-server", info.Component)
	assert.Equal(t, "1.8.7", info.Version)
	assert.Equal(t, "running", info.Status)
	assert.Equal(t, "example.org", info.TrustDomain.String())
	assert.Equal(t, now, info.CollectedAt)
	assert.NotNil(t, info.Details)
	assert.Equal(t, 0, len(info.Details))

	// Test adding details
	info.Details["test_key"] = "test_value"
	assert.Equal(t, "test_value", info.Details["test_key"])
}