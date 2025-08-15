package ephemos

import (
	"errors"
	"testing"
)

func TestValidationEngineInterface(t *testing.T) {
	// Test that the public interface correctly delegates to the domain layer
	engine := NewValidationEngine()
	if engine == nil {
		t.Fatal("NewValidationEngine() returned nil")
	}

	// Test with a simple struct
	type testStruct struct {
		Name string `validate:"required" default:"test"`
	}

	input := &testStruct{}
	err := engine.ValidateAndSetDefaults(input)
	
	if err != nil {
		t.Errorf("ValidateAndSetDefaults() failed: %v", err)
	}

	if input.Name != "test" {
		t.Errorf("expected default value 'test', got '%s'", input.Name)
	}
}

func TestValidateStructFunction(t *testing.T) {
	// Test the convenience function
	type testStruct struct {
		Value string `validate:"required"`
	}

	// Test with valid struct
	validInput := &testStruct{Value: "test"}
	err := ValidateStruct(validInput)
	if err != nil {
		t.Errorf("ValidateStruct() failed with valid input: %v", err)
	}

	// Test with invalid struct
	invalidInput := &testStruct{} // missing required value
	err = ValidateStruct(invalidInput)
	if err == nil {
		t.Error("ValidateStruct() should have failed with invalid input")
	}
}

func TestValidateStructWithEngineFunction(t *testing.T) {
	engine := NewValidationEngine()
	engine.StopOnFirstError = true

	type testStruct struct {
		Value string `validate:"required"`
	}

	input := &testStruct{}
	err := ValidateStructWithEngine(input, engine)
	if err == nil {
		t.Error("ValidateStructWithEngine() should have failed with invalid input")
	}
}

func TestGetValidationErrorsFunction(t *testing.T) {
	// Test with non-validation error
	err := errors.New("not a validation error")
	validationErrors := GetValidationErrors(err)
	if validationErrors != nil {
		t.Error("GetValidationErrors() should return nil for non-validation errors")
	}

	// Test with validation error by creating one
	type testStruct struct {
		Value string `validate:"required"`
	}

	input := &testStruct{} // missing required value
	err = ValidateStruct(input)
	if err == nil {
		t.Fatal("ValidateStruct() should have failed")
	}

	validationErrors = GetValidationErrors(err)
	if validationErrors == nil {
		t.Error("GetValidationErrors() should return errors for validation errors")
	}

	if len(validationErrors) == 0 {
		t.Error("GetValidationErrors() returned empty slice")
	}
}