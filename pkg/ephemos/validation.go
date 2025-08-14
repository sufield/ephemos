// Package ephemos provides comprehensive validation using struct tags with defaults and aggregated error handling.
package ephemos

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Validation rule constants.
const (
	validationRuleMin      = "min"
	validationRuleMax      = "max"
	validationRuleLen      = "len"
	validationRuleRequired = "required"
	unknownRuleError       = "unknown rule"
)

// ValidationEngine provides struct tag-based validation with defaults and aggregated errors.
type ValidationEngine struct {
	// TagName is the struct tag name to use for validation rules (default: "validate")
	TagName string
	// DefaultTagName is the struct tag name for default values (default: "default")
	DefaultTagName string
	// StopOnFirstError controls whether to fail fast or collect all errors
	StopOnFirstError bool
}

// NewValidationEngine creates a new validation engine with sensible defaults.
func NewValidationEngine() *ValidationEngine {
	return &ValidationEngine{
		TagName:          "validate",
		DefaultTagName:   "default",
		StopOnFirstError: false, // Collect all errors by default
	}
}

// ValidationCollectionError aggregates multiple validation errors.
type ValidationCollectionError struct {
	Errors []ValidationError
}

// Error implements the error interface, returning a summary of all validation errors.
func (vec *ValidationCollectionError) Error() string {
	if len(vec.Errors) == 0 {
		return "no validation errors"
	}

	if len(vec.Errors) == 1 {
		return vec.Errors[0].Error()
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("validation failed with %d errors:\n", len(vec.Errors)))
	for i, err := range vec.Errors {
		builder.WriteString(fmt.Sprintf("  %d. %s\n", i+1, err.Error()))
	}
	return strings.TrimSuffix(builder.String(), "\n")
}

// Add appends a validation error to the collection.
func (vec *ValidationCollectionError) Add(field, message string, value any) {
	vec.Errors = append(vec.Errors, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if there are any validation errors.
func (vec *ValidationCollectionError) HasErrors() bool {
	return len(vec.Errors) > 0
}

// GetFieldErrors returns all errors for a specific field.
func (vec *ValidationCollectionError) GetFieldErrors(field string) []ValidationError {
	var fieldErrors []ValidationError
	for _, err := range vec.Errors {
		if err.Field == field {
			fieldErrors = append(fieldErrors, err)
		}
	}
	return fieldErrors
}

// ValidateAndSetDefaults validates a struct and sets default values based on struct tags.
// This method aggregates all validation errors and provides comprehensive feedback.
func (ve *ValidationEngine) ValidateAndSetDefaults(v any) error {
	if v == nil {
		return &ValidationError{
			Field:   "root",
			Message: "cannot validate nil value",
			Value:   nil,
		}
	}

	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return &ValidationError{
			Field:   "root",
			Message: "must pass a pointer to struct for validation and default setting",
			Value:   v,
		}
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return &ValidationError{
			Field:   "root",
			Message: "can only validate struct types",
			Value:   v,
		}
	}

	errCollection := &ValidationCollectionError{}
	ve.validateStruct(elem, elem.Type(), "", errCollection)

	if errCollection.HasErrors() {
		return errCollection
	}

	return nil
}

// validateStruct recursively validates a struct and its nested structs.
func (ve *ValidationEngine) validateStruct(val reflect.Value, typ reflect.Type, prefix string, errCollection *ValidationCollectionError) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !fieldVal.CanSet() {
			continue
		}

		fieldName := ve.buildFieldName(prefix, &field)

		// Validate and process this field
		ve.validateStructField(fieldVal, &field, fieldName, errCollection)

		// Stop early if configured to do so and we have errors
		if ve.StopOnFirstError && errCollection.HasErrors() {
			return
		}
	}
}

// validateStructField validates a single field within a struct.
func (ve *ValidationEngine) validateStructField(
	fieldVal reflect.Value,
	field *reflect.StructField,
	fieldName string,
	errCollection *ValidationCollectionError,
) {
	// Initialize nested structs if they are required and zero
	ve.initializeRequiredNestedStruct(fieldVal, field, fieldName, errCollection)

	// Set defaults first
	ve.setDefaults(fieldVal, field, fieldName, errCollection)

	// Validate field
	ve.validateField(fieldVal, field, fieldName, errCollection)

	// Recursively validate nested types
	ve.validateNestedType(fieldVal, field, fieldName, errCollection)
}

// validateNestedType handles recursive validation of nested types.
func (ve *ValidationEngine) validateNestedType(
	fieldVal reflect.Value,
	field *reflect.StructField,
	fieldName string,
	errCollection *ValidationCollectionError,
) {
	switch fieldVal.Kind() {
	case reflect.Struct:
		ve.validateStruct(fieldVal, fieldVal.Type(), fieldName, errCollection)
	case reflect.Ptr:
		ve.validatePointerField(fieldVal, fieldName, errCollection)
	case reflect.Slice:
		ve.validateSlice(fieldVal, field, fieldName, errCollection)
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Array,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.String, reflect.UnsafePointer:
		// No special handling needed for primitive types
	default:
		// No special handling needed for unknown types
	}
}

// validatePointerField validates pointer fields that may contain structs.
func (ve *ValidationEngine) validatePointerField(
	fieldVal reflect.Value,
	fieldName string,
	errCollection *ValidationCollectionError,
) {
	if !fieldVal.IsNil() && fieldVal.Elem().Kind() == reflect.Struct {
		ve.validateStruct(fieldVal.Elem(), fieldVal.Elem().Type(), fieldName, errCollection)
	}
}

// validateSlice validates slice and its elements.
func (ve *ValidationEngine) validateSlice(
	val reflect.Value,
	field *reflect.StructField,
	fieldName string,
	errCollection *ValidationCollectionError,
) {
	// First validate the slice itself (length constraints)
	ve.validateSliceSelf(val, field, fieldName, errCollection)

	// Then validate each element
	ve.validateSliceElements(val, field, fieldName, errCollection)
}

// validateSliceSelf validates slice-level constraints (length, etc.).
func (ve *ValidationEngine) validateSliceSelf(
	val reflect.Value,
	field *reflect.StructField,
	fieldName string,
	errCollection *ValidationCollectionError,
) {
	validateTag := field.Tag.Get(ve.TagName)
	if validateTag == "" {
		return
	}

	rules := ve.parseValidationRules(validateTag)
	sliceRules := ve.extractSliceRules(rules)

	if len(sliceRules) > 0 {
		ve.applyValidationRules(val, sliceRules, fieldName, errCollection)
	}
}

// extractSliceRules extracts validation rules that apply to the slice itself.
func (ve *ValidationEngine) extractSliceRules(rules map[string]string) map[string]string {
	sliceRules := make(map[string]string)
	for rule, param := range rules {
		switch rule {
		case validationRuleMin, validationRuleMax, validationRuleLen, validationRuleRequired:
			sliceRules[rule] = param
		}
	}
	return sliceRules
}

// validateSliceElements validates individual slice elements.
func (ve *ValidationEngine) validateSliceElements(
	val reflect.Value,
	field *reflect.StructField,
	fieldName string,
	errCollection *ValidationCollectionError,
) {
	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i)
		elemFieldName := fmt.Sprintf("%s[%d]", fieldName, i)
		ve.validateSliceElementByType(elem, field, elemFieldName, errCollection)
	}
}

// validateSliceElementByType validates a single slice element based on its type.
func (ve *ValidationEngine) validateSliceElementByType(
	elem reflect.Value,
	field *reflect.StructField,
	elemFieldName string,
	errCollection *ValidationCollectionError,
) {
	switch elem.Kind() {
	case reflect.Struct:
		ve.validateStruct(elem, elem.Type(), elemFieldName, errCollection)
	case reflect.Ptr:
		ve.validateSlicePointerElement(elem, elemFieldName, errCollection)
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Array,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Slice, reflect.String, reflect.UnsafePointer:
		// All other types use element-specific validation rules
		ve.validateSliceElement(elem, field, elemFieldName, errCollection)
	default:
		// All other types use element-specific validation rules
		ve.validateSliceElement(elem, field, elemFieldName, errCollection)
	}
}

// validateSlicePointerElement validates pointer elements in slices.
func (ve *ValidationEngine) validateSlicePointerElement(
	elem reflect.Value,
	elemFieldName string,
	errCollection *ValidationCollectionError,
) {
	if !elem.IsNil() && elem.Elem().Kind() == reflect.Struct {
		ve.validateStruct(elem.Elem(), elem.Elem().Type(), elemFieldName, errCollection)
	}
}

// validateSliceElement validates individual slice elements.
func (ve *ValidationEngine) validateSliceElement(
	val reflect.Value,
	field *reflect.StructField,
	fieldName string,
	errCollection *ValidationCollectionError,
) {
	validateTag := field.Tag.Get(ve.TagName)
	if validateTag == "" {
		return
	}

	// Only apply validation rules that make sense for individual elements
	rules := ve.parseValidationRules(validateTag)

	// Filter out rules that should apply to the slice itself, not elements
	elementRules := make(map[string]string)
	for rule, param := range rules {
		switch rule {
		case validationRuleMin, validationRuleMax, validationRuleLen: // These apply to slice length, not elements
			continue
		default:
			elementRules[rule] = param
		}
	}

	if len(elementRules) > 0 {
		ve.applyValidationRules(val, elementRules, fieldName, errCollection)
	}
}

// setDefaults sets default values for fields that have default tags.
func (ve *ValidationEngine) setDefaults(
	val reflect.Value,
	field *reflect.StructField,
	fieldName string,
	errCollection *ValidationCollectionError,
) {
	defaultTag := field.Tag.Get(ve.DefaultTagName)
	if defaultTag == "" {
		return
	}

	// Only set defaults for zero values
	if !ve.isZeroValue(val) {
		return
	}

	if err := ve.setDefaultValue(val, defaultTag, fieldName); err != nil {
		errCollection.Add(fieldName, fmt.Sprintf("failed to set default value: %v", err), defaultTag)
	}
}

// validateField validates a single field using its validation tags.
func (ve *ValidationEngine) validateField(
	val reflect.Value,
	field *reflect.StructField,
	fieldName string,
	errCollection *ValidationCollectionError,
) {
	validateTag := field.Tag.Get(ve.TagName)
	if validateTag == "" {
		return
	}

	rules := ve.parseValidationRules(validateTag)

	// For slices, we need to handle validation differently
	if val.Kind() == reflect.Slice {
		// Apply slice-level validation (length constraints)
		sliceRules := make(map[string]string)
		for rule, param := range rules {
			switch rule {
			case validationRuleMin, validationRuleMax, validationRuleLen, validationRuleRequired:
				sliceRules[rule] = param
			}
		}

		if len(sliceRules) > 0 {
			ve.applyValidationRules(val, sliceRules, fieldName, errCollection)
		}

		// Element validation is handled in validateSlice function
		return
	}

	ve.applyValidationRules(val, rules, fieldName, errCollection)
}

// parseValidationRules parses validation rules from struct tag.
func (ve *ValidationEngine) parseValidationRules(tag string) map[string]string {
	rules := make(map[string]string)

	// Split by comma and parse each rule
	parts := strings.Split(tag, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check for rule with parameter (e.g., "min=5")
		if idx := strings.Index(part, "="); idx != -1 {
			key := strings.TrimSpace(part[:idx])
			value := strings.TrimSpace(part[idx+1:])
			rules[key] = value
		} else {
			// Rule without parameter (e.g., "required")
			rules[part] = ""
		}
	}

	return rules
}

// applyValidationRules applies validation rules to a field value.
func (ve *ValidationEngine) applyValidationRules(
	val reflect.Value,
	rules map[string]string,
	fieldName string,
	errCollection *ValidationCollectionError,
) {
	for rule, param := range rules {
		if err := ve.validateRule(val, rule, param, fieldName); err != nil {
			errCollection.Add(fieldName, err.Error(), val.Interface())

			if ve.StopOnFirstError {
				return
			}
		}
	}
}

// validateRule validates a single rule against a field value.
func (ve *ValidationEngine) validateRule(val reflect.Value, rule, param, fieldName string) error {
	// Handle basic validation rules
	if err := ve.validateBasicRule(val, rule, param, fieldName); err != nil {
		if err.Error() != unknownRuleError {
			return err
		}
	} else {
		return nil
	}

	// Handle parameter-based validation rules
	if err := ve.validateParameterRule(val, rule, param, fieldName); err != nil {
		if err.Error() != unknownRuleError {
			return err
		}
	} else {
		return nil
	}

	// Handle field-based validation rules
	if err := ve.validateFieldRule(val, rule, fieldName); err != nil {
		if err.Error() != unknownRuleError {
			return err
		}
	} else {
		return nil
	}

	return fmt.Errorf("unknown validation rule: %s", rule)
}

// validateBasicRule handles basic validation rules.
func (ve *ValidationEngine) validateBasicRule(val reflect.Value, rule, param, fieldName string) error {
	switch rule {
	case validationRuleRequired:
		return ve.validateRequired(val, fieldName)
	case validationRuleMin:
		return ve.validateMin(val, param, fieldName)
	case validationRuleMax:
		return ve.validateMax(val, param, fieldName)
	case validationRuleLen:
		return ve.validateLen(val, param, fieldName)
	default:
		return errors.New(unknownRuleError)
	}
}

// validateParameterRule handles validation rules that require parameters.
func (ve *ValidationEngine) validateParameterRule(val reflect.Value, rule, param, fieldName string) error {
	switch rule {
	case "regex":
		return ve.validateRegex(val, param, fieldName)
	case "oneof":
		return ve.validateOneOf(val, param, fieldName)
	default:
		return errors.New(unknownRuleError)
	}
}

// validateFieldRule handles validation rules that only need the field value.
func (ve *ValidationEngine) validateFieldRule(val reflect.Value, rule, fieldName string) error {
	switch rule {
	case "ip":
		return ve.validateIP(val, fieldName)
	case "port":
		return ve.validatePort(val, fieldName)
	case "file_exists":
		return ve.validateFileExists(val, fieldName)
	case "dir_exists":
		return ve.validateDirExists(val, fieldName)
	case "spiffe_id":
		return ve.validateSPIFFEID(val, fieldName)
	case "domain":
		return ve.validateDomain(val, fieldName)
	case "duration":
		return ve.validateDuration(val, fieldName)
	case "abs_path":
		return ve.validateAbsolutePath(val, fieldName)
	default:
		return errors.New(unknownRuleError)
	}
}

// Validation rule implementations

func (ve *ValidationEngine) validateRequired(val reflect.Value, _ string) error {
	// For struct types, we consider them valid if they are initialized (not nil for pointers)
	// The actual validation of their contents happens during recursive validation
	if val.Kind() == reflect.Struct {
		return nil // Structs are always considered "present" once initialized
	}

	if ve.isZeroValue(val) {
		return fmt.Errorf("field is required")
	}
	return nil
}

func (ve *ValidationEngine) validateMin(val reflect.Value, param, _ string) error {
	minVal, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("invalid min parameter: %s", param)
	}
	return ve.validateMinMax(val, minVal, true)
}

func (ve *ValidationEngine) validateMax(val reflect.Value, param, _ string) error {
	maxVal, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("invalid max parameter: %s", param)
	}
	return ve.validateMinMax(val, maxVal, false)
}

// validateMinMax is a shared helper for min/max validation to avoid code duplication.
func (ve *ValidationEngine) validateMinMax(val reflect.Value, limit int, isMin bool) error {
	switch val.Kind() {
	case reflect.String:
		return ve.validateStringMinMax(val.String(), limit, isMin, "characters long")
	case reflect.Slice, reflect.Array:
		return ve.validateLenMinMax(val.Len(), limit, isMin, "elements")
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return ve.validateIntMinMax(val.Int(), int64(limit), isMin)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return ve.validateUintMinMax(val.Uint(), uint64(limit), isMin)
	case reflect.Invalid, reflect.Bool, reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func, reflect.Interface,
		reflect.Map, reflect.Ptr, reflect.Struct, reflect.UnsafePointer:
		validationType := "min"
		if !isMin {
			validationType = validationRuleMax
		}
		return fmt.Errorf("%s validation not supported for type %s", validationType, val.Kind())
	default:
		validationType := "min"
		if !isMin {
			validationType = validationRuleMax
		}
		return fmt.Errorf("%s validation not supported for type %s", validationType, val.Kind())
	}
}

// validateStringMinMax validates string length against min/max limit.
func (ve *ValidationEngine) validateStringMinMax(str string, limit int, isMin bool, unit string) error {
	actualLen := len(str)
	if isMin && actualLen < limit {
		return fmt.Errorf("must be at least %d %s", limit, unit)
	}
	if !isMin && actualLen > limit {
		return fmt.Errorf("must be at most %d %s", limit, unit)
	}
	return nil
}

// validateLenMinMax validates length against min/max limit.
func (ve *ValidationEngine) validateLenMinMax(actualLen, limit int, isMin bool, unit string) error {
	if isMin && actualLen < limit {
		return fmt.Errorf("must have at least %d %s", limit, unit)
	}
	if !isMin && actualLen > limit {
		return fmt.Errorf("must have at most %d %s", limit, unit)
	}
	return nil
}

// validateIntMinMax validates signed integer against min/max limit.
func (ve *ValidationEngine) validateIntMinMax(actualVal, limitVal int64, isMin bool) error {
	if isMin && actualVal < limitVal {
		return fmt.Errorf("must be at least %d", limitVal)
	}
	if !isMin && actualVal > limitVal {
		return fmt.Errorf("must be at most %d", limitVal)
	}
	return nil
}

// validateUintMinMax validates unsigned integer against min/max limit.
func (ve *ValidationEngine) validateUintMinMax(actualVal, limitVal uint64, isMin bool) error {
	if isMin && actualVal < limitVal {
		return fmt.Errorf("must be at least %d", limitVal)
	}
	if !isMin && actualVal > limitVal {
		return fmt.Errorf("must be at most %d", limitVal)
	}
	return nil
}

func (ve *ValidationEngine) validateLen(val reflect.Value, param, _ string) error {
	expectedLen, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("invalid len parameter: %s", param)
	}

	switch val.Kind() {
	case reflect.String:
		if len(val.String()) != expectedLen {
			return fmt.Errorf("must be exactly %d characters long", expectedLen)
		}
	case reflect.Slice, reflect.Array:
		if val.Len() != expectedLen {
			return fmt.Errorf("must have exactly %d elements", expectedLen)
		}
	case reflect.Invalid, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Map, reflect.Ptr, reflect.Struct, reflect.UnsafePointer:
		return fmt.Errorf("len validation not supported for type %s", val.Kind())
	default:
		return fmt.Errorf("len validation not supported for type %s", val.Kind())
	}
	return nil
}

func (ve *ValidationEngine) validateRegex(val reflect.Value, param, _ string) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("regex validation only supported for strings")
	}

	regex, err := regexp.Compile(param)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %s", param)
	}

	if !regex.MatchString(val.String()) {
		return fmt.Errorf("must match pattern: %s", param)
	}
	return nil
}

func (ve *ValidationEngine) validateOneOf(val reflect.Value, param, _ string) error {
	options := strings.Split(param, "|")
	value := fmt.Sprintf("%v", val.Interface())

	for _, option := range options {
		if strings.TrimSpace(option) == value {
			return nil
		}
	}

	return fmt.Errorf("must be one of: %s", strings.Join(options, ", "))
}

func (ve *ValidationEngine) validateIP(val reflect.Value, _ string) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("IP validation only supported for strings")
	}

	ip := net.ParseIP(val.String())
	if ip == nil {
		return fmt.Errorf("must be a valid IP address")
	}
	return nil
}

func (ve *ValidationEngine) validatePort(val reflect.Value, _ string) error {
	var port int

	switch val.Kind() {
	case reflect.String:
		// Handle port in format ":8080" or "8080"
		portStr := strings.TrimPrefix(val.String(), ":")
		var err error
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return fmt.Errorf("must be a valid port number")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		port = int(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		port = int(val.Uint())
	case reflect.Invalid, reflect.Bool, reflect.Uintptr, reflect.Float32, reflect.Float64,
		reflect.Complex64, reflect.Complex128, reflect.Array, reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice, reflect.Struct, reflect.UnsafePointer:
		return fmt.Errorf("port validation not supported for type %s", val.Kind())
	default:
		return fmt.Errorf("port validation not supported for type %s", val.Kind())
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("must be a valid port number (1-65535)")
	}
	return nil
}

func (ve *ValidationEngine) validateFileExists(val reflect.Value, _ string) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("file_exists validation only supported for strings")
	}

	path := val.String()
	if path == "" {
		return nil // Empty paths are allowed unless required
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist: %s", path)
		}
		return fmt.Errorf("cannot access file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}

	return nil
}

func (ve *ValidationEngine) validateDirExists(val reflect.Value, _ string) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("dir_exists validation only supported for strings")
	}

	path := val.String()
	if path == "" {
		return nil // Empty paths are allowed unless required
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("directory does not exist: %s", path)
		}
		return fmt.Errorf("cannot access directory: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is a file, not a directory: %s", path)
	}

	return nil
}

func (ve *ValidationEngine) validateSPIFFEID(val reflect.Value, _ string) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("spiffe_id validation only supported for strings")
	}

	id := strings.TrimSpace(val.String())
	if id == "" {
		return nil // Empty SPIFFE IDs are allowed unless required
	}

	// Validate SPIFFE ID format
	if !strings.HasPrefix(id, "spiffe://") {
		return fmt.Errorf("SPIFFE ID must start with 'spiffe://' (e.g., 'spiffe://example.org/service')")
	}

	// Basic structure validation - must have trust domain and path
	parts := strings.SplitN(id[9:], "/", 2) // Remove "spiffe://" prefix
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("SPIFFE ID must have format 'spiffe://trust-domain/path' (e.g., 'spiffe://example.org/service')")
	}

	return nil
}

func (ve *ValidationEngine) validateDomain(val reflect.Value, _ string) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("domain validation only supported for strings")
	}

	domain := strings.TrimSpace(val.String())
	if domain == "" {
		return nil // Empty domains are allowed unless required
	}

	// Basic domain validation - contains dots and valid characters
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("must be a valid domain name (e.g., 'example.org')")
	}

	// Additional domain format validation
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	if !domainRegex.MatchString(domain) {
		return fmt.Errorf("must be a valid domain name format")
	}

	return nil
}

func (ve *ValidationEngine) validateDuration(val reflect.Value, _ string) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("duration validation only supported for strings")
	}

	duration := val.String()
	if duration == "" {
		return nil // Empty durations are allowed unless required
	}

	_, err := time.ParseDuration(duration)
	if err != nil {
		return fmt.Errorf("must be a valid duration (e.g., '10s', '5m', '1h')")
	}

	return nil
}

func (ve *ValidationEngine) validateAbsolutePath(val reflect.Value, _ string) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("abs_path validation only supported for strings")
	}

	path := val.String()
	if path == "" {
		return nil // Empty paths are allowed unless required
	}

	// Check for null bytes (security risk)
	if strings.Contains(path, "\x00") {
		return fmt.Errorf("path contains null bytes")
	}

	// Check for control characters including newlines (security risk)
	for _, r := range path {
		if r < 32 || r == 127 { // ASCII control characters
			return fmt.Errorf("path contains control characters")
		}
	}

	// Check if path is absolute
	if !filepath.IsAbs(path) {
		return fmt.Errorf("must be an absolute path")
	}

	// Additional security checks for socket paths
	// Reject paths that try to escape with relative components
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return fmt.Errorf("path contains unsafe components")
	}

	return nil
}

// Helper functions

func (ve *ValidationEngine) buildFieldName(prefix string, field *reflect.StructField) string {
	// Use yaml tag name if available, otherwise use field name
	yamlTag := field.Tag.Get("yaml")
	if yamlTag != "" && yamlTag != "-" {
		// Extract just the field name from yaml tag (ignore options like omitempty)
		if idx := strings.Index(yamlTag, ","); idx != -1 {
			yamlTag = yamlTag[:idx]
		}
		if yamlTag != "" {
			if prefix != "" {
				return prefix + "." + yamlTag
			}
			return yamlTag
		}
	}

	fieldName := field.Name
	if prefix != "" {
		return prefix + "." + fieldName
	}
	return fieldName
}

func (ve *ValidationEngine) isZeroValue(val reflect.Value) bool {
	switch val.Kind() {
	case reflect.String:
		return val.String() == ""
	case reflect.Bool:
		return !val.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return val.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return val.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return val.Float() == 0
	case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
		return val.IsNil()
	case reflect.Array, reflect.Struct:
		return ve.isZeroCompoundValue(val)
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128, reflect.UnsafePointer:
		return false
	default:
		return false
	}
}

// isZeroCompoundValue checks if compound values (arrays, structs) are zero.
func (ve *ValidationEngine) isZeroCompoundValue(val reflect.Value) bool {
	zero := reflect.Zero(val.Type())
	return reflect.DeepEqual(val.Interface(), zero.Interface())
}

func (ve *ValidationEngine) setDefaultValue(val reflect.Value, defaultValue, fieldName string) error {
	if !val.CanSet() {
		return fmt.Errorf("cannot set field %s", fieldName)
	}

	switch val.Kind() {
	case reflect.String:
		val.SetString(defaultValue)
	case reflect.Bool:
		return ve.setBoolDefault(val, defaultValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return ve.setIntDefault(val, defaultValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return ve.setUintDefault(val, defaultValue)
	case reflect.Float32, reflect.Float64:
		return ve.setFloatDefault(val, defaultValue)
	case reflect.Slice:
		return ve.setSliceDefault(val, defaultValue, fieldName)
	case reflect.Invalid, reflect.Uintptr, reflect.Complex64, reflect.Complex128, reflect.Array,
		reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Struct, reflect.UnsafePointer:
		return fmt.Errorf("default value setting not supported for type %s", val.Kind())
	default:
		return fmt.Errorf("default value setting not supported for type %s", val.Kind())
	}

	return nil
}

// setBoolDefault sets a boolean default value.
func (ve *ValidationEngine) setBoolDefault(val reflect.Value, defaultValue string) error {
	boolVal, err := strconv.ParseBool(defaultValue)
	if err != nil {
		return fmt.Errorf("invalid bool default value '%s': %w", defaultValue, err)
	}
	val.SetBool(boolVal)
	return nil
}

// setIntDefault sets an integer default value.
func (ve *ValidationEngine) setIntDefault(val reflect.Value, defaultValue string) error {
	intVal, err := strconv.ParseInt(defaultValue, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid int default value '%s': %w", defaultValue, err)
	}
	val.SetInt(intVal)
	return nil
}

// setUintDefault sets an unsigned integer default value.
func (ve *ValidationEngine) setUintDefault(val reflect.Value, defaultValue string) error {
	uintVal, err := strconv.ParseUint(defaultValue, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid uint default value '%s': %w", defaultValue, err)
	}
	val.SetUint(uintVal)
	return nil
}

// setFloatDefault sets a float default value.
func (ve *ValidationEngine) setFloatDefault(val reflect.Value, defaultValue string) error {
	floatVal, err := strconv.ParseFloat(defaultValue, 64)
	if err != nil {
		return fmt.Errorf("invalid float default value '%s': %w", defaultValue, err)
	}
	val.SetFloat(floatVal)
	return nil
}

// setSliceDefault sets a slice default value from comma-separated string.
func (ve *ValidationEngine) setSliceDefault(val reflect.Value, defaultValue, fieldName string) error {
	if defaultValue == "" {
		return nil
	}

	parts := strings.Split(defaultValue, ",")
	slice := reflect.MakeSlice(val.Type(), len(parts), len(parts))
	for i, part := range parts {
		elem := slice.Index(i)
		if err := ve.setDefaultValue(elem, strings.TrimSpace(part), fmt.Sprintf("%s[%d]", fieldName, i)); err != nil {
			return err
		}
	}
	val.Set(slice)
	return nil
}

// GlobalValidationEngine provides a default validation engine instance.
//
//nolint:gochecknoglobals // Global instance for convenience
var GlobalValidationEngine = NewValidationEngine()

// ValidateStruct is a convenience function that uses the global validation engine.
func ValidateStruct(v any) error {
	return GlobalValidationEngine.ValidateAndSetDefaults(v)
}

// ValidateStructWithEngine validates a struct with a custom validation engine.
func ValidateStructWithEngine(v any, engine *ValidationEngine) error {
	return engine.ValidateAndSetDefaults(v)
}

// GetValidationErrors extracts all validation errors from an error.
func GetValidationErrors(err error) []ValidationError {
	var validationErr *ValidationError
	var collectionErr *ValidationCollectionError

	if errors.As(err, &collectionErr) {
		return collectionErr.Errors
	}

	if errors.As(err, &validationErr) {
		return []ValidationError{*validationErr}
	}

	return nil
}

// initializeRequiredNestedStruct initializes nested structs that are marked as required.
func (ve *ValidationEngine) initializeRequiredNestedStruct(
	fieldVal reflect.Value,
	field *reflect.StructField,
	fieldName string,
	errCollection *ValidationCollectionError,
) {
	if !fieldVal.CanSet() {
		return
	}

	// Check if this field is required
	validateTag := field.Tag.Get(ve.TagName)
	if validateTag == "" {
		return
	}

	rules := ve.parseValidationRules(validateTag)
	if _, isRequired := rules[validationRuleRequired]; !isRequired {
		return
	}

	// Initialize nested struct if it's zero and a struct type
	if fieldVal.Kind() == reflect.Struct && ve.isZeroValue(fieldVal) {
		// Create a new instance of the struct type
		newStruct := reflect.New(fieldVal.Type()).Elem()
		fieldVal.Set(newStruct)
	}
}
