package telebit

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
)

// A Listener transforms a multiplexed websocket connection into individual net.Conn-like connections.
type Listener struct {
	//wsconn       *websocket.Conn
	tun          net.Conn
	incoming     chan *Conn
	close        chan struct{}
	encoder      *Encoder
	chunksParsed int
	bytesRead    int
	conns        map[string]net.Conn
	//conns        map[string]*Conn
}

// Listen creates a new Listener and sets it up to receive and distribute connections.
func Listen(tun net.Conn) *Listener {
	ctx := context.TODO()

	// Feed the socket into the Encoder and Decoder
	listener := &Listener{
		tun:      tun,
		incoming: make(chan *Conn, 1), // buffer ever so slightly
		close:    make(chan struct{}),
		encoder:  NewEncoder(ctx, tun),
		conns:    map[string]net.Conn{},
		//conns:    map[string]*Conn{},
	}

	// TODO perhaps the wrapper should have a mutex
	// rather than having a goroutine in the encoder
	go func() {
		err := listener.encoder.Run()
		fmt.Printf("encoder stopped entirely: %q", err)
		tun.Close()
	}()

	// Decode the stream as it comes in
	decoder := NewDecoder(tun)
	go func() {
		// TODO pass error to Accept()
		err := decoder.Decode(listener)

		// The listener itself must be closed explicitly because
		// there's an encoder with a callback between the websocket
		// and the multiplexer, so it doesn't know to stop listening otherwise
		listener.Close()
		fmt.Printf("the main stream is done: %q\n", err)
	}()

	return listener
}

// ListenAndServe listens on a websocket and handles the incomming net.Conn-like connections with a Handler
func ListenAndServe(tun net.Conn, mux Handler) error {
	listener := Listen(tun)
	return Serve(listener, mux)
}

// Serve Accept()s connections which have already been unwrapped and serves them with the given Handler
func Serve(listener *Listener, mux Handler) error {
	for {
		client, err := listener.Accept()
		if nil != err {
			return err
		}

		go func() {
			err = mux.Serve(client)
			if nil != err {
				if io.EOF != err {
					fmt.Printf("client could not be served: %q\n", err.Error())
				}
			}
			client.Close()
		}()
	}
}

// Accept returns a tunneled network connection
func (l *Listener) Accept() (net.Conn, error) {
	select {
	case rconn, ok := <-l.incoming:
		if ok {
			return rconn, nil
		}
		return nil, io.EOF

	case <-l.close:
		return nil, http.ErrServerClosed
	}
}

// Close stops accepting new connections and closes the underlying websocket.
// TODO return errors.
func (l *Listener) Close() error {
	l.tun.Close()
	close(l.incoming)
	l.close <- struct{}{}
	return nil
}

// RouteBytes receives address information and a buffer and creates or re-uses a pipe that can be Accept()ed.
func (l *Listener) RouteBytes(srcAddr, dstAddr Addr, b []byte) {
	// TODO use context to be able to cancel many at once?
	l.chunksParsed++

	src := &srcAddr
	dst := &dstAddr
	pipe := l.getPipe(src, dst, len(b))
	//fmt.Printf("%s\n", b)

	// handle errors before data writes because I don't
	// remember where the error message goes
	if "error" == string(dst.scheme) {
		pipe.Close()
		delete(l.conns, src.Network())
		fmt.Printf("a stream errored remotely: %v\n", src)
	}

	// write data, if any
	if len(b) > 0 {
		l.bytesRead += len(b)
		pipe.Write(b)
	}
	// EOF, if needed
	if "end" == string(dst.scheme) {
		fmt.Println("[debug] end")
		pipe.Close()
		delete(l.conns, src.Network())
	}
}

func (l *Listener) getPipe(src, dst *Addr, count int) net.Conn {
	connID := src.Network()
	pipe, ok := l.conns[connID]

	// Pipe exists
	if ok {
		return pipe
	}
	fmt.Printf("New client (%d byte hello)\n\tfrom %#v\n\tto %#v:\n", count, src, dst)

	// Create pipe
	rawPipe, pipe := net.Pipe()
	newconn := &Conn{
		//updated:         time.Now(),
		relaySourceAddr: *src,
		relayTargetAddr: *dst,
		relay:           rawPipe,
	}
	l.conns[connID] = pipe
	l.incoming <- newconn

	// Handle encoding
	go func() {
		// TODO handle err
		err := l.encoder.Encode(pipe, *src, *dst)
		// the error may be EOF or ErrServerClosed or ErrGoingAwawy or some such
		// or it might be an actual error
		// In any case, we'll just close it all
		newconn.Close()
		pipe.Close()
		fmt.Printf("a stream is done: %q\n", err)
	}()

	return pipe
}
