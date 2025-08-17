package domain

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

// Test configuration structs for domain layer testing
type testConfiguration struct {
	Service   testServiceConfig   `yaml:"service"`
	Transport testTransportConfig `yaml:"transport"`
}

type testServiceConfig struct {
	Name   string `yaml:"name" validate:"required,min=1,max=100,regex=^[a-zA-Z0-9_-]+$" default:"ephemos-service"`
	Domain string `yaml:"domain,omitempty" validate:"domain" default:"default.local"`
}

type testTransportConfig struct {
	Type    string `yaml:"type" validate:"required,oneof=http|https|grpc" default:"http"`
	Address string `yaml:"address" validate:"required,regex=^:[0-9]+$" default:":8080"`
}

func TestValidationEngine_ValidateAndSetDefaults(t *testing.T) {
	tests := []struct {
		name         string
		input        any
		wantErr      bool
		errSubstring string
		validate     func(t *testing.T, result any, err error)
	}{
		{
			name:    "nil input",
			input:   nil,
			wantErr: true,
		},
		{
			name:    "non-pointer input",
			input:   testConfiguration{},
			wantErr: true,
		},
		{
			name:    "pointer to non-struct",
			input:   new(string),
			wantErr: true,
		},
		{
			name:  "valid config with defaults",
			input: &testConfiguration{},
			validate: func(t *testing.T, result any, _ error) {
				t.Helper()
				config := result.(*testConfiguration)
				if config.Service.Name != "ephemos-service" {
					t.Errorf("expected default service name 'ephemos-service', got '%s'", config.Service.Name)
				}
				if config.Service.Domain != "default.local" {
					t.Errorf("expected default domain 'default.local', got '%s'", config.Service.Domain)
				}
				if config.Transport.Type != "http" {
					t.Errorf("expected default transport type 'http', got '%s'", config.Transport.Type)
				}
				if config.Transport.Address != ":8080" {
					t.Errorf("expected default address ':8080', got '%s'", config.Transport.Address)
				}
			},
		},
		{
			name: "validation errors",
			input: &testConfiguration{
				Service: testServiceConfig{
					Name:   "", // required field empty
					Domain: "invalid",
				},
			},
			wantErr: true,
		},
	}

	engine := NewValidationEngine()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateAndSetDefaults(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAndSetDefaults() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validate != nil && !tt.wantErr {
				tt.validate(t, tt.input, err)
			}

			if tt.errSubstring != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errSubstring) {
					t.Errorf("expected error to contain '%s', got: %v", tt.errSubstring, err)
				}
			}
		})
	}
}

func TestValidationEngine_CollectionError(t *testing.T) {
	// Test with configuration that has multiple validation errors
	config := &testConfiguration{
		Service: testServiceConfig{
			Name:   "", // required field empty
			Domain: "invalid-domain",
		},
		Transport: testTransportConfig{
			Type:    "invalid",
			Address: "invalid-address",
		},
	}

	engine := NewValidationEngine()
	err := engine.ValidateAndSetDefaults(config)

	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	var collectionErr *ValidationCollectionError
	if !errors.As(err, &collectionErr) {
		t.Fatalf("expected ValidationCollectionError, got %T", err)
	}

	if !collectionErr.HasErrors() {
		t.Error("expected collection to have errors")
	}

	if len(collectionErr.Errors) == 0 {
		t.Error("expected multiple validation errors")
	}

	t.Logf("Validation errors: %v", err)
}

func TestValidationEngine_DefaultsAndValidation(t *testing.T) {
	config := &testConfiguration{}

	engine := NewValidationEngine()
	err := engine.ValidateAndSetDefaults(config)

	if err != nil {
		t.Fatalf("validation failed: %v", err)
	}

	// Check that defaults were set
	if config.Service.Name == "" {
		t.Error("default service name was not set")
	}

	if config.Service.Domain == "" {
		t.Error("default domain was not set")
	}

	if config.Transport.Type == "" {
		t.Error("default transport type was not set")
	}

	if config.Transport.Address == "" {
		t.Error("default transport address was not set")
	}
}

func TestValidationRules(t *testing.T) {
	tests := []struct {
		name   string
		input  testStruct
		hasErr bool
	}{
		{
			name: "valid required field",
			input: testStruct{
				RequiredField: "value",
				MinField:      "hello",
				MaxField:      "short",
			},
			hasErr: false,
		},
		{
			name: "empty required field",
			input: testStruct{
				RequiredField: "",
			},
			hasErr: true,
		},
		{
			name: "valid min length",
			input: testStruct{
				RequiredField: "value",
				MinField:      "hello",
				MaxField:      "short",
			},
			hasErr: false,
		},
		{
			name: "invalid min length",
			input: testStruct{
				MinField: "hi",
			},
			hasErr: true,
		},
		{
			name: "valid max length",
			input: testStruct{
				RequiredField: "value",
				MinField:      "hello",
				MaxField:      "short",
			},
			hasErr: false,
		},
		{
			name: "invalid max length",
			input: testStruct{
				MaxField: "this is a very long string that exceeds maximum",
			},
			hasErr: true,
		},
	}

	engine := NewValidationEngine()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateAndSetDefaults(&tt.input)
			hasErr := err != nil

			if hasErr != tt.hasErr {
				t.Errorf("expected hasErr=%v, got hasErr=%v, err=%v", tt.hasErr, hasErr, err)
			}
		})
	}
}

func TestFileValidation(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "validation_test")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	tests := []struct {
		name     string
		filePath string
		hasErr   bool
	}{
		{
			name:     "existing file",
			filePath: tmpFile.Name(),
			hasErr:   false,
		},
		{
			name:     "non-existing file",
			filePath: "/non/existent/file",
			hasErr:   true,
		},
		{
			name:     "empty path",
			filePath: "",
			hasErr:   false, // Empty paths are allowed unless required
		},
	}

	engine := NewValidationEngine()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &testFileStruct{
				FilePath: tt.filePath,
			}

			err := engine.ValidateAndSetDefaults(input)
			hasErr := err != nil

			if hasErr != tt.hasErr {
				t.Errorf("expected hasErr=%v, got hasErr=%v, err=%v", tt.hasErr, hasErr, err)
			}
		})
	}
}

func TestGetValidationErrors(t *testing.T) {
	// Test with single validation error
	singleErr := &ValidationError{
		Field:   "test",
		Message: "test error",
		Value:   "value",
	}

	validationErrors := GetValidationErrors(singleErr)
	if len(validationErrors) != 1 {
		t.Errorf("expected 1 error, got %d", len(validationErrors))
	}

	// Test with collection error
	collectionErr := &ValidationCollectionError{
		Errors: []ValidationError{
			{Field: "field1", Message: "error1", Value: "value1"},
			{Field: "field2", Message: "error2", Value: "value2"},
		},
	}

	validationErrors2 := GetValidationErrors(collectionErr)
	if len(validationErrors2) != 2 {
		t.Errorf("expected 2 errors, got %d", len(validationErrors2))
	}

	// Test with non-validation error
	otherErr := errors.New("not a validation error")
	validationErrors3 := GetValidationErrors(otherErr)
	if validationErrors3 != nil {
		t.Errorf("expected nil for non-validation error, got %v", validationErrors3)
	}
}

func TestValidationEngineOptions(t *testing.T) {
	engine := NewValidationEngine()
	engine.StopOnFirstError = true

	// Use a struct with multiple validation errors
	config := &testConfiguration{
		Service: testServiceConfig{
			Name:   "",        // error 1
			Domain: "invalid", // error 2
		},
	}

	err := engine.ValidateAndSetDefaults(config)
	if err == nil {
		t.Fatal("expected validation error")
	}

	// With StopOnFirstError=true, we should get a single ValidationError,
	// not a ValidationCollectionError
	var singleErr *ValidationError
	if !errors.As(err, &singleErr) {
		// Or we might get a collection with just one error
		var collectionErr *ValidationCollectionError
		if errors.As(err, &collectionErr) && len(collectionErr.Errors) > 1 {
			t.Error("expected to stop on first error, but got multiple errors")
		}
	}
}

// Test structs for validation testing
type testStruct struct {
	RequiredField string `validate:"required"`
	MinField      string `validate:"min=5"`
	MaxField      string `validate:"max=10"`
	DefaultField  string `default:"default_value"`
}

type testFileStruct struct {
	FilePath string `validate:"file_exists"`
}

func TestReflectZeroValue(t *testing.T) {
	engine := NewValidationEngine()

	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{"empty string", reflect.ValueOf(""), true},
		{"non-empty string", reflect.ValueOf("test"), false},
		{"zero int", reflect.ValueOf(0), true},
		{"non-zero int", reflect.ValueOf(42), false},
		{"false bool", reflect.ValueOf(false), true},
		{"true bool", reflect.ValueOf(true), false},
		{"nil pointer", reflect.ValueOf((*string)(nil)), true},
		{"non-nil pointer", reflect.ValueOf(&[]string{"test"}[0]), false},
		{"empty slice", reflect.ValueOf([]string{}), true},
		{"nil slice", reflect.ValueOf([]string(nil)), true},
		{"non-empty slice", reflect.ValueOf([]string{"test"}), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.isZeroValue(tt.value.(reflect.Value))
			if result != tt.expected {
				t.Errorf("isZeroValue(%v) = %v, expected %v", tt.value, result, tt.expected)
			}
		})
	}
}
