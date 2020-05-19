package packer

import (
	"context"
	"errors"
	"io"
	"sync"
)

type Encoder struct {
	ctx    context.Context
	subctx context.Context
	mux    sync.Mutex
	w      io.WriteCloser
	wErr   chan error
}

func NewEncoder(ctx context.Context, w io.WriteCloser) *Encoder {
	enc := &Encoder{
		ctx:  ctx,
		w:    w,
		wErr: make(chan error),
	}
	return enc
}

func (enc *Encoder) Run() error {
	ctx, cancel := context.WithCancel(enc.ctx)
	defer cancel()

	enc.subctx = ctx

	for {
		select {
		// TODO: do children respond to children cancelling?
		case <-enc.ctx.Done():
			// TODO
			_ = enc.w.Close()
			return errors.New("context cancelled")
		case err := <-enc.wErr:
			return err
		}
	}
}

func (enc *Encoder) Start() error {
	go enc.Run()
	return nil
}

// TODO inverse

// StreamEncode can (and should) be called multiple times (once per client).
func (enc *Encoder) StreamEncode(src Addr, r io.ReadCloser, bufferSize int) error {
	rx := make(chan []byte)
	rxErr := make(chan error)

	if 0 == bufferSize {
		bufferSize = 8192
	}

	go func() {
		for {
			b := make([]byte, bufferSize)
			//fmt.Println("loopers gonna loop")
			n, err := r.Read(b)
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
		case <-enc.ctx.Done():
			// TODO: verify that closing the reader will cause the goroutine to be released
			r.Close()
			return errors.New("cancelled by context")
		case <-enc.subctx.Done():
			r.Close()
			return errors.New("cancelled by context")
		case b := <-rx:
			header, _, err := Encode(src, Addr{}, "", b)
			if nil != err {
				r.Close()
				return err
			}
			_, err = enc.write(header)
			if nil != err {
				r.Close()
				return err
			}
			_, err = enc.write(b)
			if nil != err {
				r.Close()
				return err
			}
		case err := <-rxErr:
			// it can be assumed that err will close though, right?
			r.Close()
			if io.EOF == err {
				header, _, _ := Encode(src, Addr{scheme: "end"}, "", nil)
				// ignore err, which may have already closed
				_, _ = enc.write(header)
				return nil
			}
			header, _, _ := Encode(src, Addr{scheme: "error"}, "", []byte(err.Error()))
			// ignore err, which may have already closed
			_, _ = enc.write(header)
			return err
		}

	}
}

func (enc *Encoder) write(b []byte) (int, error) {
	// mutex here so that we can get back error info
	enc.mux.Lock()
	n, err := enc.w.Write(b)
	enc.mux.Unlock()
	if nil != err {
		enc.wErr <- err
	}
	return n, err
}
