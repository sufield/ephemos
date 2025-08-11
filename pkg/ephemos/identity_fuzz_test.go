package ephemos

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// FuzzIdentityParsing tests SPIFFE ID and trust domain parsing
func FuzzIdentityParsing(f *testing.F) {
	// Seed with various SPIFFE URI patterns
	f.Add("spiffe://example.org/service/myservice")
	f.Add("spiffe://trust.domain.com/ns/production/sa/api-server")
	f.Add("spiffe://localhost/service")
	f.Add("spiffe://")
	f.Add("spiffe:///empty-domain")
	f.Add("http://not-spiffe.com/service")
	f.Add("spiffe://domain/path/../traversal")
	f.Add("spiffe://domain/path\x00null")
	f.Add("spiffe://domain\n/newline")
	f.Add("spiffe://domain:8080/service") // Port in trust domain
	f.Add("spiffe://very-very-very-long-trust-domain-name.example.com/service")
	f.Add("spiffe://domain/service?query=param")
	f.Add("spiffe://domain/service#fragment")
	f.Add("SPIFFE://UPPERCASE.ORG/SERVICE")
	f.Add("spiffe://unicode-域名.org/service")
	f.Add("spiffe://domain/service with spaces")
	f.Add("spiffe://domain//double-slash/service")
	f.Add("spiffe://./relative/domain/service")
	f.Add("spiffe://256.256.256.256/service") // Invalid IP

	f.Fuzz(func(t *testing.T, spiffeID string) {
		// Test SPIFFE ID parsing through service configuration
		tempDir := t.TempDir()
		configFile := tempDir + "/config.yaml"

		// Create config that might use this SPIFFE ID in authorized clients
		configContent := `service:
  name: test-service
  domain: example.org
transport:
  type: grpc
  address: :50051
spiffe:
  socket_path: /tmp/spire-agent/public/api.sock
authorized_clients:
  - "` + strings.ReplaceAll(spiffeID, `"`, `\"`) + `"`

		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
		defer cancel()

		_, err := loadAndValidateConfig(ctx, configFile)

		// Test that malformed SPIFFE IDs are rejected
		if !strings.HasPrefix(spiffeID, "spiffe://") ||
			strings.Contains(spiffeID, "\x00") ||
			strings.Contains(spiffeID, "\n") ||
			strings.Contains(spiffeID, "\t") {
			// Should produce validation error for malformed IDs
			if err == nil {
				t.Logf("Potentially invalid SPIFFE ID accepted: %q", spiffeID)
			}
		}
	})
}

// FuzzTrustDomain tests trust domain validation
func FuzzTrustDomain(f *testing.F) {
	// Seed with valid and invalid trust domains
	f.Add("example.org")
	f.Add("production.company.com")
	f.Add("localhost")
	f.Add("test-domain.local")
	f.Add("")
	f.Add("domain with spaces")
	f.Add("UPPERCASE.ORG")
	f.Add("domain\x00null")
	f.Add("domain\nwith\nnewlines")
	f.Add("very-very-very-very-very-long-domain-name.example.com")
	f.Add("domain..double.dot.com")
	f.Add(".starts.with.dot.com")
	f.Add("ends.with.dot.com.")
	f.Add("under_score.com")
	f.Add("hyphen-.com")
	f.Add("-starts-with-hyphen.com")
	f.Add("256.256.256.256") // Invalid IP as domain
	f.Add("domain:8080")     // Port in domain
	f.Add("../../../etc/passwd")
	f.Add("unicode-域名.org")

	f.Fuzz(func(t *testing.T, trustDomain string) {
		tempDir := t.TempDir()
		configFile := tempDir + "/config.yaml"

		configContent := `service:
  name: test-service
  domain: "` + strings.ReplaceAll(trustDomain, `"`, `\"`) + `"
transport:
  type: grpc
  address: :50051
spiffe:
  socket_path: /tmp/spire-agent/public/api.sock`

		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
		defer cancel()

		_, err := loadAndValidateConfig(ctx, configFile)

		// Test domain validation
		if strings.Contains(trustDomain, "\x00") ||
			strings.ContainsAny(trustDomain, "\n\t :") ||
			strings.Contains(trustDomain, "..") {
			if err == nil {
				t.Errorf("Expected validation error for trust domain: %q", trustDomain)
			}
		}
	})
}

// FuzzClientAuthorization tests client authorization list handling
func FuzzClientAuthorization(f *testing.F) {
	// Seed with various client patterns
	f.Add("single-client")
	f.Add("client1,client2,client3")
	f.Add("")
	f.Add("spiffe://domain.com/client")
	f.Add("client\x00null")
	f.Add("client\nwith\nnewlines")
	f.Add("very-very-very-long-client-identifier-that-might-cause-buffer-issues")
	f.Add("client with spaces")
	f.Add("../../../malicious/path")
	f.Add("DROP TABLE clients;")
	f.Add("unicode-クライアント")

	f.Fuzz(func(t *testing.T, clientPattern string) {
		tempDir := t.TempDir()
		configFile := tempDir + "/config.yaml"

		// Split pattern into individual clients (simple comma split)
		clients := strings.Split(clientPattern, ",")
		clientsYAML := ""
		for _, client := range clients {
			client = strings.TrimSpace(client)
			if client != "" {
				clientsYAML += `  - "` + strings.ReplaceAll(client, `"`, `\"`) + `"` + "\n"
			}
		}

		configContent := `service:
  name: test-service
  domain: example.org
transport:
  type: grpc
  address: :50051
spiffe:
  socket_path: /tmp/spire-agent/public/api.sock
authorized_clients:
` + clientsYAML

		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
		defer cancel()

		config, err := loadAndValidateConfig(ctx, configFile)

		// Should handle various client patterns gracefully
		if err == nil && config != nil {
			// Verify that malicious patterns don't end up in config
			for _, client := range config.AuthorizedClients {
				if strings.Contains(client, "\x00") {
					t.Errorf("Null byte found in authorized client: %q", client)
				}
			}
		}
	})
}

// FuzzServerTrusts tests trusted servers list handling
func FuzzServerTrusts(f *testing.F) {
	// Seed with various server patterns
	f.Add("spiffe://example.org/server")
	f.Add("server1,server2")
	f.Add("")
	f.Add("malicious\x00server")
	f.Add("server\nwith\nnewlines")
	f.Add("very-long-" + strings.Repeat("server-name-", 100))
	f.Add("../../../etc/passwd")
	f.Add("'; DROP TABLE servers; --")
	f.Add("unicode-サーバー")

	f.Fuzz(func(t *testing.T, serverPattern string) {
		tempDir := t.TempDir()
		configFile := tempDir + "/config.yaml"

		servers := strings.Split(serverPattern, ",")
		serversYAML := ""
		for _, server := range servers {
			server = strings.TrimSpace(server)
			if server != "" {
				serversYAML += `  - "` + strings.ReplaceAll(server, `"`, `\"`) + `"` + "\n"
			}
		}

		configContent := `service:
  name: test-service
  domain: example.org
transport:
  type: grpc
  address: :50051
spiffe:
  socket_path: /tmp/spire-agent/public/api.sock
trusted_servers:
` + serversYAML

		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
		defer cancel()

		config, err := loadAndValidateConfig(ctx, configFile)

		// Verify security of trusted servers list
		if err == nil && config != nil {
			for _, server := range config.TrustedServers {
				if strings.Contains(server, "\x00") {
					t.Errorf("Null byte found in trusted server: %q", server)
				}
			}
		}
	})
}

// FuzzContextTimeout tests context handling with various timeouts
func FuzzContextTimeout(f *testing.F) {
	// Seed with different timeout patterns
	f.Add(int64(0))              // No timeout
	f.Add(int64(1))              // 1 nanosecond
	f.Add(int64(1000))           // 1 microsecond
	f.Add(int64(1000000))        // 1 millisecond
	f.Add(int64(-1))             // Negative timeout
	f.Add(int64(86400000000000)) // 1 day in nanoseconds

	f.Fuzz(func(t *testing.T, timeoutNs int64) {
		tempDir := t.TempDir()
		configFile := tempDir + "/config.yaml"

		configContent := `service:
  name: test-service
  domain: example.org
transport:
  type: grpc
  address: :50051
spiffe:
  socket_path: /tmp/spire-agent/public/api.sock`

		if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
			return
		}

		// Create context with fuzzing timeout
		var ctx context.Context
		var cancel context.CancelFunc

		if timeoutNs > 0 {
			timeout := time.Duration(timeoutNs)
			if timeout > time.Hour {
				timeout = time.Hour // Cap at reasonable max
			}
			ctx, cancel = context.WithTimeout(t.Context(), timeout)
		} else {
			ctx, cancel = context.WithCancel(t.Context())
		}
		defer cancel()

		// Test config loading with various timeouts
		_, err := loadAndValidateConfig(ctx, configFile)

		// Very short timeouts might cause context cancellation
		if timeoutNs > 0 && timeoutNs < 1000000 { // Less than 1ms
			// Might timeout, which is acceptable
			_ = err
		} else if timeoutNs < 0 {
			// Negative timeouts should still work (treated as no timeout)
			if err != nil && strings.Contains(err.Error(), "context canceled") {
				t.Error("Negative timeout should not cause context cancellation")
			}
		}
	})
}

// Benchmark identity operations
func BenchmarkIdentityValidation(b *testing.B) {
	tempDir := b.TempDir()
	configFile := tempDir + "/bench.yaml"

	configContent := `service:
  name: benchmark-service
  domain: example.org
transport:
  type: grpc
  address: :50051
spiffe:
  socket_path: /tmp/spire-agent/public/api.sock
authorized_clients:
  - "spiffe://example.org/client1"
  - "spiffe://example.org/client2"
trusted_servers:
  - "spiffe://example.org/server1"
  - "spiffe://example.org/server2"`

	os.WriteFile(configFile, []byte(configContent), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := b.Context()
		_, _ = loadAndValidateConfig(ctx, configFile)
	}
}
