package packer

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

type testHandler struct {
	conns        map[string]*Conn
	chunksParsed int
	bytesRead    int
}

func (th *testHandler) WriteMessage(a Addr, b []byte) {
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
	}
	th.chunksParsed += 1
	th.bytesRead += len(b)
}

func TestParseWholeBlock(t *testing.T) {
	ctx := context.Background()
	//ctx, cancel := context.WithCancel(ctx)

	th := &testHandler{
		conns: map[string]*Conn{},
	}

	p := NewParser(ctx, th)
	payload := []byte(`Hello, World!`)
	fmt.Println("payload len", len(payload))
	src := Addr{
		family: "IPv4",
		addr:   "192.168.1.101",
		port:   6743,
	}
	dst := Addr{
		family: "IPv4",
		port:   80,
		scheme: "http",
	}
	domain := "ex1.telebit.io"
	h, b, err := Encode(src, dst, domain, payload)
	if nil != err {
		t.Fatal(err)
	}
	raw := append(h, b...)
	n, err := p.Write(raw)
	if nil != err {
		t.Fatal(err)
	}

	if 1 != len(th.conns) {
		t.Fatal("should have parsed one connection")
	}
	if 1 != th.chunksParsed {
		t.Fatal("should have parsed one chunck")
	}
	if len(payload) != th.bytesRead {
		t.Fatalf("should have parsed a payload of %d bytes, but saw %d\n", len(payload), th.bytesRead)
	}
	if n != len(raw) {
		t.Fatalf("should have parsed all %d bytes, not just %d\n", n, len(raw))
	}
}
