// Package ports defines the identity verification and diagnostics interfaces for Ephemos.
// These interfaces follow the hexagonal architecture pattern and enable
// integration with SPIRE's built-in identity verification and diagnostic capabilities.
package ports

import (
	"context"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/sufield/ephemos/internal/core/domain"
)

// IdentityVerificationResult contains the result of an identity verification
type IdentityVerificationResult struct {
	// Valid indicates if the identity verification passed
	Valid bool `json:"valid"`
	// Identity is the verified SPIFFE ID
	Identity spiffeid.ID `json:"identity"`
	// TrustDomain is the trust domain of the identity
	TrustDomain domain.TrustDomain `json:"trust_domain"`
	// NotBefore is when the identity becomes valid
	NotBefore time.Time `json:"not_before"`
	// NotAfter is when the identity expires
	NotAfter time.Time `json:"not_after"`
	// SerialNumber is the certificate serial number
	SerialNumber string `json:"serial_number"`
	// Subject contains the certificate subject information
	Subject string `json:"subject"`
	// Issuer contains the certificate issuer information
	Issuer string `json:"issuer"`
	// Note: Key usage details available via SPIRE CLI tools
	// Message provides additional details about the verification
	Message string `json:"message,omitempty"`
	// VerifiedAt is when the verification was performed
	VerifiedAt time.Time `json:"verified_at"`
	// Details contains additional verification information
	Details map[string]interface{} `json:"details,omitempty"`
}

// IdentityInfo contains comprehensive identity information
type IdentityInfo struct {
	// SPIFFEID is the workload's SPIFFE ID
	SPIFFEID spiffeid.ID `json:"spiffe_id"`
	// SVID contains the X.509 SVID
	SVID *x509svid.SVID `json:"svid"`
	// TrustBundle contains the trust bundle
	TrustBundle *x509bundle.Bundle `json:"trust_bundle"`
	// FetchedAt is when the identity was fetched
	FetchedAt time.Time `json:"fetched_at"`
	// Source indicates where the identity was obtained from
	Source string `json:"source"`
}

// DiagnosticInfo contains SPIRE diagnostic information
type DiagnosticInfo struct {
	// Component is the SPIRE component being diagnosed (server/agent)
	Component string `json:"component"`
	// Version is the SPIRE version
	Version string `json:"version"`
	// Status indicates the overall component status
	Status string `json:"status"`
	// Uptime is how long the component has been running
	Uptime time.Duration `json:"uptime"`
	// TrustDomain is the configured trust domain
	TrustDomain domain.TrustDomain `json:"trust_domain"`
	// Entries contains information about registration entries
	Entries *RegistrationEntryInfo `json:"entries,omitempty"`
	// Bundles contains information about trust bundles
	Bundles *TrustBundleInfo `json:"bundles,omitempty"`
	// Agents contains information about agents (server only)
	Agents *AgentInfo `json:"agents,omitempty"`
	// CollectedAt is when the diagnostic info was collected
	CollectedAt time.Time `json:"collected_at"`
	// Details contains component-specific diagnostic data
	Details map[string]interface{} `json:"details,omitempty"`
}

// RegistrationEntryInfo contains information about registration entries
type RegistrationEntryInfo struct {
	// Total number of registration entries
	Total int `json:"total"`
	// Recent entries created in the last 24 hours
	Recent int `json:"recent"`
	// Expired entries
	Expired int `json:"expired"`
	// Entries by selector type
	BySelector map[string]int `json:"by_selector"`
}

// TrustBundleInfo contains information about trust bundles
type TrustBundleInfo struct {
	// Local trust bundle information
	Local *BundleInfo `json:"local"`
	// Federated bundles
	Federated map[string]*BundleInfo `json:"federated"`
}

// BundleInfo contains trust bundle details
type BundleInfo struct {
	// TrustDomain of the bundle
	TrustDomain domain.TrustDomain `json:"trust_domain"`
	// CertificateCount is the number of certificates in the bundle
	CertificateCount int `json:"certificate_count"`
	// LastUpdated is when the bundle was last updated
	LastUpdated time.Time `json:"last_updated"`
	// ExpiresAt is when the bundle expires
	ExpiresAt time.Time `json:"expires_at"`
}

// AgentInfo contains information about SPIRE agents
type AgentInfo struct {
	// Total number of agents
	Total int `json:"total"`
	// Active agents
	Active int `json:"active"`
	// Inactive agents
	Inactive int `json:"inactive"`
	// Banned agents
	Banned int `json:"banned"`
	// Recent agents attested in the last 24 hours
	Recent int `json:"recent"`
}

// IdentityVerifierPort defines the interface for verifying SPIFFE identities
type IdentityVerifierPort interface {
	// VerifyIdentity verifies a SPIFFE identity using the Workload API
	VerifyIdentity(ctx context.Context, expectedID spiffeid.ID) (*IdentityVerificationResult, error)
	// GetCurrentIdentity fetches the current workload identity
	GetCurrentIdentity(ctx context.Context) (*IdentityInfo, error)
	// ValidateConnection validates a connection to a specific SPIFFE ID
	ValidateConnection(ctx context.Context, targetID spiffeid.ID, address string) (*IdentityVerificationResult, error)
	// RefreshIdentity forces a refresh of the workload identity
	RefreshIdentity(ctx context.Context) (*IdentityInfo, error)
}

// DiagnosticsProviderPort defines the interface for SPIRE diagnostics
type DiagnosticsProviderPort interface {
	// GetServerDiagnostics retrieves SPIRE server diagnostic information
	GetServerDiagnostics(ctx context.Context) (*DiagnosticInfo, error)
	// GetAgentDiagnostics retrieves SPIRE agent diagnostic information
	GetAgentDiagnostics(ctx context.Context) (*DiagnosticInfo, error)
	// ListRegistrationEntries lists all registration entries
	ListRegistrationEntries(ctx context.Context) ([]*RegistrationEntry, error)
	// ShowTrustBundle displays trust bundle information
	ShowTrustBundle(ctx context.Context, trustDomain domain.TrustDomain) (*TrustBundleInfo, error)
	// ListAgents lists all connected agents (server only)
	ListAgents(ctx context.Context) ([]*Agent, error)
	// GetComponentVersion gets the version of a SPIRE component
	GetComponentVersion(ctx context.Context, component string) (string, error)
}

// RegistrationEntry represents a SPIRE registration entry
type RegistrationEntry struct {
	// ID is the entry ID
	ID string `json:"id"`
	// SPIFFEID is the SPIFFE ID assigned to matching workloads
	SPIFFEID spiffeid.ID `json:"spiffe_id"`
	// ParentID is the SPIFFE ID of the parent
	ParentID spiffeid.ID `json:"parent_id"`
	// Selectors are the workload selectors
	Selectors []string `json:"selectors"`
	// TTL is the time-to-live for SVIDs issued for this entry
	TTL int32 `json:"ttl"`
	// FederatesWith lists trust domains this entry federates with
	FederatesWith []domain.TrustDomain `json:"federates_with"`
	// DNSNames are DNS SANs for X.509 SVIDs
	DNSNames []string `json:"dns_names"`
	// Admin indicates if this is an admin entry
	Admin bool `json:"admin"`
	// CreatedAt is when the entry was created
	CreatedAt time.Time `json:"created_at"`
	// Downstream indicates if this is a downstream entry
	Downstream bool `json:"downstream"`
}

// Agent represents a SPIRE agent
type Agent struct {
	// ID is the agent ID (usually the SPIFFE ID)
	ID spiffeid.ID `json:"id"`
	// AttestationType is how the agent was attested
	AttestationType string `json:"attestation_type"`
	// SerialNumber is the agent's certificate serial number
	SerialNumber string `json:"serial_number"`
	// ExpiresAt is when the agent's certificate expires
	ExpiresAt time.Time `json:"expires_at"`
	// Banned indicates if the agent is banned
	Banned bool `json:"banned"`
	// CanReattest indicates if the agent can re-attest
	CanReattest bool `json:"can_reattest"`
	// Selectors are the selectors used for agent attestation
	Selectors []string `json:"selectors"`
}

// VerificationConfig configures identity verification behavior
type VerificationConfig struct {
	// WorkloadAPISocket is the path to the Workload API socket
	WorkloadAPISocket string `json:"workload_api_socket"`
	// Timeout for verification operations
	Timeout time.Duration `json:"timeout"`
	// TrustDomain to verify against
	TrustDomain domain.TrustDomain `json:"trust_domain"`
	// AllowedSPIFFEIDs restricts which SPIFFE IDs are accepted
	AllowedSPIFFEIDs []spiffeid.ID `json:"allowed_spiffe_ids"`
	// RequireSVID indicates if SVID presence is required
	RequireSVID bool `json:"require_svid"`
}

// DiagnosticsConfig configures SPIRE diagnostics behavior
type DiagnosticsConfig struct {
	// ServerSocketPath is the path to the SPIRE server socket
	ServerSocketPath string `json:"server_socket_path"`
	// AgentSocketPath is the path to the SPIRE agent socket (for workload API)
	AgentSocketPath string `json:"agent_socket_path"`
	// ServerAddress is the SPIRE server API address
	ServerAddress string `json:"server_address"`
	// Timeout for diagnostic operations
	Timeout time.Duration `json:"timeout"`
	// UseServerAPI indicates whether to use the server API vs CLI
	UseServerAPI bool `json:"use_server_api"`
	// ServerAPIToken for authentication (if required)
	ServerAPIToken string `json:"server_api_token"`
}
