package tunnel

import (
	"io"
	"net"
)

// Listener defines a listener for use with http servers
type Listener struct {
	Conns chan net.Conn
	//ParentAddr net.Addr
}

// NewListener creates a channel for connections and returns the listener
func NewListener() *Listener {
	return &Listener{
		Conns: make(chan net.Conn),
	}
}

// Feed will block while pushing a net.Conn onto Conns
func (l *Listener) Feed(conn net.Conn) {
	l.Conns <- conn
}

// net.Listener interface

// Accept will block and wait for a new net.Conn
func (l *Listener) Accept() (net.Conn, error) {
	conn, ok := <-l.Conns
	if ok {
		return conn, nil
	}
	return nil, io.EOF
}

// Close will close the Conns channel
func (l *Listener) Close() error {
	close(l.Conns)
	return nil
}

// Addr returns nil to fulfill the net.Listener interface
func (l *Listener) Addr() net.Addr {
	// Addr may (or may not) return the original TCP or TLS listener's address
	//return l.ParentAddr
	return nil
}
