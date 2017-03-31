package client

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/packer"
)

// WsHandler handles all of reading and writing for the websocket connection to the RVPN server
// and the TCP connections to the local servers.
type WsHandler struct {
	lock       sync.Mutex
	localConns map[string]net.Conn

	servicePorts map[string]int

	ctx      context.Context
	dataChan chan *packer.Packer
}

// NewWsHandler creates a new handler ready to be given a websocket connection. The services
// argument specifies what port each service type should be directed to on the local interface.
func NewWsHandler(services map[string]int) *WsHandler {
	h := new(WsHandler)
	h.servicePorts = services
	h.localConns = make(map[string]net.Conn)
	return h
}

// HandleConn handles all of the traffic on the provided websocket connection. The function
// will not return until the connection ends.
//
// The WsHandler is designed to handle exactly one connection at a time. If HandleConn is called
// again while the instance is still handling another connection (or if the previous connection
// failed to fully cleanup) calling HandleConn again will panic.
func (h *WsHandler) HandleConn(ctx context.Context, conn *websocket.Conn) {
	if h.dataChan != nil {
		panic("WsHandler.HandleConn called while handling a previous connection")
	}
	if len(h.localConns) > 0 {
		panic(fmt.Sprintf("WsHandler has lingering local connections: %v", h.localConns))
	}
	h.dataChan = make(chan *packer.Packer)

	// The sub context allows us to clean up all of the goroutines associated with this websocket
	// if it closes at any point for any reason.
	subCtx, socketQuit := context.WithCancel(ctx)
	defer socketQuit()
	h.ctx = subCtx

	// Start the routine that will write all of the data from the local connection to the
	// remote websocket connection.
	go h.writeRemote(conn)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			loginfo.Println("failed to read message from websocket", err)
			return
		}

		p, err := packer.ReadMessage(message)
		if err != nil {
			loginfo.Println("failed to parse message from websocket", err)
			return
		}

		h.writeLocal(p)
	}
}

func (h *WsHandler) writeRemote(conn *websocket.Conn) {
	defer h.closeConnections()
	defer func() { h.dataChan = nil }()

	for {
		select {
		case <-h.ctx.Done():
			// We can't tell if this happened because the websocket is already closed/errored or
			// if it happened because the main context closed (in which case it would be preferable
			// to properly close the connection). As such we try to close the connection and ignore
			// all errors if it doesn't work.
			message := websocket.FormatCloseMessage(websocket.CloseGoingAway, "closing connection")
			deadline := time.Now().Add(10 * time.Second)
			conn.WriteControl(websocket.CloseMessage, message, deadline)
			conn.Close()
			return

		case p := <-h.dataChan:
			packed := p.PackV1()
			conn.WriteMessage(websocket.BinaryMessage, packed.Bytes())
		}
	}
}

func (h *WsHandler) sendSpecial(header *packer.Header, service string) {
	p := packer.NewPacker(header)
	p.SetService(service)

	// Avoid blocking on the data channel if the websocket is already closed
	select {
	case h.dataChan <- p:
	case <-h.ctx.Done():
	}
}

func (h *WsHandler) getLocalConn(p *packer.Packer) net.Conn {
	h.lock.Lock()
	defer h.lock.Unlock()

	key := fmt.Sprintf("%s:%d", p.Address(), p.Port())
	// Simplest case: it's already open, just return it.
	if conn := h.localConns[key]; conn != nil {
		return conn
	}

	port := h.servicePorts[p.Service()]
	if port == 0 {
		loginfo.Println("cannot open connection for invalid service", p.Service())
		return nil
	}

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		loginfo.Println("unable to open local connection on port", port, err)
		return nil
	}

	loginfo.Println("opened new connection to port", port, "for", key)
	h.localConns[key] = conn
	go h.readLocal(key, &p.Header)
	return conn
}

func (h *WsHandler) writeLocal(p *packer.Packer) {
	conn := h.getLocalConn(p)
	if conn == nil {
		h.sendSpecial(&p.Header, "error")
		return
	}

	if p.Service() == "error" || p.Service() == "end" {
		conn.Close()
		return
	}

	if _, err := conn.Write(p.Data.Data()); err != nil {
		h.sendSpecial(&p.Header, "error")
		loginfo.Println("failed to write to local connection", err)
	}
}

func (h *WsHandler) readLocal(key string, header *packer.Header) {
	h.lock.Lock()
	conn := h.localConns[key]
	h.lock.Unlock()

	defer conn.Close()
	defer func() {
		h.lock.Lock()
		delete(h.localConns, key)
		h.lock.Unlock()
	}()
	defer loginfo.Println("finished with client", key)

	buf := make([]byte, 4096)
	for {
		size, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				h.sendSpecial(header, "end")
			} else {
				loginfo.Println("failed to read from local connection for", key, err)
				h.sendSpecial(header, "error")
			}
			return
		}

		p := packer.NewPacker(header)
		p.Data.AppendBytes(buf[:size])
		h.dataChan <- p
	}
}

func (h *WsHandler) closeConnections() {
	h.lock.Lock()
	defer h.lock.Unlock()

	for _, conn := range h.localConns {
		conn.Close()
	}
}
