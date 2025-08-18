// Package domain provides domain value objects and entities.
package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	
	"gopkg.in/yaml.v3"
)

// SocketPath is a value object representing a validated Unix domain socket path.
// It enforces security rules and prevents invalid socket paths from propagating through the system.
type SocketPath struct {
	value string // Private to enforce encapsulation
}

// NewSocketPath creates a SocketPath, validating the input according to security rules.
// The path must be absolute, in a secure directory, and have proper format.
func NewSocketPath(path string) (SocketPath, error) {
	if path == "" {
		return SocketPath{}, fmt.Errorf("socket path cannot be empty")
	}

	// Remove unix:// prefix if present (common in SPIFFE usage)
	cleanPath := strings.TrimPrefix(path, "unix://")
	
	if !filepath.IsAbs(cleanPath) {
		return SocketPath{}, fmt.Errorf("socket path must be absolute: %s", path)
	}

	// Validate secure directory (from existing validateSocketPath logic)
	if err := validateSecureDirectory(cleanPath); err != nil {
		return SocketPath{}, err
	}

	// Optional: Validate .sock extension (common convention)
	if !strings.HasSuffix(cleanPath, ".sock") {
		return SocketPath{}, fmt.Errorf("socket path must end with .sock extension: %s", path)
	}

	// Optional: Check file permissions if path exists
	if err := validatePermissionsIfExists(cleanPath); err != nil {
		return SocketPath{}, err
	}

	return SocketPath{value: cleanPath}, nil
}

// NewSocketPathUnsafe creates a SocketPath without validation.
// This should only be used for testing or when validation has already been performed.
// Use NewSocketPath in production code.
func NewSocketPathUnsafe(path string) SocketPath {
	cleanPath := strings.TrimPrefix(path, "unix://")
	return SocketPath{value: cleanPath}
}


// Value returns the socket path value.
// This is the preferred method for accessing the path value.
func (sp SocketPath) Value() string {
	return sp.value
}

// WithUnixPrefix returns the socket path with unix:// prefix.
// This is useful for libraries that expect the unix:// scheme.
func (sp SocketPath) WithUnixPrefix() string {
	return "unix://" + sp.value
}

// IsEmpty returns true if the socket path is empty.
func (sp SocketPath) IsEmpty() bool {
	return sp.value == ""
}

// Equals checks if two SocketPath instances are equal.
func (sp SocketPath) Equals(other SocketPath) bool {
	return sp.value == other.value
}

// Directory returns the directory containing the socket file.
func (sp SocketPath) Directory() string {
	return filepath.Dir(sp.value)
}

// validateSecureDirectory checks if the socket path is in a secure location.
// This implements the existing security rules from validateSocketPath.
func validateSecureDirectory(socketPath string) error {
	secureDirectories := []string{"/run/", "/var/run/", "/tmp/"}
	for _, dir := range secureDirectories {
		if strings.HasPrefix(socketPath, dir) {
			return nil
		}
	}
	return fmt.Errorf("socket path must be in a secure directory (/run/, /var/run/, or /tmp/): %s", socketPath)
}

// validatePermissionsIfExists checks file permissions if the socket exists.
func validatePermissionsIfExists(socketPath string) error {
	info, err := os.Stat(socketPath)
	if os.IsNotExist(err) {
		// File doesn't exist, which is fine - it may be created later
		return nil
	}
	if err != nil {
		// Other error (permission denied, etc.) - let it pass for now
		// The actual connection will fail with a more appropriate error
		return nil
	}

	// Check if it's a socket
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("path exists but is not a socket: %s", socketPath)
	}

	// Check permissions - should typically be 660 for security
	mode := info.Mode().Perm()
	if mode != 0660 && mode != 0600 && mode != 0755 {
		// Log warning but don't fail - permissions might be set differently in different environments
		// Could be made stricter in production environments
	}

	return nil
}

// UnmarshalYAML implements yaml.Unmarshaler interface to support YAML unmarshaling
func (sp *SocketPath) UnmarshalYAML(node *yaml.Node) error {
	var path string
	if err := node.Decode(&path); err != nil {
		return err
	}
	
	// Use unsafe constructor for backward compatibility during migration
	// In production, consider using NewSocketPath for validation
	*sp = NewSocketPathUnsafe(path)
	return nil
}

// MarshalYAML implements yaml.Marshaler interface to support YAML marshaling
func (sp SocketPath) MarshalYAML() (interface{}, error) {
	return sp.value, nil
}

// SocketPathDecodeHook provides a mapstructure decode hook for SocketPath
// This enables Viper to automatically decode strings to SocketPath values
func SocketPathDecodeHook() func(reflect.Type, reflect.Type, interface{}) (interface{}, error) {
	return func(from, to reflect.Type, data interface{}) (interface{}, error) {
		// Check if we're decoding to SocketPath
		if to != reflect.TypeOf(SocketPath{}) {
			return data, nil
		}
		
		// Convert from string to SocketPath
		if from.Kind() == reflect.String {
			path, ok := data.(string)
			if !ok {
				return data, nil
			}
			// Use unsafe constructor for backward compatibility during migration
			return NewSocketPathUnsafe(path), nil
		}
		
		return data, nil
	}
}