package telebit

import (
	"bufio"
	"net"
	"time"
)

var encryptedSchemes = map[string]struct{}{
	"tls":   struct{}{},
	"https": struct{}{},
	"wss":   struct{}{},
	"smtps": struct{}{},
}

// Conn TODO rename to Pipe, perhaps?
type Conn struct {
	relaySourceAddr Addr
	relayTargetAddr Addr
	peeker          *bufio.Reader
	relay           net.Conn
	local           net.Conn
	//terminated      bool
	//updated         time.Time
}

// TODO conn.go -> conn/conn.go
// TODO NewConn -> New

/*
// NewConn checks to see if the connection is already terminated and such
func NewConn(conn *Conn) *Conn {
	conn.relayTargetAddr.scheme
	return conn
}
*/

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *Conn) Read(b []byte) (n int, err error) {
	if nil != c.peeker {
		return c.peeker.Read(b)
	}
	return c.relay.Read(b)
}

func (c *Conn) Peek(n int) (b []byte, err error) {
	if nil == c.peeker {
		c.peeker = bufio.NewReaderSize(c.relay, defaultPeekerSize)
	}
	return c.peeker.Peek(n)
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *Conn) Write(b []byte) (n int, err error) {
	return c.relay.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Conn) Close() error {
	return c.relay.Close()
}

/*
// Error signals an error back to the relay
func (c *Conn) Error(err error) error {
	panic(errors.New("not implemented"))
	return nil
}
*/

/*
// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	panic(errors.New("not implemented"))
	return &net.IPAddr{}
}
*/

// LocalAddr returns the local network address.
func (c *Conn) LocalAddr() net.Addr {
	//fmt.Println("[warn] LocalAddr() address source/target switch?")
	return &c.relayTargetAddr
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	//fmt.Println("[warn] RemoteAddr() address source/target switch?")
	return &c.relaySourceAddr
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
//
// Note that if a TCP connection has keep-alive turned on,
// which is the default unless overridden by Dialer.KeepAlive
// or ListenConfig.KeepAlive, then a keep-alive failure may
// also return a timeout error. On Unix systems a keep-alive
// failure on I/O can be detected using
// errors.Is(err, syscall.ETIMEDOUT).
func (c *Conn) SetDeadline(t time.Time) error {
	return c.relay.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *Conn) SetReadDeadline(t time.Time) error {
	return c.relay.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *Conn) SetWriteDeadline(t time.Time) error {
	return c.relay.SetWriteDeadline(t)
}
