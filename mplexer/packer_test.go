package telebit

import (
	"strconv"
	"testing"
)

func TestEncodeDataMessage(t *testing.T) {
	id := Addr{
		family: "IPv4",
		addr:   "192.168.1.101",
		port:   6743,
	}
	tun := Addr{
		family: id.family,
		addr:   "ex1.telebit.io",
		port:   80,
		scheme: "http",
	}

	payload := []byte("Hello, World!")
	header := []byte("IPv4,192.168.1.101,6743," + strconv.Itoa(len(payload)) + ",http,80,ex1.telebit.io,\n")
	//header = append([]byte{V1, byte(len(header))}, header...)
	header = append([]byte{254, byte(len(header))}, header...)

	h, b, err := Encode(payload, id, tun)
	if nil != err {
		t.Fatal(err)
	}

	if string(header) != string(h) {
		t.Fatalf("header %q should have matched %q", h, header)
	}
	if string(b) != string(payload) {
		t.Fatal("payload should be the exact reference to the original slice")
	}
}
