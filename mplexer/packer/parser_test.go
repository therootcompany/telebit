package packer

import (
	"math/rand"
	"net"
	"testing"
)

var srcTestAddr = Addr{
	family: "IPv4",
	addr:   "192.168.1.101",
	port:   6743,
}
var dstTestAddr = Addr{
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

func (th *testHandler) RouteBytes(srcAddr, dstAddr Addr, b []byte) {
	th.chunksParsed++
	src := &srcAddr
	dst := &dstAddr
	_, ok := th.conns[src.Network()]
	if !ok {
		rconn, wconn := net.Pipe()
		conn := &Conn{
			//updated:         time.Now(),
			relaySourceAddr: *src,
			relayTargetAddr: *dst,
			relay:           rconn,
			local:           wconn,
		}
		th.conns[src.Network()] = conn
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

func TestParseBy1(t *testing.T) {
	testParseByN(t, 1)
}

func TestParseByPrimes(t *testing.T) {
	testParseByN(t, 2)
	testParseByN(t, 3)
	testParseByN(t, 5)
	testParseByN(t, 7)
	testParseByN(t, 11)
	testParseByN(t, 13)
	testParseByN(t, 17)
	testParseByN(t, 19)
	testParseByN(t, 23)
	testParseByN(t, 29)
	testParseByN(t, 31)
	testParseByN(t, 37)
	testParseByN(t, 41)
	testParseByN(t, 43)
	testParseByN(t, 47)
}

func TestParseByRand(t *testing.T) {
	testParseByN(t, 0)
}

func TestParse1AndRest(t *testing.T) {
	th := &testHandler{
		conns: map[string]*Conn{},
	}

	p := NewParser(th)

	h, b, err := Encode(payload, srcTestAddr, dstTestAddr)
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
	th := &testHandler{
		conns: map[string]*Conn{},
	}

	p := NewParser(th)

	h, b, err := Encode(payload, srcTestAddr, dstTestAddr)
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

func testParseByN(t *testing.T, n int) {
	//fmt.Printf("[debug] parse by %d\n", n)
	th := &testHandler{
		conns: map[string]*Conn{},
	}

	p := NewParser(th)

	h, b, err := Encode(payload, srcTestAddr, dstTestAddr)
	if nil != err {
		t.Fatal(err)
	}
	raw := append(h, b...)
	count := 0
	nChunk := 0
	b = raw
	for {
		r := 24
		c := len(b)
		if 0 == c {
			break
		}
		i := n
		if 0 == n {
			if c < r {
				r = c
			}
			i = 1 + rand.Intn(r+1)
		}
		if c < i {
			i = c
		}
		// TODO shouldn't this cause an error?
		//a := b[:i][0]
		a := b[:i]
		b = b[i:]
		nw, err := p.Write(a)
		if nil != err {
			t.Fatal(err)
		}
		count += nw
		if count > len(h) {
			nChunk++
		}
	}

	if 1 != len(th.conns) {
		t.Fatalf("should have parsed one connection, not %d", len(th.conns))
	}
	if nChunk != th.chunksParsed {
		t.Fatalf("should have parsed %d chunk(s), not %d", nChunk, th.chunksParsed)
	}
	if len(payload) != th.bytesRead {
		t.Fatalf("should have parsed a payload of %d bytes, but saw %d\n", len(payload), th.bytesRead)
	}
	if count != len(raw) {
		t.Fatalf("should have parsed all %d bytes, not just %d\n", len(raw), count)
	}
}

func testParseNBlocks(t *testing.T, count int) {
	th := &testHandler{
		conns: map[string]*Conn{},
	}

	nAddr := 1
	if count > 2 {
		nAddr = count - 2
	}
	p := NewParser(th)
	raw := []byte{}
	for i := 0; i < count; i++ {
		if i > 2 {
			copied := srcTestAddr
			srcTestAddr = copied
			srcTestAddr.port += i
		}
		h, b, err := Encode(payload, srcTestAddr, dstTestAddr)
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
