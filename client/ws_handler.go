package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"git.coolaj86.com/coolaj86/go-telebitd/packer"
	"git.coolaj86.com/coolaj86/go-telebitd/sni"
)

var hostRegexp = regexp.MustCompile(`(?im)(?:^|[\r\n])Host: *([^\r\n]+)[\r\n]`)

// WsHandler handles all of reading and writing for the websocket connection to the RVPN server
// and the TCP connections to the local servers.
type WsHandler struct {
	lock       sync.Mutex
	localConns map[string]net.Conn

	servicePorts RouteMap

	ctx       context.Context
	dataChan  chan *packer.Packer
	tlsConfig *tls.Config
}

// NewWsHandler creates a new handler ready to be given a websocket connection. The services
// argument specifies what port each service type should be directed to on the local interface.
func NewWsHandler(services RouteMap, tlsConfig *tls.Config) *WsHandler {
	return &WsHandler{
		servicePorts: services,
		localConns:   make(map[string]net.Conn),
		tlsConfig:    tlsConfig,
	}
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

func (h *WsHandler) writeRemote(wsconn *websocket.Conn) {
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
			wsconn.WriteControl(websocket.CloseMessage, message, deadline)
			wsconn.Close()
			return

		case p := <-h.dataChan:
			packed := p.PackV1()
			wsconn.WriteMessage(websocket.BinaryMessage, packed.Bytes())
		}
	}
}

func (h *WsHandler) sendPackedMessage(header *packer.Header, data []byte, service string) {
	p := packer.NewPacker(header)
	if len(data) > 0 {
		p.Data.AppendBytes(data)
	}
	if service != "" {
		p.SetService(service)
	}

	// Avoid blocking on the data channel if the websocket closes or is already closed
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

	service := strings.ToLower(p.Service())
	portList := h.servicePorts[service]
	if portList == nil {
		loginfo.Println("cannot open connection for invalid service", service)
		return nil
	}

	var hostname string
	//var terminate bool
	if service == "http" {
		if match := hostRegexp.FindSubmatch(p.Data.Data()); match != nil {
			hostname = strings.Split(string(match[1]), ":")[0]
			// TODO remove Hostname
		}
	} else if service == "https" {
		hostname, _ = sni.GetHostname(p.Data.Data())
		//terminate = true
	} else {
		hostname = "*"
	}
	if hostname == "" {
		loginfo.Println("missing servername for", service, key)
		return nil
	}
	hostname = strings.ToLower(hostname)

	term := portList[hostname]
	fmt.Println("route to", hostname, term)
	if term == nil {
		portList[hostname] = portList["*"]
		term = portList[hostname]
	}
	if term.Port == 0 {
		portList[hostname] = portList["*"]
	}
	if term.Port == 0 {
		loginfo.Println("unable to determine local port for", service, hostname)
		return nil
	}

	// TODO allow jumping
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", term.Port))
	if err != nil {
		loginfo.Println("unable to open local connection on port", term.Port, err)
		return nil
	}

	rconn := conn
	/*
		if terminate {
			rconn = tls.Server(conn, h.tlsConfig)
			//rconn = tls.Client(conn, h.tlsConfig)
		}
	*/
	h.localConns[key] = rconn
	loginfo.Printf("new client %q for %s:%d (%d clients)\n", key, hostname, term.Port, len(h.localConns))
	go h.readLocal(key, &p.Header)
	return conn
}

func (h *WsHandler) writeLocal(p *packer.Packer) {
	conn := h.getLocalConn(p)
	if conn == nil {
		h.sendPackedMessage(&p.Header, nil, "error")
		return
	}

	if p.Service() == "error" || p.Service() == "end" {
		// TODO XXX where's the opposite of this?
		conn.Close()
		return
	}

	if _, err := conn.Write(p.Data.Data()); err != nil {
		h.sendPackedMessage(&p.Header, nil, "error")
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
		loginfo.Printf("closing client %q: (%d clients)\n", key, len(h.localConns))
		h.lock.Unlock()
	}()

	buf := make([]byte, 4096)
	for {
		size, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "use of closed network connection") {
				h.sendPackedMessage(header, nil, "end")
			} else {
				loginfo.Println("failed to read from local connection for", key, err)
				h.sendPackedMessage(header, nil, "error")
			}
			return
		}

		h.sendPackedMessage(header, buf[:size], "")
	}
}

func (h *WsHandler) closeConnections() {
	h.lock.Lock()
	defer h.lock.Unlock()

	for _, conn := range h.localConns {
		conn.Close()
	}
}
