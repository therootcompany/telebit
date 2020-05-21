package packer

import (
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

// Note: 64k is the TCP max, but 1460b is the 100mbit Ethernet max (1500 MTU - overhead),
// but 1Gbit Ethernet (Jumbo frame) has an 9000b MTU
// Nerds posting benchmarks on SO show that 8k seems about right,
// but even 1024b could work well.
var defaultBufferSize = 8192

// ErrBadGateway means that the target did not accept the connection
var ErrBadGateway = errors.New("EBADGATEWAY")

// A Handler routes, proxies, terminates, or responds to a net.Conn.
type Handler interface {
	Serve(*Conn) error
}

type HandlerFunc func(*Conn) error

// Serve calls f(conn).
func (f HandlerFunc) Serve(conn *Conn) error {
	return f(conn)
}

// NewForwarder creates a handler that port-forwards to a target
func NewForwarder(target string, timeout time.Duration) HandlerFunc {
	return func(client *Conn) error {
		tconn, err := net.Dial("tcp", target)
		if nil != err {
			return err
		}
		return Forward(client, tconn, timeout)
	}
}

// Forward port-forwards a relay (websocket) client to a target (local) server
func Forward(client *Conn, target net.Conn, timeout time.Duration) error {

	// Something like ReadAhead(size) should signal
	// to read and send up to `size` bytes without waiting
	// for a response - since we can't signal 'non-read' as
	// is the normal operation of tcp... or can we?
	// And how do we distinguish idle from dropped?
	// Maybe this should have been a udp protocol???

	defer client.Close()
	defer target.Close()

	srcCh := make(chan []byte)
	dstCh := make(chan []byte)
	srcErrCh := make(chan error)
	dstErrCh := make(chan error)

	// Source (Relay) Read Channel
	go func() {
		for {
			b := make([]byte, defaultBufferSize)
			n, err := client.Read(b)
			if n > 0 {
				srcCh <- b[:n]
			}
			if nil != err {
				// TODO let client log this server-side error (unless EOF)
				// (nil here because we probably can't send the error to the relay)
				srcErrCh <- err
				break
			}
		}
	}()

	// Target (Local) Read Channel
	go func() {
		for {
			b := make([]byte, defaultBufferSize)
			n, err := target.Read(b)
			if n > 0 {
				dstCh <- b[:n]
			}
			if nil != err {
				if io.EOF == err {
					err = nil
				}
				dstErrCh <- err
				break
			}
		}
	}()

	var err error = nil
	for {
		select {
		// TODO do we need a context here?
		//case <-ctx.Done():
		//		break
		case b := <-srcCh:
			client.SetDeadline(time.Now().Add(timeout))
			_, err = target.Write(b)
			if nil != err {
				fmt.Printf("write to target failed: %q", err.Error())
				break
			}
		case b := <-dstCh:
			target.SetDeadline(time.Now().Add(timeout))
			_, err = client.Write(b)
			if nil != err {
				fmt.Printf("write to remote failed: %q", err.Error())
				break
			}
		case err = <-srcErrCh:
			if nil != err {
				fmt.Printf("read from remote failed: %q", err.Error())
			}
			break
		case err = <-dstErrCh:
			if nil != err {
				fmt.Printf("read from target failed: %q", err.Error())
			}
			break

		}
	}

	client.Close()
	return err
}
