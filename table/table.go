package table

import (
	"fmt"
	"net"
	"sync"

	"io"
	"strconv"
	"strings"

	telebit "git.coolaj86.com/coolaj86/go-telebitd/mplexer"
	"github.com/gorilla/websocket"
)

// Servers represent actual connections
var Servers *sync.Map

// Table makes sense to be in-memory, but it could be serialized if needed
var Table *sync.Map

func init() {
	Servers = &sync.Map{}
	Table = &sync.Map{}
}

func Add(server *SubscriberConn) {
	var srvMap *sync.Map
	srvMapX, ok := Servers.Load(server.Grants.Subject)
	if ok {
		srvMap = srvMapX.(*sync.Map)
	} else {
		srvMap = &sync.Map{}
	}
	srvMap.Store(server.RemoteAddr, server)
	Servers.Store(server.Grants.Subject, srvMap)

	// Add this server to the domain name matrix
	for _, name := range server.Grants.Domains {
		var srvMap *sync.Map
		srvMapX, ok := Table.Load(name)
		if ok {
			srvMap = srvMapX.(*sync.Map)
		} else {
			srvMap = &sync.Map{}
		}
		srvMap.Store(server.RemoteAddr, server)
		Table.Store(name, srvMap)
	}
}

func Remove(subject string) bool {
	srvMapX, ok := Servers.Load(subject)
	if !ok {
		return false
	}

	srvMap := srvMapX.(*sync.Map)
	srvMap.Range(func(k, v interface{}) bool {
		srv := v.(*SubscriberConn)
		srv.Clients.Range(func(k, v interface{}) bool {
			conn := v.(net.Conn)
			_ = conn.Close()
			return true
		})
		srv.WSConn.Close()
		return true
	})
	Servers.Delete(subject)

	return true
}

// SubscriberConn represents a tunneled server, its grants, and its clients
type SubscriberConn struct {
	RemoteAddr string
	WSConn     *websocket.Conn
	WSTun      net.Conn // *telebit.WebsocketTunnel
	Grants     *telebit.Grants
	Clients    *sync.Map

	// TODO is this the right codec type?
	MultiEncoder *telebit.Encoder
	MultiDecoder *telebit.Decoder

	// to fulfill Router interface
}

func (s *SubscriberConn) RouteBytes(src, dst telebit.Addr, payload []byte) {
	id := src.String()
	fmt.Println("Routing some more bytes:")
	fmt.Println("src", id, src)
	fmt.Println("dst", dst)
	clientX, ok := s.Clients.Load(id)
	if !ok {
		// TODO send back closed client error
		return
	}

	client, _ := clientX.(net.Conn)
	for {
		n, err := client.Write(payload)
		if nil != err {
			if n > 0 && io.ErrShortWrite == err {
				payload = payload[n:]
				continue
			}
			// TODO send back closed client error
			break
		}
	}
}

func (s *SubscriberConn) Serve(client net.Conn) error {
	var wconn *telebit.ConnWrap
	switch conn := client.(type) {
	case *telebit.ConnWrap:
		wconn = conn
	default:
		// this probably isn't strictly necessary
		panic("*SubscriberConn.Serve is special in that it must receive &ConnWrap{ Conn: conn }")
	}

	id := client.RemoteAddr().String()
	s.Clients.Store(id, client)

	fmt.Println("[debug] cancel all the clients")
	_ = client.Close()

	// TODO
	// - Encode each client to the tunnel
	// - Find the right client for decoded messages

	// TODO which order is remote / local?
	srcParts := strings.Split(client.RemoteAddr().String(), ":")
	srcAddr := srcParts[0]
	srcPort, _ := strconv.Atoi(srcParts[1])

	dstParts := strings.Split(client.LocalAddr().String(), ":")
	dstAddr := dstParts[0]
	dstPort, _ := strconv.Atoi(dstParts[1])

	termination := telebit.Unknown
	scheme := telebit.None
	if 80 == dstPort {
		// TODO dstAddr = wconn.Servername()
		scheme = telebit.HTTP
	} else if 443 == dstPort {
		dstAddr = wconn.Servername()
		scheme = telebit.HTTPS
	}

	src := telebit.NewAddr(
		scheme,
		termination,
		srcAddr,
		srcPort,
	)
	dst := telebit.NewAddr(
		scheme,
		termination,
		dstAddr,
		dstPort,
	)

	err := s.MultiEncoder.Encode(wconn, *src, *dst)
	s.Clients.Delete(id)
	return err
}

func GetServer(servername string) (*SubscriberConn, bool) {
	var srv *SubscriberConn
	load := -1
	// TODO match *.whatever.com
	srvMapX, ok := Table.Load(servername)
	if !ok {
		return nil, false
	}

	srvMap := srvMapX.(*sync.Map)
	srvMap.Range(func(k, v interface{}) bool {
		myLoad := 0
		mySrv := v.(*SubscriberConn)
		mySrv.Clients.Range(func(k, v interface{}) bool {
			load += 1
			return true
		})
		// pick the least loaded server
		if -1 == load || myLoad < load {
			load = myLoad
			srv = mySrv
		}
		return true
	})

	return srv, true
}