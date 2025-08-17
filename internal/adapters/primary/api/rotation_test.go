package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
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
)

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

// FakeRotatableSource implements both x509svid.Source and x509bundle.Source
// with the ability to rotate certificates dynamically
type FakeRotatableSource struct {
	mu            sync.RWMutex
	currentSVID   atomic.Value // *x509svid.SVID
	currentBundle atomic.Value // *x509bundle.Bundle
	rotateCount   int
}

// NewFakeRotatableSource creates a new fake source with initial SVID and bundle
func NewFakeRotatableSource(t *testing.T, spiffeID string) *FakeRotatableSource {
	source := &FakeRotatableSource{}

	// Initialize test CA
	initTestCA(t)

	// Create initial SVID and bundle
	svid, bundle := createTestSVIDAndBundle(t, spiffeID, 1)
	source.currentSVID.Store(svid)
	source.currentBundle.Store(bundle)

	return source
}

// GetX509SVID implements x509svid.Source
func (f *FakeRotatableSource) GetX509SVID() (*x509svid.SVID, error) {
	svid := f.currentSVID.Load()
	if svid == nil {
		return nil, fmt.Errorf("no SVID available")
	}
	return svid.(*x509svid.SVID), nil
}

// GetX509BundleForTrustDomain implements x509bundle.Source
func (f *FakeRotatableSource) GetX509BundleForTrustDomain(td spiffeid.TrustDomain) (*x509bundle.Bundle, error) {
	bundle := f.currentBundle.Load()
	if bundle == nil {
		return nil, fmt.Errorf("no bundle available")
	}
	return bundle.(*x509bundle.Bundle), nil
}

// Rotate simulates SVID rotation by creating a new certificate with different serial
func (f *FakeRotatableSource) Rotate(t *testing.T, spiffeID string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.rotateCount++
	// Create new SVID with different serial number
	svid, bundle := createTestSVIDAndBundle(t, spiffeID, f.rotateCount+1)

	// Atomically update the current SVID and bundle
	f.currentSVID.Store(svid)
	f.currentBundle.Store(bundle)
}

// GetRotateCount returns how many times rotation has occurred
func (f *FakeRotatableSource) GetRotateCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.rotateCount
}

// createTestSVIDAndBundle creates a test SVID and trust bundle
func createTestSVIDAndBundle(t *testing.T, spiffeID string, serial int) (*x509svid.SVID, *x509bundle.Bundle) {
	// Use the shared test CA
	caPrivateKey, caCert := initTestCA(t)

	// Parse SPIFFE ID
	id, err := spiffeid.FromString(spiffeID)
	require.NoError(t, err)

	// Generate leaf key pair
	leafPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create leaf certificate template with SPIFFE ID in URI SAN
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

	// Sign the leaf certificate with the CA
	leafCertDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafPrivateKey.PublicKey, caPrivateKey)
	require.NoError(t, err)

	leafCert, err := x509.ParseCertificate(leafCertDER)
	require.NoError(t, err)

	// Create SVID with both leaf and CA cert in the chain
	svid := &x509svid.SVID{
		ID:           id,
		Certificates: []*x509.Certificate{leafCert, caCert},
		PrivateKey:   leafPrivateKey,
	}

	// Create trust bundle with the CA cert
	bundle := x509bundle.New(id.TrustDomain())
	bundle.AddX509Authority(caCert)

	return svid, bundle
}

// TestSVIDRotationCapability verifies that new TLS handshakes pick up rotated SVIDs
func TestSVIDRotationCapability(t *testing.T) {
	// Create fake rotatable sources for client and server
	clientSource := NewFakeRotatableSource(t, "spiffe://test.example.org/client")
	serverSource := NewFakeRotatableSource(t, "spiffe://test.example.org/server")

	// Create TLS configs using go-spiffe tlsconfig (the pattern ephemos should use)
	serverTLSConfig := tlsconfig.MTLSServerConfig(
		serverSource,
		serverSource,
		tlsconfig.AuthorizeAny(),
	)

	clientTLSConfig := tlsconfig.MTLSClientConfig(
		clientSource,
		clientSource,
		tlsconfig.AuthorizeAny(),
	)
	// For test purposes, we need to skip hostname verification since we're using IP addresses
	// In production, SPIFFE uses URIs for identity, not hostnames
	clientTLSConfig.InsecureSkipVerify = true

	// Start TLS server
	listener, err := tls.Listen("tcp", "127.0.0.1:0", serverTLSConfig)
	require.NoError(t, err)
	defer listener.Close()

	// Server handler that echoes back the client's SPIFFE ID
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			tlsConn := conn.(*tls.Conn)
			if err := tlsConn.Handshake(); err != nil {
				conn.Close()
				continue
			}

			// Extract client's SPIFFE ID from peer certificate
			state := tlsConn.ConnectionState()
			if len(state.PeerCertificates) > 0 {
				cert := state.PeerCertificates[0]
				// Send back the certificate serial number
				fmt.Fprintf(conn, "Serial:%d", cert.SerialNumber)
			}
			conn.Close()
		}
	}()

	// Helper to dial and get the serial number
	getSerial := func() (int64, error) {
		conn, err := tls.Dial("tcp", listener.Addr().String(), clientTLSConfig)
		if err != nil {
			return 0, err
		}
		defer conn.Close()

		// Read the serial number from server
		buf := make([]byte, 100)
		n, err := conn.Read(buf)
		if err != nil {
			return 0, err
		}

		var serial int64
		_, err = fmt.Sscanf(string(buf[:n]), "Serial:%d", &serial)
		return serial, err
	}

	// Test 1: Initial handshake before rotation
	serial1, err := getSerial()
	require.NoError(t, err)
	assert.Equal(t, int64(1), serial1, "Initial serial should be 1")

	// Test 2: Rotate the client certificate
	clientSource.Rotate(t, "spiffe://test.example.org/client")

	// Test 3: New handshake should see the rotated certificate
	serial2, err := getSerial()
	require.NoError(t, err)
	assert.Equal(t, int64(2), serial2, "Serial after rotation should be 2")

	// Test 4: Rotate again to ensure multiple rotations work
	clientSource.Rotate(t, "spiffe://test.example.org/client")

	serial3, err := getSerial()
	require.NoError(t, err)
	assert.Equal(t, int64(3), serial3, "Serial after second rotation should be 3")

	// Verify rotation count
	assert.Equal(t, 2, clientSource.GetRotateCount(), "Should have rotated twice")
}

// TestLongLivedSourcePattern verifies that sources should be long-lived, not per-request
func TestLongLivedSourcePattern(t *testing.T) {
	// This test demonstrates the correct pattern: create source once, use many times
	source := NewFakeRotatableSource(t, "spiffe://test.example.org/service")

	// Simulate multiple requests using the same source
	var serials []int64
	for i := 0; i < 5; i++ {
		svid, err := source.GetX509SVID()
		require.NoError(t, err)
		serials = append(serials, svid.Certificates[0].SerialNumber.Int64())
	}

	// All requests should see the same certificate (serial 1)
	for i, serial := range serials {
		assert.Equal(t, int64(1), serial, "Request %d should see serial 1", i)
	}

	// Now rotate
	source.Rotate(t, "spiffe://test.example.org/service")

	// New requests should see the new certificate
	serials = nil
	for i := 0; i < 5; i++ {
		svid, err := source.GetX509SVID()
		require.NoError(t, err)
		serials = append(serials, svid.Certificates[0].SerialNumber.Int64())
	}

	// All new requests should see the rotated certificate (serial 2)
	for i, serial := range serials {
		assert.Equal(t, int64(2), serial, "Request %d after rotation should see serial 2", i)
	}
}

// TestVerifyRotationWithHTTPTransport verifies rotation works with HTTP transport
func TestVerifyRotationWithHTTPTransport(t *testing.T) {
	// Create rotatable source
	source := NewFakeRotatableSource(t, "spiffe://test.example.org/service")

	// Create TLS config from source (mimics what ephemos should do)
	tlsConfig := tlsconfig.MTLSClientConfig(
		source,
		source,
		tlsconfig.AuthorizeAny(),
	)

	// Create server with TLS
	serverTLSConfig := tlsconfig.MTLSServerConfig(
		source,
		source,
		tlsconfig.AuthorizeAny(),
	)

	listener, err := tls.Listen("tcp", "127.0.0.1:0", serverTLSConfig)
	require.NoError(t, err)
	defer listener.Close()

	// Simple HTTPS server that returns certificate info
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleHTTPSConnection(conn)
		}
	}()

	// Helper to make HTTPS request and get serial
	makeRequest := func() (int64, error) {
		conn, err := tls.Dial("tcp", listener.Addr().String(), tlsConfig)
		if err != nil {
			return 0, err
		}
		defer conn.Close()

		// Send minimal HTTP request
		_, err = conn.Write([]byte("GET / HTTP/1.1\r\nHost: test\r\n\r\n"))
		if err != nil {
			return 0, err
		}

		// Read response
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			return 0, err
		}

		// Parse serial from response
		var serial int64
		_, err = fmt.Sscanf(string(buf[:n]), "HTTP/1.1 200 OK\r\n\r\nSerial:%d", &serial)
		return serial, err
	}

	// Test before rotation
	serial1, err := makeRequest()
	require.NoError(t, err)
	assert.Equal(t, int64(1), serial1)

	// Rotate
	source.Rotate(t, "spiffe://test.example.org/service")

	// Test after rotation - new connection should see new certificate
	serial2, err := makeRequest()
	require.NoError(t, err)
	assert.Equal(t, int64(2), serial2)
}

func handleHTTPSConnection(conn net.Conn) {
	defer conn.Close()

	tlsConn := conn.(*tls.Conn)
	if err := tlsConn.Handshake(); err != nil {
		return
	}

	// Read request (minimal parsing)
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		return
	}

	// Get client certificate serial
	state := tlsConn.ConnectionState()
	var serial int64 = 0
	if len(state.PeerCertificates) > 0 {
		serial = state.PeerCertificates[0].SerialNumber.Int64()
	}

	// Send response with serial
	response := fmt.Sprintf("HTTP/1.1 200 OK\r\n\r\nSerial:%d", serial)
	conn.Write([]byte(response))
}
