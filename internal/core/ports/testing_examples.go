// Package ports - Example implementations showing how to use the new abstractions in tests.
// These examples demonstrate the testing benefits of proper port abstractions.
package ports

import (
	"context"
	"io"
	"strings"
)

// Example HTTP Client Mock - Simple and Clean
type MockHTTPClient struct {
	responses map[string]*HTTPResponse
	closed    bool
}

func NewMockHTTPClient() *MockHTTPClient {
	return &MockHTTPClient{
		responses: make(map[string]*HTTPResponse),
	}
}

func (m *MockHTTPClient) SetResponse(url string, response *HTTPResponse) {
	m.responses[url] = response
}

func (m *MockHTTPClient) Do(ctx context.Context, req *HTTPRequest) (*HTTPResponse, error) {
	if m.closed {
		return nil, io.ErrClosedPipe
	}
	
	if resp, exists := m.responses[req.URL]; exists {
		return resp, nil
	}
	
	// Default response
	return &HTTPResponse{
		StatusCode: 200,
		Headers:    map[string][]string{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(`{"status": "ok"}`)),
	}, nil
}

func (m *MockHTTPClient) Close() error {
	m.closed = true
	return nil
}

// Example Network Listener Mock - In-Memory Implementation
type MockNetworkListener struct {
	connections chan io.ReadWriteCloser
	address     string
	closed      bool
}

func NewMockNetworkListener(address string) *MockNetworkListener {
	return &MockNetworkListener{
		connections: make(chan io.ReadWriteCloser, 10),
		address:     address,
	}
}

func (m *MockNetworkListener) Accept() (io.ReadWriteCloser, error) {
	if m.closed {
		return nil, io.ErrClosedPipe
	}
	
	select {
	case conn := <-m.connections:
		return conn, nil
	default:
		// Return a mock connection for testing
		return NewMockReadWriteCloser(), nil
	}
}

func (m *MockNetworkListener) Addr() string {
	return m.address
}

func (m *MockNetworkListener) Close() error {
	m.closed = true
	close(m.connections)
	return nil
}

func (m *MockNetworkListener) AddConnection(conn io.ReadWriteCloser) {
	if !m.closed {
		m.connections <- conn
	}
}

// Example ReadWriteCloser Mock - For Connection Testing
type MockReadWriteCloser struct {
	data   []byte
	pos    int
	closed bool
}

func NewMockReadWriteCloser() *MockReadWriteCloser {
	return &MockReadWriteCloser{
		data: make([]byte, 1024),
	}
}

func (m *MockReadWriteCloser) Read(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func (m *MockReadWriteCloser) Write(p []byte) (n int, err error) {
	if m.closed {
		return 0, io.ErrClosedPipe
	}
	
	if m.pos+len(p) > len(m.data) {
		// Expand buffer if needed
		newData := make([]byte, m.pos+len(p))
		copy(newData, m.data)
		m.data = newData
	}
	
	n = copy(m.data[m.pos:], p)
	m.pos += n
	return n, nil
}

func (m *MockReadWriteCloser) Close() error {
	m.closed = true
	return nil
}

func (m *MockReadWriteCloser) GetData() []byte {
	return m.data[:m.pos]
}

// Example usage in tests:
//
// func TestHTTPClientIntegration(t *testing.T) {
//     // Create mock HTTP client
//     mockClient := NewMockHTTPClient()
//     mockClient.SetResponse("https://api.example.com/health", &HTTPResponse{
//         StatusCode: 200,
//         Body:       io.NopCloser(strings.NewReader(`{"healthy": true}`)),
//     })
//     
//     // Test your service that uses HTTPClient interface
//     service := NewMyService(mockClient)
//     result, err := service.CheckHealth()
//     
//     assert.NoError(t, err)
//     assert.True(t, result.Healthy)
// }
//
// func TestNetworkListener(t *testing.T) {
//     // Create mock network listener
//     listener := NewMockNetworkListener("localhost:8080")
//     
//     // Add a mock connection
//     conn := NewMockReadWriteCloser()
//     listener.AddConnection(conn)
//     
//     // Test your server that uses NetworkListener interface
//     server := NewMyServer(listener)
//     go server.Start()
//     
//     // Verify connection handling
//     acceptedConn, err := listener.Accept()
//     assert.NoError(t, err)
//     assert.NotNil(t, acceptedConn)
// }