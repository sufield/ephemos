// Package domain contains core business logic and domain models.
package domain

import (
	"fmt"
	"time"
)

// WorkloadStatus represents the current status of a workload.
type WorkloadStatus string

const (
	WorkloadStatusActive     WorkloadStatus = "active"
	WorkloadStatusInactive   WorkloadStatus = "inactive" 
	WorkloadStatusPending    WorkloadStatus = "pending"
	WorkloadStatusTerminated WorkloadStatus = "terminated"
)

// Workload represents a SPIFFE workload with its associated identity and security context.
// This is a domain entity that aggregates identity namespace, identity documents, and trust bundles.
type Workload struct {
	// Core identity information using our value objects
	identity      IdentityNamespace
	trustDomain   TrustDomain
	
	// Security materials
	identityDoc   *IdentityDocument
	trustBundle   *TrustBundle
	
	// Workload metadata
	id            string
	status        WorkloadStatus
	createdAt     time.Time
	lastUpdated   time.Time
	
	// Optional metadata
	labels        map[string]string
	annotations   map[string]string
}

// WorkloadConfig provides configuration for creating a new workload.
type WorkloadConfig struct {
	ID              string
	Identity        IdentityNamespace
	TrustDomain     TrustDomain
	IdentityDoc     *IdentityDocument
	TrustBundle     *TrustBundle
	Labels          map[string]string
	Annotations     map[string]string
	Status          WorkloadStatus
}

// NewWorkload creates a new Workload with validation.
func NewWorkload(config WorkloadConfig) (*Workload, error) {
	// Use domain predicate instead of primitive string check
	workloadID := WorkloadID(config.ID)
	if workloadID.IsEmpty() {
		return nil, fmt.Errorf("workload ID cannot be empty")
	}
	
	if config.Identity.IsZero() {
		return nil, fmt.Errorf("workload identity namespace cannot be empty")
	}
	
	if config.TrustDomain.IsZero() {
		return nil, fmt.Errorf("workload trust domain cannot be empty")
	}
	
	// Validate that identity namespace trust domain matches workload trust domain
	if !config.Identity.GetTrustDomain().Equals(config.TrustDomain) {
		return nil, fmt.Errorf("workload trust domain %q does not match identity namespace trust domain %q",
			config.TrustDomain.String(), config.Identity.GetTrustDomain().String())
	}
	
	// Validate identity document if provided
	if config.IdentityDoc != nil {
		if err := config.IdentityDoc.Validate(); err != nil {
			return nil, fmt.Errorf("invalid identity document: %w", err)
		}
		
		// Verify that identity document SPIFFE ID matches workload identity
		docIdentity, err := config.IdentityDoc.GetIdentityNamespace()
		if err != nil {
			return nil, fmt.Errorf("failed to extract identity from document: %w", err)
		}
		
		if !docIdentity.Equals(config.Identity) {
			return nil, fmt.Errorf("identity document SPIFFE ID %q does not match workload identity %q",
				docIdentity.String(), config.Identity.String())
		}
	}
	
	// Validate trust bundle if provided
	if config.TrustBundle != nil {
		if err := config.TrustBundle.Validate(); err != nil {
			return nil, fmt.Errorf("invalid trust bundle: %w", err)
		}
	}
	
	// Set default status if not provided
	status := config.Status
	if status == "" {
		status = WorkloadStatusPending
	}
	
	// Copy labels and annotations to avoid external mutation
	labels := make(map[string]string)
	for k, v := range config.Labels {
		labels[k] = v
	}
	
	annotations := make(map[string]string)
	for k, v := range config.Annotations {
		annotations[k] = v
	}
	
	now := time.Now()
	
	return &Workload{
		identity:      config.Identity,
		trustDomain:   config.TrustDomain,
		identityDoc:   config.IdentityDoc,
		trustBundle:   config.TrustBundle,
		id:            config.ID,
		status:        status,
		createdAt:     now,
		lastUpdated:   now,
		labels:        labels,
		annotations:   annotations,
	}, nil
}

// ID returns the workload's unique identifier.
func (w *Workload) ID() string {
	return w.id
}

// Identity returns the workload's identity namespace.
func (w *Workload) Identity() IdentityNamespace {
	return w.identity
}

// TrustDomain returns the workload's trust domain.
func (w *Workload) TrustDomain() TrustDomain {
	return w.trustDomain
}

// IdentityDocument returns the workload's identity document.
func (w *Workload) IdentityDocument() *IdentityDocument {
	return w.identityDoc
}

// TrustBundle returns the workload's trust bundle.
func (w *Workload) TrustBundle() *TrustBundle {
	return w.trustBundle
}

// Status returns the workload's current status.
func (w *Workload) Status() WorkloadStatus {
	return w.status
}

// CreatedAt returns when the workload was created.
func (w *Workload) CreatedAt() time.Time {
	return w.createdAt
}

// LastUpdated returns when the workload was last updated.
func (w *Workload) LastUpdated() time.Time {
	return w.lastUpdated
}

// Labels returns a copy of the workload's labels.
func (w *Workload) Labels() map[string]string {
	labels := make(map[string]string)
	for k, v := range w.labels {
		labels[k] = v
	}
	return labels
}

// Annotations returns a copy of the workload's annotations.
func (w *Workload) Annotations() map[string]string {
	annotations := make(map[string]string)
	for k, v := range w.annotations {
		annotations[k] = v
	}
	return annotations
}

// UpdateIdentityDocument updates the workload's identity document with validation.
func (w *Workload) UpdateIdentityDocument(doc *IdentityDocument) error {
	if doc == nil {
		w.identityDoc = nil
		w.lastUpdated = time.Now()
		return nil
	}
	
	// Validate the new identity document
	if err := doc.Validate(); err != nil {
		return fmt.Errorf("invalid identity document: %w", err)
	}
	
	// Verify that identity document SPIFFE ID matches workload identity
	docIdentity, err := doc.GetIdentityNamespace()
	if err != nil {
		return fmt.Errorf("failed to extract identity from document: %w", err)
	}
	
	if !docIdentity.Equals(w.identity) {
		return fmt.Errorf("identity document SPIFFE ID %q does not match workload identity %q",
			docIdentity.String(), w.identity.String())
	}
	
	w.identityDoc = doc
	w.lastUpdated = time.Now()
	return nil
}

// UpdateTrustBundle updates the workload's trust bundle with validation.
func (w *Workload) UpdateTrustBundle(bundle *TrustBundle) error {
	if bundle == nil {
		w.trustBundle = nil
		w.lastUpdated = time.Now()
		return nil
	}
	
	// Validate the new trust bundle
	if err := bundle.Validate(); err != nil {
		return fmt.Errorf("invalid trust bundle: %w", err)
	}
	
	w.trustBundle = bundle
	w.lastUpdated = time.Now()
	return nil
}

// UpdateStatus updates the workload's status.
func (w *Workload) UpdateStatus(status WorkloadStatus) {
	w.status = status
	w.lastUpdated = time.Now()
}

// AddLabel adds or updates a label on the workload.
func (w *Workload) AddLabel(key, value string) {
	if w.labels == nil {
		w.labels = make(map[string]string)
	}
	w.labels[key] = value
	w.lastUpdated = time.Now()
}

// RemoveLabel removes a label from the workload.
func (w *Workload) RemoveLabel(key string) {
	if w.labels != nil {
		delete(w.labels, key)
		w.lastUpdated = time.Now()
	}
}

// AddAnnotation adds or updates an annotation on the workload.
func (w *Workload) AddAnnotation(key, value string) {
	if w.annotations == nil {
		w.annotations = make(map[string]string)
	}
	w.annotations[key] = value
	w.lastUpdated = time.Now()
}

// RemoveAnnotation removes an annotation from the workload.
func (w *Workload) RemoveAnnotation(key string) {
	if w.annotations != nil {
		delete(w.annotations, key)
		w.lastUpdated = time.Now()
	}
}

// IsActive returns true if the workload is in an active state.
func (w *Workload) IsActive() bool {
	return w.status == WorkloadStatusActive
}

// HasValidIdentity returns true if the workload has a valid, non-expired identity document.
func (w *Workload) HasValidIdentity() bool {
	if w.identityDoc == nil {
		return false
	}
	
	return !w.identityDoc.IsExpired(time.Now())
}

// GetServiceName extracts the service name from the workload's identity.
func (w *Workload) GetServiceName() string {
	return w.identity.GetServiceName()
}

// Validate performs comprehensive validation of the workload's state.
func (w *Workload) Validate() error {
	// Use domain predicate instead of primitive string check
	workloadID := WorkloadID(w.id)
	if workloadID.IsEmpty() {
		return fmt.Errorf("workload ID cannot be empty")
	}
	
	if w.identity.IsZero() {
		return fmt.Errorf("workload identity namespace cannot be empty")
	}
	
	if w.trustDomain.IsZero() {
		return fmt.Errorf("workload trust domain cannot be empty")
	}
	
	// Validate that identity namespace trust domain matches workload trust domain
	if !w.identity.GetTrustDomain().Equals(w.trustDomain) {
		return fmt.Errorf("workload trust domain %q does not match identity namespace trust domain %q",
			w.trustDomain.String(), w.identity.GetTrustDomain().String())
	}
	
	// Validate identity document if present
	if w.identityDoc != nil {
		if err := w.identityDoc.Validate(); err != nil {
			return fmt.Errorf("invalid identity document: %w", err)
		}
		
		// Verify that identity document SPIFFE ID matches workload identity
		docIdentity, err := w.identityDoc.GetIdentityNamespace()
		if err != nil {
			return fmt.Errorf("failed to extract identity from document: %w", err)
		}
		
		if !docIdentity.Equals(w.identity) {
			return fmt.Errorf("identity document SPIFFE ID %q does not match workload identity %q",
				docIdentity.String(), w.identity.String())
		}
	}
	
	// Validate trust bundle if present
	if w.trustBundle != nil {
		if err := w.trustBundle.Validate(); err != nil {
			return fmt.Errorf("invalid trust bundle: %w", err)
		}
	}
	
	return nil
}

// String returns a string representation of the workload for debugging.
func (w *Workload) String() string {
	return fmt.Sprintf("Workload{ID:%s, Identity:%s, Status:%s}", 
		w.id, w.identity.String(), w.status)
}