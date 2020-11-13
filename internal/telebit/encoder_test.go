package telebit

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestEncodeWholeBlock(t *testing.T) {
	ch := make(chan string)

	a1 := "A.1: hello"
	a2 := "A.2: smello"
	b1 := "B.1: hello again"
	b2 := "B.2: hello a third time"
	m := map[string]bool{
		a1: false,
		a2: false,
		b1: false,
		b2: false,
	}

	go func() {
		for {
			str := <-ch
			// TODO check the headers too
			if len(str) > 0 && 0xFE == str[0] {
				fmt.Printf("TODO header: %q\n", str)
				parts := strings.Split(str, "\n")
				if len(parts) <= 1 {
					continue
				}
				str = parts[1]
			}

			b, ok := m[str]
			if !ok {
				// possible corruption
				t.Fatalf("unexpected string %q", str)
			}
			if b {
				// possible corruption also
				t.Fatalf("duplicate string %q", str)
			}

			m[str] = true
		}
	}()

	// TODO nix context here
	ctx := context.TODO()
	rin, wout := net.Pipe()
	go func() {
		for {
			b := make([]byte, 1024)
			n, err := rin.Read(b)
			if nil != err {
				t.Fatalf("Error: %s\n", err)
				return
			}
			r := b[:n]
			ch <- string(r)
		}
	}()
	encoder := NewEncoder(ctx, wout)

	go func() {
		err := encoder.Run()
		if nil != err {
			if io.EOF != err {
				t.Fatalf("Encoder Run Err: %q\n", err)
			}
		}
		wout.Close()
	}()

	// TODO eliminate Run and don't sleep here
	time.Sleep(time.Millisecond)

	// single client
	go func() {
		wout, rin := net.Pipe()

		go func() {
			wout.Write([]byte(a1))
			wout.Write([]byte(a2))
		}()

		err := encoder.Encode(rin, Addr{
			family: "IPv4",
			addr:   "192.168.1.102",
			port:   4834,
		}, Addr{
			scheme: "https",
			addr:   "example.com",
			port:   443,
		})
		if nil != err {
			t.Fatalf("Enc Err 1: %q\n", err)
		}
	}()

	// single client
	go func() {
		wout, rin := net.Pipe()

		go func() {
			wout.Write([]byte(b1))
			wout.Write([]byte(b2))
		}()

		err := encoder.Encode(rin, Addr{
			family: "IPv4",
			addr:   "192.168.1.103",
			port:   4834,
		}, Addr{
			scheme: "https",
			addr:   "example.com",
			port:   443,
		})
		if nil != err {
			t.Fatalf("Enc Err 2: %q\n", err)
		}
	}()

	// TODO use wait group
	time.Sleep(10 * time.Millisecond)

	for k, v := range m {
		if !v {
			t.Fatalf("failed to encode and transmit a value: %q", k)
		}
	}
}

/*
func TestEncodeEnd(t *testing.T) {
}

func TestEncodeError(t *testing.T) {
}
*/
