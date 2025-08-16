package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/sufield/ephemos/internal/core/errors"
	"github.com/sufield/ephemos/internal/core/ports"
)

func TestPrinter(t *testing.T) {
	tests := []struct {
		name     string
		emoji    bool
		quiet    bool
		method   string
		message  string
		expected string
		useErr   bool
	}{
		{
			name:     "success with emoji",
			emoji:    true,
			method:   "Success",
			message:  "Test passed",
			expected: "✅ Test passed\n",
		},
		{
			name:     "success without emoji",
			emoji:    false,
			method:   "Success",
			message:  "Test passed",
			expected: "Test passed\n",
		},
		{
			name:     "quiet mode suppresses success",
			emoji:    true,
			quiet:    true,
			method:   "Success",
			message:  "Test passed",
			expected: "",
		},
		{
			name:     "error always shows",
			emoji:    true,
			quiet:    true,
			method:   "Error",
			message:  "Test failed",
			expected: "❌ Test failed\n",
			useErr:   true,
		},
		{
			name:     "bullet with emoji",
			emoji:    true,
			method:   "Bullet",
			message:  "Item",
			expected: "  • Item\n",
		},
		{
			name:     "bullet without emoji",
			emoji:    false,
			method:   "Bullet",
			message:  "Item",
			expected: "  - Item\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, err bytes.Buffer
			printer := NewPrinter(&out, &err, tt.emoji, tt.quiet)

			switch tt.method {
			case "Success":
				printer.Success(tt.message)
			case "Error":
				printer.Error(tt.message)
			case "Bullet":
				printer.Bullet(tt.message)
			}

			var result string
			if tt.useErr {
				result = err.String()
			} else {
				result = out.String()
			}

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestGetProductionTips(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected []string
	}{
		{
			name: "example trust domain",
			err:  errors.ErrExampleTrustDomain,
			expected: []string{
				"Set EPHEMOS_TRUST_DOMAIN to your production domain (e.g., 'prod.company.com')",
			},
		},
		{
			name: "localhost trust domain",
			err:  errors.ErrLocalhostTrustDomain,
			expected: []string{
				"Set EPHEMOS_TRUST_DOMAIN to a proper domain (not localhost)",
			},
		},
		{
			name: "demo service name",
			err:  errors.ErrDemoServiceName,
			expected: []string{
				"Set EPHEMOS_SERVICE_NAME to your production service name (not demo)",
			},
		},
		{
			name: "debug enabled",
			err:  errors.ErrDebugEnabled,
			expected: []string{
				"Set EPHEMOS_DEBUG_ENABLED=false for production",
			},
		},
		{
			name: "insecure skip verify",
			err:  errors.ErrInsecureSkipVerify,
			expected: []string{
				"Remove EPHEMOS_INSECURE_SKIP_VERIFY or set to false for production",
			},
		},
		{
			name: "verbose logging",
			err:  errors.ErrVerboseLogging,
			expected: []string{
				"Set EPHEMOS_LOG_LEVEL to 'info' or 'error' for production (not debug/trace)",
			},
		},
		{
			name: "wildcard clients",
			err:  errors.ErrWildcardClients,
			expected: []string{
				"Use specific SPIFFE IDs instead of wildcards in EPHEMOS_AUTHORIZED_CLIENTS",
			},
		},
		{
			name: "insecure socket path",
			err:  errors.ErrInsecureSocketPath,
			expected: []string{
				"Set EPHEMOS_SPIFFE_SOCKET to a secure path like '/run/spire/sockets/api.sock'",
			},
		},
		{
			name: "multiple errors",
			err: errors.NewProductionValidationError(
				errors.ErrExampleTrustDomain,
				errors.ErrDebugEnabled,
			),
			expected: []string{
				"Set EPHEMOS_TRUST_DOMAIN to your production domain (e.g., 'prod.company.com')",
				"Set EPHEMOS_DEBUG_ENABLED=false for production",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tips := getProductionTips(tt.err)

			if len(tips) != len(tt.expected) {
				t.Errorf("expected %d tips, got %d", len(tt.expected), len(tips))
				t.Errorf("tips: %v", tips)
				return
			}

			for i, tip := range tips {
				if tip != tt.expected[i] {
					t.Errorf("tip %d: expected %q, got %q", i, tt.expected[i], tip)
				}
			}
		})
	}
}

func TestLoadConfiguration(t *testing.T) {
	// Save and restore environment
	originalServiceName := os.Getenv(ports.EnvServiceName)
	originalTrustDomain := os.Getenv(ports.EnvTrustDomain)
	defer func() {
		os.Setenv(ports.EnvServiceName, originalServiceName)
		os.Setenv(ports.EnvTrustDomain, originalTrustDomain)
	}()

	tests := []struct {
		name       string
		configFile string
		envOnly    bool
		envVars    map[string]string
		expectErr  bool
		errMsg     string
	}{
		{
			name:      "neither flag specified",
			expectErr: true,
			errMsg:    "either --config or --env-only must be specified",
		},
		{
			name:    "env-only success",
			envOnly: true,
			envVars: map[string]string{
				ports.EnvServiceName: "test-service",
				ports.EnvTrustDomain: "prod.company.com",
			},
			expectErr: false,
		},
		{
			name:      "env-only missing required",
			envOnly:   true,
			envVars:   map[string]string{},
			expectErr: true,
			errMsg:    "failed to load configuration from environment",
		},
		{
			name:       "config file not found",
			configFile: "/nonexistent/config.yaml",
			expectErr:  true,
			errMsg:     "failed to load configuration file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}
			defer func() {
				for k := range tt.envVars {
					os.Unsetenv(k)
				}
			}()

			var out, err bytes.Buffer
			printer := NewPrinter(&out, &err, false, false)
			ctx := context.Background()

			cfg, loadErr := loadConfigurationCobra(ctx, printer, tt.configFile, tt.envOnly)

			if tt.expectErr {
				if loadErr == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(loadErr.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, loadErr.Error())
				}
			} else {
				if loadErr != nil {
					t.Errorf("unexpected error: %v", loadErr)
				}
				if cfg == nil {
					t.Error("expected configuration, got nil")
				}
			}
		})
	}
}

func TestJSONOutput(t *testing.T) {
	tests := []struct {
		name     string
		result   *Result
		validate func(t *testing.T, output map[string]interface{})
	}{
		{
			name: "basic validation success",
			result: &Result{
				BasicValid:      true,
				ProductionValid: true,
				Messages:        []string{"Basic validation passed"},
				Configuration: &Config{
					ServiceName: "test-service",
					TrustDomain: "test.example.org",
				},
			},
			validate: func(t *testing.T, output map[string]interface{}) {
				if !output["basic_valid"].(bool) {
					t.Error("expected basic_valid to be true")
				}
				if !output["production_valid"].(bool) {
					t.Error("expected production_valid to be true")
				}
			},
		},
		{
			name: "production validation failure with tips",
			result: &Result{
				BasicValid:      true,
				ProductionValid: false,
				Tips: []string{
					"Set EPHEMOS_TRUST_DOMAIN to your production domain",
					"Set EPHEMOS_DEBUG_ENABLED=false for production",
				},
				Errors: []string{"production validation failed"},
			},
			validate: func(t *testing.T, output map[string]interface{}) {
				if !output["basic_valid"].(bool) {
					t.Error("expected basic_valid to be true")
				}
				if output["production_valid"].(bool) {
					t.Error("expected production_valid to be false")
				}
				tips := output["tips"].([]interface{})
				if len(tips) != 2 {
					t.Errorf("expected 2 tips, got %d", len(tips))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printer := NewPrinter(&buf, &buf, false, false)

			if err := printer.PrintJSON(tt.result); err != nil {
				t.Fatalf("failed to print JSON: %v", err)
			}

			var output map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			tt.validate(t, output)
		})
	}
}

func TestExitCodes(t *testing.T) {
	// Verify exit codes are unique and meaningful
	codes := map[string]int{
		"ExitSuccess":             ExitSuccess,
		"ExitUsageError":          ExitUsageError,
		"ExitBasicValidation":     ExitBasicValidation,
		"ExitProductionReadiness": ExitProductionReadiness,
		"ExitLoadError":           ExitLoadError,
	}

	seen := make(map[int]string)
	for name, code := range codes {
		if existing, ok := seen[code]; ok {
			t.Errorf("duplicate exit code %d for %s and %s", code, name, existing)
		}
		seen[code] = name
	}

	// Verify specific values
	if ExitSuccess != 0 {
		t.Error("ExitSuccess should be 0")
	}
	if ExitUsageError == ExitSuccess {
		t.Error("ExitUsageError should not equal ExitSuccess")
	}
}