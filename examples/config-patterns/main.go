// Package main demonstrates different configuration patterns with Ephemos.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sufield/ephemos/pkg/ephemos"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== Ephemos Configuration Patterns Demo ===")
	fmt.Println()

	// Pattern 1: Traditional YAML with Environment Overrides (Default)
	demonstrateYAMLWithEnvOverrides(ctx)

	// Pattern 2: Environment Variables Only
	demonstrateEnvironmentOnly(ctx)

	// Pattern 3: Pure Code Configuration
	demonstratePureCode(ctx)

	// Pattern 4: Flexible Configuration API
	demonstrateFlexibleAPI(ctx)
}

func demonstrateYAMLWithEnvOverrides(ctx context.Context) {
	fmt.Println("1. YAML Configuration with Environment Overrides")
	fmt.Println("   Default pattern: YAML file + env var overrides")
	fmt.Println()

	// Set an environment override
	os.Setenv("EPHEMOS_SERVICE_NAME", "yaml-with-env-override")
	defer os.Unsetenv("EPHEMOS_SERVICE_NAME")

	config, err := ephemos.NewConfigBuilder().
		WithSource(ephemos.ConfigSourceYAML).
		WithYAMLFile("config/echo-server.yaml"). // Falls back to default if file doesn't exist
		Build(ctx)

	if err != nil {
		log.Printf("   ⚠️  Error (expected if no YAML file): %v", err)
		return
	}

	fmt.Printf("   ✅ Service Name: %s (from env override)\n", config.Service.Name)
	fmt.Printf("   ✅ Transport: %s on %s\n", config.Transport.Type, config.Transport.Address)
	fmt.Println()
}

func demonstrateEnvironmentOnly(ctx context.Context) {
	fmt.Println("2. Environment Variables Only")
	fmt.Println("   Pattern: No YAML files, pure environment configuration")
	fmt.Println()

	// Set environment variables
	envVars := map[string]string{
		"EPHEMOS_SERVICE_NAME":       "env-only-service",
		"EPHEMOS_SERVICE_DOMAIN":     "env.example.com",
		"EPHEMOS_TRANSPORT_TYPE":     "http",
		"EPHEMOS_TRANSPORT_ADDRESS":  ":8080",
		"EPHEMOS_AUTHORIZED_CLIENTS": "client-a, client-b, client-c",
	}

	for key, value := range envVars {
		os.Setenv(key, value)
	}
	defer func() {
		for key := range envVars {
			os.Unsetenv(key)
		}
	}()

	config, err := ephemos.NewConfigBuilder().
		WithSource(ephemos.ConfigSourceEnvOnly).
		WithEnvPrefix("EPHEMOS").
		Build(ctx)

	if err != nil {
		log.Printf("   ❌ Error: %v", err)
		return
	}

	fmt.Printf("   ✅ Service: %s.%s\n", config.Service.Name, config.Service.Domain)
	fmt.Printf("   ✅ Transport: %s on %s\n", config.Transport.Type, config.Transport.Address)
	fmt.Printf("   ✅ Authorized Clients: %v\n", config.AuthorizedClients)
	fmt.Println()
}

func demonstratePureCode(ctx context.Context) {
	fmt.Println("3. Pure Code Configuration")
	fmt.Println("   Pattern: All configuration in code, no external files or env vars")
	fmt.Println()

	config, err := ephemos.NewConfigBuilder().
		WithSource(ephemos.ConfigSourcePureCode).
		WithServiceName("code-configured-service").
		WithServiceDomain("code.example.com").
		WithSPIFFESocket("/tmp/spire-agent/public/api.sock").
		WithTransport("grpc", ":50052").
		WithAuthorizedClients([]string{"microservice-a", "microservice-b"}).
		WithTrustedServers([]string{"auth-service", "data-service"}).
		Build(ctx)

	if err != nil {
		log.Printf("   ❌ Error: %v", err)
		return
	}

	fmt.Printf("   ✅ Service: %s.%s\n", config.Service.Name, config.Service.Domain)
	fmt.Printf("   ✅ Transport: %s on %s\n", config.Transport.Type, config.Transport.Address)
	fmt.Printf("   ✅ SPIFFE Socket: %s\n", config.SPIFFE.SocketPath)
	fmt.Printf("   ✅ Authorized Clients: %v\n", config.AuthorizedClients)
	fmt.Printf("   ✅ Trusted Servers: %v\n", config.TrustedServers)
	fmt.Println()
}

func demonstrateFlexibleAPI(ctx context.Context) {
	fmt.Println("4. Flexible Configuration API")
	fmt.Println("   Pattern: Functional options for clean, composable configuration")
	fmt.Println()

	// Example: Microservice with environment-specific overrides
	config, err := ephemos.LoadConfigFlexible(ctx,
		ephemos.WithPureCodeSource(),
		ephemos.WithService("api-gateway", "prod.company.com"),
		ephemos.WithTransportOption("grpc", ":443"),
	)

	if err != nil {
		log.Printf("   ❌ Error: %v", err)
		return
	}

	fmt.Printf("   ✅ Flexible Service: %s.%s\n", config.Service.Name, config.Service.Domain)
	fmt.Printf("   ✅ Flexible Transport: %s on %s\n", config.Transport.Type, config.Transport.Address)
	fmt.Println()

	// Example: Development environment with custom env prefix
	os.Setenv("DEV_SERVICE_NAME", "dev-api-gateway")
	defer os.Unsetenv("DEV_SERVICE_NAME")

	devConfig, err := ephemos.LoadConfigFlexible(ctx,
		ephemos.WithEnvSource("DEV"),
		ephemos.WithService("default-service", "dev.company.com"),
	)

	if err != nil {
		log.Printf("   ❌ Dev Config Error: %v", err)
		return
	}

	fmt.Printf("   ✅ Dev Service: %s (env override working)\n", devConfig.Service.Name)
	fmt.Println()
}

// Example of how this would be used in a real application
func realWorldExample(ctx context.Context) (*ephemos.ManagedIdentityServer, error) {
	// Production: YAML + env overrides for sensitive data
	config, err := ephemos.LoadConfigFlexible(ctx,
		ephemos.WithYAMLSource("config/production.yaml"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load production config: %w", err)
	}

	// In a real implementation, we would pass the config to the server
	// For now, we'll simulate server creation (config validation already passed)
	if config != nil {
		// Create server with production configuration
		server, err := ephemos.NewManagedIdentityServer(ctx, &ephemos.ServerOptions{
			ConfigPath: "", // We already have the config
			// Alternative: pass config directly if API supports it
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create server: %w", err)
		}

		return server, nil
	}

	return nil, fmt.Errorf("config is nil")
}

// Example for testing environments
func testingExample(ctx context.Context) (*ephemos.ManagedIdentityServer, error) {
	// Testing: Pure code configuration for predictable tests
	config, err := ephemos.LoadConfigFlexible(ctx,
		ephemos.WithPureCodeSource(),
		ephemos.WithService("test-service", "test.local"),
		ephemos.WithTransportOption("grpc", ":0"), // Random port for tests
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load test config: %w", err)
	}

	// In a real implementation, we'd need to pass this config to the server
	if config != nil {
		// Server would use this config
		return nil, nil
	}

	return nil, nil
}
