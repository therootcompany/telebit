package telebit

import (
	"bufio"
	"encoding/hex"
	"net"
	"time"

	"git.rootprojects.org/root/telebit/internal/dbg"
	"git.rootprojects.org/root/telebit/internal/sni"
)

// ConnWrap is just a cheap way to DRY up some switch conn.(type) statements to handle special features of Conn
type ConnWrap struct {
	// TODO use io.MultiReader to unbuffer the peeker
	//Conn  net.Conn
	peeker     *bufio.Reader
	servername string
	scheme     string
	Conn       net.Conn
	Plain      net.Conn
	encrypted  *bool
}

type Peeker interface {
	Peek(n int) ([]byte, error)
}

func (c *ConnWrap) Peek(n int) ([]byte, error) {
	if nil != c.peeker {
		return c.peeker.Peek(n)
	}

	switch conn := c.Conn.(type) {
	case *ConnWrap:
		return conn.Peek(n)
	case *Conn:
		return conn.Peek(n)
	default:
		// *net.UDPConn,*net.TCPConn,*net.IPConn,*net.UnixConn
		if nil == c.peeker {
			c.peeker = bufio.NewReaderSize(c.Conn, defaultPeekerSize)
		}
		return c.peeker.Peek(n)
	}
}

func (c *ConnWrap) Read(b []byte) (n int, err error) {
	if nil != c.peeker {
		return c.peeker.Read(b)
	}
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
	if "" != c.scheme {
		return c.scheme
	}

	/*
		if nil != c.Plain {
			tlsConn := &ConnWrap{Conn: c.Plain}
			// TODO upgrade tls+http => https
			c.scheme = tlsConn.Scheme()
			return c.scheme
		}
	*/

	switch conn := c.Conn.(type) {
	case *ConnWrap:
		return conn.Scheme()
	case *Conn:
		return string(conn.relayTargetAddr.scheme)
	}
	return ""
}

// CheckServername returns the servername without detection
func (c *ConnWrap) CheckServername() string {
	return c.servername
}

// SetServername sets the servername without detection
func (c *ConnWrap) SetServername(name string) {
	c.servername = name
}

// Servername may return Servername or Hostname as hinted by a tunnel or buffered peeking
func (c *ConnWrap) Servername() string {
	if "" != c.servername {
		return c.servername
	}

	if nil != c.Plain {
		tlsConn := &ConnWrap{Conn: c.Plain}
		c.servername = tlsConn.Servername()
		return c.servername
	}

	switch conn := c.Conn.(type) {
	case *ConnWrap:
		//c.servername = conn.Servername()
		return conn.Servername()
	case *Conn:
		// TODO XXX
		//c.servername = string(conn.relayTargetAddr.addr)
		return string(conn.relayTargetAddr.addr)
	}

	// this will get the servername
	_ = c.isEncrypted()
	return c.servername
}

// isEncrypted returns true if peeking at net.Conn reveals that it is TLS-encrypted
func (c *ConnWrap) isEncrypted() bool {
	if nil != c.encrypted {
		return *c.encrypted
	}

	var encrypted bool

	// TODO: how to allow / detect / handle protocols where the server hello happens first?
	c.SetDeadline(time.Now().Add(5 * time.Second))
	n := 6
	b, err := c.Peek(n)
	defer c.SetDeadline(time.Time{})
	if dbg.Debug {
		dbg.Debugf("[wrap] Peek(%d): %q %v\n", n, hex.EncodeToString(b), err)
	}
	if nil != err {
		// TODO return error on error?
		return encrypted
	}
	if len(b) >= n {
		// SSL v3.x / TLS v1.x
		// 0: TLS Byte
		// 1: Major Version
		// 2: Minor Version - 1
		// 3-4: Header Length

		// Payload
		// 5: TLS Client Hello Marker Byte
		if 0x16 == b[0] && 0x03 == b[1] && 0x01 == b[5] {
			length := (int(b[3]) << 8) + int(b[4])
			b, err := c.Peek(n - 1 + length)
			if nil != err {
				c.encrypted = &encrypted
				return *c.encrypted
			}
			c.servername, _ = sni.GetHostname(b)
			encrypted = true
			c.encrypted = &encrypted
			return *c.encrypted
		}
	}
	c.encrypted = &encrypted
	return *c.encrypted
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
