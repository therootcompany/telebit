package packer

import (
	"io"
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
			//fmt.Println("loopers gonna loop")
			n, err := d.in.Read(b)
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
		case b := <-rx:
			_, err := p.Write(b)
			if nil != err {
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
