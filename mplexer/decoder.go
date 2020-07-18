package telebit

import (
	"fmt"
	"io"
	"log"

	"git.coolaj86.com/coolaj86/go-telebitd/dbg"
)

// Decoder handles a Reader stream containing mplexy-encoded clients
type Decoder struct {
	in         io.Reader
	bufferSize int
}

// NewDecoder returns an initialized Decoder
func NewDecoder(rin io.Reader) *Decoder {
	return &Decoder{
		in:         rin,
		bufferSize: defaultBufferSize,
	}
}

// Decode will call WriteMessage as often as addressable data exists,
// reading up to bufferSize (default 8192) at a time
// (header + data, though headers are often sent separately from data).
func (d *Decoder) Decode(out Router) error {
	p := NewParser(out)
	rx := make(chan []byte)
	rxErr := make(chan error)

	go func() {
		for {
			b := make([]byte, d.bufferSize)
			n, err := d.in.Read(b)
			if dbg.Debug {
				log.Println("[debug] [decoder] [srv] Tunnel read", n, dbg.Trunc(b, n))
			}
			if n > 0 {
				rx <- b[:n]
			}
			if nil != err {
				fmt.Println("[debug] [decoder] [srv] Tunnel read err", err)
				rxErr <- err
				return
			}
		}
	}()

	for {
		select {
		case b := <-rx:
			n, err := p.Write(b)
			if dbg.Debug {
				fmt.Println("[debug] [decoder] [srv] Tunnel write", n, len(b), dbg.Trunc(b, len(b)))
			}
			// TODO BUG: handle when 'n' bytes written is less than len(b)
			if nil != err {
				fmt.Println("[debug] [decoder] [srv] Tunnel write error")
				// an error to write represents an unrecoverable error,
				// not just a downstream client error
				//d.in.Close()
				return err
			}
		case err := <-rxErr:
			//d.in.Close()
			if io.EOF == err {
				// it can be assumed that err will close though, right
				return nil
			}
			return err
		}

	}
}
