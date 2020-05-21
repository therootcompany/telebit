package packer

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

func TestDialServer(t *testing.T) {
	// TODO replace the websocket connection with a mock server

	//ctx := context.Background()
	wsconn := &WSTestConn{
		rwt: &RWTest{},
	}

	mux := NewRouteMux()
	t.Fatal(ListenAndServe(wsconn, mux))
}

var ErrNoImpl error = errors.New("not implemented")

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

// RWTest is a fake buffer
type RWTest struct {
	closed bool
	tmpr   []byte
}

func (rwt *RWTest) Read(dst []byte) (int, error) {
	if rwt.closed {
		return 0, io.EOF
	}

	id := Addr{
		scheme: "http",
		addr:   "192.168.1.108",
		port:   6732,
	}
	tun := Addr{
		scheme:      "http",
		termination: TLS,
		addr:        "abc.example.com",
		port:        443,
	}

	if 0 == len(rwt.tmpr) {
		b := []byte("Hello, World!")
		h, _, _ := Encode(b, id, tun)
		rwt.tmpr = append(h, b...)
	}

	n := copy(dst, rwt.tmpr)
	rwt.tmpr = rwt.tmpr[n:]

	return n, nil
}

func (rwt *RWTest) Write(int, []byte) error {
	if rwt.closed {
		return io.EOF
	}
	return nil
}

func (rwt *RWTest) Close() error {
	rwt.closed = true
	return nil
}
