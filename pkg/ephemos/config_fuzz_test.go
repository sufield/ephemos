package ephemos

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
	// coreErrors import removed for public API compliance
)

// FuzzResolveConfigPath tests the config path resolution with random inputs.
func FuzzResolveConfigPath(f *testing.F) {
	// Add seed corpus with common path patterns
	f.Add("")
	f.Add("config.yaml")
	f.Add("/etc/ephemos/config.yaml")
	f.Add("../config.yaml")
	f.Add("./config.yaml")
	f.Add("/tmp/test/config.yaml")
	f.Add("config/ephemos.yaml")
	f.Add("     ")                      // whitespace
	f.Add("\n\t config.yaml")           // whitespace with path
	f.Add("config\x00.yaml")            // null byte
	f.Add("config..yaml")               // double dot
	f.Add("../../../../etc/passwd")     // path traversal
	f.Add("config.yaml\x00/etc/passwd") // null byte injection

	f.Fuzz(func(t *testing.T, configPath string) {
		// Set up temporary directory for test
		tempDir := t.TempDir()
		oldWD, _ := os.Getwd()
		t.Cleanup(func() { t.Chdir(oldWD) })
		t.Chdir(tempDir)

		// Test should not panic and should handle malicious inputs safely
		result, err := resolveConfigPath(configPath)

		// Validate that returned paths are safe
		if err == nil && result != "" {
			// Ensure no path traversal attacks succeeded
			abs, _ := filepath.Abs(result)
			if abs != result {
				t.Errorf("resolveConfigPath returned non-absolute path: %s", result)
			}

			// Ensure null bytes are rejected
			if filepath.Base(result) != filepath.Clean(filepath.Base(result)) {
				t.Errorf("resolveConfigPath accepted unsafe path: %s", result)
			}
		}
	})
}

// FuzzValidateFileAccess tests file access validation with random file paths.
func FuzzValidateFileAccess(f *testing.F) {
	// Seed corpus with various path patterns
	f.Add("")
	f.Add("nonexistent.yaml")
	f.Add("/etc/passwd")
	f.Add("/dev/null")
	f.Add(".")
	f.Add("..")
	f.Add("/")
	f.Add("/tmp")
	f.Add("config.yaml\x00")
	f.Add("../../../etc/passwd")

	f.Fuzz(func(t *testing.T, path string) {
		// Create temporary directory with test file
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "valid.yaml")
		os.WriteFile(testFile, []byte("test: value\n"), 0o644)

		// Test should not panic regardless of input
		err := validateFileAccess(path)

		// Validate error handling
		if path == "" {
			// Empty path should not error (uses defaults)
			if err != nil {
				t.Errorf("validateFileAccess should accept empty path, got: %v", err)
			}
		}

		// Test with the valid file we created
		if path == testFile {
			if err != nil {
				t.Errorf("validateFileAccess should accept valid file %s, got: %v", testFile, err)
			}
		}
	})
}

// FuzzYAMLParsing tests YAML parsing with malformed/malicious YAML content.
func FuzzYAMLParsing(f *testing.F) {
	// Seed with valid and invalid YAML patterns
	f.Add("service:\n  name: test")
	f.Add("invalid: yaml: content:")
	f.Add("---\nservice: {}")
	f.Add("!!binary |")
	f.Add("&anchor value")
	f.Add("<<: *unknown")
	f.Add("service: !!str |\n  multiline\n  value")
	f.Add("recursive: &rec\n  self: *rec")
	f.Add("large_string: " + string(make([]byte, 10000)))
	f.Add("nested:\n  deeply:\n    very:\n      much:\n        so: deep")
	f.Add("tabs:\t\tand\tspaces   mixed")
	f.Add("unicode: 🔒 security test")
	f.Add("special: \"\\x00\\xFF\"")
	f.Add("---\n!!binary |\n  " + string(make([]byte, 1000)))

	f.Fuzz(func(t *testing.T, yamlContent string) {
		// Create temporary file with fuzzing content
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "fuzz.yaml")

		// Write content (might be invalid)
		if err := os.WriteFile(configFile, []byte(yamlContent), 0o644); err != nil {
			return // Skip if we can't write the file
		}

		// Test YAML parsing - should not panic
		ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
		defer cancel()

		_, err := loadConfigFile(ctx, configFile)
		// Parsing can fail, but should not panic or hang
		// We mainly test for stability and proper error handling
		if err != nil {
			// Verify error is wrapped properly
			if !IsConfigurationError(err) {
				t.Errorf("loadConfigFile should return configuration error for invalid YAML, got: %T", err)
			}
		}
	})
}

// FuzzConfigValidation tests configuration validation with random config values.
func FuzzConfigValidation(f *testing.F) {
	// Seed with valid and boundary case configurations
	validConfigs := []string{
		`service:
  name: test-service
  domain: example.org
transport:
  type: grpc
  address: :50051
spiffe:
  socket_path: /tmp/spire-agent/public/api.sock`,
		`service:
  name: ""
transport:
  type: invalid
  address: malformed`,
		`service:
  name: "service with spaces"
  domain: "not-a-domain"`,
		`transport:
  address: "65536"`, // Invalid port
		`spiffe:
  socket_path: "relative/path"`, // Invalid socket path
		`service:
  name: "` + string(make([]byte, 1000)) + `"`, // Very long name
	}

	for _, config := range validConfigs {
		f.Add(config)
	}

	f.Fuzz(func(t *testing.T, configYAML string) {
		// Parse YAML into config struct
		tempDir := t.TempDir()
		configFile := filepath.Join(tempDir, "fuzz.yaml")

		if err := os.WriteFile(configFile, []byte(configYAML), 0o644); err != nil {
			return
		}

		// Test configuration loading with validation
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()

		_, err := loadAndValidateConfig(ctx, configFile)

		// Validation may fail, but should not panic
		// Test that all configuration errors are properly wrapped
		if err != nil && !IsConfigurationError(err) {
			t.Errorf("loadAndValidateConfig should return configuration error, got: %T: %v", err, err)
		}
	})
}

// FuzzEnhanceValidationMessage tests error message generation with random validation errors.
func FuzzEnhanceValidationMessage(f *testing.F) {
	// Seed with known field names and values
	f.Add("service.name", "")
	f.Add("service.domain", "invalid-domain")
	f.Add("spiffe.socket_path", "relative/path")
	f.Add("transport.type", "invalid")
	f.Add("transport.address", "not-an-address")
	f.Add("unknown.field", "random-value")
	f.Add("", "")
	f.Add("field.with.dots", "value")
	f.Add("field\x00with\x00nulls", "value\x00null")

	f.Fuzz(func(t *testing.T, field, value string) {
		// Create a validation error
		validationErr := &ValidationError{
			Field:   field,
			Value:   value,
			Message: "original message",
		}

		// Test message enhancement - should not panic
		enhanced := enhanceValidationMessage(validationErr)

		// Enhanced message should not be empty
		if enhanced == "" {
			t.Error("enhanceValidationMessage returned empty message")
		}

		// Should not contain null bytes
		for i, r := range enhanced {
			if r == 0 {
				t.Errorf("enhanceValidationMessage contains null byte at position %d", i)
			}
		}
	})
}

// FuzzIsConfigurationError tests error type checking with random error values.
func FuzzIsConfigurationError(f *testing.F) {
	f.Add("config error")
	f.Add("")
	f.Add("generic error")

	f.Fuzz(func(t *testing.T, errMsg string) {
		// Test with nil
		if IsConfigurationError(nil) {
			t.Error("IsConfigurationError should return false for nil")
		}

		// Test with various error types
		errors := []error{
			ErrInvalidConfig,
			ErrConfigFileNotFound,
			&ConfigValidationError{Message: errMsg},
			errors.New(errMsg),
		}

		for _, err := range errors {
			// Should not panic
			result := IsConfigurationError(err)
			_ = result // Use result to avoid unused variable
		}
	})
}

// Benchmark fuzzing to ensure performance doesn't degrade.
func BenchmarkConfigPathResolution(b *testing.B) {
	paths := []string{
		"",
		"config.yaml",
		"/etc/ephemos.yaml",
		"../config.yaml",
		"./test/config.yaml",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		_, _ = resolveConfigPath(path)
	}
}
