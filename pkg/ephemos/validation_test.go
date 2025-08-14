package ephemos

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

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
			input:   Configuration{},
			wantErr: true,
		},
		{
			name:    "pointer to non-struct",
			input:   new(string),
			wantErr: true,
		},
		{
			name:  "valid config with defaults",
			input: &Configuration{},
			validate: func(t *testing.T, result any, _ error) {
				t.Helper()
				config := result.(*Configuration)
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
			name: "invalid service name",
			input: &Configuration{
				Service: ServiceConfig{
					Name: "invalid name with spaces",
				},
			},
			wantErr:      true,
			errSubstring: "must match pattern",
		},
		{
			name: "invalid transport type",
			input: &Configuration{
				Service: ServiceConfig{
					Name: "valid-service",
				},
				Transport: TransportConfig{
					Type: "invalid-type",
				},
			},
			wantErr:      true,
			errSubstring: "must be one of",
		},
		{
			name: "multiple validation errors",
			input: &Configuration{
				Service: ServiceConfig{
					Name:   "",               // Required field missing
					Domain: "invalid-domain", // Invalid domain format
				},
				Transport: TransportConfig{
					Type:    "invalid", // Invalid transport type
					Address: "invalid", // Invalid address format
				},
			},
			wantErr: true,
			validate: func(t *testing.T, _ any, err error) {
				t.Helper()
				// Should have multiple errors
				var errCollection *ValidationCollectionError
				if !errors.As(err, &errCollection) {
					t.Fatal("expected ValidationErrors")
				}
				if len(errCollection.Errors) < 2 {
					t.Errorf("expected multiple errors, got %d", len(errCollection.Errors))
				}
			},
		},
	}

	engine := NewValidationEngine()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateAndSetDefaults(tt.input)

			validateTestResult(t, tt, err)
		})
	}
}

// testStruct is used for validation rule testing.
type testStruct struct {
	RequiredField    string   `validate:"required"`
	MinLengthField   string   `validate:"min=5"`
	MaxLengthField   string   `validate:"max=10"`
	ExactLengthField string   `validate:"len=8"`
	RegexField       string   `validate:"regex=^[A-Z]+$"`
	OneOfField       string   `validate:"oneof=red|green|blue"`
	IPField          string   `validate:"ip"`
	PortField        string   `validate:"port"`
	SPIFFEIDField    string   `validate:"spiffe_id"`
	DomainField      string   `validate:"domain"`
	DurationField    string   `validate:"duration"`
	AbsPathField     string   `validate:"abs_path"`
	SliceMinField    []string `validate:"min=2"`
	SliceMaxField    []string `validate:"max=3"`
}

func TestValidationRules(t *testing.T) {
	tests := []struct {
		name     string
		data     testStruct
		wantErr  bool
		errField string
	}{
		{
			name: "all valid",
			data: testStruct{
				RequiredField:    "present",
				MinLengthField:   "12345",
				MaxLengthField:   "short",
				ExactLengthField: "exactly8",
				RegexField:       "UPPERCASE",
				OneOfField:       "red",
				IPField:          "192.168.1.1",
				PortField:        "8080",
				SPIFFEIDField:    "spiffe://example.org/service",
				DomainField:      "example.org",
				DurationField:    "5m",
				AbsPathField:     "/absolute/path",
				SliceMinField:    []string{"one", "two"},
				SliceMaxField:    []string{"a", "b", "c"},
			},
			wantErr: false,
		},
		{
			name: "required field missing",
			data: testStruct{
				RequiredField: "",
			},
			wantErr:  true,
			errField: "RequiredField",
		},
		{
			name: "min length violation",
			data: testStruct{
				RequiredField:  "present",
				MinLengthField: "123",
			},
			wantErr:  true,
			errField: "MinLengthField",
		},
		{
			name: "max length violation",
			data: testStruct{
				RequiredField:  "present",
				MaxLengthField: "this is too long",
			},
			wantErr:  true,
			errField: "MaxLengthField",
		},
		{
			name: "exact length violation",
			data: testStruct{
				RequiredField:    "present",
				ExactLengthField: "wrong",
			},
			wantErr:  true,
			errField: "ExactLengthField",
		},
		{
			name: "regex violation",
			data: testStruct{
				RequiredField: "present",
				RegexField:    "lowercase",
			},
			wantErr:  true,
			errField: "RegexField",
		},
		{
			name: "oneof violation",
			data: testStruct{
				RequiredField: "present",
				OneOfField:    "yellow",
			},
			wantErr:  true,
			errField: "OneOfField",
		},
		{
			name: "invalid IP",
			data: testStruct{
				RequiredField: "present",
				IPField:       "not.an.ip",
			},
			wantErr:  true,
			errField: "IPField",
		},
		{
			name: "invalid port",
			data: testStruct{
				RequiredField: "present",
				PortField:     "99999",
			},
			wantErr:  true,
			errField: "PortField",
		},
		{
			name: "invalid SPIFFE ID",
			data: testStruct{
				RequiredField: "present",
				SPIFFEIDField: "not-a-spiffe-id",
			},
			wantErr:  true,
			errField: "SPIFFEIDField",
		},
		{
			name: "invalid domain",
			data: testStruct{
				RequiredField: "present",
				DomainField:   "invalid",
			},
			wantErr:  true,
			errField: "DomainField",
		},
		{
			name: "invalid duration",
			data: testStruct{
				RequiredField: "present",
				DurationField: "not-a-duration",
			},
			wantErr:  true,
			errField: "DurationField",
		},
		{
			name: "invalid absolute path",
			data: testStruct{
				RequiredField: "present",
				AbsPathField:  "relative/path",
			},
			wantErr:  true,
			errField: "AbsPathField",
		},
		{
			name: "slice too small",
			data: testStruct{
				RequiredField: "present",
				SliceMinField: []string{"one"},
			},
			wantErr:  true,
			errField: "SliceMinField",
		},
		{
			name: "slice too large",
			data: testStruct{
				RequiredField: "present",
				SliceMaxField: []string{"a", "b", "c", "d"},
			},
			wantErr:  true,
			errField: "SliceMaxField",
		},
	}

	engine := NewValidationEngine()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateAndSetDefaults(&tt.data)

			validateRuleTestResult(t, &tt, err)
		})
	}
}

func TestDefaultValues(t *testing.T) {
	type testStruct struct {
		StringField    string   `default:"default-value"`
		BoolField      bool     `default:"true"`
		IntField       int      `default:"42"`
		UintField      uint     `default:"100"`
		FloatField     float64  `default:"3.14"`
		SliceField     []string `default:"one,two,three"`
		NoDefaultField string
	}

	data := &testStruct{}
	engine := NewValidationEngine()

	err := engine.ValidateAndSetDefaults(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.StringField != "default-value" {
		t.Errorf("expected string default 'default-value', got '%s'", data.StringField)
	}
	if !data.BoolField {
		t.Error("expected bool default true, got false")
	}
	if data.IntField != 42 {
		t.Errorf("expected int default 42, got %d", data.IntField)
	}
	if data.UintField != 100 {
		t.Errorf("expected uint default 100, got %d", data.UintField)
	}
	if data.FloatField != 3.14 {
		t.Errorf("expected float default 3.14, got %f", data.FloatField)
	}
	if !reflect.DeepEqual(data.SliceField, []string{"one", "two", "three"}) {
		t.Errorf("expected slice default [one two three], got %v", data.SliceField)
	}
	if data.NoDefaultField != "" {
		t.Errorf("expected no default field to remain empty, got '%s'", data.NoDefaultField)
	}
}

func TestNestedStructValidation(t *testing.T) {
	type nestedStruct struct {
		RequiredField string `validate:"required"`
		DefaultField  string `default:"nested-default"`
	}

	type parentStruct struct {
		Nested      nestedStruct `validate:"required"`
		NestedPtr   *nestedStruct
		NestedSlice []nestedStruct `validate:"min=1"`
		ParentField string         `validate:"required"`
	}

	// Test with valid nested struct
	t.Run("valid nested", func(t *testing.T) {
		data := &parentStruct{
			Nested: nestedStruct{
				RequiredField: "present",
			},
			NestedPtr: &nestedStruct{
				RequiredField: "also-present",
			},
			NestedSlice: []nestedStruct{
				{RequiredField: "first"},
				{RequiredField: "second"},
			},
			ParentField: "parent-value",
		}

		engine := NewValidationEngine()
		err := engine.ValidateAndSetDefaults(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Check that defaults were set
		if data.Nested.DefaultField != "nested-default" {
			t.Error("nested default not set")
		}
		if data.NestedPtr.DefaultField != "nested-default" {
			t.Error("nested pointer default not set")
		}
	})

	// Test with invalid nested struct
	t.Run("invalid nested", func(t *testing.T) {
		data := &parentStruct{
			Nested: nestedStruct{
				RequiredField: "", // Missing required field
			},
			ParentField: "parent-value",
		}

		engine := NewValidationEngine()
		err := engine.ValidateAndSetDefaults(data)
		if err == nil {
			t.Error("expected error for missing nested required field")
		}

		if !strings.Contains(err.Error(), "Nested.RequiredField") {
			t.Errorf("expected error for Nested.RequiredField, got: %v", err)
		}
	})
}

func TestStopOnFirstError(t *testing.T) {
	type testStruct struct {
		Field1 string `validate:"required"`
		Field2 string `validate:"required"`
		Field3 string `validate:"required"`
	}

	data := &testStruct{} // All fields are missing

	// Test with stop on first error enabled
	t.Run("stop on first error", func(t *testing.T) {
		engine := NewValidationEngine()
		engine.StopOnFirstError = true

		err := engine.ValidateAndSetDefaults(data)
		if err == nil {
			t.Error("expected error")
			return
		}

		// Should only have one error
		var errCollection *ValidationCollectionError
		if errors.As(err, &errCollection) {
			if len(errCollection.Errors) != 1 {
				t.Errorf("expected 1 error with stop on first error, got %d", len(errCollection.Errors))
			}
		}
	})

	// Test with collect all errors
	t.Run("collect all errors", func(t *testing.T) {
		engine := NewValidationEngine()
		engine.StopOnFirstError = false

		err := engine.ValidateAndSetDefaults(data)
		if err == nil {
			t.Error("expected error")
			return
		}

		// Should have multiple errors
		var errCollection *ValidationCollectionError
		if errors.As(err, &errCollection) {
			if len(errCollection.Errors) < 3 {
				t.Errorf("expected 3 errors with collect all errors, got %d", len(errCollection.Errors))
			}
		}
	})
}

func TestConfigurationValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Configuration
		wantErr bool
		setup   func()
		cleanup func()
	}{
		{
			name: "valid configuration with defaults",
			config: &Configuration{
				Service: ServiceConfig{
					Name: "test-service",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid service name",
			config: &Configuration{
				Service: ServiceConfig{
					Name: "invalid service name",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid SPIFFE socket path",
			config: &Configuration{
				Service: ServiceConfig{
					Name: "test-service",
				},
				SPIFFE: &SPIFFEConfig{
					SocketPath: "relative/path",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid transport type",
			config: &Configuration{
				Service: ServiceConfig{
					Name: "test-service",
				},
				Transport: TransportConfig{
					Type: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid transport address",
			config: &Configuration{
				Service: ServiceConfig{
					Name: "test-service",
				},
				Transport: TransportConfig{
					Address: "invalid-address",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			if tt.cleanup != nil {
				defer tt.cleanup()
			}

			err := tt.config.ValidateAndSetDefaults()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	collection := &ValidationCollectionError{}

	// Test empty collection
	if collection.HasErrors() {
		t.Error("empty collection should not have errors")
	}

	// Add some errors
	collection.Add("field1", "error message 1", "value1")
	collection.Add("field2", "error message 2", "value2")
	collection.Add("field1", "error message 3", "value3")

	if !collection.HasErrors() {
		t.Error("collection should have errors")
	}

	if len(collection.Errors) != 3 {
		t.Errorf("expected 3 errors, got %d", len(collection.Errors))
	}

	// Test getting field-specific errors
	field1Errors := collection.GetFieldErrors("field1")
	if len(field1Errors) != 2 {
		t.Errorf("expected 2 errors for field1, got %d", len(field1Errors))
	}

	field2Errors := collection.GetFieldErrors("field2")
	if len(field2Errors) != 1 {
		t.Errorf("expected 1 error for field2, got %d", len(field2Errors))
	}

	// Test error message formatting
	errMsg := collection.Error()
	if !strings.Contains(errMsg, "validation failed with 3 errors") {
		t.Errorf("unexpected error message format: %s", errMsg)
	}
}

func TestValidationHelperFunctions(t *testing.T) {
	// Test IsValidationError
	validationErr := &ValidationError{Field: "test", Message: "test error"}
	collectionErr := &ValidationCollectionError{
		Errors: []ValidationError{*validationErr},
	}
	regularErr := errors.New("regular error")

	if !IsValidationError(validationErr) {
		t.Error("should detect ValidationError")
	}
	if !IsValidationError(collectionErr) {
		t.Error("should detect ValidationErrors")
	}
	if IsValidationError(regularErr) {
		t.Error("should not detect regular error as validation error")
	}
	if IsValidationError(nil) {
		t.Error("should not detect nil as validation error")
	}

	// Test GetValidationErrors
	validationErrors := GetValidationErrors(validationErr)
	if len(validationErrors) != 1 {
		t.Errorf("expected 1 validation error, got %d", len(validationErrors))
	}

	collectionErrors := GetValidationErrors(collectionErr)
	if len(collectionErrors) != 1 {
		t.Errorf("expected 1 validation error from collection, got %d", len(collectionErrors))
	}

	regularErrors := GetValidationErrors(regularErr)
	if len(regularErrors) != 0 {
		t.Errorf("expected 0 validation errors from regular error, got %d", len(regularErrors))
	}
}

// Benchmark tests for performance.
func BenchmarkValidationEngine_ValidateAndSetDefaults(b *testing.B) {
	engine := NewValidationEngine()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a fresh copy for each iteration
		testConfig := &Configuration{
			Service: ServiceConfig{
				Name: "benchmark-service",
			},
		}
		_ = engine.ValidateAndSetDefaults(testConfig)
	}
}

func BenchmarkValidationEngine_AggregatedErrors(b *testing.B) {
	engine := NewValidationEngine()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a fresh copy for each iteration with multiple errors
		testConfig := &Configuration{
			Service: ServiceConfig{
				Name:   "",               // Missing required
				Domain: "invalid-domain", // Invalid format
			},
			Transport: TransportConfig{
				Type:    "invalid", // Invalid type
				Address: "invalid", // Invalid address
			},
		}
		_ = engine.ValidateAndSetDefaults(testConfig)
	}
}

// Helper function to create temporary files for testing file validation.
func createTempFile(t *testing.T, content string) (string, func()) {
	t.Helper()

	tmpFile, err := os.CreateTemp(t.TempDir(), "validation_test_*.txt")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	if content != "" {
		if _, err := tmpFile.WriteString(content); err != nil {
			tmpFile.Close()
			os.Remove(tmpFile.Name())
			t.Fatalf("failed to write to temp file: %v", err)
		}
	}

	tmpFile.Close()

	return tmpFile.Name(), func() {
		os.Remove(tmpFile.Name())
	}
}

func TestFileValidation(t *testing.T) {
	type fileStruct struct {
		ExistingFile    string `validate:"file_exists"`
		NonExistingFile string `validate:"file_exists"`
		ExistingDir     string `validate:"dir_exists"`
		NonExistingDir  string `validate:"dir_exists"`
	}

	// Create temporary file and directory
	tmpFile, cleanupFile := createTempFile(t, "test content")
	defer cleanupFile()

	tmpDir := t.TempDir()

	tests := []struct {
		name    string
		data    fileStruct
		wantErr bool
	}{
		{
			name: "valid files and dirs",
			data: fileStruct{
				ExistingFile:    tmpFile,
				NonExistingFile: "", // Empty is allowed
				ExistingDir:     tmpDir,
				NonExistingDir:  "", // Empty is allowed
			},
			wantErr: false,
		},
		{
			name: "non-existing file",
			data: fileStruct{
				ExistingFile:    tmpFile,
				NonExistingFile: "/path/that/does/not/exist",
				ExistingDir:     tmpDir,
			},
			wantErr: true,
		},
		{
			name: "non-existing directory",
			data: fileStruct{
				ExistingFile:   tmpFile,
				ExistingDir:    tmpDir,
				NonExistingDir: "/dir/that/does/not/exist",
			},
			wantErr: true,
		},
		{
			name: "file as directory",
			data: fileStruct{
				ExistingFile: tmpFile,
				ExistingDir:  tmpFile, // File path where directory expected
			},
			wantErr: true,
		},
		{
			name: "directory as file",
			data: fileStruct{
				ExistingFile: tmpDir, // Directory path where file expected
				ExistingDir:  tmpDir,
			},
			wantErr: true,
		},
	}

	engine := NewValidationEngine()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateAndSetDefaults(&tt.data)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	engine := NewValidationEngine()

	// Test with custom tag names
	t.Run("custom tag names", func(t *testing.T) {
		type customStruct struct {
			Field string `customValidate:"required" customDefault:"default-value"`
		}

		customEngine := NewValidationEngine()
		customEngine.TagName = "customValidate"
		customEngine.DefaultTagName = "customDefault"

		data := &customStruct{}
		err := customEngine.ValidateAndSetDefaults(data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if data.Field != "default-value" {
			t.Errorf("expected default value to be set, got '%s'", data.Field)
		}
	})

	// Test with complex nested structures
	t.Run("deeply nested structures", func(t *testing.T) {
		type level3 struct {
			Field string `validate:"required" default:"level3-default"`
		}
		type level2 struct {
			Level3 level3 `validate:"required"`
			Field  string `validate:"required" default:"level2-default"`
		}
		type level1 struct {
			Level2 level2 `validate:"required"`
			Field  string `validate:"required" default:"level1-default"`
		}

		data := &level1{}
		err := engine.ValidateAndSetDefaults(data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if data.Field != "level1-default" {
			t.Error("level1 default not set")
		}
		if data.Level2.Field != "level2-default" {
			t.Error("level2 default not set")
		}
		if data.Level2.Level3.Field != "level3-default" {
			t.Error("level3 default not set")
		}
	})

	// Test with interface{} fields
	t.Run("interface fields", func(t *testing.T) {
		type interfaceStruct struct {
			InterfaceField any `validate:"required"`
		}

		data := &interfaceStruct{
			InterfaceField: "some value",
		}

		err := engine.ValidateAndSetDefaults(data)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// validateTestResult is a helper function to reduce nested complexity in tests.
func validateTestResult(t *testing.T, tt struct {
	name         string
	input        any
	wantErr      bool
	errSubstring string
	validate     func(t *testing.T, result any, err error)
}, err error,
) {
	t.Helper()

	if !tt.wantErr {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
			return
		}
		if tt.validate != nil {
			tt.validate(t, tt.input, nil)
		}
		return
	}

	// tt.wantErr is true
	if err == nil {
		t.Error("expected error, got nil")
		return
	}

	if tt.errSubstring != "" && !strings.Contains(err.Error(), tt.errSubstring) {
		t.Errorf("expected error containing '%s', got '%s'", tt.errSubstring, err.Error())
	}

	if tt.validate != nil {
		tt.validate(t, tt.input, err)
	}
}

// validateRuleTestResult is a helper function to reduce nested complexity in rule tests.
func validateRuleTestResult(t *testing.T, tt *struct {
	name     string
	data     testStruct
	wantErr  bool
	errField string
}, err error,
) {
	t.Helper()

	if !tt.wantErr {
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		return
	}

	// tt.wantErr is true
	if err == nil {
		t.Error("expected error, got nil")
		return
	}

	if tt.errField != "" && !strings.Contains(err.Error(), tt.errField) {
		t.Errorf("expected error for field '%s', got '%s'", tt.errField, err.Error())
	}
}
