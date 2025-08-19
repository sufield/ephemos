// Package domain handles service identity and authentication policies.
package domain

// AuthenticationPolicy defines authentication context for identity verification.
// This focuses solely on identity authentication, not authorization.
type AuthenticationPolicy struct {
	ServiceIdentity *ServiceIdentity
	TrustDomain     TrustDomain // Trust domain for identity verification
	RequireAuth     bool        // Whether authentication is required
}

// NewAuthenticationPolicy creates a policy for identity authentication only.
// This policy focuses solely on identity verification, not authorization.
func NewAuthenticationPolicy(identity *ServiceIdentity) *AuthenticationPolicy {
	policy := &AuthenticationPolicy{
		ServiceIdentity: identity,
		RequireAuth:     true, // Always require authentication for security
	}

	// Auto-set trust domain from identity if available
	if identity != nil {
		td, _ := NewTrustDomain(identity.Domain())
		policy.TrustDomain = td
	}

	return policy
}
