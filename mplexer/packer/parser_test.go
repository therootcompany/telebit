package packer

import (
	"context"
	"net"
	"testing"
	"time"
)

var src = Addr{
	family: "IPv4",
	addr:   "192.168.1.101",
	port:   6743,
}
var dst = Addr{
	family: "IPv4",
	port:   80,
	scheme: "http",
}
var domain = "ex1.telebit.io"
var payload = []byte("Hello, World!")

type testHandler struct {
	conns        map[string]*Conn
	chunksParsed int
	bytesRead    int
}

func (th *testHandler) WriteMessage(a Addr, b []byte) {
	th.chunksParsed += 1
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
	th.bytesRead += len(b)
}

func TestParse1WholeBlock(t *testing.T) {
	testParseNBlocks(t, 1)
}

func TestParse2WholeBlocks(t *testing.T) {
	testParseNBlocks(t, 2)
}

func TestParse3WholeBlocks(t *testing.T) {
	testParseNBlocks(t, 3)
}

func TestParse2Addrs(t *testing.T) {
	testParseNBlocks(t, 4)
}

func TestParse3Addrs(t *testing.T) {
	testParseNBlocks(t, 5)
}

func TestParse1AndRest(t *testing.T) {
	ctx := context.Background()
	//ctx, cancel := context.WithCancel(ctx)

	th := &testHandler{
		conns: map[string]*Conn{},
	}

	p := NewParser(ctx, th)

	h, b, err := Encode(src, dst, domain, payload)
	if nil != err {
		t.Fatal(err)
	}
	raw := append(h, b...)
	n, err := p.Write(raw[:1])
	if nil != err {
		t.Fatal(err)
	}
	m, err := p.Write(raw[1:])
	if nil != err {
		t.Fatal(err)
	}

	if 1 != len(th.conns) {
		t.Fatal("should have parsed one connection")
	}
	if 1 != th.chunksParsed {
		t.Fatal("should have parsed 1 chunck(s)")
	}
	if len(payload) != th.bytesRead {
		t.Fatalf("should have parsed a payload of %d bytes, but saw %d\n", len(payload), th.bytesRead)
	}
	if n+m != len(raw) {
		t.Fatalf("should have parsed all %d bytes, not just %d\n", n, len(raw))
	}
}

func TestParseRestAnd1(t *testing.T) {
	ctx := context.Background()
	//ctx, cancel := context.WithCancel(ctx)

	th := &testHandler{
		conns: map[string]*Conn{},
	}

	p := NewParser(ctx, th)

	h, b, err := Encode(src, dst, domain, payload)
	if nil != err {
		t.Fatal(err)
	}
	raw := append(h, b...)
	i := len(raw)
	n, err := p.Write(raw[:i-1])
	if nil != err {
		t.Fatal(err)
	}
	m, err := p.Write(raw[i-1:])
	if nil != err {
		t.Fatal(err)
	}

	if 1 != len(th.conns) {
		t.Fatal("should have parsed one connection")
	}
	if 2 != th.chunksParsed {
		t.Fatal("should have parsed 2 chunck(s)")
	}
	if len(payload) != th.bytesRead {
		t.Fatalf("should have parsed a payload of %d bytes, but saw %d\n", len(payload), th.bytesRead)
	}
	if n+m != len(raw) {
		t.Fatalf("should have parsed all %d bytes, not just %d\n", n, len(raw))
	}
}

func TestParse1By1(t *testing.T) {
	ctx := context.Background()
	//ctx, cancel := context.WithCancel(ctx)

	th := &testHandler{
		conns: map[string]*Conn{},
	}

	p := NewParser(ctx, th)

	h, b, err := Encode(src, dst, domain, payload)
	if nil != err {
		t.Fatal(err)
	}
	raw := append(h, b...)
	count := 0
	for _, b := range raw {
		n, err := p.Write([]byte{b})
		if nil != err {
			t.Fatal(err)
		}
		count += n
	}

	if 1 != len(th.conns) {
		t.Fatal("should have parsed one connection")
	}
	if len(payload) != th.chunksParsed {
		t.Fatalf("should have parsed %d chunck(s), not %d", len(payload), th.chunksParsed)
	}
	if len(payload) != th.bytesRead {
		t.Fatalf("should have parsed a payload of %d bytes, but saw %d\n", len(payload), th.bytesRead)
	}
	if count != len(raw) {
		t.Fatalf("should have parsed all %d bytes, not just %d\n", len(raw), count)
	}
}

func testParseNBlocks(t *testing.T, count int) {
	ctx := context.Background()
	//ctx, cancel := context.WithCancel(ctx)

	th := &testHandler{
		conns: map[string]*Conn{},
	}

	nAddr := 1
	if count > 2 {
		nAddr = count - 2
	}
	p := NewParser(ctx, th)
	raw := []byte{}
	for i := 0; i < count; i++ {
		if i > 2 {
			copied := src
			src = copied
			src.port += i
		}
		h, b, err := Encode(src, dst, domain, payload)
		if nil != err {
			t.Fatal(err)
		}
		raw = append(raw, h...)
		raw = append(raw, b...)
	}
	n, err := p.Write(raw)
	if nil != err {
		t.Fatal(err)
	}

	if nAddr != len(th.conns) {
		t.Fatalf("should have parsed %d connection(s)", nAddr)
	}
	if count != th.chunksParsed {
		t.Fatalf("should have parsed %d chunk(s)", count)
	}
	if count*len(payload) != th.bytesRead {
		t.Fatalf("should have parsed a payload of %d bytes, but saw %d\n", count*len(payload), th.bytesRead)
	}
	if n != len(raw) {
		t.Fatalf("should have parsed all %d bytes, not just %d\n", len(raw), n)
	}
}
