package packer

import (
	"context"
	"fmt"
	"net"
	"strconv"
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
	body := []byte(`Hello, World!`)
	fmt.Println("payload len", len(body))
	header := []byte("IPv4,192.168.1.101,6743," + strconv.Itoa(len(body)) + ",http,80,ex1.telebit.io,\n")
	fmt.Println("header len", len(header))
	raw := []byte{255 - 1, byte(len(header))}
	raw = append(raw, header...)
	raw = append(raw, body...)
	fmt.Println("total len", len(raw))
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
	if len(body) != th.bytesRead {
		t.Fatalf("should have parsed a body of %d bytes, but saw %d\n", len(body), th.bytesRead)
	}
	if n != len(raw) {
		t.Fatalf("should have parsed all %d bytes, not just %d\n", n, len(raw))
	}
}
