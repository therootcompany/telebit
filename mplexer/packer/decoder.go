package packer

import (
	"context"
	"errors"
	"io"
)

// Decoder handles a ReadCloser stream containing mplexy-encoded clients
type Decoder struct {
	ctx context.Context
	r   io.ReadCloser
}

// NewDecoder returns an initialized Decoder
func NewDecoder(ctx context.Context, r io.ReadCloser) *Decoder {
	return &Decoder{
		ctx: ctx,
		r:   r,
	}
}

// StreamDecode will call WriteMessage as often as addressable data exists,
// reading up to bufferSize (default 8192) at a time
// (header + data, though headers are often sent separately from data).
func (d *Decoder) StreamDecode(handler Handler, bufferSize int) error {
	p := NewParser(handler)
	rx := make(chan []byte)
	rxErr := make(chan error)

	if 0 == bufferSize {
		bufferSize = 8192
	}

	go func() {
		for {
			b := make([]byte, bufferSize)
			//fmt.Println("loopers gonna loop")
			n, err := d.r.Read(b)
			if n > 0 {
				rx <- b[:n]
			}
			if nil != err {
				rxErr <- err
				return
			}
		}
	}()

	for {
		//fmt.Println("poopers gonna poop")
		select {
		// TODO, do we actually need ctx here?
		// would it be sufficient to expect the reader to be closed by the caller instead?
		case <-d.ctx.Done():
			// TODO: verify that closing the reader will cause the goroutine to be released
			d.r.Close()
			return errors.New("cancelled by context")
		case b := <-rx:
			_, err := p.Write(b)
			if nil != err {
				// an error to write represents an unrecoverable error,
				// not just a downstream client error
				d.r.Close()
				return err
			}
		case err := <-rxErr:
			d.r.Close()
			if io.EOF == err {
				// it can be assumed that err will close though, right
				return nil
			}
			return err
		}

	}
}
