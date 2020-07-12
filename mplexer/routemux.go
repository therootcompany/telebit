package telebit

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// A RouteMux is a net.Conn multiplexer.
//
// It matches the port, domain, or connection type of a connection
// and selects the matching handler.
type RouteMux struct {
	defaultTimeout time.Duration
	routes         []meta
}

var ErrNotHandled = errors.New("connection not handled")

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
	var wconn *ConnWrap
	switch conn := client.(type) {
	case *ConnWrap:
		wconn = conn
	default:
		wconn = &ConnWrap{Conn: client}
	}

	var servername string
	var port string
	// TODO go back to Servername on conn, but with SNI
	//servername := wconn.Servername()
	fam := wconn.LocalAddr().Network()
	if "tun" == fam {
		switch laddr := wconn.LocalAddr().(type) {
		case *Addr:
			servername = laddr.Hostname()
			port = ":" + strconv.Itoa(laddr.Port())
		default:
			panic("impossible type switch: Addr is 'tun' but didn't match")
		}
	} else {
		// TODO make an AddrWrap to do this switch
		addr := wconn.LocalAddr().String()
		parts := strings.Split(addr, ":")
		port = ":" + parts[len(parts)-1]
		servername = strings.Join(parts[:len(parts)-1], ":")
	}
	fmt.Println("Addr:", fam, servername, port)

	for _, meta := range m.routes {
		// TODO '*.example.com'
		if meta.terminate {
			servername = wconn.Servername()
		}
		fmt.Println("Meta:", meta.addr, servername)
		if servername == meta.addr || "*" == meta.addr || port == meta.addr {
			//fmt.Println("[debug] test of route:", meta)
			// Only keep trying handlers if ErrNotHandled was returned
			if err := meta.handler.Serve(wconn); ErrNotHandled != err {
				return err
			}
		}
	}

	fmt.Println("No match found for", wconn.Scheme(), wconn.Servername())
	return client.Close()
}

// ForwardTCP creates and returns a connection to a local handler target.
func (m *RouteMux) ForwardTCP(servername string, target string, timeout time.Duration) error {
	// TODO check servername
	m.routes = append(m.routes, meta{
		addr:      servername,
		terminate: false,
		handler:   NewForwarder(target, timeout),
	})
	return nil
}

// HandleTCP creates and returns a connection to a local handler target.
func (m *RouteMux) HandleTCP(servername string, handler Handler) error {
	// TODO check servername
	m.routes = append(m.routes, meta{
		addr:      servername,
		terminate: false,
		handler:   handler,
	})
	return nil
}

// HandleTLS creates and returns a connection to a local handler target.
func (m *RouteMux) HandleTLS(servername string, acme *ACME, handler Handler) error {
	// TODO check servername
	m.routes = append(m.routes, meta{
		addr:      servername,
		terminate: true,
		handler: HandlerFunc(func(client net.Conn) error {
			var wconn *ConnWrap
			switch conn := client.(type) {
			case *ConnWrap:
				wconn = conn
			default:
				panic("HandleTLS is special in that it must receive &ConnWrap{ Conn: conn }")
			}

			if !wconn.isEncrypted() {
				fmt.Println("[debug] conn is not encrypted")
				return ErrNotHandled
			}

			fmt.Println("[debug] terminated encrypted connection")

			//NewTerminator(acme, handler)(client)
			//return handler.Serve(client)
			return handler.Serve(TerminateTLS(wconn, acme))
		}),
	})
	return nil
}
