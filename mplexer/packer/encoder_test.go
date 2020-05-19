package packer

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestEncodeWholeBlock(t *testing.T) {
	ch := make(chan string)
	go func() {
		for {
			str := <-ch
			fmt.Printf("Read: %q\n", str)
		}
	}()

	ctx := context.Background()
	rp, wp := net.Pipe()
	go func() {
		for {
			b := make([]byte, 1024)
			n, err := rp.Read(b)
			if nil != err {
				fmt.Printf("Error: %s\n", err)
				return
			}
			r := b[:n]
			ch <- string(r)
		}
	}()
	encoder := NewEncoder(ctx, wp)
	encoder.Start()

	time.Sleep(time.Millisecond)

	// single client
	go func() {
		wp, rp := net.Pipe()

		go func() {
			wp.Write([]byte("hello"))
			wp.Write([]byte("smello"))
		}()

		err := encoder.StreamEncode(Addr{
			family: "IPv4",
			addr:   "192.168.1.102",
			port:   4834,
		}, rp, 0)
		if nil != err {
			fmt.Printf("Enc Err: %q\n", err)
		}
	}()

	// single client
	go func() {
		wp, rp := net.Pipe()

		go func() {
			wp.Write([]byte("hello again"))
			wp.Write([]byte("hello a third time"))
		}()

		err := encoder.StreamEncode(Addr{
			family: "IPv4",
			addr:   "192.168.1.103",
			port:   4834,
		}, rp, 0)
		if nil != err {
			fmt.Printf("Enc Err 2: %q\n", err)
		}
	}()

	time.Sleep(time.Second)
}
