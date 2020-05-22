package telebit

import (
	"io"
	"net"
	"time"
)

// WSTestConn is a fake websocket connection
type WSTestConn struct {
	closed bool
	rwt    *RWTest
}

func (wst *WSTestConn) NextReader() (messageType int, r io.Reader, err error) {
	return 0, nil, ErrNoImpl
}
func (wst *WSTestConn) NextWriter(messageType int) (io.WriteCloser, error) {
	return nil, ErrNoImpl
}
func (wst *WSTestConn) WriteControl(messageType int, data []byte, deadline time.Time) error {
	if wst.closed {
		return io.EOF
	}
	return nil
}
func (wst *WSTestConn) WriteMessage(messageType int, data []byte) error {
	if wst.closed {
		return io.EOF
	}
	return nil
}
func (wst *WSTestConn) SetReadDeadline(t time.Time) error {
	return ErrNoImpl
}
func (wst *WSTestConn) Close() error {
	wst.closed = true
	return nil
}
func (wst *WSTestConn) RemoteAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:8443")
	return addr
}
