package ports

import (
	"errors"
	"testing"
)

// Mock implementation of ServiceIdentity for testing
type mockServiceIdentity struct {
	domain   string
	name     string
	shouldFailValidation bool
}

func (m *mockServiceIdentity) GetDomain() string {
	return m.domain
}

func (m *mockServiceIdentity) GetName() string {
	return m.name
}

func (m *mockServiceIdentity) Validate() error {
	if m.shouldFailValidation {
		return &ValidationError{
			Field:   "validation",
			Value:   "mock",
			Message: "mock validation failure",
		}
	}
	return nil
}

func (m *mockServiceIdentity) Close() error {
	return nil
}

// Mock implementation of IdentityProvider for testing
type mockIdentityProvider struct {
	identity       ServiceIdentity
	shouldFail     bool
	errorToReturn  error
	closeCalled    bool
}

func (m *mockIdentityProvider) GetServiceIdentity() (ServiceIdentity, error) {
	if m.shouldFail {
		if m.errorToReturn != nil {
			return nil, m.errorToReturn
		}
		return nil, ErrIdentityNotFound
	}
	return m.identity, nil
}

func (m *mockIdentityProvider) Close() error {
	m.closeCalled = true
	return nil
}

func TestIdentityProvider_Interface(t *testing.T) {
	// Test that mock implements the interface correctly
	var provider IdentityProvider = &mockIdentityProvider{
		identity: &mockServiceIdentity{
			domain: "example.com",
			name:   "test-service",
		},
	}

	// Test GetServiceIdentity
	identity, err := provider.GetServiceIdentity()
	if err != nil {
		t.Errorf("GetServiceIdentity() failed: %v", err)
	}

	if identity == nil {
		t.Error("GetServiceIdentity() returned nil identity")
	}

	// Test Close
	err = provider.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestServiceIdentity_Interface(t *testing.T) {
	// Test that mock implements the interface correctly
	var identity ServiceIdentity = &mockServiceIdentity{
		domain: "example.com",
		name:   "test-service",
	}

	// Test GetDomain
	domain := identity.GetDomain()
	if domain != "example.com" {
		t.Errorf("GetDomain() = %v, want example.com", domain)
	}

	// Test GetName  
	name := identity.GetName()
	if name != "test-service" {
		t.Errorf("GetName() = %v, want test-service", name)
	}

	// Test Validate
	err := identity.Validate()
	if err != nil {
		t.Errorf("Validate() failed: %v", err)
	}

	// Test Close
	err = identity.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

func TestIdentityProvider_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		provider      *mockIdentityProvider
		wantErr       bool
		expectedError error
	}{
		{
			name: "successful operation",
			provider: &mockIdentityProvider{
				identity: &mockServiceIdentity{
					domain: "example.com",
					name:   "test-service",
				},
			},
			wantErr: false,
		},
		{
			name: "identity not found",
			provider: &mockIdentityProvider{
				shouldFail: true,
			},
			wantErr:       true,
			expectedError: ErrIdentityNotFound,
		},
		{
			name: "custom error",
			provider: &mockIdentityProvider{
				shouldFail:    true,
				errorToReturn: errors.New("custom error"),
			},
			wantErr:       true,
			expectedError: errors.New("custom error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity, err := tt.provider.GetServiceIdentity()
			
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServiceIdentity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.expectedError != nil && !errors.Is(err, tt.expectedError) {
					// For custom errors, just check the message
					if err.Error() != tt.expectedError.Error() {
						t.Errorf("GetServiceIdentity() error = %v, want %v", err, tt.expectedError)
					}
				}
			} else {
				if identity == nil {
					t.Error("GetServiceIdentity() returned nil identity")
				}
			}
		})
	}
}

func TestServiceIdentity_Validation(t *testing.T) {
	tests := []struct {
		name     string
		identity *mockServiceIdentity
		wantErr  bool
	}{
		{
			name: "valid identity",
			identity: &mockServiceIdentity{
				domain: "example.com",
				name:   "test-service",
			},
			wantErr: false,
		},
		{
			name: "validation failure",
			identity: &mockServiceIdentity{
				domain: "example.com",
				name:   "test-service",
				shouldFailValidation: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.identity.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIdentityProvider_Lifecycle(t *testing.T) {
	// Test the complete lifecycle of an identity provider
	provider := &mockIdentityProvider{
		identity: &mockServiceIdentity{
			domain: "example.com",
			name:   "test-service",
		},
	}

	// Get identity
	identity, err := provider.GetServiceIdentity()
	if err != nil {
		t.Errorf("GetServiceIdentity() failed: %v", err)
	}

	if identity == nil {
		t.Error("GetServiceIdentity() returned nil")
	}

	// Use identity
	domain := identity.GetDomain()
	name := identity.GetName()

	if domain != "example.com" {
		t.Errorf("GetDomain() = %v, want example.com", domain)
	}

	if name != "test-service" {
		t.Errorf("GetName() = %v, want test-service", name)
	}

	// Validate identity
	err = identity.Validate()
	if err != nil {
		t.Errorf("Validate() failed: %v", err)
	}

	// Close identity
	err = identity.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	// Close provider
	err = provider.Close()
	if err != nil {
		t.Errorf("Close() failed: %v", err)
	}

	if !provider.closeCalled {
		t.Error("Close() was not called on provider")
	}
}

func TestErrIdentityNotFound(t *testing.T) {
	// Test the standard error
	if ErrIdentityNotFound == nil {
		t.Error("ErrIdentityNotFound should not be nil")
	}

	expectedMsg := "identity not found"
	if ErrIdentityNotFound.Error() != expectedMsg {
		t.Errorf("ErrIdentityNotFound.Error() = %v, want %v", ErrIdentityNotFound.Error(), expectedMsg)
	}

	// Test error comparison
	provider := &mockIdentityProvider{shouldFail: true}
	_, err := provider.GetServiceIdentity()

	if !errors.Is(err, ErrIdentityNotFound) {
		t.Error("Error should be ErrIdentityNotFound")
	}
}

func TestIdentityProvider_Concurrent(t *testing.T) {
	// Test concurrent access to identity provider
	provider := &mockIdentityProvider{
		identity: &mockServiceIdentity{
			domain: "example.com", 
			name:   "test-service",
		},
	}

	done := make(chan bool, 10)
	
	// Start multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			for j := 0; j < 100; j++ {
				identity, err := provider.GetServiceIdentity()
				if err != nil {
					t.Errorf("GetServiceIdentity() failed: %v", err)
					return
				}
				
				if identity == nil {
					t.Error("GetServiceIdentity() returned nil")
					return
				}
				
				// Use the identity
				domain := identity.GetDomain()
				name := identity.GetName()
				
				if domain != "example.com" || name != "test-service" {
					t.Errorf("Identity values incorrect: domain=%v, name=%v", domain, name)
					return
				}
				
				identity.Close()
			}
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkIdentityProvider_GetServiceIdentity(b *testing.B) {
	provider := &mockIdentityProvider{
		identity: &mockServiceIdentity{
			domain: "example.com",
			name:   "test-service",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		identity, err := provider.GetServiceIdentity()
		if err != nil {
			b.Errorf("GetServiceIdentity() failed: %v", err)
		}
		if identity != nil {
			identity.Close()
		}
	}
}

func BenchmarkServiceIdentity_Operations(b *testing.B) {
	identity := &mockServiceIdentity{
		domain: "example.com",
		name:   "test-service",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		domain := identity.GetDomain()
		name := identity.GetName()
		err := identity.Validate()
		
		if domain == "" || name == "" || err != nil {
			b.Error("Identity operations failed")
		}
	}
}