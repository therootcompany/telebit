package packer

import (
	"errors"
	"time"
)

// A RouteMux is a net.Conn multiplexer.
//
// It matches the port, domain, or connection type of a connection
// and selects the matching handler.
type RouteMux struct {
	defaultTimeout time.Duration
	list           []meta
}

type meta struct {
	addr    string
	handler Handler
}

// NewRouteMux allocates and returns a new RouteMux.
func NewRouteMux() *RouteMux {
	mux := &RouteMux{
		defaultTimeout: 45 * time.Second,
	}
	return mux
}

// Serve dispatches the connection to the handler whose selectors matches the attributes.
func (m *RouteMux) Serve(client *Conn) error {
	addr := client.RemoteAddr()

	for _, meta := range m.list {
		if addr.addr == meta.addr || "*" == meta.addr {
			if err := meta.handler.Serve(client); nil != err {
				return err
			}
		}
	}

	return client.Close()
}

// ForwardTCP creates and returns a connection to a local handler target.
func (m *RouteMux) ForwardTCP(servername string, target string, timeout time.Duration) error {
	// TODO check servername
	m.list = append(m.list, meta{
		addr:    servername,
		handler: NewForwarder(target, timeout),
	})
	return nil
}

// HandleTCP creates and returns a connection to a local handler target.
func (m *RouteMux) HandleTCP(servername string, handler Handler) error {
	// TODO check servername
	m.list = append(m.list, meta{
		addr:    servername,
		handler: handler,
	})
	return nil
}

// HandleTLS creates and returns a connection to a local handler target.
func (m *RouteMux) HandleTLS(servername string, serve Handler) error {
	return errors.New("not implemented")
}
