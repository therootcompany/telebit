package mplexer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"git.coolaj86.com/coolaj86/go-telebitd/mplexer/packer"

	"github.com/gorilla/websocket"
)

// Listener defines a listener for use with http servers
type Listener struct {
	//ParentAddr net.Addr
	//Conns  chan *Conn
	ws     *websocket.Conn
	ctx    context.Context
	parser *packer.Parser
}

// Listen creates a channel for connections and returns the listener
func (m *MultiplexLocal) Listen(ctx context.Context) (*Listener, error) {
	authz, err := m.SortingHat.Authz()
	if nil != err {
		return nil, err
	}

	wsd := websocket.Dialer{}
	headers := http.Header{}
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", authz))
	// *http.Response
	wsconn, _, err := wsd.DialContext(ctx, m.Relay, headers)
	if nil != err {
		return nil, err
	}

	//conns := make(chan *packer.Conn)
	//parser := &packer.NewParser(ctx, conns)

	/*
		go func() {
			conn, err := packer.Accept()
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to accept new relayed connection: %s\n", err)
				return
			}
			conns <- conn
		}()
	*/

	handler := &Handler{}
	listener := &Listener{
		//Conns:  conns,
		parser: packer.NewParser(ctx, handler),
	}
	go m.listen(ctx, wsconn, listener)
	return listener, nil
}

type Handler struct {
}

func (h *Handler) WriteMessage(packer.Addr, []byte) {
	panic(errors.New("not implemented"))
}

func (m *MultiplexLocal) listen(ctx context.Context, wsconn *websocket.Conn, listener *Listener) {
	// will cancel if ws errors out or closes
	// (TODO: this may also be redundant)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Ping every 15 seconds, or stop listening
	go func() {
		for {
			time.Sleep(15 * time.Second)
			deadline := time.Now().Add(45 * time.Second)
			if err := wsconn.WriteControl(websocket.PingMessage, []byte(""), deadline); nil != err {
				fmt.Fprintf(os.Stderr, "failed to write ping message to websocket: %s\n", err)
				cancel()
				break
			}
		}
	}()

	// The write loop (which fails if ping fails)
	go func() {
		// TODO optimal buffer size
		b := make([]byte, 128*1024)
		for {
			n, err := listener.parser.Read(b)
			if n > 0 {
				if err := wsconn.WriteMessage(websocket.BinaryMessage, b); nil != err {
					fmt.Fprintf(os.Stderr, "failed to write packer message to websocket: %s\n", err)
					break
				}
			}
			if nil != err {
				if io.EOF != err {
					fmt.Fprintf(os.Stderr, "failed to read message from packer: %s\n", err)
					break
				}
				fmt.Fprintf(os.Stderr, "[TODO debug] closed packer: %s\n", err)
				break
			}
		}
		// TODO handle EOF as websocket.CloseNormal?
		message := websocket.FormatCloseMessage(websocket.CloseGoingAway, "closing connection")
		deadline := time.Now().Add(10 * time.Second)
		if err := wsconn.WriteControl(websocket.CloseMessage, message, deadline); nil != err {
			fmt.Fprintf(os.Stderr, "failed to write close message to websocket: %s\n", err)
		}
		_ = wsconn.Close()
	}()

	// The read loop (also fails if ping fails)
	for {
		_, message, err := wsconn.ReadMessage()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read message from websocket: %s\n", err)
			break
		}

		//
		_, err = listener.packer.Write(message)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to process message from websocket: %s\n", err)
			break
		}
	}

	// just to be sure
	listener.packer.Close()
	wsconn.Close()

	return
}

/*
// Feed will block while pushing a net.Conn onto Conns
func (l *Listener) Feed(conn *Conn) {
	l.Conns <- conn
}
*/

// net.Listener interface

/*
// Accept will block and wait for a new net.Conn
func (l *Listener) Accept() (*Conn, error) {
	select {
	case conn, ok := <-l.Conns:
		if ok {
			return conn, nil
		}
		return nil, io.EOF

	case <-l.ctx.Done():
		// TODO is another error more suitable?
		// TODO is this redundant?
		return nil, io.EOF
	}
}
*/

func (l *Listener) Accept() (*packer.Conn, error) {
	return l.Accept()
}

// Close will close the Conns channel
func (l *Listener) Close() error {
	//close(l.Conns)
	//return nil
	return l.packer.Close()
}

// Addr returns nil to fulfill the net.Listener interface
func (l *Listener) Addr() net.Addr {
	// Addr may (or may not) return the original TCP or TLS listener's address
	//return l.ParentAddr
	return nil
}
