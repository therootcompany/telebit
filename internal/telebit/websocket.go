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

	"git.rootprojects.org/root/telebit/internal/dbg"

	"github.com/gorilla/websocket"
)

var defaultReadWait = 20 * time.Second
var defaultWriteWait = 20 * time.Second

// WebsocketTunnel wraps a websocket.Conn instance to behave like net.Conn.
type WebsocketTunnel struct {
	wsconn    WSConn
	readWait  time.Duration
	writeWait time.Duration
	tmpr      io.Reader
	//w      io.WriteCloser
	//pingCh chan struct{}
}

// WSConn defines a interface for gorilla websockets for the purpose of testing
type WSConn interface {
	NextReader() (messageType int, r io.Reader, err error)
	NextWriter(messageType int) (io.WriteCloser, error)
	WriteControl(messageType int, data []byte, deadline time.Time) error
	WriteMessage(messageType int, data []byte) error
	SetPongHandler(h func(appData string) error)
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
	Close() error
	RemoteAddr() net.Addr
	// LocalAddr() net.Addr
}

// NewWebsocketTunnel allocates a new websocket connection wrapper
func NewWebsocketTunnel(wsconn WSConn) net.Conn {
	// TODO only set ping when SetReadDeadline would otherwise fail
	// See https://github.com/gorilla/websocket/blob/a6870891/examples/chat/conn.go#L86
	writeWait := defaultWriteWait
	readWait := defaultReadWait
	go func() {
		// Ping every 15 seconds, or stop listening
		for {
			time.Sleep(15 * time.Second)
			deadline := time.Now().Add(writeWait)
			// https://www.gorillatoolkit.org/pkg/websocket
			// "The Close and WriteControl methods can be called concurrently with all other methods."
			if dbg.Debug {
				fmt.Fprintf(os.Stderr, "[debug] [wstun] sending ping (set write deadline %s)\n", writeWait)
			}
			if err := wsconn.WriteControl(websocket.PingMessage, []byte(""), deadline); nil != err {
				wsconn.Close()
				fmt.Fprintf(os.Stderr, "failed to write ping message to websocket: %s\n", err)
				break
			}
			if dbg.Debug {
				fmt.Fprintf(os.Stderr, "[debug] [wstun] sent ping (cleared write deadline)\n")
			}
		}
	}()

	wsconn.SetPongHandler(func(pong string) error {
		if dbg.Debug {
			fmt.Fprintf(os.Stderr, "[debug] [wstun] received pong (reset read deadline %s): %q\n", readWait, pong)
		}
		wsconn.SetReadDeadline(time.Now().Add(readWait))
		return nil
	})

	return &WebsocketTunnel{
		wsconn:    wsconn,
		readWait:  readWait,
		writeWait: writeWait,
		tmpr:      nil,
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
		if dbg.Debug {
			fmt.Fprintf(os.Stderr, "[debug] [wstun] simple dial failed %q %v %v\n", err, wsconn, ctx)
		}
		return nil, err
	}
	return NewWebsocketTunnel(wsconn), err
}

func (wsw *WebsocketTunnel) Read(b []byte) (int, error) {
	wsw.wsconn.SetReadDeadline(time.Now().Add(wsw.readWait))
	if nil == wsw.tmpr {
		_, msgr, err := wsw.wsconn.NextReader()
		if nil != err {
			if dbg.Debug {
				fmt.Fprintf(os.Stderr, "[debug] [wstun] NextReader err: %q\n", err)
			}
			return 0, err
		}
		wsw.tmpr = msgr
	}

	n, err := wsw.tmpr.Read(b)
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] [wstun] Read %d %v\n", n, dbg.Trunc(b, n))
	}
	if nil != err {
		if dbg.Debug {
			fmt.Fprintf(os.Stderr, "[debug] [wstun] Read (EOF=WS packet complete) err: %q\n", err)
		}
		if io.EOF == err {
			wsw.tmpr = nil
			// ignore the message EOF because it's not the websocket EOF
			err = nil
		}
	}
	return n, err
}

func (wsw *WebsocketTunnel) Write(b []byte) (int, error) {
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] [wstun] Write %d\n", len(b))
	}
	// TODO create or reset ping deadline
	// TODO document that more complete writes are preferred?

	wsw.wsconn.SetWriteDeadline(time.Now().Add(wsw.writeWait))
	msgw, err := wsw.wsconn.NextWriter(websocket.BinaryMessage)
	if nil != err {
		if dbg.Debug {
			fmt.Fprintf(os.Stderr, "[debug] [wstun] NextWriter err: %q\n", err)
		}
		return 0, err
	}
	n, err := msgw.Write(b)
	if nil != err {
		if dbg.Debug {
			fmt.Fprintf(os.Stderr, "[debug] [wstun] Write err: %q\n", err)
		}
		return n, err
	}
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] [wstun] Write n %d = %d\n", n, len(b))
	}

	// if the message error fails, we can assume the websocket is damaged
	return n, msgw.Close()
}

// Close will close the websocket with a control message
func (wsw *WebsocketTunnel) Close() error {
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] [wstun] closing the websocket.Conn\n")
	}

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
	fmt.Fprintf(os.Stderr, "no LocalAddr() implementation\n")
	return nil
}

// RemoteAddr is not implemented and will panic. Additionally, it wouldn't mean anything useful anyway.
func (wsw *WebsocketTunnel) RemoteAddr() net.Addr {
	// TODO do we reverse this since the "remote" address means nothing / is that of one of the clients?
	// return wsw.wsconn.LocalAddr()
	fmt.Fprintf(os.Stderr, "no RemoteAddr() implementation\n")
	return nil
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
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] [wstun] read deadline\n")
	}
	return wsw.wsconn.SetReadDeadline(t)
}

// SetWriteDeadline sets the deadline for future Write calls
func (wsw *WebsocketTunnel) SetWriteDeadline(t time.Time) error {
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] [wstun] write deadline\n")
	}
	return wsw.wsconn.SetWriteDeadline(t)
}
