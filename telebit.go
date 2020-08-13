package telebit

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"git.rootprojects.org/root/telebit/dbg"
	httpshim "git.rootprojects.org/root/telebit/tunnel"

	"github.com/caddyserver/certmagic"
	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/challenge/dns01"
)

// Note: 64k is the TCP max, but 1460b is the 100mbit Ethernet max (1500 MTU - overhead),
// but 1Gbit Ethernet (Jumbo frame) has an 9000b MTU
// Nerds posting benchmarks on SO show that 8k seems about right,
// but even 1024b could work well.
var defaultBufferSize = 8192
var defaultPeekerSize = 1024
var defaultWriteTimeout = 10 * time.Second

// ErrBadGateway means that the target did not accept the connection
var ErrBadGateway = errors.New("EBADGATEWAY")

// The proper handling of this error
// is still being debated as of Jun 9, 2020
// https://github.com/golang/go/issues/4373
var errNetClosing = "use of closed network connection"

// A Handler routes, proxies, terminates, or responds to a net.Conn.
type Handler interface {
	// TODO ServeTCP
	Serve(net.Conn) error
}

// HandlerFunc should handle, proxy, or terminate the connection
type HandlerFunc func(net.Conn) error

// Authorizer is called when a new client connects and we need to know something about it
type Authorizer func(*http.Request) (*Grants, error)

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
		go Forward(client, tconn, timeout)
		return nil
	}
}

// NewTheatricalProxier exists because... reasons... but should not be used
func NewTheatricalProxier(target string, timeout time.Duration) HandlerFunc {
	return newReverseProxier(target, timeout, true)
}

func NewReverseProxier(target string, timeout time.Duration) HandlerFunc {
	return newReverseProxier(target, timeout, false)
}

func newReverseProxier(target string, timeout time.Duration, theatre bool) HandlerFunc {
	// TODO accept listener?
	proxyListener := httpshim.NewListener()
	scheme := "http://"
	if theatre {
		scheme = "https://"
	}
	targetURL, err := url.Parse(scheme + target)
	if nil != err {
		panic(err)
	}
	//proxyHandler := httputil.NewSingleHostReverseProxy(targetURL)
	proxyHandler := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.Header.Del("X-Forwarded-For")
			req.Header.Del("X-Forwarded-Proto")
			req.Header.Del("X-Forwarded-Port")

			targetQuery := targetURL.RawQuery
			req.URL.Scheme = targetURL.Scheme
			req.URL.Host = targetURL.Host
			req.URL.Path, req.URL.RawPath = joinURLPath(targetURL, req.URL)
			if targetQuery == "" || req.URL.RawQuery == "" {
				req.URL.RawQuery = targetQuery + req.URL.RawQuery
			} else {
				req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
			}
			if _, ok := req.Header["User-Agent"]; !ok {
				// explicitly disable User-Agent so it's not set to default value
				req.Header.Set("User-Agent", "")
			}
		},
	}
	if theatre {
		/*
			// TODO we could take control of the SNI here
			proxyHandler.Transport = &http.Transport{
				DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					// would need timeout
					dialer := tls.Dialer{
						Config: &tls.Config{
							ServerName:         "localhost",
							InsecureSkipVerify: true,
							TLSHandshakeTimeout: 10 * time.Second,
						},
					}
					return dialer.DialContext(ctx, network, addr)
				},
			}
			//*/
		///*
		proxyHandler.Transport = &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			TLSHandshakeTimeout: 10 * time.Second,
		}
		//*/
	}
	proxyServer := &http.Server{
		Handler: proxyHandler,
	}
	go func() {
		proxyServer.Serve(proxyListener)
	}()

	return func(client net.Conn) error {
		// TODO Peek to see if this is HTTP
		proxyListener.Feed(client)
		return nil
	}
}

// Taken from https://golang.org/src/net/http/httputil/reverseproxy.go
func joinURLPath(a, b *url.URL) (path, rawpath string) {
	if a.RawPath == "" && b.RawPath == "" {
		return singleJoiningSlash(a.Path, b.Path), ""
	}
	// Same as singleJoiningSlash, but uses EscapedPath to determine
	// whether a slash should be added
	apath := a.EscapedPath()
	bpath := b.EscapedPath()

	aslash := strings.HasSuffix(apath, "/")
	bslash := strings.HasPrefix(bpath, "/")

	switch {
	case aslash && bslash:
		return a.Path + b.Path[1:], apath + bpath[1:]
	case !aslash && !bslash:
		return a.Path + "/" + b.Path, apath + "/" + bpath
	}
	return a.Path + b.Path, apath + bpath
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
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

	noDeadline := time.Time{}
	writeTimeout := defaultWriteTimeout
	if timeout < defaultWriteTimeout {
		writeTimeout = timeout
	}

	srcCh := make(chan []byte)
	dstCh := make(chan []byte)
	srcErrCh := make(chan error)
	dstErrCh := make(chan error)

	// Source (Relay) Read Channel
	go func() {
		for {
			b := make([]byte, defaultBufferSize)
			client.SetReadDeadline(time.Now().Add(timeout))
			target.SetReadDeadline(time.Now().Add(timeout))
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
			target.SetReadDeadline(time.Now().Add(timeout))
			client.SetReadDeadline(time.Now().Add(timeout))
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

	fmt.Printf(
		"[mux] Forwarding TCP connection\n\t%s => %s\n\t(%s => %s)\n",
		client.RemoteAddr(),
		client.LocalAddr(),
		target.LocalAddr(),
		target.RemoteAddr(),
	)
	var err error = nil

ForwardData:
	for {
		select {
		// TODO do we need a context here?
		//case <-ctx.Done():
		//		break
		case b := <-srcCh:
			//fmt.Println("Read(): ", len(b))
			target.SetWriteDeadline(time.Now().Add(writeTimeout))
			_, err = target.Write(b)
			target.SetWriteDeadline(noDeadline)
			if nil != err {
				fmt.Printf("write to target failed: %q\n", err.Error())
				break ForwardData
			}
		case b := <-dstCh:
			//fmt.Println("Write(): ", len(b))
			client.SetWriteDeadline(time.Now().Add(writeTimeout))
			_, err = client.Write(b)
			client.SetWriteDeadline(noDeadline)
			if nil != err {
				fmt.Printf("write to remote failed: %q\n", err.Error())
				break ForwardData
			}
		case err = <-srcErrCh:
			if nil == err {
				break ForwardData
			}
			if io.EOF != err && io.ErrClosedPipe != err && !strings.Contains(err.Error(), errNetClosing) {
				fmt.Printf("error: data source (websocket client) read failed: %q\n", err.Error())
			} else {
				fmt.Printf("Connection closed (possibly by remote client)\n")
			}
			break ForwardData
		case err = <-dstErrCh:
			if nil == err {
				break ForwardData
			}
			if io.EOF != err && io.ErrClosedPipe != err && !strings.Contains(err.Error(), errNetClosing) {
				fmt.Printf("error: data sink (local target) read failed: %q\n", err.Error())
			} else {
				fmt.Printf("Connection closed (possibly by local target)\n")
			}
			break ForwardData

		}
	}

	return err
}

type ACME struct {
	Agree                  bool
	Email                  string
	Directory              string
	DNSProvider            challenge.Provider
	DNSChallengeOption     dns01.ChallengeOption
	Storage                certmagic.Storage
	StoragePath            string
	EnableHTTPChallenge    bool
	EnableTLSALPNChallenge bool
}

var acmecert *certmagic.Config = nil

/*
func NewTerminator(servername string, acme *ACME, handler Handler) HandlerFunc {
	return func(client net.Conn) error {
		return handler.Serve(TerminateTLS("", client, acme))
	}
}
*/

//func TerminateTLS(client *ConnWrap, acme *ACME) net.Conn

func TerminateTLS(client net.Conn, acme *ACME) net.Conn {
	var magic *certmagic.Config = nil

	if nil == acmecert {
		acme.Storage = &certmagic.FileStorage{Path: acme.StoragePath}

		if "" == acme.Directory {
			acme.Directory = certmagic.LetsEncryptProductionCA
		}

		var err error
		magic, err = NewCertMagic(acme)
		if nil != err {
			fmt.Fprintf(
				os.Stderr,
				"failed to initialize certificate management (discovery url? local folder perms?): %s\n",
				err,
			)
			os.Exit(1)
		}
		acmecert = magic
	}

	// TODO NextProtos: []string{ "h2", "http/1.1" }
	tlsConfig := &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return acmecert.GetCertificate(hello)
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

	var servername string
	var scheme string
	// I think this must always be ConnWrap, but I'm not sure
	switch conn := client.(type) {
	case *ConnWrap:
		servername = conn.Servername()
		scheme = conn.Scheme()
		client = conn
	default:
		wconn := &ConnWrap{
			Conn: client,
		}
		_ = wconn.isEncrypted()
		servername = wconn.Servername()
		scheme = wconn.Scheme()
		client = wconn
	}

	/*
		// TODO ?
		if "" == scheme {
			scheme = "tls"
		}
		if "http" == scheme {
			scheme = "https"
		}
	*/

	tlsconn := tls.Server(client, tlsConfig)
	return &ConnWrap{
		Conn:       tlsconn,
		Plain:      client,
		servername: servername,
		scheme:     scheme,
	}
}

func NewCertMagic(acme *ACME) (*certmagic.Config, error) {
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
	fmt.Printf("ACME Email: %q\n", acme.Email)
	magic.Issuer = certmagic.NewACMEManager(magic, certmagic.ACMEManager{
		DNSProvider:             acme.DNSProvider,
		DNSChallengeOption:      acme.DNSChallengeOption,
		CA:                      acme.Directory,
		Email:                   acme.Email,
		Agreed:                  acme.Agree,
		DisableHTTPChallenge:    !acme.EnableHTTPChallenge,
		DisableTLSALPNChallenge: !acme.EnableTLSALPNChallenge,
		// plus any other customizations you need
	})
	return magic, nil
}

type Grants struct {
	Subject  string   `json:"sub"`
	Audience string   `json:"aud"`
	Domains  []string `json:"domains"`
	Ports    []int    `json:"ports"`
}

func Inspect(authURL, token string) (*Grants, error) {
	inspectURL := strings.TrimSuffix(authURL, "/inspect") + "/inspect"
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] telebit.Inspect(\n\tinspectURL = %s,\n\ttoken = %s,\n)\n", inspectURL, token)
	}
	msg, err := Request("GET", inspectURL, token, nil)
	if nil != err {
		return nil, err
	}
	if nil == msg {
		return nil, fmt.Errorf("invalid response")
	}

	grants := &Grants{}
	err = json.NewDecoder(msg).Decode(grants)
	if err != nil {
		return nil, err
	}
	if "" == grants.Subject {
		fmt.Fprintf(os.Stderr, "TODO update mgmt server to show Subject: %q\n", msg)
		grants.Subject = strings.Split(grants.Domains[0], ".")[0]
	}
	return grants, nil
}

func Request(method, fullurl, token string, payload io.Reader) (io.Reader, error) {
	HTTPClient := &http.Client{
		Timeout: 15 * time.Second,
	}
	req, err := http.NewRequest(method, fullurl, payload)
	if err != nil {
		return nil, err
	}
	if len(token) > 0 {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if nil != payload {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%d: failed to read response body: %w", resp.StatusCode, err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("%d: request failed: %v", resp.StatusCode, string(body))
	}

	return bytes.NewBuffer(body), nil
}
