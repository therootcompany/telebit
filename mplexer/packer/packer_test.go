package packer

import (
	"strconv"
	"testing"
)

func TestEncodeDataMessage(t *testing.T) {
	src := Addr{
		family: "IPv4",
		addr:   "192.168.1.101",
		port:   6743,
	}
	dst := Addr{
		family: src.family,
		port:   80,
		scheme: "http",
	}
	domain := "ex1.telebit.io"

	payload := []byte("Hello, World!")
	header := []byte("IPv4,192.168.1.101,6743," + strconv.Itoa(len(payload)) + ",http,80,ex1.telebit.io,\n")
	//header = append([]byte{V1, byte(len(header))}, header...)
	header = append([]byte{254, byte(len(header))}, header...)

	h, b, err := Encode(src, dst, domain, payload)
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
