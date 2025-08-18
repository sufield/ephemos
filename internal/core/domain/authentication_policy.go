// Package domain handles service identity and authentication policies.
package domain

import (
	"fmt"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
)

// AuthenticationPolicy defines authentication and authorization context.
// This includes both identity verification and access control policies.
type AuthenticationPolicy struct {
	ServiceIdentity   *ServiceIdentity
	AuthorizedClients []spiffeid.ID // For server-side: SPIFFE IDs allowed to connect
	TrustedServers    []spiffeid.ID // For client-side: SPIFFE IDs this service trusts
	TrustDomain       TrustDomain   // Trust domain for authorization
	AllowedSPIFFEIDs  []spiffeid.ID // Specific SPIFFE IDs for precise authorization
	RequireAuth       bool          // Whether authentication is required
}

// NewAuthenticationPolicy creates a policy for authentication only.
// For authorization support, use NewAuthorizationPolicy instead.
func NewAuthenticationPolicy(identity *ServiceIdentity) *AuthenticationPolicy {
	policy := &AuthenticationPolicy{
		ServiceIdentity: identity,
	}

	// Auto-set trust domain from identity if available
	if identity != nil {
		td, _ := NewTrustDomain(identity.Domain())
		policy.TrustDomain = td
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
	if policy.TrustDomain.IsZero() && identity != nil {
		td, _ := NewTrustDomain(identity.Domain())
		policy.TrustDomain = td
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
		if !p.TrustDomain.IsZero() {
			clientTD := TrustDomain(clientSPIFFEID.TrustDomain().String())
			return p.TrustDomain.Equals(clientTD)
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
		if !p.TrustDomain.IsZero() {
			serverTD := TrustDomain(serverSPIFFEID.TrustDomain().String())
			return p.TrustDomain.Equals(serverTD)
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