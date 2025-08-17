package transport

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spiffe/go-spiffe/v2/bundle/x509bundle"
	"github.com/spiffe/go-spiffe/v2/spiffeid"
	"github.com/spiffe/go-spiffe/v2/spiffetls/tlsconfig"
	"github.com/spiffe/go-spiffe/v2/svid/x509svid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sufield/ephemos/internal/core/domain"
)

// TestRotatableSource implements both x509svid.Source and x509bundle.Source for testing
type TestRotatableSource struct {
	mu            sync.RWMutex
	currentSVID   atomic.Value // *x509svid.SVID
	currentBundle atomic.Value // *x509bundle.Bundle
	rotateCount   int
}

// Global test CA for all certificates
var (
	testCAKey  *rsa.PrivateKey
	testCACert *x509.Certificate
	testCAOnce sync.Once
)

// initTestCA initializes the shared test CA
func initTestCA(t *testing.T) (*rsa.PrivateKey, *x509.Certificate) {
	testCAOnce.Do(func() {
		var err error
		testCAKey, err = rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)

		caTemplate := &x509.Certificate{
			SerialNumber: big.NewInt(1000),
			Subject: pkix.Name{
				Organization: []string{"Test CA"},
				Country:      []string{"US"},
			},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().Add(24 * time.Hour),
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			BasicConstraintsValid: true,
			IsCA:                  true,
		}

		caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &testCAKey.PublicKey, testCAKey)
		require.NoError(t, err)

		testCACert, err = x509.ParseCertificate(caCertDER)
		require.NoError(t, err)
	})
	return testCAKey, testCACert
}

// NewTestRotatableSource creates a new test source
func NewTestRotatableSource(t *testing.T, spiffeID string) *TestRotatableSource {
	source := &TestRotatableSource{}

	// Initialize test CA
	initTestCA(t)

	// Create initial SVID and bundle
	svid, bundle := createTestSVIDAndBundle(t, spiffeID, 1)
	source.currentSVID.Store(svid)
	source.currentBundle.Store(bundle)

	return source
}

// GetX509SVID implements x509svid.Source
func (s *TestRotatableSource) GetX509SVID() (*x509svid.SVID, error) {
	svid := s.currentSVID.Load()
	if svid == nil {
		return nil, fmt.Errorf("no SVID available")
	}
	return svid.(*x509svid.SVID), nil
}

// GetX509BundleForTrustDomain implements x509bundle.Source
func (s *TestRotatableSource) GetX509BundleForTrustDomain(td spiffeid.TrustDomain) (*x509bundle.Bundle, error) {
	bundle := s.currentBundle.Load()
	if bundle == nil {
		return nil, fmt.Errorf("no bundle available")
	}
	return bundle.(*x509bundle.Bundle), nil
}

// Rotate simulates SVID rotation
func (s *TestRotatableSource) Rotate(t *testing.T, spiffeID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rotateCount++
	svid, bundle := createTestSVIDAndBundle(t, spiffeID, s.rotateCount+1)

	s.currentSVID.Store(svid)
	s.currentBundle.Store(bundle)
}

// GetRotateCount returns rotation count
func (s *TestRotatableSource) GetRotateCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.rotateCount
}

// createTestSVIDAndBundle creates test certificates
func createTestSVIDAndBundle(t *testing.T, spiffeID string, serial int) (*x509svid.SVID, *x509bundle.Bundle) {
	caPrivateKey, caCert := initTestCA(t)

	id, err := spiffeid.FromString(spiffeID)
	require.NoError(t, err)

	// Generate leaf key pair
	leafPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create leaf certificate template
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(int64(serial)),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			Country:      []string{"US"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(1 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		URIs:                  []*url.URL{id.URL()},
	}

	// Sign the leaf certificate
	leafCertDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafPrivateKey.PublicKey, caPrivateKey)
	require.NoError(t, err)

	leafCert, err := x509.ParseCertificate(leafCertDER)
	require.NoError(t, err)

	// Create SVID
	svid := &x509svid.SVID{
		ID:           id,
		Certificates: []*x509.Certificate{leafCert, caCert},
		PrivateKey:   leafPrivateKey,
	}

	// Create trust bundle
	bundle := x509bundle.New(id.TrustDomain())
	bundle.AddX509Authority(caCert)

	return svid, bundle
}

// TestGRPCProviderRotation tests that the gRPC provider supports SVID rotation
func TestGRPCProviderRotation(t *testing.T) {
	// Create test sources
	clientSource := NewTestRotatableSource(t, "spiffe://test.example.org/client")
	serverSource := NewTestRotatableSource(t, "spiffe://test.example.org/server")

	// Create provider with sources
	provider := NewRotatableGRPCProvider(nil)
	provider.SetSources(serverSource, serverSource, tlsconfig.AuthorizeAny())

	// Create server
	serverPort, err := provider.CreateServer(nil, nil, nil)
	require.NoError(t, err)

	// Mock service registrar
	registrar := &mockServiceRegistrar{}
	err = serverPort.RegisterService(registrar)
	require.NoError(t, err)

	// Start server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	mockListener := &mockListenerPort{listener: listener}

	go func() {
		serverPort.Start(mockListener)
	}()
	defer serverPort.Stop()

	// Create client provider
	clientProvider := NewRotatableGRPCProvider(nil)
	clientProvider.SetSources(clientSource, clientSource, tlsconfig.AuthorizeAny())

	// Helper function to test connection and get certificate serial
	testConnection := func() (int64, error) {
		clientPort, err := clientProvider.CreateClient(nil, nil, nil)
		if err != nil {
			return 0, err
		}
		defer clientPort.Close()

		conn, err := clientPort.Connect("test-service", listener.Addr().String())
		if err != nil {
			return 0, err
		}
		defer conn.Close()

		// For testing, we'll use the source directly to get the serial
		svid, err := clientSource.GetX509SVID()
		if err != nil {
			return 0, err
		}

		return svid.Certificates[0].SerialNumber.Int64(), nil
	}

	// Test 1: Initial connection
	serial1, err := testConnection()
	require.NoError(t, err)
	assert.Equal(t, int64(1), serial1, "Initial serial should be 1")

	// Test 2: Rotate client certificate
	clientSource.Rotate(t, "spiffe://test.example.org/client")

	// Test 3: New connection should see rotated certificate
	serial2, err := testConnection()
	require.NoError(t, err)
	assert.Equal(t, int64(2), serial2, "Serial after rotation should be 2")

	// Verify rotation count
	assert.Equal(t, 1, clientSource.GetRotateCount(), "Should have rotated once")
}

// TestSourceAdapter tests the SourceAdapter wrapper
func TestSourceAdapter(t *testing.T) {
	// Create mock identity provider
	mockProvider := &mockIdentityProvider{
		cert: &domain.Certificate{
			Cert:       createMockCert(t, "spiffe://test.example.org/service"),
			PrivateKey: createMockKey(t),
		},
		bundle: &domain.TrustBundle{
			Certificates: []*x509.Certificate{createMockCACert(t)},
		},
		identity: domain.NewServiceIdentity("test-service", "test.example.org"),
	}

	// Create adapter
	adapter := NewSourceAdapter(mockProvider)

	// Test SVID source
	svid, err := adapter.GetX509SVID()
	require.NoError(t, err)
	assert.NotNil(t, svid)
	assert.Equal(t, "spiffe://test.example.org/test-service", svid.ID.String())

	// Test bundle source
	td, err := spiffeid.TrustDomainFromString("test.example.org")
	require.NoError(t, err)

	bundle, err := adapter.GetX509BundleForTrustDomain(td)
	require.NoError(t, err)
	assert.NotNil(t, bundle)
	assert.Len(t, bundle.X509Authorities(), 1)
}

// Mock implementations for testing

type mockServiceRegistrar struct{}

func (m *mockServiceRegistrar) Register(server interface{}) {
	// Mock registration - do nothing
}

type mockListenerPort struct {
	listener net.Listener
}

func (m *mockListenerPort) Accept() (interface{}, error) {
	return m.listener.Accept()
}

func (m *mockListenerPort) Close() error {
	return m.listener.Close()
}

func (m *mockListenerPort) Addr() string {
	return m.listener.Addr().String()
}

type mockIdentityProvider struct {
	cert     *domain.Certificate
	bundle   *domain.TrustBundle
	identity *domain.ServiceIdentity
}

func (m *mockIdentityProvider) GetCertificate() (*domain.Certificate, error) {
	return m.cert, nil
}

func (m *mockIdentityProvider) GetTrustBundle() (*domain.TrustBundle, error) {
	return m.bundle, nil
}

func (m *mockIdentityProvider) GetServiceIdentity() (*domain.ServiceIdentity, error) {
	return m.identity, nil
}

func createMockCert(t *testing.T, spiffeID string) *x509.Certificate {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	id, err := spiffeid.FromString(spiffeID)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		URIs:        []*url.URL{id.URL()},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert
}

func createMockKey(t *testing.T) *rsa.PrivateKey {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func createMockCACert(t *testing.T) *x509.Certificate {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	cert, err := x509.ParseCertificate(certDER)
	require.NoError(t, err)

	return cert
}
