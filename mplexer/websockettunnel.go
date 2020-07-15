package telebit

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// WebsocketTunnel wraps a websocket.Conn instance to behave like net.Conn.
// TODO make conform.
type WebsocketTunnel struct {
	wsconn WSConn
	tmpr   io.Reader
	//w      io.WriteCloser
	//pingCh chan struct{}
}

// WSConn defines a interface for gorilla websockets for the purpose of testing
type WSConn interface {
	NextReader() (messageType int, r io.Reader, err error)
	NextWriter(messageType int) (io.WriteCloser, error)
	WriteControl(messageType int, data []byte, deadline time.Time) error
	WriteMessage(messageType int, data []byte) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	Close() error
	RemoteAddr() net.Addr
	// LocalAddr() net.Addr
}

// NewWebsocketTunnel allocates a new websocket connection wrapper
func NewWebsocketTunnel(wsconn WSConn) net.Conn {
	return &WebsocketTunnel{
		wsconn: wsconn,
		tmpr:   nil,
	}
}

// DialWebsocketTunnel connects to the given websocket relay as wraps it as net.Conn
func DialWebsocketTunnel(ctx context.Context, relay, authz string) (net.Conn, error) {
	wsd := websocket.Dialer{}
	headers := http.Header{}
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", authz))
	// *http.Response
	sep := "?"
	if strings.Contains(relay, sep) {
		sep = "&"
	}
	wsconn, _, err := wsd.DialContext(ctx, relay+sep+"access_token="+authz+"&versions=v1", headers)
	if nil != err {
		fmt.Println("[debug] [wstun] simple dial failed", err, wsconn, ctx)
	}
	return NewWebsocketTunnel(wsconn), err
}

func (wsw *WebsocketTunnel) Read(b []byte) (int, error) {
	if nil == wsw.tmpr {
		_, msgr, err := wsw.wsconn.NextReader()
		if nil != err {
			fmt.Println("[debug] [wstun] NextReader err:", err)
			return 0, err
		}
		wsw.tmpr = msgr
	}

	n, err := wsw.tmpr.Read(b)
	fmt.Println("[debug] [wstun] Read", n)
	if nil != err {
		fmt.Println("[debug] [wstun] Read err:", err)
		if io.EOF == err {
			wsw.tmpr = nil
			// ignore the message EOF because it's not the websocket EOF
			err = nil
		}
	}
	return n, err
}

func (wsw *WebsocketTunnel) Write(b []byte) (int, error) {
	fmt.Println("[debug] [wstun] Write", len(b))
	// TODO create or reset ping deadline
	// TODO document that more complete writes are preferred?

	msgw, err := wsw.wsconn.NextWriter(websocket.BinaryMessage)
	if nil != err {
		fmt.Println("[debug] [wstun] NextWriter err:", err)
		return 0, err
	}
	n, err := msgw.Write(b)
	if nil != err {
		fmt.Println("[debug] [wstun] Write err:", err)
		return n, err
	}

	// if the message error fails, we can assume the websocket is damaged
	return n, msgw.Close()
}

// Close will close the websocket with a control message
func (wsw *WebsocketTunnel) Close() error {
	fmt.Println("[debug] [wstun] closing the websocket.Conn")

	// TODO handle EOF as websocket.CloseNormal?
	message := websocket.FormatCloseMessage(websocket.CloseGoingAway, "closing connection")
	deadline := time.Now().Add(10 * time.Second)
	err := wsw.wsconn.WriteControl(websocket.CloseMessage, message, deadline)
	if nil != err {
		fmt.Fprintf(os.Stderr, "failed to write close message to websocket: %s\n", err)
	}
	_ = wsw.wsconn.Close()
	return err
}

// LocalAddr is not implemented and will panic
func (wsw *WebsocketTunnel) LocalAddr() net.Addr {
	// TODO do we reverse this since the "local" address is that of the relay?
	// return wsw.wsconn.RemoteAddr()
	panic("no LocalAddr() implementation")
}

// RemoteAddr is not implemented and will panic. Additionally, it wouldn't mean anything useful anyway.
func (wsw *WebsocketTunnel) RemoteAddr() net.Addr {
	// TODO do we reverse this since the "remote" address means nothing / is that of one of the clients?
	// return wsw.wsconn.LocalAddr()
	panic("no RemoteAddr() implementation")
}

// SetDeadline sets the read and write deadlines associated
func (wsw *WebsocketTunnel) SetDeadline(t time.Time) error {
	err := wsw.SetReadDeadline(t)
	if nil == err {
		err = wsw.SetWriteDeadline(t)
	}
	return err
}

// SetReadDeadline sets the deadline for future Read calls
func (wsw *WebsocketTunnel) SetReadDeadline(t time.Time) error {
	fmt.Println("[debug] [wstun] read deadline")
	return wsw.wsconn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
func (wsw *WebsocketTunnel) SetWriteDeadline(t time.Time) error {
	fmt.Println("[debug] [wstun] write deadline")
	return wsw.wsconn.SetWriteDeadline(t)
}
