package packer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"

	"github.com/gorilla/websocket"
)

func TestDialServer(t *testing.T) {
	// TODO replace the websocket connection with a mock server

	relay := "wss://roottest.duckdns.org:8443"
	authz, err := getToken("xxxxyyyyssss8347")
	if nil != err {
		panic(err)
	}

	ctx := context.Background()
	wsd := websocket.Dialer{}
	headers := http.Header{}
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", authz))
	// *http.Response
	sep := "?"
	if strings.Contains(relay, sep) {
		sep = "&"
	}
	wsconn, _, err := wsd.DialContext(ctx, relay+sep+"access_token="+authz, headers)
	if nil != err {
		fmt.Println("relay:", relay)
		t.Fatal(err)
		return
	}

	/*
		t := telebit.New(token)
		mux := telebit.RouteMux{}
		mux.HandleTLS("*", mux) // go back to itself
		mux.HandleProxy("example.com", "localhost:3000")
		mux.HandleTCP("example.com", func (c *telebit.Conn) {
			return httpmux.Serve()
		})

		l := t.Listen("wss://example.com")
		conn := l.Accept()
		telebit.Serve(listener, mux)
		t.ListenAndServe("wss://example.com", mux)
	*/

	mux := NewRouteMux()
	// TODO set failure
	t.Fatal(ListenAndServe(wsconn, mux))
}

func getToken(secret string) (token string, err error) {
	domains := []string{"dandel.duckdns.org"}
	tokenData := jwt.MapClaims{"domains": domains}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenData)
	if token, err = jwtToken.SignedString([]byte(secret)); err != nil {
		return "", err
	}
	return token, nil
}

type Listener struct {
	ws           *websocket.Conn
	incoming     chan *Conn
	close        chan struct{}
	encoder      *Encoder
	conns        map[string]*Conn
	chunksParsed int
	bytesRead    int
}

func ListenAndServe(ws *websocket.Conn, mux Handler) error {
	listener := Listen(ws)
	return Serve(listener, mux)
}

func Listen(ws *websocket.Conn) *Listener {
	ctx := context.TODO()

	// Wrap the websocket and feed it into the Encoder and Decoder
	rw := &WSConn{c: ws, nr: nil}
	listener := &Listener{
		ws:       ws,
		conns:    map[string]*Conn{},
		incoming: make(chan *Conn, 1), // buffer ever so slightly
		close:    make(chan struct{}),
		encoder:  NewEncoder(ctx, rw),
	}
	// TODO perhaps the wrapper should have a mutex
	// rather than having a goroutine in the encoder
	go func() {
		err := listener.encoder.Run()
		fmt.Printf("encoder stopped entirely: %q", err)
		rw.c.Close()
	}()

	// Decode the stream as it comes in
	decoder := NewDecoder(rw)
	go func() {
		// TODO pass error to Accept()
		err := decoder.Decode(listener)
		rw.Close()
		fmt.Printf("the main stream is done: %q\n", err)
	}()

	return listener
}

func (l *Listener) RouteBytes(a Addr, b []byte) {
	// TODO use context to be able to cancel many at once?
	l.chunksParsed++

	addr := &a
	pipe := l.getPipe(addr)

	// handle errors before data writes because I don't
	// remember where the error message goes
	if "error" == string(addr.scheme) {
		pipe.Close()
		delete(l.conns, addr.Network())
		fmt.Printf("a stream errored remotely: %v\n", addr)
	}

	// write data, if any
	if len(b) > 0 {
		l.bytesRead += len(b)
		pipe.Write(b)
	}
	// EOF, if needed
	if "end" == string(addr.scheme) {
		pipe.Close()
		delete(l.conns, addr.Network())
	}
}

func (l *Listener) getPipe(addr *Addr) *Conn {
	connID := addr.Network()
	pipe, ok := l.conns[connID]

	// Pipe exists
	if ok {
		return pipe
	}

	// Create pipe
	rawPipe, encodable := net.Pipe()
	pipe = &Conn{
		//updated:         time.Now(),
		relayRemoteAddr: *addr,
		relay:           rawPipe,
	}
	l.conns[connID] = pipe
	l.incoming <- pipe

	// Handle encoding
	go func() {
		// TODO handle err
		err := l.encoder.Encode(encodable, *pipe.LocalAddr())
		// the error may be EOF or ErrServerClosed or ErrGoingAwawy or some such
		// or it might be an actual error
		// In any case, we'll just close it all
		encodable.Close()
		pipe.Close()
		fmt.Printf("a stream is done: %q\n", err)
	}()

	return pipe
}

func Serve(listener *Listener, mux Handler) error {
	for {
		client, err := listener.Accept()
		if nil != err {
			return err
		}

		go func() {
			err = mux.Serve(client)
			if nil != err {
				if io.EOF != err {
					fmt.Printf("client could not be served: %q\n", err.Error())
				}
			}
			client.Close()
		}()
	}
}

func (l *Listener) Accept() (*Conn, error) {
	select {
	case rconn, ok := <-l.incoming:
		if ok {
			return rconn, nil
		}
		return nil, io.EOF

	case <-l.close:
		l.ws.Close()
		// TODO is another error more suitable?
		return nil, http.ErrServerClosed
	}
}

type Handler interface {
	Serve(*Conn) error
	GetTargetConn(*Addr) (net.Conn, error)
}

type RouteMux struct {
	defaultTimeout time.Duration
}

func NewRouteMux() *RouteMux {
	mux := &RouteMux{
		defaultTimeout: 45 * time.Second,
	}
	return mux
}

func (m *RouteMux) Serve(client *Conn) error {
	// TODO could proxy or handle directly, etc
	target, err := m.GetTargetConn(client.RemoteAddr())
	if nil != err {
		return err
	}

	return Forward(client, target, m.defaultTimeout)
}

// Forward port-forwards a relay (websocket) client to a target (local) server
func Forward(client *Conn, target net.Conn, timeout time.Duration) error {

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
				srcCh <- b
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
				dstCh <- b
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
				fmt.Printf("write to target failed: %q", err.Error())
				break
			}
		case b := <-dstCh:
			target.SetDeadline(time.Now().Add(timeout))
			_, err = client.Write(b)
			if nil != err {
				fmt.Printf("write to remote failed: %q", err.Error())
				break
			}
		case err = <-srcErrCh:
			if nil != err {
				fmt.Printf("read from remote failed: %q", err.Error())
			}
			break
		case err = <-dstErrCh:
			if nil != err {
				fmt.Printf("read from target failed: %q", err.Error())
			}
			break

		}
	}

	client.Close()
	return err
}

// this function is very client-specific logic
func (m *RouteMux) GetTargetConn(paddr *Addr) (net.Conn, error) {
	//if target := GetTargetByPort(paddr.Port()); nil != target { }
	if target := m.GetTargetByServername(paddr.Hostname()); nil != target {
		tconn, err := net.Dial(target.Network(), target.Hostname())
		if nil != err {
			return nil, err
		}
		/*
			// TODO for http proxy
			return mplexer.TargetOptions {
				Hostname // default localhost
				Termination // default TLS
				XFWD // default... no?
				Port // default 0
				Conn // should be dialed beforehand
			}, nil
		*/
		return tconn, nil
	}
	// TODO
	return nil, errors.New("Bad Gateway")
}

func (m *RouteMux) GetTargetByServername(servername string) *Addr {
	return NewAddr(
		HTTPS,
		TCP, // TCP -> termination.None? / Plain?
		"localhost",
		3000,
	)
}

type WSConn struct {
	c  *websocket.Conn
	nr io.Reader
	//w      io.WriteCloser
	//pingCh chan struct{}
}

func (ws *WSConn) Read(b []byte) (int, error) {
	if nil == ws.nr {
		_, r, err := ws.c.NextReader()
		if nil != err {
			return 0, err
		}
		ws.nr = r
	}
	n, err := ws.nr.Read(b)
	if io.EOF == err {
		err = nil
	}
	return n, err
}

func (ws *WSConn) Write(b []byte) (int, error) {
	// TODO create or reset ping deadline
	// TODO document that more complete writes are preferred?

	w, err := ws.c.NextWriter(websocket.BinaryMessage)
	if nil != err {
		return 0, err
	}
	n, err := w.Write(b)
	if nil != err {
		return n, err
	}
	err = w.Close()
	return n, err
}

func (ws *WSConn) Close() error {
	// TODO handle EOF as websocket.CloseNormal?
	message := websocket.FormatCloseMessage(websocket.CloseGoingAway, "closing connection")
	deadline := time.Now().Add(10 * time.Second)
	err := ws.c.WriteControl(websocket.CloseMessage, message, deadline)
	if nil != err {
		fmt.Fprintf(os.Stderr, "failed to write close message to websocket: %s\n", err)
	}
	_ = ws.c.Close()
	return err
}
