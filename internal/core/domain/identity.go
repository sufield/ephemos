// Package domain contains core business logic and domain models.
//
// This package implements the domain layer of the hexagonal architecture pattern
// and contains the following key components:
//
// - ServiceIdentity: SPIFFE service identity representation
// - Certificate: X.509 SVID certificate management
// - TrustBundle: Trust anchor certificate bundles
// - AuthenticationPolicy: Authentication and authorization policies
// - TrustDomain: Trust domain value object
//
// The domain layer is independent of external frameworks and infrastructure
// concerns, ensuring clean separation of business logic from technical implementation details.
package domain