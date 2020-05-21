package packer

import (
	"errors"
	"io"
	"testing"
)

func TestDialServer(t *testing.T) {
	// TODO replace the websocket connection with a mock server

	//ctx := context.Background()
	wsw := &WSWrap{}

	mux := NewRouteMux()
	t.Fatal(ListenAndServe(wsw, mux))
}

var ErrNoImpl error = errors.New("not implemented")

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
