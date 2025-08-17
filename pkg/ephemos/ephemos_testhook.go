package ephemos

// TestOnlyNewClientConnection creates a ClientConnection for testing purposes only.
// This should not be used in production code.
func TestOnlyNewClientConnection(cc clientConn) ClientConnection {
	return &clientConnectionImpl{conn: cc}
}
