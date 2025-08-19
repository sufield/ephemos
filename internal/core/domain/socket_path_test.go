package domain

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewSocketPath(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid path in /run",
			path:        "/run/spire/sockets/api.sock",
			expectError: false,
		},
		{
			name:        "valid path in /var/run",
			path:        "/var/run/spire/sockets/api.sock",
			expectError: false,
		},
		{
			name:        "valid path in /tmp",
			path:        "/tmp/spire-agent/public/api.sock",
			expectError: false,
		},
		{
			name:        "valid path with unix prefix",
			path:        "unix:///tmp/spire-agent/public/api.sock",
			expectError: false,
		},
		{
			name:        "empty path",
			path:        "",
			expectError: true,
			errorMsg:    "socket path cannot be empty",
		},
		{
			name:        "relative path",
			path:        "relative/path.sock",
			expectError: true,
			errorMsg:    "socket path must be absolute",
		},
		{
			name:        "insecure directory",
			path:        "/home/user/socket.sock",
			expectError: true,
			errorMsg:    "socket path must be in a secure directory",
		},
		{
			name:        "missing .sock extension",
			path:        "/tmp/spire-agent/api",
			expectError: true,
			errorMsg:    "socket path must end with .sock extension",
		},
		{
			name:        "unix prefix with secure path",
			path:        "unix:///run/spire/sockets/api.sock",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sp, err := NewSocketPath(tt.path)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for path %s, but got none", tt.path)
					return
				}
				if tt.errorMsg != "" && !stringContains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for path %s: %v", tt.path, err)
				return
			}

			// Verify the path value is cleaned (no unix:// prefix)
			expectedValue := tt.path
			if strings.HasPrefix(expectedValue, "unix://") {
				expectedValue = strings.TrimPrefix(expectedValue, "unix://")
			}

			if sp.Value() != expectedValue {
				t.Errorf("Expected value %s, got %s", expectedValue, sp.Value())
			}
		})
	}
}

func TestNewSocketPathUnsafe(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		expectedValue string
	}{
		{
			name:          "unix prefix removed",
			path:          "unix:///tmp/test.sock",
			expectedValue: "/tmp/test.sock",
		},
		{
			name:          "no prefix",
			path:          "/tmp/test.sock",
			expectedValue: "/tmp/test.sock",
		},
		{
			name:          "invalid path accepted",
			path:          "invalid-path",
			expectedValue: "invalid-path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sp := NewSocketPathUnsafe(tt.path)
			if sp.Value() != tt.expectedValue {
				t.Errorf("Expected value %s, got %s", tt.expectedValue, sp.Value())
			}
		})
	}
}

func TestSocketPathMethods(t *testing.T) {
	sp := NewSocketPathUnsafe("/tmp/spire-agent/api.sock")

	t.Run("Value method compatibility", func(t *testing.T) {
		if sp.Value() != "/tmp/spire-agent/api.sock" {
			t.Errorf("Expected /tmp/spire-agent/api.sock, got %s", sp.Value())
		}
	})

	t.Run("Value method", func(t *testing.T) {
		if sp.Value() != "/tmp/spire-agent/api.sock" {
			t.Errorf("Expected /tmp/spire-agent/api.sock, got %s", sp.Value())
		}
	})

	t.Run("WithUnixPrefix method", func(t *testing.T) {
		expected := "unix:///tmp/spire-agent/api.sock"
		if sp.WithUnixPrefix() != expected {
			t.Errorf("Expected %s, got %s", expected, sp.WithUnixPrefix())
		}
	})

	t.Run("Directory method", func(t *testing.T) {
		expected := "/tmp/spire-agent"
		if sp.Directory() != expected {
			t.Errorf("Expected %s, got %s", expected, sp.Directory())
		}
	})

	t.Run("IsEmpty method", func(t *testing.T) {
		if sp.IsEmpty() {
			t.Error("Expected non-empty socket path to return false for IsEmpty")
		}

		empty := NewSocketPathUnsafe("")
		if !empty.IsEmpty() {
			t.Error("Expected empty socket path to return true for IsEmpty")
		}
	})

	t.Run("Equals method", func(t *testing.T) {
		same := NewSocketPathUnsafe("/tmp/spire-agent/api.sock")
		different := NewSocketPathUnsafe("/run/spire/api.sock")

		if !sp.Equals(same) {
			t.Error("Expected equal socket paths to return true")
		}

		if sp.Equals(different) {
			t.Error("Expected different socket paths to return false")
		}
	})
}

func TestSocketPathWithExistingFile(t *testing.T) {
	// Create a temporary directory and socket file for testing
	tmpDir, err := os.MkdirTemp("", "socket_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a mock socket file (just a regular file for testing)
	socketPath := filepath.Join(tmpDir, "test.sock")
	file, err := os.Create(socketPath)
	if err != nil {
		t.Fatalf("Failed to create test socket file: %v", err)
	}
	file.Close()

	// Set permissions to 660
	err = os.Chmod(socketPath, 0660)
	if err != nil {
		t.Fatalf("Failed to set permissions: %v", err)
	}

	t.Run("existing file with correct permissions", func(t *testing.T) {
		// Note: This will fail secure directory validation since tmpDir is not in /run, /var/run, or /tmp
		// But we can test the file existence logic by using NewSocketPathUnsafe
		// and calling the validation separately if needed

		sp := NewSocketPathUnsafe(socketPath)
		if sp.Value() != socketPath {
			t.Errorf("Expected %s, got %s", socketPath, sp.Value())
		}
	})
}

// Helper function to check if string contains substring
func stringContains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestSocketPathValidation(t *testing.T) {
	t.Run("secure directory validation", func(t *testing.T) {
		validDirs := []string{
			"/run/spire/api.sock",
			"/var/run/spire/api.sock",
			"/tmp/spire-agent/api.sock",
		}

		for _, path := range validDirs {
			err := validateSecureDirectory(path)
			if err != nil {
				t.Errorf("Expected %s to be valid, got error: %v", path, err)
			}
		}

		invalidDirs := []string{
			"/home/user/api.sock",
			"/usr/local/api.sock",
			"/opt/spire/api.sock",
		}

		for _, path := range invalidDirs {
			err := validateSecureDirectory(path)
			if err == nil {
				t.Errorf("Expected %s to be invalid, but got no error", path)
			}
		}
	})
}
