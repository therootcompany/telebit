package packer

import (
	"net"
	"testing"
)

func TestDecode1WholeBlock(t *testing.T) {
	testDecodeNBlocks(t, 1)
}

func testDecodeNBlocks(t *testing.T, count int) {
	wp, rp := net.Pipe()

	//ctx := context.Background()
	decoder := NewDecoder(rp)
	nAddr := 1
	if count > 2 {
		nAddr = count - 2
	}

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

	var nw int
	go func() {
		var err error
		//fmt.Println("writers gonna write")
		nw, err = wp.Write(raw)
		if nil != err {
			//fmt.Println("writer died")
			t.Fatal(err)
		}
		// very important: don't forget to close when done!
		wp.Close()
		//fmt.Println("writer done wrote")
	}()

	th := &testHandler{
		conns: map[string]*Conn{},
	}
	//fmt.Println("streamers gonna stream")
	err := decoder.Decode(th)
	if nil != err {
		t.Fatalf("failed to decode stream: %s", err)
	}
	//fmt.Println("streamer done streamed")

	if nAddr != len(th.conns) {
		t.Fatalf("should have parsed %d connection(s)", nAddr)
	}
	if count != th.chunksParsed {
		t.Fatalf("should have parsed %d chunk(s)", count)
	}
	if count*len(payload) != th.bytesRead {
		t.Fatalf("should have parsed a payload of %d bytes, but saw %d\n", count*len(payload), th.bytesRead)
	}
	if nw != len(raw) {
		t.Fatalf("should have parsed all %d bytes, not just %d\n", len(raw), nw)
	}
}
