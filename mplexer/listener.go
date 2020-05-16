package mplexer

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/gorilla/websocket"
)

// Listener defines a listener for use with http servers
type Listener struct {
	//ParentAddr net.Addr
	Conns chan *Conn
	ws    *websocket.Conn
}

// NewListener creates a channel for connections and returns the listener
func (m *MultiplexLocal) Listen(ctx context.Context) (*Listener, error) {
	authz, err := m.SortingHat.Authz()
	if nil != err {
		return nil, err
	}

	wsd := websocket.Dialer{}
	headers := http.Header{}
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", authz))
	// *http.Response
	wsconn, _, err := wsd.DialContext(ctx, m.Relay, headers)
	if nil != err {
		return nil, err
	}
	listener := &Listener{
		Conns: make(chan *Conn),
	}
	return listener, nil
}

// Feed will block while pushing a net.Conn onto Conns
func (l *Listener) Feed(conn *Conn) {
	l.Conns <- conn
}

// net.Listener interface

// Accept will block and wait for a new net.Conn
func (l *Listener) Accept() (*Conn, error) {
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
