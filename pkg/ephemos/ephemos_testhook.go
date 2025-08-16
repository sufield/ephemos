//go:build test

package ephemos

func TestOnlyNewClientConnection(cc clientConn) *ClientConnection {
	return &ClientConnection{conn: cc}
}
