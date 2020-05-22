package telebit

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/go-acme/lego/v3/challenge"
)

// Note: 64k is the TCP max, but 1460b is the 100mbit Ethernet max (1500 MTU - overhead),
// but 1Gbit Ethernet (Jumbo frame) has an 9000b MTU
// Nerds posting benchmarks on SO show that 8k seems about right,
// but even 1024b could work well.
var defaultBufferSize = 8192

// ErrBadGateway means that the target did not accept the connection
var ErrBadGateway = errors.New("EBADGATEWAY")

// A Handler routes, proxies, terminates, or responds to a net.Conn.
type Handler interface {
	Serve(net.Conn) error
}

// HandlerFunc should handle, proxy, or terminate the connection
type HandlerFunc func(net.Conn) error

// Serve calls f(conn).
func (f HandlerFunc) Serve(conn net.Conn) error {
	return f(conn)
}

// NewForwarder creates a handler that port-forwards to a target
func NewForwarder(target string, timeout time.Duration) HandlerFunc {
	return func(client net.Conn) error {
		tconn, err := net.Dial("tcp", target)
		if nil != err {
			return err
		}
		return Forward(client, tconn, timeout)
	}
}

// Forward port-forwards a relay (websocket) client to a target (local) server
func Forward(client net.Conn, target net.Conn, timeout time.Duration) error {

	// Something like ReadAhead(size) should signal
	// to read and send up to `size` bytes without waiting
	// for a response - since we can't signal 'non-read' as
	// is the normal operation of tcp... or can we?
	// And how do we distinguish idle from dropped?
	// Maybe this should have been a udp protocol???

	defer client.Close()
	defer target.Close()

	srcCh := make(chan []byte)
	dstCh := make(chan []byte)
	srcErrCh := make(chan error)
	dstErrCh := make(chan error)

	// Source (Relay) Read Channel
	go func() {
		for {
			b := make([]byte, defaultBufferSize)
			n, err := client.Read(b)
			if n > 0 {
				srcCh <- b[:n]
			}
			if nil != err {
				// TODO let client log this server-side error (unless EOF)
				// (nil here because we probably can't send the error to the relay)
				srcErrCh <- err
				break
			}
		}
	}()

	// Target (Local) Read Channel
	go func() {
		for {
			b := make([]byte, defaultBufferSize)
			n, err := target.Read(b)
			if n > 0 {
				dstCh <- b[:n]
			}
			if nil != err {
				if io.EOF == err {
					err = nil
				}
				dstErrCh <- err
				break
			}
		}
	}()

	fmt.Println("[debug] forwarding tcp connection")
	var err error = nil
	for {
		select {
		// TODO do we need a context here?
		//case <-ctx.Done():
		//		break
		case b := <-srcCh:
			client.SetDeadline(time.Now().Add(timeout))
			_, err = target.Write(b)
			if nil != err {
				fmt.Printf("write to target failed: %q\n", err.Error())
				break
			}
		case b := <-dstCh:
			target.SetDeadline(time.Now().Add(timeout))
			_, err = client.Write(b)
			if nil != err {
				fmt.Printf("write to remote failed: %q\n", err.Error())
				break
			}
		case err = <-srcErrCh:
			if nil == err {
				break
			}
			if io.EOF != err {
				fmt.Printf("read from remote client failed: %q\n", err.Error())
			} else {
				fmt.Printf("Connection closed (possibly by remote client)\n")
			}
			break
		case err = <-dstErrCh:
			if nil == err {
				break
			}
			if io.EOF != err {
				fmt.Printf("read from local target failed: %q\n", err.Error())
			} else {
				fmt.Printf("Connection closed (possibly by local target)\n")
			}
			break

		}
	}

	client.Close()
	return err
}

type ACME struct {
	Agree                  bool
	Email                  string
	Directory              string
	DNSProvider            challenge.Provider
	Storage                certmagic.Storage
	StoragePath            string
	EnableHTTPChallenge    bool
	EnableTLSALPNChallenge bool
}

var acmecert *certmagic.Config = nil

func NewTerminator(acme *ACME, handler Handler) HandlerFunc {
	return func(client net.Conn) error {
		return handler.Serve(TerminateTLS(client, acme))
	}
}

func TerminateTLS(client net.Conn, acme *ACME) net.Conn {
	var magic *certmagic.Config = nil

	if nil == acmecert {
		acme.Storage = &certmagic.FileStorage{Path: acme.StoragePath}

		if "" == acme.Directory {
			acme.Directory = certmagic.LetsEncryptProductionCA
		}

		var err error
		magic, err = newCertMagic(acme)
		if nil != err {
			fmt.Fprintf(os.Stderr, "failed to initialize certificate management (discovery url? local folder perms?): %s\n", err)
			os.Exit(1)
		}
		acmecert = magic
	}

	tlsConfig := &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return magic.GetCertificate(hello)
			/*
				if false {
					_, _ = magic.GetCertificate(hello)
				}

				// TODO
				// 1. call out to greenlock for validation
				// 2. push challenges through http channel
				// 3. receive certificates (or don't)
				certbundleT, err := tls.LoadX509KeyPair("certs/fullchain.pem", "certs/privkey.pem")
				certbundle := &certbundleT
				if err != nil {
					return nil, err
				}
				return certbundle, nil
			*/
		},
	}

	tlsconn := tls.Server(client, tlsConfig)
	return &ConnWrap{
		Conn:  tlsconn,
		Plain: client,
	}
}

func newCertMagic(acme *ACME) (*certmagic.Config, error) {
	if !acme.Agree {
		fmt.Fprintf(
			os.Stderr,
			"\n\nError: must --acme-agree to terms to use Let's Encrypt / ACME issued certificates\n\n",
		)
		os.Exit(1)
	}

	cache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(cert certmagic.Certificate) (*certmagic.Config, error) {
			// do whatever you need to do to get the right
			// configuration for this certificate; keep in
			// mind that this config value is used as a
			// template, and will be completed with any
			// defaults that are set in the Default config
			return &certmagic.Config{}, nil
		},
	})
	magic := certmagic.New(cache, certmagic.Config{
		Storage: acme.Storage,
		OnDemand: &certmagic.OnDemandConfig{
			DecisionFunc: func(name string) error {
				return nil
			},
		},
	})
	// yes, a circular reference, passing `magic` to its own Issuer
	magic.Issuer = certmagic.NewACMEManager(magic, certmagic.ACMEManager{
		DNSProvider:             acme.DNSProvider,
		CA:                      acme.Directory,
		Email:                   acme.Email,
		Agreed:                  acme.Agree,
		DisableHTTPChallenge:    !acme.EnableHTTPChallenge,
		DisableTLSALPNChallenge: !acme.EnableTLSALPNChallenge,
		// plus any other customizations you need
	})
	return magic, nil
}