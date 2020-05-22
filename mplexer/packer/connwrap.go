package packer

import (
	"fmt"
	"net"
	"time"
)

// ConnWrap is just a cheap way to DRY up some switch conn.(type) statements to handle special features of Conn
type ConnWrap struct {
	Conn  net.Conn
	Plain net.Conn
}

func (c *ConnWrap) Read(b []byte) (n int, err error) {
	return c.Conn.Read(b)
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *ConnWrap) Write(b []byte) (n int, err error) {
	return c.Conn.Write(b)
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *ConnWrap) Close() error {
	return c.Conn.Close()
}

// Scheme returns one of "https", "http", "tcp", "tls", or ""
func (c *ConnWrap) Scheme() string {
	if nil != c.Plain {
		tlsConn := &ConnWrap{Conn: c.Plain}
		return tlsConn.Scheme()
	}

	switch conn := c.Conn.(type) {
	case *ConnWrap:
		return conn.Scheme()
	case *Conn:
		return string(conn.relayTargetAddr.scheme)
	}
	return ""
}

// Servername may return Servername or Hostname as hinted by a tunnel or buffered peeking
func (c *ConnWrap) Servername() string {
	if nil != c.Plain {
		tlsConn := &ConnWrap{Conn: c.Plain}
		return tlsConn.Scheme()
	}

	switch conn := c.Conn.(type) {
	case *ConnWrap:
		return conn.Scheme()
	case *Conn:
		return string(conn.relaySourceAddr.scheme)
	}
	return ""
}

// isTerminated returns true if it is certain that the connection has been decrypted at least once
func (c *ConnWrap) isTerminated() bool {
	if nil != c.Plain {
		return true
	}

	switch conn := c.Conn.(type) {
	case *ConnWrap:
		return conn.isTerminated()
	case *Conn:
		fmt.Printf("[debug] isTerminated: %#v\n", conn.relayTargetAddr)
		_, ok := encryptedSchemes[string(conn.relayTargetAddr.scheme)]
		return !ok
	}
	return false
}

// LocalAddr returns the local network address.
func (c *ConnWrap) LocalAddr() net.Addr {
	// TODO is this the right one?
	return c.Conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (c *ConnWrap) RemoteAddr() net.Addr {
	// TODO is this the right one?
	return c.Conn.RemoteAddr()
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
func (c *ConnWrap) SetDeadline(t time.Time) error {
	return c.Conn.SetDeadline(t)
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *ConnWrap) SetReadDeadline(t time.Time) error {
	return c.Conn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
func (c *ConnWrap) SetWriteDeadline(t time.Time) error {
	return c.Conn.SetWriteDeadline(t)
}
