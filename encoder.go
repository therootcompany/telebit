package telebit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"git.rootprojects.org/root/telebit/dbg"
)

// TODO: try to be more like encoding/csv, or more like encoding/pem and encoding/json?

// Encoder converts TCP to MPLEXY-TCP
type Encoder struct {
	ctx context.Context
	//subctx context.Context
	mux sync.Mutex
	//out        io.WriteCloser
	out        io.Writer
	outErr     chan error
	bufferSize int
}

// NewEncoder returns an Encoder instance
func NewEncoder(ctx context.Context, wout io.Writer) *Encoder {
	enc := &Encoder{
		ctx:        ctx,
		out:        wout,
		outErr:     make(chan error),
		bufferSize: defaultBufferSize,
	}
	return enc
}

// Run loops over a select of contexts and error channels
// to cancel and close south-side connections, if needed.
// TODO should this be pushed elsewhere to handled?
func (enc *Encoder) Run() error {
	//ctx, cancel := context.WithCancel(enc.ctx)
	//defer cancel()

	//enc.subctx = ctx

	for {
		select {
		// TODO: do children respond to children cancelling?
		case <-enc.ctx.Done():
			// TODO
			//_ = enc.out.Close()
			return errors.New("context cancelled")
		case err := <-enc.outErr:
			// if a write fails for one, it fail for all
			return err
		}
	}
}

func (enc *Encoder) Start() error {
	go enc.Run()
	return nil
}

// Encode adds MPLEXY headers to raw net traffic, and is intended to be used on each client connection
func (enc *Encoder) Encode(rin io.Reader, src, dst Addr) error {
	rx := make(chan []byte)
	rxErr := make(chan error)

	//fmt.Fprintf(os.Stderr, "[debug] what's the source to encode? %v\n", src)

	go func() {
		for {
			b := make([]byte, enc.bufferSize)
			//fmt.Println("loopers gonna loop")
			n, err := rin.Read(b)
			if dbg.Debug {
				fmt.Fprintf(os.Stderr, "[debug] [encoder] [srv] Browser read %d %s\n", n, dbg.Trunc(b, n))
			}
			if n > 0 {
				rx <- b[:n]
			}
			if nil != err {
				if io.EOF != err {
					fmt.Fprintf(os.Stderr, "[encoder] [srv] Browser read err: %s\n", err)
				}
				rxErr <- err
				return
			}
		}
	}()

	// TODO set a timeout as a failsafe
	for {
		//fmt.Println("poopers gonna poop")
		select {
		// TODO, do we actually need ctx here?
		// would it be sufficient to expect the reader to be closed by the caller instead?
		case <-enc.ctx.Done():
			// TODO: verify that closing the reader will cause the goroutine to be released
			//rin.Close()
			if dbg.Debug {
				fmt.Fprintf(os.Stderr, "[debug] [encoder] [srv] Browser ctx.Done()\n")
			}
			return errors.New("cancelled by encoder read or parent context")
		/*
			case <-enc.subctx.Done():
				//rin.Close()
				fmt.Fprintf(os.Stderr, "[debug] [encoder] [srv] Browser subctx.Done()\n")
				return errors.New("cancelled by encoder write context")
		*/
		case b := <-rx:
			header, _, err := Encode(b, src, Addr{scheme: src.scheme, addr: dst.Hostname(), port: dst.Port()})
			if nil != err {
				//rin.Close()
				fmt.Fprintf(os.Stderr, "[encoder] [srv] Browser Encode err: %s\n", err)
				return err
			}
			//fmt.Fprintf(os.Stderr, "[debug] encode header: %s\n", string(header))
			//fmt.Fprintf(os.Stderr, "[debug] encode payload: %s\n", string(b))

			_, err = enc.write(header, b)
			if dbg.Debug {
				fmt.Fprintf(os.Stderr, "[debug] [encoder] [srv] Browser-to-tun write %d %s\n", len(header), strings.TrimSpace(string(header)))
				fmt.Fprintf(os.Stderr, "[debug] [encoder] [srv] %d %s\n", len(b), dbg.Trunc(b, len(b)))
			}
			if nil != err {
				fmt.Fprintf(os.Stderr, "[encoder] [srv] Browser-to-tun write err: %s\n", err)
				//rin.Close()
				return err
			}
		case err := <-rxErr:
			// it can be assumed that err will close though, right?
			//rin.Close()
			if io.EOF == err {
				header, _, _ := Encode(nil, src, Addr{scheme: "end"})
				//fmt.Fprintf(os.Stderr, "[debug] encode end: %s\n", string(header))
				// ignore err, which may have already closed
				_, _ = enc.write(header, nil)
				return nil
			}
			// TODO transmit message , []byte(err.Error())
			header, _, _ := Encode(nil, src, Addr{scheme: "error"})
			// ignore err, which may have already closed
			_, _ = enc.write(header, nil)
			return err
		}

	}
}

func (enc *Encoder) write(h, b []byte) (int, error) {
	// mutex here so that we can get back error info
	enc.mux.Lock()
	var m int
	n, err := enc.out.Write(append(h, b...))
	/*
		n, err := enc.out.Write(h)
		if nil == err && len(b) > 0 {
			m, err = enc.out.Write(b)
		}
	*/
	enc.mux.Unlock()
	if nil != err {
		enc.outErr <- err
	}
	return n + m, err
}
