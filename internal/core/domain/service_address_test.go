package domain

import (
	"testing"
)

func TestNewServiceAddress(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid host:port address",
			input:     "api.service.com:8080",
			wantError: false,
		},
		{
			name:      "valid IP:port address",
			input:     "192.168.1.100:3000",
			wantError: false,
		},
		{
			name:      "valid HTTPS URL",
			input:     "https://api.service.com:8080/v1",
			wantError: false,
		},
		{
			name:      "valid HTTP URL",
			input:     "http://localhost:8080",
			wantError: false,
		},
		{
			name:      "valid hostname only",
			input:     "api.service.local",
			wantError: false,
		},
		{
			name:      "valid IP address only",
			input:     "10.0.0.1",
			wantError: false,
		},
		{
			name:      "empty string",
			input:     "",
			wantError: true,
			errorMsg:  "service address cannot be empty or whitespace-only",
		},
		{
			name:      "whitespace only",
			input:     "   ",
			wantError: true,
			errorMsg:  "service address cannot be empty or whitespace-only",
		},
		{
			name:      "too long",
			input:     "https://" + generateLongString(500) + ".com",
			wantError: true,
			errorMsg:  "service address too long",
		},
		{
			name:      "example.com domain",
			input:     "api.example.com:8080",
			wantError: true,
			errorMsg:  "service address cannot use example domains",
		},
		{
			name:      "example.org domain",
			input:     "https://service.example.org",
			wantError: true,
			errorMsg:  "service address cannot use example domains",
		},
		{
			name:      "invalid port range",
			input:     "api.service.com:99999",
			wantError: true,
			errorMsg:  "port number must be between 1 and 65535",
		},
		{
			name:      "zero port",
			input:     "api.service.com:0",
			wantError: true,
			errorMsg:  "port number must be between 1 and 65535",
		},
		{
			name:      "invalid hostname characters",
			input:     "api_service.com:8080",
			wantError: true,
			errorMsg:  "invalid host:port format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewServiceAddress(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("NewServiceAddress(%q) expected error, got nil", tt.input)
					return
				}
				if tt.errorMsg != "" && !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("NewServiceAddress(%q) error = %v, expected to contain %q", tt.input, err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("NewServiceAddress(%q) unexpected error: %v", tt.input, err)
					return
				}
				if result.Value() != tt.input {
					t.Errorf("NewServiceAddress(%q) value = %q, expected %q", tt.input, result.Value(), tt.input)
				}
			}
		})
	}
}

func TestServiceAddressMethods(t *testing.T) {
	// Test host:port format
	hostPort, err := NewServiceAddress("api.service.com:8080")
	if err != nil {
		t.Fatalf("Failed to create ServiceAddress: %v", err)
	}

	// Test String() method
	if hostPort.String() != "api.service.com:8080" {
		t.Errorf("String() = %q, expected %q", hostPort.String(), "api.service.com:8080")
	}

	// Test Value() method
	if hostPort.Value() != "api.service.com:8080" {
		t.Errorf("Value() = %q, expected %q", hostPort.Value(), "api.service.com:8080")
	}

	// Test IsEmpty() method
	if hostPort.IsEmpty() {
		t.Errorf("IsEmpty() = true, expected false")
	}

	// Test format detection methods
	if hostPort.IsURL() {
		t.Errorf("IsURL() = true, expected false for host:port")
	}
	if !hostPort.IsHostPort() {
		t.Errorf("IsHostPort() = false, expected true")
	}
	if hostPort.IsHostnameOnly() {
		t.Errorf("IsHostnameOnly() = true, expected false for host:port")
	}

	// Test GetHost() method
	host, err := hostPort.GetHost()
	if err != nil {
		t.Errorf("GetHost() error: %v", err)
	}
	if host != "api.service.com" {
		t.Errorf("GetHost() = %q, expected %q", host, "api.service.com")
	}

	// Test GetPort() method
	port, err := hostPort.GetPort()
	if err != nil {
		t.Errorf("GetPort() error: %v", err)
	}
	if port != 8080 {
		t.Errorf("GetPort() = %d, expected 8080", port)
	}

	// Test Equals() method
	other, _ := NewServiceAddress("api.service.com:8080")
	if !hostPort.Equals(other) {
		t.Errorf("Equals() = false, expected true for identical addresses")
	}

	different, _ := NewServiceAddress("api.service.com:3000")
	if hostPort.Equals(different) {
		t.Errorf("Equals() = true, expected false for different addresses")
	}
}

func TestServiceAddressURLMethods(t *testing.T) {
	// Test HTTPS URL
	httpsURL, err := NewServiceAddress("https://api.service.com:8080/v1")
	if err != nil {
		t.Fatalf("Failed to create ServiceAddress: %v", err)
	}

	if !httpsURL.IsURL() {
		t.Errorf("IsURL() = false, expected true for HTTPS URL")
	}
	if httpsURL.IsHostPort() {
		t.Errorf("IsHostPort() = true, expected false for URL")
	}
	if httpsURL.IsHostnameOnly() {
		t.Errorf("IsHostnameOnly() = true, expected false for URL")
	}
	if !httpsURL.IsSecure() {
		t.Errorf("IsSecure() = false, expected true for HTTPS")
	}

	// Test GetHost() for URL
	host, err := httpsURL.GetHost()
	if err != nil {
		t.Errorf("GetHost() error: %v", err)
	}
	if host != "api.service.com" {
		t.Errorf("GetHost() = %q, expected %q", host, "api.service.com")
	}

	// Test GetPort() for URL
	port, err := httpsURL.GetPort()
	if err != nil {
		t.Errorf("GetPort() error: %v", err)
	}
	if port != 8080 {
		t.Errorf("GetPort() = %d, expected 8080", port)
	}

	// Test HTTP URL
	httpURL, err := NewServiceAddress("http://api.service.com")
	if err != nil {
		t.Fatalf("Failed to create ServiceAddress: %v", err)
	}

	if httpURL.IsSecure() {
		t.Errorf("IsSecure() = true, expected false for HTTP")
	}

	// Test ToSecure() method
	secure := httpURL.ToSecure()
	if !secure.IsSecure() {
		t.Errorf("ToSecure() didn't create secure URL")
	}
	expectedSecure := "https://api.service.com"
	if secure.Value() != expectedSecure {
		t.Errorf("ToSecure() = %q, expected %q", secure.Value(), expectedSecure)
	}

	// Test default port for HTTP
	port, err = httpURL.GetPort()
	if err != nil {
		t.Errorf("GetPort() error: %v", err)
	}
	if port != 80 {
		t.Errorf("GetPort() = %d, expected 80 for HTTP default", port)
	}

	// Test default port for HTTPS
	httpsDefault, _ := NewServiceAddress("https://api.service.com")
	port, err = httpsDefault.GetPort()
	if err != nil {
		t.Errorf("GetPort() error: %v", err)
	}
	if port != 443 {
		t.Errorf("GetPort() = %d, expected 443 for HTTPS default", port)
	}
}

func TestServiceAddressHostnameOnly(t *testing.T) {
	hostname, err := NewServiceAddress("api.service.com")
	if err != nil {
		t.Fatalf("Failed to create ServiceAddress: %v", err)
	}

	if hostname.IsURL() {
		t.Errorf("IsURL() = true, expected false for hostname")
	}
	if hostname.IsHostPort() {
		t.Errorf("IsHostPort() = true, expected false for hostname")
	}
	if !hostname.IsHostnameOnly() {
		t.Errorf("IsHostnameOnly() = false, expected true")
	}

	// Test GetHost() for hostname-only
	host, err := hostname.GetHost()
	if err != nil {
		t.Errorf("GetHost() error: %v", err)
	}
	if host != "api.service.com" {
		t.Errorf("GetHost() = %q, expected %q", host, "api.service.com")
	}

	// Test GetPort() for hostname-only (should error)
	_, err = hostname.GetPort()
	if err == nil {
		t.Errorf("GetPort() expected error for hostname-only address")
	}
}

func TestServiceAddressUnsafe(t *testing.T) {
	// Test that unsafe constructor doesn't validate
	unsafe := NewServiceAddressUnsafe("")
	if !unsafe.IsEmpty() {
		t.Errorf("NewServiceAddressUnsafe('') should create empty service address")
	}

	unsafe2 := NewServiceAddressUnsafe("invalid@address!")
	if unsafe2.Value() != "invalid@address!" {
		t.Errorf("NewServiceAddressUnsafe should accept invalid characters")
	}
}

func TestServiceAddressProductionValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid production address",
			input:     "https://api.production.com:443",
			wantError: false,
		},
		{
			name:      "contains localhost",
			input:     "localhost:8080",
			wantError: true,
			errorMsg:  "service address contains 'localhost': not suitable for production",
		},
		{
			name:      "contains 127.0.0.1",
			input:     "127.0.0.1:8080",
			wantError: true,
			errorMsg:  "service address contains loopback IP: not suitable for production",
		},
		{
			name:      "contains example domain",
			input:     "api.example.com:8080",
			wantError: true,
			errorMsg:  "service address uses example domain: not suitable for production",
		},
		{
			name:      "HTTP in production",
			input:     "http://api.production.com:8080",
			wantError: true,
			errorMsg:  "service address uses insecure HTTP: production should use HTTPS",
		},
		{
			name:      "HTTPS is valid",
			input:     "https://api.production.com:8080",
			wantError: false,
		},
		{
			name:      "host:port without protocol is valid",
			input:     "api.production.com:8080",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceAddress := NewServiceAddressUnsafe(tt.input)
			err := serviceAddress.IsValidForProduction()

			if tt.wantError {
				if err == nil {
					t.Errorf("IsValidForProduction() expected error, got nil")
					return
				}
				if tt.errorMsg != "" && !containsSubstring(err.Error(), tt.errorMsg) {
					t.Errorf("IsValidForProduction() error = %v, expected to contain %q", err, tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("IsValidForProduction() unexpected error: %v", err)
				}
			}
		})
	}
}

// Helper function to generate a long string for testing
func generateLongString(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a'
	}
	return string(result)
}