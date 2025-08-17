// Package verification provides identity verification implementations using SPIRE's
// built-in capabilities through the go-spiffe/v2 library rather than implementing
// custom verification logic from scratch.
package verification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/workloadapi"

	"github.com/sufield/ephemos/internal/core/ports"
)

// SpireIdentityVerifier implements identity verification using SPIRE's Workload API
// and go-spiffe/v2 library, leveraging built-in verification mechanisms
type SpireIdentityVerifier struct {
	config *ports.VerificationConfig
	source *workloadapi.X509Source
}

// NewSpireIdentityVerifier creates a new SPIRE identity verifier using the Workload API
func NewSpireIdentityVerifier(config *ports.VerificationConfig) (*SpireIdentityVerifier, error) {
	if config == nil {
		return nil, fmt.Errorf("verification config cannot be nil")
	}

	if config.WorkloadAPISocket == "" {
		config.WorkloadAPISocket = "unix:///tmp/spire-agent/public/api.sock"
	}

	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &SpireIdentityVerifier{
		config: config,
	}, nil
}

// initializeSource creates and initializes the X509Source for Workload API access
func (v *SpireIdentityVerifier) initializeSource(ctx context.Context) error {
	if v.source != nil {
		return nil // Already initialized
	}

	clientOptions := workloadapi.WithClientOptions(
		workloadapi.WithAddr(v.config.WorkloadAPISocket),
	)

	source, err := workloadapi.NewX509Source(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("failed to create X509Source: %w", err)
	}

	v.source = source
	return nil
}

// VerifyIdentity verifies a SPIFFE identity using the Workload API
func (v *SpireIdentityVerifier) VerifyIdentity(ctx context.Context, expectedID spiffeid.ID) (*ports.IdentityVerificationResult, error) {
	if err := v.initializeSource(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize source: %w", err)
	}

	// Get current SVID from Workload API
	svid, err := v.source.GetX509SVID()
	if err != nil {
		return &ports.IdentityVerificationResult{
			Valid:      false,
			Identity:   spiffeid.ID{},
			Message:    fmt.Sprintf("Failed to get SVID: %v", err),
			VerifiedAt: time.Now(),
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}, err
	}

	// Verify the identity matches expected
	valid := svid.ID.String() == expectedID.String()
	result := &ports.IdentityVerificationResult{
		Valid:        valid,
		Identity:     svid.ID,
		TrustDomain:  svid.ID.TrustDomain(),
		NotBefore:    svid.Certificates[0].NotBefore,
		NotAfter:     svid.Certificates[0].NotAfter,
		SerialNumber: svid.Certificates[0].SerialNumber.String(),
		Subject:      svid.Certificates[0].Subject.String(),
		Issuer:       svid.Certificates[0].Issuer.String(),
		VerifiedAt:   time.Now(),
		Details: map[string]interface{}{
			"certificate_count": len(svid.Certificates),
			"has_private_key":   svid.PrivateKey != nil,
		},
	}

	// Note: Key usage details available via SPIRE CLI tools if needed

	if valid {
		result.Message = "Identity verification successful"

		// Verify trust domain if configured
		if !v.config.TrustDomain.IsZero() && svid.ID.TrustDomain().String() != v.config.TrustDomain.String() {
			result.Valid = false
			result.Message = fmt.Sprintf("Trust domain mismatch: expected %s, got %s",
				v.config.TrustDomain, svid.ID.TrustDomain())
		}

		// Check allowed SPIFFE IDs if configured
		if len(v.config.AllowedSPIFFEIDs) > 0 {
			allowed := false
			for _, allowedID := range v.config.AllowedSPIFFEIDs {
				if svid.ID.String() == allowedID.String() {
					allowed = true
					break
				}
			}
			if !allowed {
				result.Valid = false
				result.Message = fmt.Sprintf("SPIFFE ID %s not in allowed list", svid.ID)
			}
		}
	} else {
		result.Message = fmt.Sprintf("Identity mismatch: expected %s, got %s", expectedID, svid.ID)
	}

	return result, nil
}

// GetCurrentIdentity fetches the current workload identity
func (v *SpireIdentityVerifier) GetCurrentIdentity(ctx context.Context) (*ports.IdentityInfo, error) {
	if err := v.initializeSource(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize source: %w", err)
	}

	// Get SVID
	svid, err := v.source.GetX509SVID()
	if err != nil {
		return nil, fmt.Errorf("failed to get SVID: %w", err)
	}

	// Get trust bundle
	bundle, err := v.source.GetX509BundleForTrustDomain(svid.ID.TrustDomain())
	if err != nil {
		return nil, fmt.Errorf("failed to get trust bundle: %w", err)
	}

	return &ports.IdentityInfo{
		SPIFFEID:    svid.ID,
		SVID:        svid,
		TrustBundle: bundle,
		FetchedAt:   time.Now(),
		Source:      "workload-api",
	}, nil
}

// ValidateConnection validates a connection to a specific SPIFFE ID
func (v *SpireIdentityVerifier) ValidateConnection(ctx context.Context, targetID spiffeid.ID, address string) (*ports.IdentityVerificationResult, error) {
	if err := v.initializeSource(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize source: %w", err)
	}

	// Create TLS config that authorizes the target SPIFFE ID
	tlsConfig := tlsconfig.MTLSClientConfig(v.source, v.source, tlsconfig.AuthorizeID(targetID))

	// Attempt to connect
	dialer := &net.Dialer{
		Timeout: v.config.Timeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return &ports.IdentityVerificationResult{
			Valid:      false,
			Identity:   spiffeid.ID{},
			Message:    fmt.Sprintf("Failed to connect to %s: %v", address, err),
			VerifiedAt: time.Now(),
			Details: map[string]interface{}{
				"target_address": address,
				"target_id":      targetID.String(),
				"error":          err.Error(),
			},
		}, err
	}
	defer conn.Close()

	// Perform TLS handshake
	tlsConn := tls.Client(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return &ports.IdentityVerificationResult{
			Valid:      false,
			Identity:   spiffeid.ID{},
			Message:    fmt.Sprintf("TLS handshake failed: %v", err),
			VerifiedAt: time.Now(),
			Details: map[string]interface{}{
				"target_address": address,
				"target_id":      targetID.String(),
				"error":          err.Error(),
			},
		}, err
	}

	// Extract peer certificate and verify identity
	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return &ports.IdentityVerificationResult{
			Valid:      false,
			Identity:   spiffeid.ID{},
			Message:    "No peer certificates received",
			VerifiedAt: time.Now(),
			Details: map[string]interface{}{
				"target_address": address,
				"target_id":      targetID.String(),
			},
		}, fmt.Errorf("no peer certificates")
	}

	peerCert := state.PeerCertificates[0]

	// Use go-spiffe to extract SPIFFE ID from certificate
	var peerID spiffeid.ID
	for _, uri := range peerCert.URIs {
		if uri.Scheme == "spiffe" {
			if id, err := spiffeid.FromURI(uri); err == nil {
				peerID = id
				break
			}
		}
	}

	if peerID.IsZero() {
		return &ports.IdentityVerificationResult{
			Valid:      false,
			Identity:   spiffeid.ID{},
			Message:    "No valid SPIFFE ID found in peer certificate",
			VerifiedAt: time.Now(),
			Details: map[string]interface{}{
				"target_address": address,
				"target_id":      targetID.String(),
			},
		}, fmt.Errorf("no SPIFFE ID found in certificate")
	}

	valid := peerID.String() == targetID.String()
	result := &ports.IdentityVerificationResult{
		Valid:        valid,
		Identity:     peerID,
		TrustDomain:  peerID.TrustDomain(),
		NotBefore:    peerCert.NotBefore,
		NotAfter:     peerCert.NotAfter,
		SerialNumber: peerCert.SerialNumber.String(),
		Subject:      peerCert.Subject.String(),
		Issuer:       peerCert.Issuer.String(),
		// KeyUsage details available via SPIRE CLI tools if needed
		VerifiedAt: time.Now(),
		Details: map[string]interface{}{
			"target_address": address,
			"target_id":      targetID.String(),
			"peer_id":        peerID.String(),
			"tls_version":    fmt.Sprintf("0x%04x", state.Version),
			"cipher_suite":   fmt.Sprintf("0x%04x", state.CipherSuite),
		},
	}

	if valid {
		result.Message = fmt.Sprintf("Successfully validated connection to %s", targetID)
	} else {
		result.Message = fmt.Sprintf("Identity mismatch: expected %s, got %s", targetID, peerID)
	}

	return result, nil
}

// RefreshIdentity forces a refresh of the workload identity
func (v *SpireIdentityVerifier) RefreshIdentity(ctx context.Context) (*ports.IdentityInfo, error) {
	// Close existing source to force refresh
	if v.source != nil {
		v.source.Close()
		v.source = nil
	}

	// Re-initialize to get fresh identity
	if err := v.initializeSource(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh source: %w", err)
	}

	return v.GetCurrentIdentity(ctx)
}

// Close cleans up the identity verifier
func (v *SpireIdentityVerifier) Close() error {
	if v.source != nil {
		v.source.Close()
		v.source = nil
	}
	return nil
}

// Note: Detailed certificate inspection (key usage, TLS details, etc.)
// is available through SPIRE's built-in CLI tools:
// - spire-agent api fetch x509 for SVID details
// - openssl x509 commands for certificate parsing
// - SPIRE server APIs for comprehensive certificate information
