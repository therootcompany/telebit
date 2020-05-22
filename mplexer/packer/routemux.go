package packer

import (
	"fmt"
	"net"
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
	addr      string
	handler   Handler
	terminate bool
}

// NewRouteMux allocates and returns a new RouteMux.
func NewRouteMux() *RouteMux {
	mux := &RouteMux{
		defaultTimeout: 45 * time.Second,
	}
	return mux
}

// Serve dispatches the connection to the handler whose selectors matches the attributes.
func (m *RouteMux) Serve(client net.Conn) error {
	wconn := &ConnWrap{Conn: client}
	servername := wconn.Servername()

	for _, meta := range m.list {
		if servername == meta.addr || "*" == meta.addr {
			//fmt.Println("[debug] test of route:", meta)
			if err := meta.handler.Serve(client); nil != err {
				// error should be EOF if successful
				return err
			}
			// nil err means skipped
		}
	}

	fmt.Println("No match found for", wconn.Scheme(), wconn.Servername())
	return client.Close()
}

// ForwardTCP creates and returns a connection to a local handler target.
func (m *RouteMux) ForwardTCP(servername string, target string, timeout time.Duration) error {
	// TODO check servername
	m.list = append(m.list, meta{
		addr:      servername,
		terminate: false,
		handler:   NewForwarder(target, timeout),
	})
	return nil
}

// HandleTCP creates and returns a connection to a local handler target.
func (m *RouteMux) HandleTCP(servername string, handler Handler) error {
	// TODO check servername
	m.list = append(m.list, meta{
		addr:      servername,
		terminate: false,
		handler:   handler,
	})
	return nil
}

// HandleTLS creates and returns a connection to a local handler target.
func (m *RouteMux) HandleTLS(servername string, acme *ACME, handler Handler) error {
	// TODO check servername
	m.list = append(m.list, meta{
		addr:      servername,
		terminate: true,
		handler: HandlerFunc(func(client net.Conn) error {
			wrap := &ConnWrap{Conn: client}
			if wrap.isTerminated() {
				// nil to skip
				return nil
			}
			//NewTerminator(acme, handler)(client)
			//return handler.Serve(client)
			return handler.Serve(TerminateTLS(client, acme))
		}),
	})
	return nil
}
