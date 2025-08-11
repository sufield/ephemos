// Package main demonstrates advanced gRPC connection management features in ephemos.
package main

import (
	"fmt"
	"time"

	"github.com/sufield/ephemos/internal/adapters/secondary/spiffe"
	"github.com/sufield/ephemos/internal/adapters/secondary/transport"
)

func main() {
	fmt.Println("Enhanced gRPC Connection Management Example")
	fmt.Println("==========================================")

	// Example 1: Default Production Configuration
	fmt.Println("\n1. Default Production Configuration:")
	defaultConfig := transport.DefaultConnectionConfig()
	printConnectionConfig("Default", defaultConfig)

	// Example 2: Development Configuration (faster timeouts, fewer retries)
	fmt.Println("\n2. Development Configuration:")
	devConfig := transport.DevelopmentConnectionConfig()
	printConnectionConfig("Development", devConfig)

	// Example 3: High-Throughput Configuration (optimized for performance)
	fmt.Println("\n3. High-Throughput Configuration:")
	highThroughputConfig := transport.HighThroughputConnectionConfig()
	printConnectionConfig("High-Throughput", highThroughputConfig)

	// Example 4: Custom Configuration
	fmt.Println("\n4. Custom Configuration Example:")
	customConfig := createCustomConfig()
	printConnectionConfig("Custom", customConfig)

	// Example 5: Using the enhanced provider
	fmt.Println("\n5. Creating Enhanced gRPC Provider:")
	demonstrateProviderUsage()

	fmt.Println("\nDemonstration completed!")
}

func printConnectionConfig(name string, config *transport.ConnectionConfig) {
	fmt.Printf("  %s Configuration:\n", name)
	fmt.Printf("    - Connect Timeout: %v\n", config.ConnectTimeout)
	fmt.Printf("    - Backoff Base Delay: %v\n", config.BackoffConfig.BaseDelay)
	fmt.Printf("    - Backoff Max Delay: %v\n", config.BackoffConfig.MaxDelay)
	fmt.Printf("    - Backoff Multiplier: %.2f\n", config.BackoffConfig.Multiplier)
	fmt.Printf("    - Backoff Jitter: %.2f\n", config.BackoffConfig.Jitter)
	fmt.Printf("    - Keepalive Time: %v\n", config.KeepaliveParams.Time)
	fmt.Printf("    - Keepalive Timeout: %v\n", config.KeepaliveParams.Timeout)
	fmt.Printf("    - Keepalive Without Stream: %v\n", config.KeepaliveParams.PermitWithoutStream)
	fmt.Printf("    - Idle Timeout: %v\n", config.IdleTimeout)
	fmt.Printf("    - Max Recv Message Size: %d MB\n", config.MaxRecvMsgSize/(1024*1024))
	fmt.Printf("    - Max Send Message Size: %d MB\n", config.MaxSendMsgSize/(1024*1024))
	fmt.Printf("    - Connection Pooling: %v\n", config.EnablePooling)
	if config.EnablePooling {
		fmt.Printf("    - Pool Size: %d\n", config.PoolSize)
	}
	fmt.Printf("    - Service Config Contains Retry Policy: %v\n", 
		len(config.ServiceConfig) > 0 && contains(config.ServiceConfig, "retryPolicy"))
}

func createCustomConfig() *transport.ConnectionConfig {
	// Create a custom configuration for a specific use case:
	// - Microservice with moderate traffic
	// - Needs resilience but not extreme performance
	// - Deployed in a reliable network environment

	config := transport.DefaultConnectionConfig()
	
	// Adjust timeouts for moderate traffic scenario
	config.ConnectTimeout = 15 * time.Second
	
	// Configure more aggressive backoff for faster recovery
	config.BackoffConfig.BaseDelay = 500 * time.Millisecond
	config.BackoffConfig.Multiplier = 2.0
	config.BackoffConfig.MaxDelay = 30 * time.Second
	
	// Adjust keepalive for moderate traffic
	config.KeepaliveParams.Time = 20 * time.Second
	config.KeepaliveParams.Timeout = 5 * time.Second
	
	// Enable pooling with moderate pool size
	config.EnablePooling = true
	config.PoolSize = 3
	
	// Set reasonable message sizes (8MB)
	config.MaxRecvMsgSize = 8 * 1024 * 1024
	config.MaxSendMsgSize = 8 * 1024 * 1024
	
	// Custom service configuration with moderate retry policy
	config.ServiceConfig = `{
		"methodConfig": [
			{
				"name": [{"service": ""}],
				"retryPolicy": {
					"maxAttempts": 4,
					"initialBackoff": "1s",
					"maxBackoff": "15s",
					"backoffMultiplier": 1.8,
					"retryableStatusCodes": ["UNAVAILABLE", "DEADLINE_EXCEEDED"]
				},
				"timeout": "30s"
			}
		]
	}`
	
	return config
}

func demonstrateProviderUsage() {
	// Create a mock SPIFFE provider (in real usage, this would be properly initialized)
	spiffeProvider := &spiffe.Provider{}
	
	// Example 1: Default provider
	fmt.Println("  Creating default provider...")
	_ = transport.NewGRPCProvider(spiffeProvider)
	fmt.Println("    ✓ Default provider created with standard configuration")
	
	// Example 2: Development provider
	fmt.Println("  Creating development provider...")
	devConfig := transport.DevelopmentConnectionConfig()
	_ = transport.NewGRPCProviderWithConfig(spiffeProvider, devConfig)
	fmt.Println("    ✓ Development provider created with fast timeouts")
	
	// Example 3: High-throughput provider
	fmt.Println("  Creating high-throughput provider...")
	htConfig := transport.HighThroughputConnectionConfig()
	_ = transport.NewGRPCProviderWithConfig(spiffeProvider, htConfig)
	fmt.Println("    ✓ High-throughput provider created with connection pooling")
	
	// Example 4: Custom provider
	fmt.Println("  Creating custom provider...")
	customConfig := createCustomConfig()
	_ = transport.NewGRPCProviderWithConfig(spiffeProvider, customConfig)
	fmt.Println("    ✓ Custom provider created with tailored configuration")
	
	// Demonstrate configuration differences
	defaultConfig := transport.DefaultConnectionConfig()
	fmt.Println("\n  Configuration Comparison:")
	fmt.Printf("    Default connect timeout: %v\n", defaultConfig.ConnectTimeout)
	fmt.Printf("    Development connect timeout: %v\n", devConfig.ConnectTimeout)
	fmt.Printf("    High-throughput pooling: %v\n", htConfig.EnablePooling)
	fmt.Printf("    Custom pool size: %d\n", customConfig.PoolSize)
}

// Note: In a real implementation, you would access configuration through provider methods
// This example demonstrates the configuration concepts and usage patterns

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		fmt.Sprintf("%s", s)[0:0] != fmt.Sprintf("%s", substr)[0:0]))) // simplified contains check
}

// Additional examples of specific use cases

func demonstrateUseCase(name string, config *transport.ConnectionConfig, description string) {
	fmt.Printf("\n=== %s ===\n", name)
	fmt.Println(description)
	fmt.Println()
	
	// Show key configuration aspects for this use case
	fmt.Printf("Key Configuration:\n")
	fmt.Printf("- Connection Timeout: %v\n", config.ConnectTimeout) 
	fmt.Printf("- Retry Max Attempts: Configured in service config\n")
	fmt.Printf("- Connection Pooling: %v\n", config.EnablePooling)
	if config.EnablePooling {
		fmt.Printf("- Pool Size: %d connections\n", config.PoolSize)
	}
	fmt.Printf("- Max Message Size: %d MB\n", config.MaxRecvMsgSize/(1024*1024))
}

func init() {
	// Additional use case examples
	go func() {
		time.Sleep(100 * time.Millisecond)
		
		// E-commerce example
		ecommerceConfig := transport.HighThroughputConnectionConfig()
		ecommerceConfig.ConnectTimeout = 5 * time.Second // Fast connection required
		demonstrateUseCase("E-commerce Platform",
			ecommerceConfig,
			"High-volume e-commerce platform requiring fast connections and high throughput for order processing and inventory updates.")
		
		// Financial services example  
		financeConfig := transport.DefaultConnectionConfig()
		financeConfig.BackoffConfig.MaxDelay = 10 * time.Second // Conservative backoff
		financeConfig.IdleTimeout = 5 * time.Minute // Shorter idle timeout for security
		demonstrateUseCase("Financial Services",
			financeConfig, 
			"Financial trading system requiring reliable connections with conservative retry policies and strict timeout controls.")
			
		// IoT example
		iotConfig := transport.DevelopmentConnectionConfig()
		iotConfig.KeepaliveParams.Time = 60 * time.Second // Less frequent keepalives
		iotConfig.MaxRecvMsgSize = 1024 * 1024 // Smaller messages
		iotConfig.MaxSendMsgSize = 1024 * 1024
		demonstrateUseCase("IoT Data Collection",
			iotConfig,
			"IoT sensor data collection system with many small messages and battery-conscious keepalive settings.")
	}()
}