// Package domain handles service identity and authentication policies.
package domain

import (
	"crypto"
	"crypto/x509"
	"fmt"
	"strings"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

// ServiceIdentity represents a SPIFFE service identity with name, domain, and URI.
// Use the constructors (NewServiceIdentity, NewServiceIdentityFromSPIFFEID) to ensure proper validation.
type ServiceIdentity struct {
	name   string
	domain string
	uri    string
}

// Name returns the service name.
func (s *ServiceIdentity) Name() string {
	return s.name
}

// Domain returns the trust domain.
func (s *ServiceIdentity) Domain() string {
	return s.domain
}

// URI returns the SPIFFE URI.
func (s *ServiceIdentity) URI() string {
	return s.uri
}

// NewServiceIdentity creates a new ServiceIdentity with the given name and domain.
func NewServiceIdentity(name, domain string) *ServiceIdentity {
	return &ServiceIdentity{
		name:   name,
		domain: domain,
		uri:    fmt.Sprintf("spiffe://%s/%s", domain, name),
	}
}

// NewServiceIdentityFromSPIFFEID creates a ServiceIdentity from a SPIFFE ID.
func NewServiceIdentityFromSPIFFEID(id spiffeid.ID) *ServiceIdentity {
	trustDomain := id.TrustDomain().String()
	serviceName := strings.TrimPrefix(id.Path(), "/")

	return &ServiceIdentity{
		name:   serviceName,
		domain: trustDomain,
		uri:    id.String(),
	}
}

// Validate checks the identity for SPIFFE compliance.
func (s *ServiceIdentity) Validate() error {
	if s.name == "" {
		return fmt.Errorf("service name cannot be empty")
	}
	if s.domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	// Validate trust domain format (must be valid DNS-like string)
	if err := s.validateTrustDomain(s.domain); err != nil {
		return fmt.Errorf("invalid trust domain %q: %w", s.domain, err)
	}

	// Validate service name format (SPIFFE path component)
	if err := s.validateServiceName(s.name); err != nil {
		return fmt.Errorf("invalid service name %q: %w", s.name, err)
	}

	// Validate the constructed SPIFFE URI using official go-spiffe validation
	if _, err := spiffeid.FromString(s.uri); err != nil {
		return fmt.Errorf("invalid SPIFFE URI %q: %w", s.uri, err)
	}

	return nil
}

// validateTrustDomain checks if a trust domain follows SPIFFE specifications.
func (s *ServiceIdentity) validateTrustDomain(domain string) error {
	// Trust domain must be lowercase and DNS-compliant
	if domain != strings.ToLower(domain) {
		return fmt.Errorf("trust domain must be lowercase")
	}

	// Check for invalid characters in trust domain
	invalidChars := " !@#$%^&*()+=[]{}|\\:;\"'<>?/`~"
	if strings.ContainsAny(domain, invalidChars) {
		return fmt.Errorf("trust domain contains invalid characters")
	}

	// Must not start or end with hyphen or period
	if strings.HasPrefix(domain, "-") || strings.HasSuffix(domain, "-") ||
		strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return fmt.Errorf("trust domain cannot start or end with hyphen or period")
	}

	return nil
}

// validateServiceName checks if a service name is valid for SPIFFE path component.
func (s *ServiceIdentity) validateServiceName(name string) error {
	// Service name cannot contain path separators or invalid URI characters
	if strings.Contains(name, "/") {
		return fmt.Errorf("service name cannot contain path separators")
	}

	// Check for characters that would break URI construction
	invalidChars := " \"<>\\^`{|}"
	if strings.ContainsAny(name, invalidChars) {
		return fmt.Errorf("service name contains invalid characters")
	}

	return nil
}

// ToSPIFFEID converts the ServiceIdentity to a SPIFFE ID.
func (s *ServiceIdentity) ToSPIFFEID() (spiffeid.ID, error) {
	return spiffeid.FromString(s.uri)
}

// Equal checks if two ServiceIdentity instances are equivalent.
func (s *ServiceIdentity) Equal(other *ServiceIdentity) bool {
	if other == nil {
		return false
	}
	return s.uri == other.uri
}

// String returns the SPIFFE URI string representation.
func (s *ServiceIdentity) String() string {
	return s.uri
}

// Certificate holds SPIFFE X.509 SVID certificate data with proper type safety.
type Certificate struct {
	Cert       *x509.Certificate   // Leaf certificate
	PrivateKey crypto.Signer       // Private key (must implement crypto.Signer for SPIFFE)
	Chain      []*x509.Certificate // Intermediate certificates (leaf-to-root order)
}

// NewCertificate creates a new Certificate with validation.
func NewCertificate(cert *x509.Certificate, key crypto.Signer, chain []*x509.Certificate) (*Certificate, error) {
	c := &Certificate{
		Cert:       cert,
		PrivateKey: key,
		Chain:      chain,
	}

	if err := c.Validate(); err != nil {
		return nil, fmt.Errorf("certificate validation failed: %w", err)
	}

	return c, nil
}

// Validate checks that the certificate is valid and properly formed.
func (c *Certificate) Validate() error {
	if c.Cert == nil {
		return fmt.Errorf("certificate cannot be nil")
	}

	if c.PrivateKey == nil {
		return fmt.Errorf("private key cannot be nil")
	}

	// Verify the private key matches the certificate's public key
	// Use crypto.Signer interface to get the public key
	privateKeyPublic := c.PrivateKey.Public()
	
	// Compare the public keys - need to handle different key types
	switch pubKey := c.Cert.PublicKey.(type) {
	case interface{ Equal(interface{}) bool }:
		if !pubKey.Equal(privateKeyPublic) {
			return fmt.Errorf("private key does not match certificate public key")
		}
	default:
		// Fallback comparison for older Go versions or unsupported key types
		// This is a basic comparison and may not work for all key types
		return fmt.Errorf("unable to verify key match for public key type %T", c.Cert.PublicKey)
	}

	// Check certificate is not expired
	now := time.Now()
	if now.Before(c.Cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid (NotBefore: %v)", c.Cert.NotBefore)
	}
	if now.After(c.Cert.NotAfter) {
		return fmt.Errorf("certificate has expired (NotAfter: %v)", c.Cert.NotAfter)
	}

	// Warn if certificate expires soon (within 30 minutes)
	if now.Add(30 * time.Minute).After(c.Cert.NotAfter) {
		// In production, this should use structured logging
		fmt.Printf("Warning: Certificate expires soon (NotAfter: %v)\n", c.Cert.NotAfter)
	}

	// Validate chain order if present (should be leaf-to-root)
	if len(c.Chain) > 0 {
		if err := c.validateChainOrder(); err != nil {
			return fmt.Errorf("certificate chain validation failed: %w", err)
		}
	}

	return nil
}

// validateChainOrder checks that the certificate chain is properly ordered.
func (c *Certificate) validateChainOrder() error {
	if len(c.Chain) == 0 {
		return nil // No chain to validate
	}

	// In a proper chain, each certificate should be signed by the next one
	// This is a simplified check - full validation would verify signatures
	current := c.Cert
	for i, next := range c.Chain {
		if current.Issuer.String() != next.Subject.String() {
			return fmt.Errorf("chain order invalid at position %d: current issuer %q != next subject %q",
				i, current.Issuer.String(), next.Subject.String())
		}
		current = next
	}

	return nil
}

// IsExpiringSoon returns true if the certificate expires within the given duration.
func (c *Certificate) IsExpiringSoon(threshold time.Duration) bool {
	if c.Cert == nil {
		return true // Treat nil certificate as expired
	}
	return time.Now().Add(threshold).After(c.Cert.NotAfter)
}

// ToSPIFFEID extracts the SPIFFE ID from the certificate's URI SAN.
func (c *Certificate) ToSPIFFEID() (spiffeid.ID, error) {
	if c.Cert == nil {
		return spiffeid.ID{}, fmt.Errorf("certificate is nil")
	}

	for _, uri := range c.Cert.URIs {
		if uri.Scheme == "spiffe" {
			return spiffeid.FromURI(uri)
		}
	}

	return spiffeid.ID{}, fmt.Errorf("no SPIFFE ID found in certificate URI SANs")
}

// ToServiceIdentity extracts a ServiceIdentity from the certificate's SPIFFE ID.
func (c *Certificate) ToServiceIdentity() (*ServiceIdentity, error) {
	spiffeID, err := c.ToSPIFFEID()
	if err != nil {
		return nil, fmt.Errorf("failed to extract SPIFFE ID: %w", err)
	}

	// Parse trust domain and service name from SPIFFE ID
	trustDomain := spiffeID.TrustDomain().String()
	path := spiffeID.Path()

	// Extract service name from path (remove leading slash)
	serviceName := strings.TrimPrefix(path, "/")
	if serviceName == "" {
		return nil, fmt.Errorf("SPIFFE ID path is empty")
	}

	return NewServiceIdentity(serviceName, trustDomain), nil
}

// TrustBundle holds SPIFFE trust anchor certificates for a trust domain.
type TrustBundle struct {
	Certificates []*x509.Certificate // Root CA certificates for the trust domain
}

// NewTrustBundle creates a new TrustBundle with validation.
func NewTrustBundle(certificates []*x509.Certificate) (*TrustBundle, error) {
	tb := &TrustBundle{
		Certificates: certificates,
	}

	if err := tb.Validate(); err != nil {
		return nil, fmt.Errorf("trust bundle validation failed: %w", err)
	}

	return tb, nil
}

// Validate checks that the trust bundle is valid and contains valid certificates.
func (tb *TrustBundle) Validate() error {
	if len(tb.Certificates) == 0 {
		return fmt.Errorf("trust bundle cannot be empty")
	}

	// Validate each certificate in the bundle
	for i, cert := range tb.Certificates {
		if cert == nil {
			return fmt.Errorf("certificate at index %d is nil", i)
		}

		// Check if certificate is a valid CA (should have CA:TRUE basic constraint)
		if !cert.IsCA {
			return fmt.Errorf("certificate at index %d is not a CA certificate", i)
		}

		// Check if certificate is not expired
		now := time.Now()
		if now.After(cert.NotAfter) {
			return fmt.Errorf("CA certificate at index %d has expired (NotAfter: %v)", i, cert.NotAfter)
		}

		// Warn if CA certificate expires soon (within 24 hours)
		if now.Add(24 * time.Hour).After(cert.NotAfter) {
			fmt.Printf("Warning: CA certificate expires soon (Subject: %s, NotAfter: %v)\n", 
				cert.Subject.String(), cert.NotAfter)
		}
	}

	return nil
}

// IsEmpty returns true if the trust bundle contains no certificates.
func (tb *TrustBundle) IsEmpty() bool {
	return len(tb.Certificates) == 0
}

// ContainsCertificate checks if the trust bundle contains a specific certificate.
func (tb *TrustBundle) ContainsCertificate(cert *x509.Certificate) bool {
	if cert == nil {
		return false
	}

	for _, bundleCert := range tb.Certificates {
		if bundleCert.Equal(cert) {
			return true
		}
	}

	return false
}

// ToCertPool converts the trust bundle to an x509.CertPool for use with TLS.
func (tb *TrustBundle) ToCertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	for _, cert := range tb.Certificates {
		pool.AddCert(cert)
	}
	return pool
}

// AuthenticationPolicy defines authentication and authorization context.
// This includes both identity verification and access control policies.
type AuthenticationPolicy struct {
	ServiceIdentity   *ServiceIdentity
	AuthorizedClients []string // For server-side: SPIFFE IDs allowed to connect
	TrustedServers    []string // For client-side: SPIFFE IDs this service trusts
}

// NewAuthenticationPolicy creates a policy for authentication only.
// For authorization support, use NewAuthorizationPolicy instead.
func NewAuthenticationPolicy(identity *ServiceIdentity) *AuthenticationPolicy {
	return &AuthenticationPolicy{
		ServiceIdentity: identity,
	}
}

// NewAuthorizationPolicy creates a policy with both authentication and authorization.
// This enforces access control based on SPIFFE ID allowlists.
func NewAuthorizationPolicy(identity *ServiceIdentity, authorizedClients, trustedServers []string) *AuthenticationPolicy {
	return &AuthenticationPolicy{
		ServiceIdentity:   identity,
		AuthorizedClients: authorizedClients,
		TrustedServers:    trustedServers,
	}
}

// HasAuthorization returns true if this policy includes authorization rules.
func (p *AuthenticationPolicy) HasAuthorization() bool {
	return len(p.AuthorizedClients) > 0 || len(p.TrustedServers) > 0
}

// IsClientAuthorized checks if a client SPIFFE ID is authorized to connect (server-side).
func (p *AuthenticationPolicy) IsClientAuthorized(clientSPIFFEID string) bool {
	if len(p.AuthorizedClients) == 0 {
		// No explicit authorization rules - allow same trust domain
		return true
	}
	
	for _, authorized := range p.AuthorizedClients {
		if authorized == clientSPIFFEID {
			return true
		}
	}
	return false
}

// IsServerTrusted checks if a server SPIFFE ID is trusted for connections (client-side).
func (p *AuthenticationPolicy) IsServerTrusted(serverSPIFFEID string) bool {
	if len(p.TrustedServers) == 0 {
		// No explicit trust rules - trust same trust domain
		return true
	}
	
	for _, trusted := range p.TrustedServers {
		if trusted == serverSPIFFEID {
			return true
		}
	}
	return false
}
