package ephemos

import (
	"context"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
)

// FuzzServiceName tests service name validation with random inputs.
func FuzzServiceName(f *testing.F) {
	// Seed with valid and invalid service names
	f.Add("valid-service")
	f.Add("service_name")
	f.Add("service123")
	f.Add("UPPERCASE")
	f.Add("")
	f.Add("with spaces")
	f.Add("with-special@chars")
	f.Add("service\x00null")
	f.Add("service\nnewline")
	f.Add("service\ttab")
	f.Add("very-very-very-very-very-long-service-name-that-might-cause-issues")
	f.Add("unicode-service-名前")
	f.Add("service/with/slashes")
	f.Add("service\\with\\backslashes")
	f.Add("../../../etc/passwd")
	f.Add("DROP TABLE services;")

	f.Fuzz(func(t *testing.T, serviceName string) {
		// Test with temporary config to avoid real SPIRE dependency
		tempDir := t.TempDir()
		configFile := tempDir + "/config.yaml"

		// Create a config with the fuzzing service name
		configContent := `service:
  name: "` + strings.ReplaceAll(serviceName, `"`, `\"`) + `"
  domain: example.org
transport:
  type: grpc
  address: :0
spiffe:
  socket_path: /nonexistent/socket`

		if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
			return
		}

		// Test configuration loading - should handle invalid names gracefully
		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
		defer cancel()

		_, err := loadAndValidateConfig(ctx, configFile)

		// Validation should catch invalid service names (empty names get defaults, so only truly invalid patterns should fail)
		if strings.ContainsAny(serviceName, "\x00\n\t@/\\") {
			if err == nil {
				t.Errorf("Expected validation error for service name: %q", serviceName)
			}
		}
	})
}

// FuzzTransportAddress tests transport address parsing with random inputs.
func FuzzTransportAddress(f *testing.F) {
	// Seed with valid and invalid addresses
	f.Add(":50051")
	f.Add("localhost:8080")
	f.Add("0.0.0.0:9999")
	f.Add("[::1]:8080")
	f.Add("")
	f.Add("invalid")
	f.Add(":99999")               // Port too high
	f.Add(":-1")                  // Negative port
	f.Add(":0")                   // Port 0 (should be valid for auto-assign)
	f.Add("host:")                // Missing port
	f.Add(":port")                // Non-numeric port
	f.Add("host:port:extra")      // Too many colons
	f.Add("256.256.256.256:8080") // Invalid IP
	f.Add("[invalid::ipv6]:8080")
	f.Add("localhost\x00:8080") // Null byte
	f.Add("very-very-very-very-very-very-long-hostname-that-might-cause-buffer-overflow.example.com:8080")

	f.Fuzz(func(t *testing.T, address string) {
		// Test address validation through config
		tempDir := t.TempDir()
		configFile := tempDir + "/config.yaml"

		configContent := `service:
  name: test-service
  domain: example.org
transport:
  type: grpc
  address: "` + strings.ReplaceAll(address, `"`, `\"`) + `"
spiffe:
  socket_path: /nonexistent/socket`

		if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
		defer cancel()

		config, err := loadAndValidateConfig(ctx, configFile)

		// Test that invalid addresses are caught (but not empty addresses, which get defaults)
		if strings.Contains(address, "\x00") {
			if err == nil {
				t.Errorf("Expected validation error for address with null byte: %q", address)
			}
		}

		// If validation passes, test actual network parsing
		if err == nil && config != nil {
			// Try to parse the address - this should not panic
			_, parseErr := net.ResolveTCPAddr("tcp", config.Transport.Address)
			_ = parseErr // We expect some addresses to fail parsing
		}
	})
}

// FuzzSPIFFESocketPath tests SPIFFE socket path validation.
func FuzzSPIFFESocketPath(f *testing.F) {
	// Seed with valid and invalid socket paths
	f.Add("/tmp/spire-agent/public/api.sock")
	f.Add("/var/run/spire/socket")
	f.Add("./relative/path/socket")
	f.Add("relative-socket")
	f.Add("")
	f.Add("/")
	f.Add("/tmp")
	f.Add("/dev/null")
	f.Add("/etc/passwd")
	f.Add("socket\x00path")
	f.Add("socket\npath")
	f.Add("../../../etc/passwd")
	f.Add("/tmp/" + strings.Repeat("a", 1000) + "/socket") // Very long path
	f.Add("\\Windows\\Path\\socket")                       // Windows-style path on Unix

	f.Fuzz(func(t *testing.T, socketPath string) {
		tempDir := t.TempDir()
		configFile := tempDir + "/config.yaml"

		configContent := `service:
  name: test-service
  domain: example.org
transport:
  type: grpc
  address: :50051
spiffe:
  socketPath: "` + strings.ReplaceAll(socketPath, `"`, `\"`) + `"`

		if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
		defer cancel()

		_, err := loadAndValidateConfig(ctx, configFile)

		// Test validation of socket paths (empty paths are allowed and get defaults)
		if (socketPath != "" && !strings.HasPrefix(socketPath, "/")) || strings.Contains(socketPath, "\x00") {
			if err == nil {
				t.Errorf("Expected validation error for socket path: %q", socketPath)
			}
		}
	})
}

// FuzzTransportType tests transport type validation.
func FuzzTransportType(f *testing.F) {
	// Seed with valid and invalid transport types
	f.Add("grpc")
	f.Add("http")
	f.Add("GRPC")
	f.Add("HTTP")
	f.Add("")
	f.Add("invalid")
	f.Add("tcp")
	f.Add("udp")
	f.Add("websocket")
	f.Add("grpc\x00")
	f.Add("grpc\n")
	f.Add("type with spaces")
	f.Add("very-long-transport-type-name")

	f.Fuzz(func(t *testing.T, transportType string) {
		tempDir := t.TempDir()
		configFile := tempDir + "/config.yaml"

		configContent := `service:
  name: test-service
  domain: example.org
transport:
  type: "` + strings.ReplaceAll(transportType, `"`, `\"`) + `"
  address: :50051
spiffe:
  socket_path: /tmp/spire-agent/public/api.sock`

		if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
			return
		}

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Second)
		defer cancel()

		_, err := loadAndValidateConfig(ctx, configFile)

		// Only "grpc", "http", and "tcp" should be valid per struct tag (empty gets default)
		// Case sensitive validation
		validTypes := map[string]bool{
			"grpc": true,
			"http": true,
			"tcp":  true,
		}

		if transportType != "" && !validTypes[transportType] {
			if err == nil {
				t.Errorf("Expected validation error for transport type: %q", transportType)
			}
		}
	})
}

// FuzzGenericServiceRegistrar tests service registrar with random function inputs.
func FuzzGenericServiceRegistrar(f *testing.F) {
	f.Add(true)  // Valid function
	f.Add(false) // Nil function

	f.Fuzz(func(t *testing.T, hasValidFunc bool) {
		var registerFunc func(*grpc.Server)

		if hasValidFunc {
			registerFunc = func(_ *grpc.Server) {
				// Mock registration - do nothing but don't panic
			}
		}
		// else registerFunc remains nil

		// Test registrar creation - should not panic
		registrar := NewServiceRegistrar(registerFunc)

		if registrar == nil {
			t.Error("NewServiceRegistrar should not return nil")
		}

		// Test registration - should handle nil function gracefully
		mockServer := grpc.NewServer()
		defer mockServer.Stop()

		// This should not panic even with nil function
		if genericRegistrar, ok := registrar.(*GenericServiceRegistrar); ok {
			genericRegistrar.Register(mockServer)
		}
	})
}

// Benchmark to ensure fuzzing doesn't introduce performance regressions.
func BenchmarkConfigValidation(b *testing.B) {
	tempDir := b.TempDir()
	configFile := tempDir + "/bench.yaml"

	configContent := `service:
  name: benchmark-service
  domain: example.org
transport:
  type: grpc
  address: :50051
spiffe:
  socket_path: /tmp/spire-agent/public/api.sock`

	os.WriteFile(configFile, []byte(configContent), 0o644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := b.Context()
		_, _ = loadAndValidateConfig(ctx, configFile)
	}
}
