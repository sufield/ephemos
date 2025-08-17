// Package ephemos provides identity-based authentication for backend services.
package ephemos

import (
	"fmt"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
)

// Authorizer validates peer certificates during mTLS handshake.
// It is used by both core and contrib components for authentication.
type Authorizer = tlsconfig.Authorizer

// AuthorizeMemberOf returns an Authorizer that validates the peer certificate
// belongs to the specified trust domain.
//
// Example:
//
//	authorizer := ephemos.AuthorizeMemberOf("prod.company.com")
//	// This will accept any service in the prod.company.com trust domain
func AuthorizeMemberOf(trustDomain string) (Authorizer, error) {
	td, err := spiffeid.TrustDomainFromString(trustDomain)
	if err != nil {
		return nil, fmt.Errorf("invalid trust domain %q: %w", trustDomain, err)
	}
	return tlsconfig.AuthorizeMemberOf(td), nil
}

// AuthorizeID returns an Authorizer that validates the peer certificate
// has exactly the specified SPIFFE ID.
//
// Example:
//
//	authorizer, err := ephemos.AuthorizeID("spiffe://prod.company.com/payment-service")
//	// This will only accept the payment-service
func AuthorizeID(spiffeID string) (Authorizer, error) {
	id, err := spiffeid.FromString(spiffeID)
	if err != nil {
		return nil, fmt.Errorf("invalid SPIFFE ID %q: %w", spiffeID, err)
	}
	return tlsconfig.AuthorizeID(id), nil
}

// AuthorizeAny returns an Authorizer that accepts any valid SPIFFE certificate.
// This is useful for development but should not be used in production.
//
// Example:
//
//	authorizer := ephemos.AuthorizeAny()
//	// This will accept any valid SPIFFE certificate
func AuthorizeAny() Authorizer {
	return tlsconfig.AuthorizeAny()
}

// AuthorizeOneOf returns an Authorizer that validates the peer certificate
// matches any of the specified SPIFFE IDs.
//
// Example:
//
//	authorizer, err := ephemos.AuthorizeOneOf(
//	    "spiffe://prod.company.com/payment-service",
//	    "spiffe://prod.company.com/billing-service",
//	)
//	// This will accept either payment-service or billing-service
func AuthorizeOneOf(spiffeIDs ...string) (Authorizer, error) {
	var ids []spiffeid.ID
	for _, idStr := range spiffeIDs {
		id, err := spiffeid.FromString(idStr)
		if err != nil {
			return nil, fmt.Errorf("invalid SPIFFE ID %q: %w", idStr, err)
		}
		ids = append(ids, id)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no SPIFFE IDs provided")
	}

	return tlsconfig.AuthorizeOneOf(ids...), nil
}

// AuthorizationPolicy represents a configurable authorization policy.
// This provides a declarative way to configure authorization.
type AuthorizationPolicy struct {
	// TrustDomain specifies which trust domain is allowed.
	// If empty, any trust domain is allowed.
	TrustDomain string

	// AllowedServices specifies exact SPIFFE IDs that are allowed.
	// If empty, this constraint is not applied.
	AllowedServices []string

	// AllowAny allows any valid SPIFFE certificate.
	// This overrides other settings and should only be used in development.
	AllowAny bool
}

// NewAuthorizerFromPolicy creates an Authorizer from an AuthorizationPolicy.
// This provides a declarative way to configure authorization.
//
// Example:
//
//	policy := &ephemos.AuthorizationPolicy{
//	    TrustDomain: "prod.company.com",
//	    AllowedServices: []string{
//	        "spiffe://prod.company.com/payment-service",
//	        "spiffe://prod.company.com/billing-service",
//	    },
//	}
//	authorizer, err := ephemos.NewAuthorizerFromPolicy(policy)
func NewAuthorizerFromPolicy(policy *AuthorizationPolicy) (Authorizer, error) {
	if policy == nil {
		return nil, fmt.Errorf("authorization policy is nil")
	}

	// If AllowAny is set, return an authorizer that accepts any SPIFFE certificate
	if policy.AllowAny {
		return AuthorizeAny(), nil
	}

	// If specific services are listed, authorize those
	if len(policy.AllowedServices) > 0 {
		return AuthorizeOneOf(policy.AllowedServices...)
	}

	// If only trust domain is specified, authorize membership
	if policy.TrustDomain != "" {
		return AuthorizeMemberOf(policy.TrustDomain)
	}

	// If no policy is specified, return an error
	return nil, fmt.Errorf("authorization policy must specify TrustDomain, AllowedServices, or AllowAny")
}
