package mplexer

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

type MultiplexLocal struct {
	Relay      string
	SortingHat SortingHat
	Timeout    time.Duration
}

func New(relay string, hat SortingHat) *MultiplexLocal {
	return &MultiplexLocal{
		Relay:      relay,
		SortingHat: hat,
		Timeout:    30 * time.Second,
	}
}

func (m *MultiplexLocal) ListenAndServe(ctx context.Context) error {
	listener, err := m.Listen(ctx)
	if nil != err {
		return err
	}

	for {
		pconn, err := listener.Accept() // packer.Conn
		if nil != err {
			return err
		}

		go m.serve(ctx, pconn)
	}
}

func (m *MultiplexLocal) serve(ctx context.Context, pconn *Conn) {
	//paddr := pconn.LocalAddr().(*Addr) // packer.Addr
	paddr := pconn.LocalAddr()
	//addr.Network()
	//addr.String()
	paddr.Scheme()
	//paddr.Encrypted()
	//paddr.Servername()

	// todo: some sort of logic to avoid infinite loop to self?
	// (that's probably not possible since the connection could
	// route several layers deep)
	if target, err := m.SortingHat.LookupTarget(paddr); nil != target {
		if nil != err {
			// TODO get a log channel or some such
			fmt.Fprintf(os.Stderr, "lookup failed for tunneled client: %s", err)
			err := pconn.Error(err)
			if nil != err {
				fmt.Fprintf(os.Stderr, "failed to signal error back to relay: %s", err)
			}
			return
		}
		pipePacker(ctx, pconn, target, m.Timeout)
	}
}

func pipePacker(ctx context.Context, pconn *Conn, target net.Conn, timeout time.Duration) {
	// how can this be done so that target errors are
	// sent back to the relay server?

	// Also something like ReadAhead(size) should signal
	// to read and send up to `size` bytes without waiting
	// for a response - since we can't signal 'non-read' as
	// is the normal operation of tcp... or can we?
	// And how do we distinguish idle from dropped?
	// Maybe this should have been a udp protocol???

	defer pconn.Close()
	defer target.Close()

	srcCh := make(chan []byte)
	dstCh := make(chan []byte)
	errCh := make(chan error)

	// Source (Relay) Read Channel
	go func() {
		// TODO what's the optimal size to buffer?
		// TODO user buffered reader
		b := make([]byte, 128*1024)
		for {
			pconn.SetDeadline(time.Now().Add(timeout))
			n, err := pconn.Read(b)
			if n > 0 {
				srcCh <- b
			}
			if nil != err {
				// TODO let client log this server-side error (unless EOF)
				// (nil here because we probably can't send the error to the relay)
				errCh <- nil
				break
			}
		}
	}()

	// Target (Local) Read Channel
	go func() {
		// TODO what's the optimal size to buffer?
		// TODO user buffered reader
		b := make([]byte, 128*1024)
		for {
			target.SetDeadline(time.Now().Add(timeout))
			n, err := target.Read(b)
			if n > 0 {
				dstCh <- b
			}
			if nil != err {
				if io.EOF == err {
					err = nil
				}
				errCh <- err
				break
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			break
		case b := <-srcCh:
			target.SetDeadline(time.Now().Add(timeout))
			_, err := target.Write(b)
			if nil != err {
				// TODO log error locally
				pconn.Error(err)
				break
			}
		case b := <-dstCh:
			pconn.SetDeadline(time.Now().Add(timeout))
			_, err := pconn.Write(b)
			if nil != err {
				// TODO log error locally
				break
			}
		}
	}
}
