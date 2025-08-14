//go:build ignore
// Package main demonstrates the new struct tag-based validation system with defaults and aggregated error handling.
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/sufield/ephemos/pkg/ephemos"
)

func main() {
	fmt.Println("🚀 Ephemos Configuration Validation Examples")
	fmt.Println("====================================================")

	// Example 1: Basic configuration with automatic defaults
	fmt.Println("\n📋 Example 1: Configuration with automatic defaults")
	config1 := &ephemos.Configuration{}

	if err := config1.ValidateAndSetDefaults(); err != nil {
		log.Printf("❌ Validation failed: %v", err)
	} else {
		fmt.Printf("✅ Configuration validated successfully!\n")
		fmt.Printf("   Service Name: %s\n", config1.Service.Name)
		fmt.Printf("   Service Domain: %s\n", config1.Service.Domain)
		fmt.Printf("   Transport Type: %s\n", config1.Transport.Type)
		fmt.Printf("   Transport Address: %s\n", config1.Transport.Address)
		if config1.SPIFFE != nil {
			fmt.Printf("   SPIFFE Socket: %s\n", config1.SPIFFE.SocketPath)
		}
	}

	// Example 2: Configuration with validation errors (aggregated)
	fmt.Println("\n❌ Example 2: Configuration with multiple validation errors")
	config2 := &ephemos.Configuration{
		Service: ephemos.ServiceConfig{
			Name:   "invalid service name with spaces", // Invalid: contains spaces
			Domain: "invalid-domain",                   // Invalid: no dots
		},
		Transport: ephemos.TransportConfig{
			Type:    "invalid-transport", // Invalid: not grpc|http|tcp
			Address: "invalid-address",   // Invalid: doesn't match regex
		},
		SPIFFE: &ephemos.SPIFFEConfig{
			SocketPath: "relative/path", // Invalid: not absolute
		},
		AuthorizedClients: []string{
			"invalid-spiffe-id", // Invalid: doesn't start with spiffe://
		},
	}

	if err := config2.ValidateAndSetDefaults(); err != nil {
		fmt.Printf("❌ Configuration validation failed with multiple errors:\n")

		// Check if it's a validation error collection
		if validationErrors := ephemos.GetValidationErrors(err); len(validationErrors) > 0 {
			for i, validationErr := range validationErrors {
				fmt.Printf("   %d. Field '%s': %s\n", i+1, validationErr.Field, validationErr.Message)
			}
		} else {
			fmt.Printf("   Error: %v\n", err)
		}
	}

	// Example 3: Environment variable configuration
	fmt.Println("\n🌍 Example 3: Configuration from environment variables")

	// Set some environment variables
	os.Setenv("EPHEMOS_SERVICE_NAME", "env-service")
	os.Setenv("EPHEMOS_TRUST_DOMAIN", "example.org")
	os.Setenv("EPHEMOS_AUTHORIZED_CLIENTS", "spiffe://example.org/client1,spiffe://example.org/client2")

	defer func() {
		os.Unsetenv("EPHEMOS_SERVICE_NAME")
		os.Unsetenv("EPHEMOS_TRUST_DOMAIN")
		os.Unsetenv("EPHEMOS_AUTHORIZED_CLIENTS")
	}()

	if envConfig, err := ephemos.LoadFromEnvironment(); err != nil {
		log.Printf("❌ Environment configuration failed: %v", err)
	} else {
		fmt.Printf("✅ Environment configuration loaded successfully!\n")
		fmt.Printf("   Service Name: %s\n", envConfig.Service.Name)
		fmt.Printf("   Service Domain: %s\n", envConfig.Service.Domain)
		fmt.Printf("   Authorized Clients: %v\n", envConfig.AuthorizedClients)
	}

	// Example 4: Custom validation engine configuration
	fmt.Println("\n⚙️  Example 4: Custom validation engine (fail-fast mode)")

	failFastEngine := ephemos.NewValidationEngine()
	failFastEngine.StopOnFirstError = true // Enable fail-fast mode

	config4 := &ephemos.Configuration{
		Service: ephemos.ServiceConfig{
			Name:   "",               // Invalid: required field missing
			Domain: "invalid-domain", // Also invalid, but won't be reported in fail-fast mode
		},
	}

	if err := ephemos.ValidateStructWithEngine(config4, failFastEngine); err != nil {
		fmt.Printf("❌ Fail-fast validation stopped at first error:\n")
		fmt.Printf("   Error: %v\n", err)

		// Check if it's a validation error collection
		if validationErrors := ephemos.GetValidationErrors(err); len(validationErrors) > 0 {
			fmt.Printf("   Total errors found: %d (stopped early)\n", len(validationErrors))
		}
	}

	// Example 5: Demonstrating default value types
	fmt.Println("\n🔧 Example 5: Various default value types")

	type ExampleConfig struct {
		StringField   string   `default:"default-string"`
		IntField      int      `default:"42"`
		BoolField     bool     `default:"true"`
		SliceField    []string `default:"item1,item2,item3"`
		RequiredField string   `validate:"required"`
		OptionalField string   `validate:"min=5"`
	}

	exampleConfig := &ExampleConfig{
		RequiredField: "present", // Satisfy required validation
	}

	if err := ephemos.ValidateStruct(exampleConfig); err != nil {
		log.Printf("❌ Example config validation failed: %v", err)
	} else {
		fmt.Printf("✅ Example config with defaults:\n")
		fmt.Printf("   StringField: %s\n", exampleConfig.StringField)
		fmt.Printf("   IntField: %d\n", exampleConfig.IntField)
		fmt.Printf("   BoolField: %t\n", exampleConfig.BoolField)
		fmt.Printf("   SliceField: %v\n", exampleConfig.SliceField)
		fmt.Printf("   RequiredField: %s\n", exampleConfig.RequiredField)
		fmt.Printf("   OptionalField: %s\n", exampleConfig.OptionalField)
	}

	// Example 6: Validation rule demonstrations
	fmt.Println("\n🔍 Example 6: Validation rule demonstrations")

	type ValidationExampleConfig struct {
		ServiceName    string `validate:"required,min=3,max=50,regex=^[a-zA-Z0-9_-]+$"`
		Port           string `validate:"port"`
		IPAddress      string `validate:"ip"`
		Domain         string `validate:"domain"`
		SPIFFEIdentity string `validate:"spiffe_id"`
		Duration       string `validate:"duration"`
		FilePath       string `validate:"abs_path"`
		TransportType  string `validate:"oneof=grpc|http|tcp"`
	}

	validationExample := &ValidationExampleConfig{
		ServiceName:    "valid-service-123",
		Port:           "8080",
		IPAddress:      "192.168.1.1",
		Domain:         "example.org",
		SPIFFEIdentity: "spiffe://example.org/service",
		Duration:       "30s",
		FilePath:       "/absolute/path/to/file",
		TransportType:  "grpc",
	}

	if err := ephemos.ValidateStruct(validationExample); err != nil {
		log.Printf("❌ Validation example failed: %v", err)
	} else {
		fmt.Printf("✅ All validation rules passed!\n")
		fmt.Printf("   Service Name: %s (✓ required, min/max length, regex pattern)\n", validationExample.ServiceName)
		fmt.Printf("   Port: %s (✓ valid port number)\n", validationExample.Port)
		fmt.Printf("   IP Address: %s (✓ valid IP format)\n", validationExample.IPAddress)
		fmt.Printf("   Domain: %s (✓ valid domain format)\n", validationExample.Domain)
		fmt.Printf("   SPIFFE ID: %s (✓ valid SPIFFE format)\n", validationExample.SPIFFEIdentity)
		fmt.Printf("   Duration: %s (✓ valid duration format)\n", validationExample.Duration)
		fmt.Printf("   File Path: %s (✓ absolute path)\n", validationExample.FilePath)
		fmt.Printf("   Transport: %s (✓ one of allowed values)\n", validationExample.TransportType)
	}

	fmt.Println("\n🎉 All examples completed!")
	fmt.Println("\n📚 Key benefits of the new validation system:")
	fmt.Println("   • ✅ Struct tag-based validation (declarative)")
	fmt.Println("   • ✅ Automatic default value setting")
	fmt.Println("   • ✅ Aggregated error reporting (see all issues at once)")
	fmt.Println("   • ✅ Fail-fast mode option")
	fmt.Println("   • ✅ Comprehensive validation rules (regex, domain, SPIFFE ID, etc.)")
	fmt.Println("   • ✅ Nested struct validation")
	fmt.Println("   • ✅ Type-safe with reflection-based implementation")
}
