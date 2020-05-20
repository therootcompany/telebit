package packer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestDialServer(t *testing.T) {

	// TODO replace the websocket connection with a mock server

	relay := os.Getenv("RELAY")
	authz := os.Getenv("SECRET")

	ctx := context.Background()
	wsd := websocket.Dialer{}
	headers := http.Header{}
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", authz))
	// *http.Response
	wsconn, _, err := wsd.DialContext(ctx, relay, headers)
	if nil != err {
		t.Fatal(err)
		return
	}

	mux := &MyMux{}
	err = ListenAndServe(wsconn, mux)
	t.Fatal(err)
}

type Listener struct {
	ws       *websocket.Conn
	incoming chan *Conn
	close    chan struct{}
}

func ListenAndServe(ws *websocket.Conn, mux Mux) error {
	listener := Listen(ws)
	return Serve(listener, mux)
}

func Listen(ws *websocket.Conn) *Listener {
	listener := &Listener{
		ws:       ws,
		incoming: make(chan *Conn, 1),
		close:    make(chan struct{}),
	}

	ctx := context.TODO()
	r := &WSConn{
		c: ws,
		r: nil,
		w: nil,
	}
	decoder := NewDecoder(ctx, r)

	// Feed websocket into Decoder
	th := &testHandler2{
		conns:  map[string]*Conn{},
		connCh: listener.incoming,
	}
	go func() {
		// TODO pass error to Accept()
		err := decoder.StreamDecode(th, 0)
		fmt.Printf("the main stream is done: %q", err)
	}()

	return listener
}

type testHandler2 struct {
	conns        map[string]*Conn
	connCh       chan *Conn
	chunksParsed int
	bytesRead    int
}

func (th *testHandler2) WriteMessage(a Addr, b []byte) {
	th.chunksParsed++
	addr := &a
	_, ok := th.conns[addr.Network()]
	if !ok {
		rconn, wconn := net.Pipe()
		conn := &Conn{
			updated:         time.Now(),
			relayRemoteAddr: *addr,
			relay:           rconn,
			local:           wconn,
		}
		th.conns[addr.Network()] = conn
		th.connCh <- conn
	}
	th.bytesRead += len(b)
}

func Serve(listener *Listener, mux Mux) error {
	w := &WSConn{
		c: listener.ws,
		r: nil,
		w: nil,
	}
	ctx := context.TODO()
	encoder := NewEncoder(ctx, w)
	encoder.Start()

	for {
		client, err := listener.Accept()
		if nil != err {
			return err
		}
		lconn, err := mux.LookupTarget(client.LocalAddr())
		if nil != err {
			conn.Close()
			continue
		}

		go func() {
			// TODO handle err
			err := encoder.StreamEncode(*conn.LocalAddr(), lconn, 0)
			fmt.Printf("a stream is done: %q", err)
		}()
	}
}

func Blah() {
		go func() {
			pipe
		}


}

func (l *Listener) Accept() (*Conn, error) {
	select {
	case conn, ok := <-l.incoming:
		if ok {
			return conn, nil
		}
		return nil, io.EOF

	case <-l.close:
		l.ws.Close()
		// TODO is another error more suitable?
		return nil, http.ErrServerClosed
	}
}

type Mux interface {
	LookupTarget(*Addr) (net.Conn, error)
}

type MyMux struct {
}

// this function is very client-specific logic
func (m *MyMux) LookupTarget(paddr *Addr) (net.Conn, error) {
	//if target := LookupPort(paddr.Port()); nil != target { }
	if target := m.LookupServername(paddr.Hostname()); nil != target {
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

func (m *MyMux) LookupServername(servername string) *Addr {
	return NewAddr(
		HTTPS,
		TCP, // TCP -> termination.None? / Plain?
		"localhost",
		3000,
	)
}

type WSConn struct {
	c      *websocket.Conn
	r      io.Reader
	w      io.WriteCloser
	pingCh chan struct{}
}

func (ws *WSConn) Read(b []byte) (int, error) {
	if nil == ws.r {
		_, r, err := ws.c.NextReader()
		if nil != err {
			return 0, err
		}
		ws.r = r
	}
	n, err := ws.r.Read(b)
	if io.EOF == err {
		err = nil
	}
	return n, err
}

func (ws *WSConn) Write(b []byte) (int, error) {
	// TODO create or reset ping deadline

	w, err := ws.c.NextWriter(websocket.BinaryMessage)
	if nil != err {
		return 0, err
	}
	ws.w = w
	n, err := ws.w.Write(b)
	if nil != err {
		return n, err
	}
	err = ws.w.Close()
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
