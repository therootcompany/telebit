package telebit

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"git.rootprojects.org/root/telebit/internal/dbg"
)

// A RouteMux is a net.Conn multiplexer.
//
// It matches the port, domain, or connection type of a connection
// and selects the matching handler.
type RouteMux struct {
	defaultTimeout time.Duration
	routes         []meta
}

// ErrNotHandled is returned when the next middleware in the stack should take over
var ErrNotHandled = errors.New("connection not handled")

type meta struct {
	addr      string
	handler   Handler
	terminate bool
	comment   string
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
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "\n\n[debug] mux.Serve(client)\n")
	}

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
			connServername := wconn.CheckServername()
			if "" == connServername {
				wconn.SetServername(servername)
			} else {
				if connServername != servername {
					fmt.Fprintf(
						os.Stderr,
						"[mux] Mismatch Servername: (current) %s != (new) %s\n",
						connServername, servername,
					)
				}
				wconn.SetServername(servername)
				//panic(errors.New("Can't SetServername() over existing servername"))
			}
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
	fmt.Println("\nAddr:", fam, servername, port)

	for _, meta := range m.routes {
		// TODO '*.example.com'
		if meta.terminate {
			servername = wconn.Servername()
		}
		if servername == meta.addr || "*" == meta.addr || port == meta.addr {
			//fmt.Fprintf(os.Stderr, "[debug] test of route: %v\n", meta)
			// Only keep trying handlers if ErrNotHandled was returned
			if err := meta.handler.Serve(wconn); ErrNotHandled != err {
				fmt.Printf(
					"[mux] Match: %s\n\tmeta.addr=%s\n\tservername=%s\n",
					meta.comment, meta.addr, servername,
				)
				return err
			}
			if dbg.Debug {
				fmt.Fprintf(
					os.Stderr,
					"[debug] [mux] Failed match: %s meta.addr=%s servername=%s\n",
					meta.comment, meta.addr, servername,
				)
			}
		} else if dbg.Debug {
			fmt.Fprintf(
				os.Stderr,
				"[debug] [mux] Skip (no match): %s meta.addr=%s servername=%s\n",
				meta.comment, meta.addr, servername,
			)
		}
	}

	fmt.Printf("No match found for %q %q\n", wconn.Scheme(), wconn.Servername())
	return client.Close()

	// TODO Chi-style route handling
	/*
		routes := m.routes
		next := func() error {
			if 0 == len(routes) {
				fmt.Println("No match found for", wconn.Scheme(), wconn.Servername())
				return client.Close()
			}
			route := routes[0]
			routes := routes[1:]
			handled := false
			handler := meta.handler(func () {
				if !handled {
					handled = true
					next()
				}
			})
			return handler.Serve(client)
		}
		return next()
	*/
}

// ForwardTCP creates and returns a connection to a local handler target.
func (m *RouteMux) ForwardTCP(servername string, target string, timeout time.Duration, comment ...string) error {
	// TODO check servername
	m.routes = append(m.routes, meta{
		addr:      servername,
		terminate: false,
		handler:   NewForwarder(target, timeout),
		comment:   append(comment, "")[0],
	})
	return nil
}

func (m *RouteMux) ReverseProxyHTTPS(servername string, target string, timeout time.Duration, comment ...string) error {
	m.routes = append(m.routes, meta{
		addr:      servername,
		terminate: false,
		handler:   NewTheatricalProxier(target, timeout),
		comment:   append(comment, "")[0],
	})
	return nil
}

func (m *RouteMux) ReverseProxyHTTP(servername string, target string, timeout time.Duration, comment ...string) error {
	m.routes = append(m.routes, meta{
		addr:      servername,
		terminate: false,
		handler:   NewReverseProxier(target, timeout),
		comment:   append(comment, "")[0],
	})
	return nil
}

// HandleTCP creates and returns a connection to a local handler target.
func (m *RouteMux) HandleTCP(servername string, handler Handler, comment ...string) error {
	// TODO check servername
	m.routes = append(m.routes, meta{
		addr:      servername,
		terminate: false,
		handler:   handler,
		comment:   append(comment, "")[0],
	})
	return nil
}

// HandleTLS creates and returns a connection to a local handler target.
func (m *RouteMux) HandleTLS(servername string, acme *ACME, next Handler, comment ...string) error {
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
				if dbg.Debug {
					fmt.Fprintf(os.Stderr, "[debug] HandleTLS: conn is not encrypted\n")
				}
				// TODO handle underlying Peek() timeout error
				return ErrNotHandled
			}

			if dbg.Debug {
				fmt.Fprintf(os.Stderr, "[debug] HandleTLS: decrypted connection, recursing\n")
			}

			//NewTerminator(acme, handler)(client)
			//return handler.Serve(client)
			go func() {
				if err := next.Serve(TerminateTLS(wconn, acme)); nil != err {
					fmt.Fprintf(os.Stderr, "error Terminating TLS: %s\n", err)
				}
			}()
			// at this point, we've handled it
			// TODO could we move termination further in?
			return nil
		}),
		comment: append(comment, "")[0],
	})
	return nil
}
