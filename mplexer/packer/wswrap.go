package packer

import (
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/gorilla/websocket"
)

// WSWrap wraps a websocket.Conn instance to behave like net.Conn.
// TODO make conform.
type WSWrap struct {
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
	Close() error
	RemoteAddr() net.Addr
	// LocalAddr() net.Addr
}

// NewWSWrap allocates a new websocket connection wrapper
func NewWSWrap(wsconn WSConn) *WSWrap {
	return &WSWrap{
		wsconn: wsconn,
		tmpr:   nil,
	}
}

func (wsw *WSWrap) Read(b []byte) (int, error) {
	if nil == wsw.tmpr {
		_, msgr, err := wsw.wsconn.NextReader()
		if nil != err {
			fmt.Println("debug wsw NextReader err:", err)
			return 0, err
		}
		wsw.tmpr = msgr
	}

	n, err := wsw.tmpr.Read(b)
	if nil != err {
		fmt.Println("debug wsw Read err:", err)
		if io.EOF == err {
			wsw.tmpr = nil
			// ignore the message EOF because it's not the websocket EOF
			err = nil
		}
	}
	return n, err
}

func (wsw *WSWrap) Write(b []byte) (int, error) {
	// TODO create or reset ping deadline
	// TODO document that more complete writes are preferred?

	msgw, err := wsw.wsconn.NextWriter(websocket.BinaryMessage)
	if nil != err {
		fmt.Println("debug wsw NextWriter err:", err)
		return 0, err
	}
	n, err := msgw.Write(b)
	if nil != err {
		fmt.Println("debug wsw Write err:", err)
		return n, err
	}

	// if the message error fails, we can assume the websocket is damaged
	return n, msgw.Close()
}

// Close will close the websocket with a control message
func (wsw *WSWrap) Close() error {
	fmt.Println("[debug] closing the websocket.Conn")

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

// LocalAddr returns the local network address.
func (wsw *WSWrap) LocalAddr() *Addr {
	panic("not implemented")
}

// RemoteAddr returns the remote network address.
func (wsw *WSWrap) RemoteAddr() *Addr {
	panic("not implemented")
}

// SetDeadline sets the read and write deadlines associated
func (wsw *WSWrap) SetDeadline(t time.Time) error {
	panic("not implemented")
}

// SetReadDeadline sets the deadline for future Read calls
func (wsw *WSWrap) SetReadDeadline(t time.Time) error {
	panic("not implemented")
}

// SetWriteDeadline sets the deadline for future Write calls
func (wsw *WSWrap) SetWriteDeadline(t time.Time) error {
	panic("not implemented")
}
