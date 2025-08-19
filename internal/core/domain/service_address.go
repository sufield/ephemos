// Package domain provides domain value objects and entities.
package domain

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ServiceAddress is a value object for validated service addresses.
// It encapsulates validation logic centrally and enforces invariants at construction.
type ServiceAddress struct {
	value string // Private to enforce encapsulation
}

// Valid address patterns for different formats
var (
	// hostPortPattern matches host:port format
	hostPortPattern = regexp.MustCompile(`^[a-zA-Z0-9.-]+:[0-9]+$`)
	// urlPattern matches basic URL format
	urlPattern = regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+(:[0-9]+)?(/.*)?$`)
	// hostnamePattern matches valid hostnames
	hostnamePattern = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9.-]*[a-zA-Z0-9])?$`)
)

// NewServiceAddress creates a ServiceAddress, applying validation.
// Returns an error if the service address is invalid.
func NewServiceAddress(address string) (ServiceAddress, error) {
	trimmed := strings.TrimSpace(address)
	
	if trimmed == "" {
		return ServiceAddress{}, fmt.Errorf("service address cannot be empty or whitespace-only")
	}
	
	// Length limits for reasonable addresses
	if len(trimmed) > 500 {
		return ServiceAddress{}, fmt.Errorf("service address too long: maximum 500 characters, got %d", len(trimmed))
	}
	
	// Validate the address format
	if err := validateAddressFormat(trimmed); err != nil {
		return ServiceAddress{}, fmt.Errorf("invalid service address format: %w", err)
	}
	
	// Additional domain-specific rules
	if strings.Contains(strings.ToLower(trimmed), "example.com") || strings.Contains(strings.ToLower(trimmed), "example.org") {
		return ServiceAddress{}, fmt.Errorf("service address cannot use example domains: use a real address")
	}
	
	return ServiceAddress{value: trimmed}, nil
}

// validateAddressFormat validates different address formats
func validateAddressFormat(address string) error {
	// Check if it's a URL format
	if strings.HasPrefix(address, "http://") || strings.HasPrefix(address, "https://") {
		return validateURLFormat(address)
	}
	
	// Check if it's a host:port format
	if strings.Contains(address, ":") {
		return validateHostPortFormat(address)
	}
	
	// Check if it's just a hostname
	return validateHostnameFormat(address)
}

// validateURLFormat validates URL-style addresses
func validateURLFormat(address string) error {
	if !urlPattern.MatchString(address) {
		return fmt.Errorf("invalid URL format")
	}
	
	parsedURL, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a host")
	}
	
	return nil
}

// validateHostPortFormat validates host:port style addresses
func validateHostPortFormat(address string) error {
	if !hostPortPattern.MatchString(address) {
		return fmt.Errorf("invalid host:port format")
	}
	
	host, portStr, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("failed to parse host:port: %w", err)
	}
	
	// Validate host
	if host == "" {
		return fmt.Errorf("host cannot be empty")
	}
	
	// Validate port
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}
	
	if port < 1 || port > 65535 {
		return fmt.Errorf("port number must be between 1 and 65535, got %d", port)
	}
	
	// Validate hostname format
	if !hostnamePattern.MatchString(host) {
		// Try to parse as IP address
		if net.ParseIP(host) == nil {
			return fmt.Errorf("invalid hostname or IP address: %s", host)
		}
	}
	
	return nil
}

// validateHostnameFormat validates hostname-only addresses
func validateHostnameFormat(address string) error {
	if !hostnamePattern.MatchString(address) {
		// Try to parse as IP address
		if net.ParseIP(address) == nil {
			return fmt.Errorf("invalid hostname or IP address: %s", address)
		}
	}
	
	return nil
}

// NewServiceAddressUnsafe creates a ServiceAddress without validation.
// This should only be used for testing or when validation has already been performed.
// Use NewServiceAddress in production code.
func NewServiceAddressUnsafe(address string) ServiceAddress {
	return ServiceAddress{value: strings.TrimSpace(address)}
}

// String returns the service address as a string.
// This is a temporary compatibility method during refactoring.
func (sa ServiceAddress) String() string {
	return sa.value
}

// Value returns the service address value.
// This is the preferred method for accessing the service address.
func (sa ServiceAddress) Value() string {
	return sa.value
}

// Equals compares two ServiceAddresses for equality.
func (sa ServiceAddress) Equals(other ServiceAddress) bool {
	return sa.value == other.value
}

// IsEmpty returns true if the service address is empty.
func (sa ServiceAddress) IsEmpty() bool {
	return sa.value == ""
}

// IsURL returns true if the address is in URL format.
func (sa ServiceAddress) IsURL() bool {
	return strings.HasPrefix(sa.value, "http://") || strings.HasPrefix(sa.value, "https://")
}

// IsHostPort returns true if the address is in host:port format.
func (sa ServiceAddress) IsHostPort() bool {
	return strings.Contains(sa.value, ":") && !sa.IsURL()
}

// IsHostnameOnly returns true if the address is just a hostname or IP.
func (sa ServiceAddress) IsHostnameOnly() bool {
	return !sa.IsURL() && !sa.IsHostPort()
}

// GetHost extracts the host part from the address.
func (sa ServiceAddress) GetHost() (string, error) {
	if sa.IsURL() {
		parsedURL, err := url.Parse(sa.value)
		if err != nil {
			return "", fmt.Errorf("failed to parse URL: %w", err)
		}
		host, _, err := net.SplitHostPort(parsedURL.Host)
		if err != nil {
			// No port in URL host
			return parsedURL.Host, nil
		}
		return host, nil
	}
	
	if sa.IsHostPort() {
		host, _, err := net.SplitHostPort(sa.value)
		if err != nil {
			return "", fmt.Errorf("failed to parse host:port: %w", err)
		}
		return host, nil
	}
	
	// Hostname only
	return sa.value, nil
}

// GetPort extracts the port from the address if present.
func (sa ServiceAddress) GetPort() (int, error) {
	if sa.IsURL() {
		parsedURL, err := url.Parse(sa.value)
		if err != nil {
			return 0, fmt.Errorf("failed to parse URL: %w", err)
		}
		if parsedURL.Port() == "" {
			// Default ports based on protocol
			protocol, err := ParseProtocol(parsedURL.Scheme)
			if err != nil {
				return 0, fmt.Errorf("failed to parse protocol %q - explicit port required: %w", parsedURL.Scheme, err)
			}
			return protocol.DefaultPort(), nil
		}
		return strconv.Atoi(parsedURL.Port())
	}
	
	if sa.IsHostPort() {
		_, portStr, err := net.SplitHostPort(sa.value)
		if err != nil {
			return 0, fmt.Errorf("failed to parse host:port: %w", err)
		}
		return strconv.Atoi(portStr)
	}
	
	return 0, fmt.Errorf("no port specified in address")
}

// IsSecure returns true if the address uses a secure protocol (HTTPS).
func (sa ServiceAddress) IsSecure() bool {
	return strings.HasPrefix(sa.value, ProtocolHTTPS.String()+"://")
}

// ToSecure converts the address to use HTTPS if it's an HTTP URL.
func (sa ServiceAddress) ToSecure() ServiceAddress {
	httpPrefix := ProtocolHTTP.String() + "://"
	if strings.HasPrefix(sa.value, httpPrefix) {
		httpsPrefix := ProtocolHTTPS.String() + "://"
		return ServiceAddress{value: strings.Replace(sa.value, httpPrefix, httpsPrefix, 1)}
	}
	return sa
}

// IsValidForProduction checks if the address is suitable for production use.
// This applies additional rules beyond basic validation.
func (sa ServiceAddress) IsValidForProduction() error {
	if strings.Contains(strings.ToLower(sa.value), "localhost") {
		return fmt.Errorf("service address contains 'localhost': not suitable for production")
	}
	
	if strings.Contains(strings.ToLower(sa.value), "127.0.0.1") {
		return fmt.Errorf("service address contains loopback IP: not suitable for production")
	}
	
	if strings.Contains(strings.ToLower(sa.value), "example.") {
		return fmt.Errorf("service address uses example domain: not suitable for production")
	}
	
	if sa.IsURL() && !sa.IsSecure() {
		return fmt.Errorf("service address uses insecure HTTP: production should use HTTPS")
	}
	
	return nil
}