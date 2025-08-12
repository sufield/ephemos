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

// ValidationErrorCollection aggregates multiple validation errors.
type ValidationErrorCollection struct {
	Errors []ValidationError
}

// Error implements the error interface, returning a summary of all validation errors.
func (vec *ValidationErrorCollection) Error() string {
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
func (vec *ValidationErrorCollection) Add(field, message string, value any) {
	vec.Errors = append(vec.Errors, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if there are any validation errors.
func (vec *ValidationErrorCollection) HasErrors() bool {
	return len(vec.Errors) > 0
}

// GetFieldErrors returns all errors for a specific field.
func (vec *ValidationErrorCollection) GetFieldErrors(field string) []ValidationError {
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

	errCollection := &ValidationErrorCollection{}
	ve.validateStruct(elem, elem.Type(), "", errCollection)

	if errCollection.HasErrors() {
		return errCollection
	}

	return nil
}

// validateStruct recursively validates a struct and its nested structs.
func (ve *ValidationEngine) validateStruct(val reflect.Value, typ reflect.Type, prefix string, errCollection *ValidationErrorCollection) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Skip unexported fields
		if !fieldVal.CanSet() {
			continue
		}

		fieldName := ve.buildFieldName(prefix, field)

		// Set defaults first
		ve.setDefaults(fieldVal, field, fieldName, errCollection)

		// Validate field
		ve.validateField(fieldVal, field, fieldName, errCollection)

		// Stop early if configured to do so and we have errors
		if ve.StopOnFirstError && errCollection.HasErrors() {
			return
		}

		// Recursively validate nested structs
		if fieldVal.Kind() == reflect.Struct {
			ve.validateStruct(fieldVal, fieldVal.Type(), fieldName, errCollection)
		} else if fieldVal.Kind() == reflect.Ptr && !fieldVal.IsNil() && fieldVal.Elem().Kind() == reflect.Struct {
			ve.validateStruct(fieldVal.Elem(), fieldVal.Elem().Type(), fieldName, errCollection)
		} else if fieldVal.Kind() == reflect.Slice {
			ve.validateSlice(fieldVal, field, fieldName, errCollection)
		}
	}
}

// validateSlice validates slice and its elements.
func (ve *ValidationEngine) validateSlice(val reflect.Value, field reflect.StructField, fieldName string, errCollection *ValidationErrorCollection) {
	// First validate the slice itself (length constraints)
	validateTag := field.Tag.Get(ve.TagName)
	if validateTag != "" {
		rules := ve.parseValidationRules(validateTag)

		// Apply slice-level validation rules
		sliceRules := make(map[string]string)
		for rule, param := range rules {
			switch rule {
			case "min", "max", "len", "required":
				sliceRules[rule] = param
			}
		}

		if len(sliceRules) > 0 {
			ve.applyValidationRules(val, sliceRules, fieldName, errCollection)
		}
	}

	// Then validate each element
	for i := 0; i < val.Len(); i++ {
		elem := val.Index(i)
		elemFieldName := fmt.Sprintf("%s[%d]", fieldName, i)

		if elem.Kind() == reflect.Struct {
			ve.validateStruct(elem, elem.Type(), elemFieldName, errCollection)
		} else if elem.Kind() == reflect.Ptr && !elem.IsNil() && elem.Elem().Kind() == reflect.Struct {
			ve.validateStruct(elem.Elem(), elem.Elem().Type(), elemFieldName, errCollection)
		} else {
			// Validate slice element using element-specific rules
			ve.validateSliceElement(elem, field, elemFieldName, errCollection)
		}
	}
}

// validateSliceElement validates individual slice elements.
func (ve *ValidationEngine) validateSliceElement(val reflect.Value, field reflect.StructField, fieldName string, errCollection *ValidationErrorCollection) {
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
		case "min", "max", "len": // These apply to slice length, not elements
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
func (ve *ValidationEngine) setDefaults(val reflect.Value, field reflect.StructField, fieldName string, errCollection *ValidationErrorCollection) {
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
func (ve *ValidationEngine) validateField(val reflect.Value, field reflect.StructField, fieldName string, errCollection *ValidationErrorCollection) {
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
			case "min", "max", "len", "required":
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
func (ve *ValidationEngine) applyValidationRules(val reflect.Value, rules map[string]string, fieldName string, errCollection *ValidationErrorCollection) {
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
	switch rule {
	case "required":
		return ve.validateRequired(val, fieldName)
	case "min":
		return ve.validateMin(val, param, fieldName)
	case "max":
		return ve.validateMax(val, param, fieldName)
	case "len":
		return ve.validateLen(val, param, fieldName)
	case "regex":
		return ve.validateRegex(val, param, fieldName)
	case "oneof":
		return ve.validateOneOf(val, param, fieldName)
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
		return fmt.Errorf("unknown validation rule: %s", rule)
	}
}

// Validation rule implementations

func (ve *ValidationEngine) validateRequired(val reflect.Value, fieldName string) error {
	if ve.isZeroValue(val) {
		return fmt.Errorf("field is required")
	}
	return nil
}

func (ve *ValidationEngine) validateMin(val reflect.Value, param, fieldName string) error {
	minVal, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("invalid min parameter: %s", param)
	}

	switch val.Kind() {
	case reflect.String:
		if len(val.String()) < minVal {
			return fmt.Errorf("must be at least %d characters long", minVal)
		}
	case reflect.Slice, reflect.Array:
		if val.Len() < minVal {
			return fmt.Errorf("must have at least %d elements", minVal)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() < int64(minVal) {
			return fmt.Errorf("must be at least %d", minVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val.Uint() < uint64(minVal) {
			return fmt.Errorf("must be at least %d", minVal)
		}
	default:
		return fmt.Errorf("min validation not supported for type %s", val.Kind())
	}
	return nil
}

func (ve *ValidationEngine) validateMax(val reflect.Value, param, fieldName string) error {
	maxVal, err := strconv.Atoi(param)
	if err != nil {
		return fmt.Errorf("invalid max parameter: %s", param)
	}

	switch val.Kind() {
	case reflect.String:
		if len(val.String()) > maxVal {
			return fmt.Errorf("must be at most %d characters long", maxVal)
		}
	case reflect.Slice, reflect.Array:
		if val.Len() > maxVal {
			return fmt.Errorf("must have at most %d elements", maxVal)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() > int64(maxVal) {
			return fmt.Errorf("must be at most %d", maxVal)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val.Uint() > uint64(maxVal) {
			return fmt.Errorf("must be at most %d", maxVal)
		}
	default:
		return fmt.Errorf("max validation not supported for type %s", val.Kind())
	}
	return nil
}

func (ve *ValidationEngine) validateLen(val reflect.Value, param, fieldName string) error {
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
	default:
		return fmt.Errorf("len validation not supported for type %s", val.Kind())
	}
	return nil
}

func (ve *ValidationEngine) validateRegex(val reflect.Value, param, fieldName string) error {
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

func (ve *ValidationEngine) validateOneOf(val reflect.Value, param, fieldName string) error {
	options := strings.Split(param, "|")
	value := fmt.Sprintf("%v", val.Interface())

	for _, option := range options {
		if strings.TrimSpace(option) == value {
			return nil
		}
	}

	return fmt.Errorf("must be one of: %s", strings.Join(options, ", "))
}

func (ve *ValidationEngine) validateIP(val reflect.Value, fieldName string) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("IP validation only supported for strings")
	}

	ip := net.ParseIP(val.String())
	if ip == nil {
		return fmt.Errorf("must be a valid IP address")
	}
	return nil
}

func (ve *ValidationEngine) validatePort(val reflect.Value, fieldName string) error {
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
	default:
		return fmt.Errorf("port validation not supported for type %s", val.Kind())
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("must be a valid port number (1-65535)")
	}
	return nil
}

func (ve *ValidationEngine) validateFileExists(val reflect.Value, fieldName string) error {
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
		return fmt.Errorf("cannot access file: %v", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not a file: %s", path)
	}

	return nil
}

func (ve *ValidationEngine) validateDirExists(val reflect.Value, fieldName string) error {
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
		return fmt.Errorf("cannot access directory: %v", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is a file, not a directory: %s", path)
	}

	return nil
}

func (ve *ValidationEngine) validateSPIFFEID(val reflect.Value, fieldName string) error {
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

func (ve *ValidationEngine) validateDomain(val reflect.Value, fieldName string) error {
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

func (ve *ValidationEngine) validateDuration(val reflect.Value, fieldName string) error {
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

func (ve *ValidationEngine) validateAbsolutePath(val reflect.Value, fieldName string) error {
	if val.Kind() != reflect.String {
		return fmt.Errorf("abs_path validation only supported for strings")
	}

	path := val.String()
	if path == "" {
		return nil // Empty paths are allowed unless required
	}

	if !filepath.IsAbs(path) {
		return fmt.Errorf("must be an absolute path")
	}

	return nil
}

// Helper functions

func (ve *ValidationEngine) buildFieldName(prefix string, field reflect.StructField) string {
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
	case reflect.Array:
		zero := reflect.Zero(val.Type())
		return reflect.DeepEqual(val.Interface(), zero.Interface())
	case reflect.Struct:
		zero := reflect.Zero(val.Type())
		return reflect.DeepEqual(val.Interface(), zero.Interface())
	default:
		return false
	}
}

func (ve *ValidationEngine) setDefaultValue(val reflect.Value, defaultValue, fieldName string) error {
	if !val.CanSet() {
		return fmt.Errorf("cannot set field %s", fieldName)
	}

	switch val.Kind() {
	case reflect.String:
		val.SetString(defaultValue)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(defaultValue)
		if err != nil {
			return fmt.Errorf("invalid bool default value '%s': %v", defaultValue, err)
		}
		val.SetBool(boolVal)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(defaultValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int default value '%s': %v", defaultValue, err)
		}
		val.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(defaultValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid uint default value '%s': %v", defaultValue, err)
		}
		val.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(defaultValue, 64)
		if err != nil {
			return fmt.Errorf("invalid float default value '%s': %v", defaultValue, err)
		}
		val.SetFloat(floatVal)
	case reflect.Slice:
		// Handle comma-separated values for slices
		if defaultValue != "" {
			parts := strings.Split(defaultValue, ",")
			slice := reflect.MakeSlice(val.Type(), len(parts), len(parts))
			for i, part := range parts {
				elem := slice.Index(i)
				if err := ve.setDefaultValue(elem, strings.TrimSpace(part), fmt.Sprintf("%s[%d]", fieldName, i)); err != nil {
					return err
				}
			}
			val.Set(slice)
		}
	default:
		return fmt.Errorf("default value setting not supported for type %s", val.Kind())
	}

	return nil
}

// GlobalValidationEngine provides a default validation engine instance.
var GlobalValidationEngine = NewValidationEngine()

// ValidateStruct is a convenience function that uses the global validation engine.
func ValidateStruct(v any) error {
	return GlobalValidationEngine.ValidateAndSetDefaults(v)
}

// ValidateStructWithEngine validates a struct with a custom validation engine.
func ValidateStructWithEngine(v any, engine *ValidationEngine) error {
	return engine.ValidateAndSetDefaults(v)
}

// IsValidationError checks if an error is a validation error.
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	var collectionErr *ValidationErrorCollection
	return errors.As(err, &validationErr) || errors.As(err, &collectionErr)
}

// GetValidationErrors extracts all validation errors from an error.
func GetValidationErrors(err error) []ValidationError {
	var validationErr *ValidationError
	var collectionErr *ValidationErrorCollection

	if errors.As(err, &collectionErr) {
		return collectionErr.Errors
	}

	if errors.As(err, &validationErr) {
		return []ValidationError{*validationErr}
	}

	return nil
}

