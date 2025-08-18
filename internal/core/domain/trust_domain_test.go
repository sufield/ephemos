package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNewTrustDomain(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple domain",
			input:   "example.org",
			wantErr: false,
		},
		{
			name:    "valid production domain",
			input:   "prod.company.com",
			wantErr: false,
		},
		{
			name:    "valid localhost",
			input:   "localhost",
			wantErr: false,
		},
		{
			name:    "valid subdomain",
			input:   "staging.prod.company.com",
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
			errMsg:  "trust domain cannot be empty",
		},
		{
			name:    "whitespace only",
			input:   "   ",
			wantErr: true,
			errMsg:  "invalid trust domain format",
		},
		{
			name:    "too long",
			input:   strings.Repeat("a", 256),
			wantErr: true,
			errMsg:  "trust domain exceeds maximum length",
		},
		{
			name:    "invalid characters",
			input:   "example.org!",
			wantErr: true,
			errMsg:  "invalid trust domain format",
		},
		{
			name:    "starts with dot",
			input:   ".example.org",
			wantErr: true,
			errMsg:  "invalid trust domain format",
		},
		{
			name:    "ends with dot",
			input:   "example.org.",
			wantErr: true,
			errMsg:  "invalid trust domain format",
		},
		{
			name:    "consecutive dots",
			input:   "example..org",
			wantErr: true,
			errMsg:  "invalid trust domain format",
		},
		{
			name:    "starts with hyphen",
			input:   "-example.org",
			wantErr: true,
			errMsg:  "invalid trust domain format",
		},
		{
			name:    "ends with hyphen",
			input:   "example.org-",
			wantErr: true,
			errMsg:  "invalid trust domain format",
		},
		{
			name:    "contains underscore",
			input:   "example_domain.org",
			wantErr: true,
			errMsg:  "invalid trust domain format",
		},
		{
			name:    "contains uppercase",
			input:   "Example.Org",
			wantErr: false, // converts to lowercase
		},
		{
			name:    "valid with numbers",
			input:   "example123.org",
			wantErr: false,
		},
		{
			name:    "valid with hyphens",
			input:   "my-company.example-domain.org",
			wantErr: false,
		},
		{
			name:    "contains spaces",
			input:   "example .org",
			wantErr: true,
			errMsg:  "trust domain must not contain spaces",
		},
		{
			name:    "contains protocol",
			input:   "https://example.org",
			wantErr: true,
			errMsg:  "trust domain must not contain protocol",
		},
		{
			name:    "contains path",
			input:   "example.org/path",
			wantErr: true,
			errMsg:  "trust domain must not contain path",
		},
		{
			name:    "contains port",
			input:   "example.org:8080",
			wantErr: true,
			errMsg:  "trust domain must not contain port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td, err := NewTrustDomain(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewTrustDomain(%q) = %v, want error", tt.input, td)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewTrustDomain(%q) error = %v, want error containing %q", tt.input, err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewTrustDomain(%q) = %v, want no error", tt.input, err)
				return
			}

			// For uppercase test, check that it gets normalized to lowercase
			if tt.input == "Example.Org" {
				if td.String() != "example.org" {
					t.Errorf("NewTrustDomain(%q).String() = %q, want %q (lowercase)", tt.input, td.String(), "example.org")
				}
			} else {
				if td.String() != tt.input {
					t.Errorf("NewTrustDomain(%q).String() = %q, want %q", tt.input, td.String(), tt.input)
				}
			}
		})
	}
}

func TestTrustDomain_String(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple domain",
			input:    "example.org",
			expected: "example.org",
		},
		{
			name:     "complex domain",
			input:    "staging.prod.company.com",
			expected: "staging.prod.company.com",
		},
		{
			name:     "localhost",
			input:    "localhost",
			expected: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td, err := NewTrustDomain(tt.input)
			if err != nil {
				t.Fatalf("NewTrustDomain(%q) failed: %v", tt.input, err)
			}

			result := td.String()
			if result != tt.expected {
				t.Errorf("TrustDomain.String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestTrustDomain_IsZero(t *testing.T) {
	tests := []struct {
		name     string
		td       TrustDomain
		expected bool
	}{
		{
			name:     "zero value",
			td:       TrustDomain(""),
			expected: true,
		},
		{
			name:     "valid domain",
			td:       TrustDomain("example.org"),
			expected: false,
		},
		{
			name:     "whitespace",
			td:       TrustDomain("   "),
			expected: false, // technically not zero, but invalid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.td.IsZero()
			if result != tt.expected {
				t.Errorf("TrustDomain(%q).IsZero() = %v, want %v", tt.td, result, tt.expected)
			}
		})
	}
}

func TestTrustDomain_Equals(t *testing.T) {
	td1, _ := NewTrustDomain("example.org")
	td2, _ := NewTrustDomain("example.org")
	td3, _ := NewTrustDomain("different.org")

	tests := []struct {
		name     string
		td1      TrustDomain
		td2      TrustDomain
		expected bool
	}{
		{
			name:     "same domains",
			td1:      td1,
			td2:      td2,
			expected: true,
		},
		{
			name:     "different domains",
			td1:      td1,
			td2:      td3,
			expected: false,
		},
		{
			name:     "zero vs non-zero",
			td1:      TrustDomain(""),
			td2:      td1,
			expected: false,
		},
		{
			name:     "both zero",
			td1:      TrustDomain(""),
			td2:      TrustDomain(""),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.td1.Equals(tt.td2)
			if result != tt.expected {
				t.Errorf("TrustDomain.Equals() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestTrustDomain_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple domain",
			input:    "example.org",
			expected: `"example.org"`,
		},
		{
			name:     "complex domain",
			input:    "staging.prod.company.com",
			expected: `"staging.prod.company.com"`,
		},
		{
			name:     "localhost",
			input:    "localhost",
			expected: `"localhost"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td, err := NewTrustDomain(tt.input)
			if err != nil {
				t.Fatalf("NewTrustDomain(%q) failed: %v", tt.input, err)
			}

			// Test marshaling
			data, err := json.Marshal(td)
			if err != nil {
				t.Errorf("json.Marshal(%v) failed: %v", td, err)
				return
			}

			if string(data) != tt.expected {
				t.Errorf("json.Marshal(%v) = %q, want %q", td, string(data), tt.expected)
			}

			// Test unmarshaling
			var unmarshaled TrustDomain
			err = json.Unmarshal(data, &unmarshaled)
			if err != nil {
				t.Errorf("json.Unmarshal(%q) failed: %v", string(data), err)
				return
			}

			if unmarshaled != td {
				t.Errorf("json.Unmarshal(%q) = %v, want %v", string(data), unmarshaled, td)
			}
		})
	}
}

func TestTrustDomain_JSONUnmarshalingErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty string",
			input:   `""`,
			wantErr: false, // allowed for optional fields
		},
		{
			name:    "invalid characters",
			input:   `"example.org!"`,
			wantErr: true,
			errMsg:  "invalid trust domain format",
		},
		{
			name:    "too long",
			input:   `"` + strings.Repeat("a", 256) + `"`,
			wantErr: true,
			errMsg:  "trust domain exceeds maximum length",
		},
		{
			name:    "non-string JSON",
			input:   `123`,
			wantErr: true,
		},
		{
			name:    "invalid JSON",
			input:   `"unclosed string`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var td TrustDomain
			err := json.Unmarshal([]byte(tt.input), &td)

			if tt.wantErr {
				if err == nil {
					t.Errorf("json.Unmarshal(%q) = %v, want error", tt.input, td)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("json.Unmarshal(%q) error = %v, want error containing %q", tt.input, err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("json.Unmarshal(%q) = %v, want no error", tt.input, err)
			}
		})
	}
}

func TestTrustDomain_ValidInSPIFFEContext(t *testing.T) {
	// Test that domains valid in our system are also valid for SPIFFE
	validDomains := []string{
		"example.org",
		"prod.company.com",
		"localhost",
		"staging.prod.company.com",
		"my-company.example-domain.org",
		"example123.org",
	}

	for _, domain := range validDomains {
		t.Run(domain, func(t *testing.T) {
			td, err := NewTrustDomain(domain)
			if err != nil {
				t.Errorf("NewTrustDomain(%q) failed: %v", domain, err)
				return
			}

			// Verify it can be used to create a SPIFFE trust domain
			// This tests compatibility with the SPIFFE ID specification
			spiffeURI := "spiffe://" + td.String() + "/service"
			if !isValidSPIFFEURI(spiffeURI) {
				t.Errorf("Trust domain %q creates invalid SPIFFE URI: %q", td, spiffeURI)
			}
		})
	}
}

// Simple SPIFFE URI validation for testing purposes
func isValidSPIFFEURI(uri string) bool {
	// Basic validation - starts with spiffe://, has domain and path
	if !strings.HasPrefix(uri, "spiffe://") {
		return false
	}
	parts := strings.Split(uri[9:], "/")
	if len(parts) < 2 {
		return false
	}
	// Domain part should not be empty
	return parts[0] != ""
}

func TestTrustDomain_Validation_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "exactly 255 characters",
			input:   strings.Repeat("a", 251) + ".org", // 255 total
			wantErr: false,
		},
		{
			name:    "exactly 256 characters",
			input:   strings.Repeat("a", 252) + ".org", // 256 total
			wantErr: true,
			errMsg:  "trust domain exceeds maximum length",
		},
		{
			name:    "single character",
			input:   "a",
			wantErr: false,
		},
		{
			name:    "numeric only",
			input:   "123",
			wantErr: false,
		},
		{
			name:    "all valid special chars",
			input:   "a-b.c-d.e",
			wantErr: false,
		},
		{
			name:    "label starting with number",
			input:   "1example.org",
			wantErr: false,
		},
		{
			name:    "label ending with number",
			input:   "example1.org",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTrustDomain(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewTrustDomain(%q) = success, want error", tt.input)
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewTrustDomain(%q) error = %v, want error containing %q", tt.input, err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewTrustDomain(%q) = %v, want no error", tt.input, err)
			}
		})
	}
}

func TestTrustDomain_ConcurrentSafety(t *testing.T) {
	// Test that TrustDomain operations are safe for concurrent use
	const numGoroutines = 100
	const numOperations = 1000

	td, err := NewTrustDomain("example.org")
	if err != nil {
		t.Fatalf("NewTrustDomain failed: %v", err)
	}

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()

			for j := 0; j < numOperations; j++ {
				// Test all read operations concurrently
				_ = td.String()
				_ = td.IsZero()
				_ = td.Equals(td)

				// Test JSON marshaling
				data, err := json.Marshal(td)
				if err != nil {
					t.Errorf("json.Marshal failed: %v", err)
				}

				var unmarshaled TrustDomain
				err = json.Unmarshal(data, &unmarshaled)
				if err != nil {
					t.Errorf("json.Unmarshal failed: %v", err)
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func BenchmarkNewTrustDomain(b *testing.B) {
	domain := "prod.company.com"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := NewTrustDomain(domain)
		if err != nil {
			b.Fatalf("NewTrustDomain failed: %v", err)
		}
	}
}

func BenchmarkTrustDomain_String(b *testing.B) {
	td, err := NewTrustDomain("prod.company.com")
	if err != nil {
		b.Fatalf("NewTrustDomain failed: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = td.String()
	}
}

func BenchmarkTrustDomain_Equals(b *testing.B) {
	td1, err := NewTrustDomain("prod.company.com")
	if err != nil {
		b.Fatalf("NewTrustDomain failed: %v", err)
	}
	td2, err := NewTrustDomain("prod.company.com")
	if err != nil {
		b.Fatalf("NewTrustDomain failed: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = td1.Equals(td2)
	}
}

func BenchmarkTrustDomain_JSONMarshal(b *testing.B) {
	td, err := NewTrustDomain("prod.company.com")
	if err != nil {
		b.Fatalf("NewTrustDomain failed: %v", err)
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(td)
		if err != nil {
			b.Fatalf("json.Marshal failed: %v", err)
		}
	}
}