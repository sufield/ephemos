// Package domain handles service identity and authentication policies.
package domain

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"log/slog"
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
	// Parse trust domain to ensure it's valid
	trustDomain, err := spiffeid.TrustDomainFromString(domain)
	if err != nil {
		// Fallback to simple construction for backward compatibility
		return &ServiceIdentity{
			name:   name,
			domain: domain,
			uri:    fmt.Sprintf("spiffe://%s/%s", domain, name),
		}
	}
	
	// Use official SPIFFE ID construction
	spiffeID, err := spiffeid.FromPath(trustDomain, "/"+name)
	if err != nil {
		// Fallback to simple construction for backward compatibility
		return &ServiceIdentity{
			name:   name,
			domain: domain,
			uri:    fmt.Sprintf("spiffe://%s/%s", domain, name),
		}
	}
	
	return &ServiceIdentity{
		name:   name,
		domain: domain,
		uri:    spiffeID.String(),
	}
}

// NewServiceIdentityWithValidation creates a new ServiceIdentity with optional validation.
// Set validate to false only in trusted contexts where performance is critical
// and you're certain the identity data is valid (e.g., internal caching).
func NewServiceIdentityWithValidation(name, domain string, validate bool) (*ServiceIdentity, error) {
	// Parse trust domain to ensure it's valid when validating
	var identity *ServiceIdentity
	if validate {
		trustDomain, err := spiffeid.TrustDomainFromString(domain)
		if err != nil {
			return nil, fmt.Errorf("invalid trust domain %q: %w", domain, err)
		}
		
		// Use official SPIFFE ID construction
		spiffeID, err := spiffeid.FromPath(trustDomain, "/"+name)
		if err != nil {
			return nil, fmt.Errorf("invalid SPIFFE path %q: %w", name, err)
		}
		
		identity = &ServiceIdentity{
			name:   name,
			domain: domain,
			uri:    spiffeID.String(),
		}
		
		if err := identity.Validate(); err != nil {
			return nil, fmt.Errorf("service identity validation failed: %w", err)
		}
	} else {
		// Simple construction without validation
		identity = &ServiceIdentity{
			name:   name,
			domain: domain,
			uri:    fmt.Sprintf("spiffe://%s/%s", domain, name),
		}
	}

	return identity, nil
}

// NewServiceIdentityFromSPIFFEID creates a ServiceIdentity from a SPIFFE ID.
// Supports multi-segment paths (e.g., "/api/v1/service" becomes "api/v1/service").
func NewServiceIdentityFromSPIFFEID(id spiffeid.ID) *ServiceIdentity {
	trustDomain := id.TrustDomain().String()
	// Support multi-segment paths by preserving the full path structure
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

	// Validate the constructed SPIFFE URI using go-spiffe validation
	spiffeID, err := spiffeid.FromString(s.uri)
	if err != nil {
		return fmt.Errorf("invalid SPIFFE URI %q: %w", s.uri, err)
	}
	
	// Additional SPIFFE spec constraints
	if err := s.validateSPIFFEConstraints(spiffeID); err != nil {
		return fmt.Errorf("SPIFFE spec violation in URI %q: %w", s.uri, err)
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
// Now supports multi-segment paths (e.g., "api/v1/service") using official SPIFFE validation.
func (s *ServiceIdentity) validateServiceName(name string) error {
	// Use official SPIFFE path validation
	path := "/" + name // SPIFFE paths must start with "/"
	if err := spiffeid.ValidatePath(path); err != nil {
		return fmt.Errorf("service name validation failed: %w", err)
	}

	return nil
}

// validateSPIFFEConstraints validates additional SPIFFE specification constraints
// beyond what the go-spiffe library checks.
func (s *ServiceIdentity) validateSPIFFEConstraints(spiffeID spiffeid.ID) error {
	path := spiffeID.Path()
	
	// Use official SPIFFE path validation if there's a path
	if path != "" {
		if err := spiffeid.ValidatePath(path); err != nil {
			return fmt.Errorf("SPIFFE path validation failed: %w", err)
		}
	}
	
	// Additional practical constraints
	// SPIFFE spec: path length should be reasonable (practical limit)
	if len(path) > 2048 {
		return fmt.Errorf("SPIFFE path exceeds maximum length of 2048 characters")
	}
	
	// Trust domain validation (additional checks beyond go-spiffe)
	trustDomain := spiffeID.TrustDomain().String()
	
	// SPIFFE spec: trust domain length limit (practical limit)
	if len(trustDomain) > 255 {
		return fmt.Errorf("trust domain exceeds maximum length of 255 characters")
	}
	
	// SPIFFE spec: trust domain must not contain uppercase (should be normalized)
	if trustDomain != strings.ToLower(trustDomain) {
		return fmt.Errorf("trust domain must be lowercase")
	}
	
	return nil
}

// ToSPIFFEID converts the ServiceIdentity to a SPIFFE ID.
func (s *ServiceIdentity) ToSPIFFEID() (spiffeid.ID, error) {
	return spiffeid.FromString(s.uri)
}

// Equal checks if two ServiceIdentity instances are equivalent.
func (s *ServiceIdentity) Equal(other *ServiceIdentity) bool {
	if s == nil {
		return other == nil
	}
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

// CertValidationOptions configures certificate validation behavior.
type CertValidationOptions struct {
	ExpectedIdentity *ServiceIdentity // Optional: Expected SPIFFE identity for matching
	WarningThreshold time.Duration    // Optional: Warning threshold for near-expiry (e.g., 1h)
	TrustBundle      *TrustBundle     // Optional: Trust bundle for chain verification
	SkipExpiry       bool             // Optional: Skip expiry checks (testing only)
	SkipChainVerify  bool             // Optional: Skip chain cryptographic verification
}

// NewCertificate creates a new Certificate with validation.
func NewCertificate(cert *x509.Certificate, key crypto.Signer, chain []*x509.Certificate) (*Certificate, error) {
	return NewCertificateWithValidation(cert, key, chain, true)
}

// NewCertificateWithValidation creates a new Certificate with optional validation.
// Set skipValidation to true only in trusted contexts where performance is critical
// and you're certain the certificate data is valid (e.g., internal caching).
func NewCertificateWithValidation(cert *x509.Certificate, key crypto.Signer, chain []*x509.Certificate, validate bool) (*Certificate, error) {
	c := &Certificate{
		Cert:       cert,
		PrivateKey: key,
		Chain:      chain,
	}

	if validate {
		if err := c.Validate(CertValidationOptions{}); err != nil {
			return nil, fmt.Errorf("certificate validation failed: %w", err)
		}
	}

	return c, nil
}

// Validate performs comprehensive certificate validation with configurable options.
// This is the primary validation method that should be used in production code.
func (c *Certificate) Validate(opts CertValidationOptions) error {
	// Basic structure validation
	if c == nil || c.Cert == nil {
		return fmt.Errorf("certificate is nil")
	}
	if c.PrivateKey == nil {
		return fmt.Errorf("private key is nil")
	}

	// Verify the private key matches the certificate's public key
	if err := c.verifyKeyMatch(); err != nil {
		return fmt.Errorf("private key validation failed: %w", err)
	}

	// Expiry checks (can be skipped for testing)
	if !opts.SkipExpiry {
		now := time.Now()
		if now.Before(c.Cert.NotBefore) {
			return fmt.Errorf("certificate is not yet valid (NotBefore: %v)", c.Cert.NotBefore)
		}
		if now.After(c.Cert.NotAfter) {
			return fmt.Errorf("certificate has expired (NotAfter: %v)", c.Cert.NotAfter)
		}

		// Near-expiry warning with configurable threshold
		warningThreshold := opts.WarningThreshold
		if warningThreshold == 0 {
			warningThreshold = 30 * time.Minute // Default warning threshold
		}
		if now.Add(warningThreshold).After(c.Cert.NotAfter) {
			slog.Warn("Certificate expires soon",
				"cert_subject", c.Cert.Subject.String(),
				"expires_at", c.Cert.NotAfter,
				"expires_in", time.Until(c.Cert.NotAfter).String(),
				"serial_number", c.Cert.SerialNumber.String(),
			)
		}
	}

	// Chain validation
	if len(c.Chain) > 0 {
		if err := c.validateChainOrder(); err != nil {
			return fmt.Errorf("certificate chain validation failed: %w", err)
		}
	}

	// Trust bundle verification (if provided)
	if opts.TrustBundle != nil && !opts.SkipChainVerify {
		if err := c.verifyWithTrustBundle(opts.TrustBundle); err != nil {
			return fmt.Errorf("trust bundle verification failed: %w", err)
		}
	}

	// SPIFFE identity matching (if expected identity provided)
	if opts.ExpectedIdentity != nil {
		actualID, err := c.ToSPIFFEID()
		if err != nil {
			return fmt.Errorf("failed to extract SPIFFE ID: %w", err)
		}
		
		expectedID, err := opts.ExpectedIdentity.ToSPIFFEID()
		if err != nil {
			return fmt.Errorf("failed to get expected SPIFFE ID: %w", err)
		}
		
		// Compare SPIFFE IDs using String() representation
		if actualID.String() != expectedID.String() {
			return fmt.Errorf("SPIFFE ID mismatch: expected %s, got %s", expectedID, actualID)
		}
	}

	return nil
}


// validateChainOrder checks that the certificate chain is properly ordered
// and cryptographically valid with full signature verification.
func (c *Certificate) validateChainOrder() error {
	if len(c.Chain) == 0 {
		return nil // No chain to validate
	}
	
	// Start with the leaf certificate and verify each link in the chain
	current := c.Cert
	for i, next := range c.Chain {
		// Check issuer-subject name matching first (fast check)
		if current.Issuer.String() != next.Subject.String() {
			return fmt.Errorf("chain order invalid at position %d: current issuer %q != next subject %q",
				i, current.Issuer.String(), next.Subject.String())
		}
		
		// Perform cryptographic signature verification
		// Create a certificate pool with just the issuer certificate
		issuerPool := x509.NewCertPool()
		issuerPool.AddCert(next)
		
		// Verify the current certificate was signed by the next certificate
		verifyOpts := x509.VerifyOptions{
			Roots:         issuerPool,
			Intermediates: x509.NewCertPool(), // Empty intermediate pool for single-step verification
			KeyUsages:     []x509.ExtKeyUsage{}, // Don't enforce key usage for chain validation
		}
		
		// Verify the signature (this checks the cryptographic validity)
		_, err := current.Verify(verifyOpts)
		if err != nil {
			return fmt.Errorf("signature verification failed at chain position %d: certificate %q was not properly signed by %q: %w",
				i, current.Subject.String(), next.Subject.String(), err)
		}
		
		// Check that the signing certificate is authorized to sign other certificates
		if !next.IsCA {
			slog.Warn("Certificate in chain is not marked as CA but is signing other certificates",
				"position", i,
				"subject", next.Subject.String(),
				"serial_number", next.SerialNumber.String(),
			)
		}
		
		// Move to the next link in the chain
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

	// Extract service name from path (supports multi-segment paths)
	serviceName := strings.TrimPrefix(path, "/")
	if serviceName == "" {
		return nil, fmt.Errorf("SPIFFE ID path is empty")
	}
	
	// Validate that the path doesn't contain invalid characters or patterns
	if strings.Contains(serviceName, "//") {
		return nil, fmt.Errorf("SPIFFE ID path contains invalid double slashes")
	}

	return NewServiceIdentity(serviceName, trustDomain), nil
}

// verifyKeyMatch verifies that the private key matches the certificate's public key
// with support for multiple key types including RSA, ECDSA, and future algorithms.
func (c *Certificate) verifyKeyMatch() error {
	if c.PrivateKey == nil {
		return fmt.Errorf("private key is nil")
	}
	if c.Cert == nil {
		return fmt.Errorf("certificate is nil")
	}
	
	privateKeyPublic := c.PrivateKey.Public()
	
	// First try the modern Equal method (available in Go 1.15+)
	switch pubKey := c.Cert.PublicKey.(type) {
	case interface{ Equal(interface{}) bool }:
		if !pubKey.Equal(privateKeyPublic) {
			return fmt.Errorf("private key does not match certificate public key")
		}
		return nil
	}
	
	// Fallback to manual field comparison for specific key types
	switch certPubKey := c.Cert.PublicKey.(type) {
	case *rsa.PublicKey:
		privPubKey, ok := privateKeyPublic.(*rsa.PublicKey)
		if !ok {
			return fmt.Errorf("certificate has RSA public key but private key is %T", privateKeyPublic)
		}
		if certPubKey.N.Cmp(privPubKey.N) != 0 || certPubKey.E != privPubKey.E {
			return fmt.Errorf("RSA private key does not match certificate public key")
		}
		return nil
		
	case *ecdsa.PublicKey:
		privPubKey, ok := privateKeyPublic.(*ecdsa.PublicKey)
		if !ok {
			return fmt.Errorf("certificate has ECDSA public key but private key is %T", privateKeyPublic)
		}
		if certPubKey.Curve != privPubKey.Curve ||
			certPubKey.X.Cmp(privPubKey.X) != 0 ||
			certPubKey.Y.Cmp(privPubKey.Y) != 0 {
			return fmt.Errorf("ECDSA private key does not match certificate public key")
		}
		return nil
		
	default:
		// For unknown key types, we can't verify the match
		return fmt.Errorf("unable to verify key match for unsupported public key type %T", c.Cert.PublicKey)
	}
}

// verifyWithTrustBundle verifies the certificate chain against a trust bundle.
func (c *Certificate) verifyWithTrustBundle(trustBundle *TrustBundle) error {
	if trustBundle == nil {
		return fmt.Errorf("trust bundle is nil")
	}
	
	// Create cert pool from trust bundle
	roots := trustBundle.CreateCertPool()
	if roots == nil {
		return fmt.Errorf("failed to create cert pool from trust bundle")
	}
	
	// Setup verification options
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: x509.NewCertPool(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}
	
	// Add intermediate certificates to the pool if present
	if len(c.Chain) > 0 {
		for _, intermediate := range c.Chain {
			opts.Intermediates.AddCert(intermediate)
		}
	}
	
	// Perform cryptographic verification
	_, err := c.Cert.Verify(opts)
	if err != nil {
		return fmt.Errorf("certificate chain cryptographic verification failed: %w", err)
	}
	
	return nil
}

// DefaultCertValidator provides the standard certificate validation implementation.
// It implements the CertValidatorPort interface from the ports package.
type DefaultCertValidator struct{}

// Validate delegates to the Certificate's Validate method.
func (v *DefaultCertValidator) Validate(cert *Certificate, opts CertValidationOptions) error {
	if cert == nil {
		return fmt.Errorf("certificate is nil")
	}
	return cert.Validate(opts)
}

// TrustBundle holds SPIFFE trust anchor certificates for a trust domain.
type TrustBundle struct {
	Certificates []*x509.Certificate // Root CA certificates for the trust domain
}

// NewTrustBundle creates a new TrustBundle with validation.
func NewTrustBundle(certificates []*x509.Certificate) (*TrustBundle, error) {
	return NewTrustBundleWithValidation(certificates, true)
}

// NewTrustBundleWithValidation creates a new TrustBundle with optional validation.
// Set skipValidation to true only in trusted contexts where performance is critical
// and you're certain the trust bundle data is valid (e.g., internal caching).
func NewTrustBundleWithValidation(certificates []*x509.Certificate, validate bool) (*TrustBundle, error) {
	tb := &TrustBundle{
		Certificates: certificates,
	}

	if validate {
		if err := tb.Validate(); err != nil {
			return nil, fmt.Errorf("trust bundle validation failed: %w", err)
		}
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

		// Check if certificate is not expired and not yet valid
		now := time.Now()
		if now.Before(cert.NotBefore) {
			return fmt.Errorf("CA certificate at index %d is not yet valid (NotBefore: %v)", i, cert.NotBefore)
		}
		if now.After(cert.NotAfter) {
			return fmt.Errorf("CA certificate at index %d has expired (NotAfter: %v)", i, cert.NotAfter)
		}

		// Warn if CA certificate expires soon (within 24 hours)
		if now.Add(24 * time.Hour).After(cert.NotAfter) {
			slog.Warn("CA certificate expires soon",
				"ca_subject", cert.Subject.String(),
				"expires_at", cert.NotAfter,
				"expires_in", time.Until(cert.NotAfter).String(),
				"serial_number", cert.SerialNumber.String(),
				"is_ca", cert.IsCA,
			)
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

// CreateCertPool creates a new x509.CertPool from the trust bundle certificates.
// Unlike a deprecated static ToCertPool, this creates a fresh pool each time
// to support dynamic reloading scenarios where trust bundles change.
func (tb *TrustBundle) CreateCertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	
	for _, cert := range tb.Certificates {
		if cert != nil {
			pool.AddCert(cert)
		}
	}
	
	return pool
}

// TrustBundleProvider defines an interface for dynamic trust bundle access.
// This supports SVID rotation scenarios where trust bundles may change over time.
type TrustBundleProvider interface {
	// GetTrustBundle returns the current trust bundle.
	// Implementations should return the most up-to-date bundle.
	GetTrustBundle() (*TrustBundle, error)
	
	// CreateCertPool creates a cert pool from the current trust bundle.
	// This is a convenience method that calls GetTrustBundle().CreateCertPool().
	CreateCertPool() (*x509.CertPool, error)
}

// StaticTrustBundleProvider provides a static trust bundle (for testing/simple cases).
type StaticTrustBundleProvider struct {
	bundle *TrustBundle
}

// NewStaticTrustBundleProvider creates a provider with a fixed trust bundle.
func NewStaticTrustBundleProvider(bundle *TrustBundle) *StaticTrustBundleProvider {
	return &StaticTrustBundleProvider{bundle: bundle}
}

// GetTrustBundle returns the static trust bundle.
func (p *StaticTrustBundleProvider) GetTrustBundle() (*TrustBundle, error) {
	if p.bundle == nil {
		return nil, fmt.Errorf("no trust bundle configured")
	}
	return p.bundle, nil
}

// CreateCertPool creates a cert pool from the static trust bundle.
func (p *StaticTrustBundleProvider) CreateCertPool() (*x509.CertPool, error) {
	bundle, err := p.GetTrustBundle()
	if err != nil {
		return nil, err
	}
	return bundle.CreateCertPool(), nil
}


// AuthenticationPolicy defines authentication and authorization context.
// This includes both identity verification and access control policies.
type AuthenticationPolicy struct {
	ServiceIdentity    *ServiceIdentity
	AuthorizedClients  []spiffeid.ID // For server-side: SPIFFE IDs allowed to connect
	TrustedServers     []spiffeid.ID // For client-side: SPIFFE IDs this service trusts
	TrustDomain        string        // Trust domain for authorization
	AllowedSPIFFEIDs   []spiffeid.ID // Specific SPIFFE IDs for precise authorization
	RequireAuth        bool          // Whether authentication is required
}

// NewAuthenticationPolicy creates a policy for authentication only.
// For authorization support, use NewAuthorizationPolicy instead.
func NewAuthenticationPolicy(identity *ServiceIdentity) *AuthenticationPolicy {
	policy := &AuthenticationPolicy{
		ServiceIdentity: identity,
	}
	
	// Auto-set trust domain from identity if available
	if identity != nil {
		policy.TrustDomain = identity.Domain()
	}
	
	return policy
}

// NewAuthorizationPolicy creates a policy with both authentication and authorization.
// This enforces access control based on SPIFFE ID allowlists.
// String inputs are parsed and validated as SPIFFE IDs at construction time.
func NewAuthorizationPolicy(identity *ServiceIdentity, authorizedClients, trustedServers []string) (*AuthenticationPolicy, error) {
	// Parse and validate authorized clients
	var clientIDs []spiffeid.ID
	for i, clientStr := range authorizedClients {
		clientID, err := spiffeid.FromString(clientStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SPIFFE ID in authorized clients at index %d (%q): %w", i, clientStr, err)
		}
		clientIDs = append(clientIDs, clientID)
	}
	
	// Parse and validate trusted servers
	var serverIDs []spiffeid.ID
	for i, serverStr := range trustedServers {
		serverID, err := spiffeid.FromString(serverStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SPIFFE ID in trusted servers at index %d (%q): %w", i, serverStr, err)
		}
		serverIDs = append(serverIDs, serverID)
	}
	
	policy := &AuthenticationPolicy{
		ServiceIdentity:   identity,
		AuthorizedClients: clientIDs,
		TrustedServers:    serverIDs,
	}
	
	// Auto-set trust domain from identity if not already set
	if policy.TrustDomain == "" && identity != nil {
		policy.TrustDomain = identity.Domain()
	}
	
	return policy, nil
}

// HasAuthorization returns true if this policy includes authorization rules.
func (p *AuthenticationPolicy) HasAuthorization() bool {
	return len(p.AuthorizedClients) > 0 || len(p.TrustedServers) > 0
}

// IsClientAuthorized checks if a client SPIFFE ID is authorized to connect (server-side).
func (p *AuthenticationPolicy) IsClientAuthorized(clientSPIFFEID spiffeid.ID) bool {
	if len(p.AuthorizedClients) == 0 {
		// No explicit authorization rules - allow same trust domain
		if p.TrustDomain != "" {
			return clientSPIFFEID.TrustDomain().String() == p.TrustDomain
		}
		return true
	}
	
	for _, authorized := range p.AuthorizedClients {
		if authorized.String() == clientSPIFFEID.String() {
			return true
		}
	}
	return false
}

// IsClientAuthorizedString checks if a client SPIFFE ID string is authorized (convenience method).
func (p *AuthenticationPolicy) IsClientAuthorizedString(clientSPIFFEID string) bool {
	clientID, err := spiffeid.FromString(clientSPIFFEID)
	if err != nil {
		return false // Invalid SPIFFE ID is not authorized
	}
	return p.IsClientAuthorized(clientID)
}

// IsServerTrusted checks if a server SPIFFE ID is trusted for connections (client-side).
func (p *AuthenticationPolicy) IsServerTrusted(serverSPIFFEID spiffeid.ID) bool {
	if len(p.TrustedServers) == 0 {
		// No explicit trust rules - trust same trust domain
		if p.TrustDomain != "" {
			return serverSPIFFEID.TrustDomain().String() == p.TrustDomain
		}
		return true
	}
	
	for _, trusted := range p.TrustedServers {
		if trusted.String() == serverSPIFFEID.String() {
			return true
		}
	}
	return false
}

// IsServerTrustedString checks if a server SPIFFE ID string is trusted (convenience method).
func (p *AuthenticationPolicy) IsServerTrustedString(serverSPIFFEID string) bool {
	serverID, err := spiffeid.FromString(serverSPIFFEID)
	if err != nil {
		return false // Invalid SPIFFE ID is not trusted
	}
	return p.IsServerTrusted(serverID)
}
