package packer

import (
	"context"
	"errors"
)

type Server struct {
	ctx       context.Context
	newConns  chan *Conn
	data      []byte
	dataReady chan struct{}
}

func (s *Server) Accept() (*Conn, error) {
	select {
	case <-s.ctx.Done():
		return nil, errors.New("TODO: ErrClosed")
	case conn := <-s.newConns:
		return conn, nil
	}
}

// Read packs transforms local responses into wrapped data for the tunnel
func (s *Server) Read(b []byte) (int, error) {
	select {
	case <-s.ctx.Done():
		return 0, errors.New("TODO: EOF / ErrClosed")
	case <-s.dataReady:
		if 0 == len(s.data) {
			return s.Read(b)
		}
		return s.read(b)
	}
}

func (s *Server) read(b []byte) (int, error) {
	// TODO mutex data while reading, against writing?

	c := len(b)      // capacity
	a := len(s.data) // available
	n := c

	// see if the available data is smaller than the receiving buffer
	if a < c {
		n = a
	}

	// copy available data up to capacity
	for i := 0; i < n; i++ {
		b[i] = s.data[i]
	}
	// shrink the data slice by amount read
	s.data = s.data[n:]

	// if there's data left over, flag as ready to read again
	// otherwise... flag as ready to write?
	if len(b) > 0 {
		s.dataReady <- struct{}{}
	} else {
		//p.writeReady <- struct{}{}
	}

	// Note a read error should not be possible here
	// as all traffic (including errors) can be wrapped
	return n, nil
}

// Close (TODO) should politely close all connections, if possible (set Read() to io.EOF, or use ErrClosed?)
func (s *Server) Close() error {
	return errors.New("not implemented")
}
