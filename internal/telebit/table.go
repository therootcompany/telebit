package telebit

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"git.rootprojects.org/root/telebit/internal/dbg"

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
	for _, domainname := range server.Grants.Domains {
		var srvMap *sync.Map
		srvMapX, ok := Table.Load(domainname)
		if ok {
			srvMap = srvMapX.(*sync.Map)
		} else {
			srvMap = &sync.Map{}
		}
		srvMap.Store(server.RemoteAddr, server)
		Table.Store(domainname, srvMap)
	}
}

func RemoveServer(server *SubscriberConn) bool {
	// TODO remove by RemoteAddr
	//return false
	fmt.Fprintf(
		os.Stderr,
		"[warn] RemoveServer() still calls Remove(subject) instead of removing by RemoteAddr\n",
	)
	return Remove(server.Grants.Subject)
}

func Remove(subject string) bool {
	srvMapX, ok := Servers.Load(subject)
	fmt.Printf("Remove(%s): exists? %t\n", subject, ok)

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
		for _, domainname := range srv.Grants.Domains {
			srvMapX, ok := Table.Load(domainname)
			if !ok {
				continue
			}
			srvMap = srvMapX.(*sync.Map)
			srvMap.Delete(srv.RemoteAddr)
			n := 0
			srvMap.Range(func(k, v interface{}) bool {
				n++
				return true
			})
			if 0 == n {
				// TODO comment out to handle the bad case of 0 servers / empty map
				Table.Delete(domainname)
			}
		}
		return true
	})
	Servers.Delete(subject)

	return true
}

// SubscriberConn represents a tunneled server, its grants, and its clients
type SubscriberConn struct {
	Since      *time.Time
	RemoteAddr string
	WSConn     *websocket.Conn
	WSTun      net.Conn // *WebsocketTunnel
	Grants     *Grants
	Clients    *sync.Map

	// TODO is this the right codec type?
	MultiEncoder *Encoder
	MultiDecoder *Decoder

	// to fulfill Router interface
}

func (s *SubscriberConn) RouteBytes(src, dst Addr, payload []byte) {
	id := fmt.Sprintf("%s:%d", src.Hostname(), src.Port())
	if dbg.Debug {
		fmt.Fprintf(
			os.Stderr,
			"[debug] Routing some more bytes: %s\n",
			dbg.Trunc(payload, len(payload)),
		)
		fmt.Printf("\tid %s\nsrc %+v\n", id, src)
		fmt.Printf("\tdst %s %+v\n", dst.Scheme(), dst)
	}
	clientX, ok := s.Clients.Load(id)
	if !ok {
		// TODO send back closed client error
		fmt.Printf("RouteBytes({ %s }, %v, ...) [debug] no client found for %s\n", id, dst)
		return
	}

	client, _ := clientX.(net.Conn)
	if "end" == dst.Scheme() {
		fmt.Printf("RouteBytes: { %s }.Close(): %v\n", id, dst)
		_ = client.Close()
		return
	}

	for {
		n, err := client.Write(payload)
		if dbg.Debug {
			fmt.Fprintf(os.Stderr, "[debug] table Write %s\n", dbg.Trunc(payload, len(payload)))
		}
		if nil == err || io.EOF == err {
			break
		}
		if n > 0 && io.ErrShortWrite == err {
			payload = payload[n:]
			continue
		}
		break
		// TODO send back closed client error
		//return err
	}
}

func (s *SubscriberConn) Serve(client net.Conn) error {
	var wconn *ConnWrap
	switch conn := client.(type) {
	case *ConnWrap:
		wconn = conn
	default:
		// this probably isn't strictly necessary
		panic("*SubscriberConn.Serve is special in that it must receive &ConnWrap{ Conn: conn }")
	}

	id := client.RemoteAddr().String()
	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] NEW ID (ip:port) %s\n", id)
	}
	s.Clients.Store(id, client)

	//fmt.Fprintf(os.Stderr, "[debug] immediately cancel client to simplify testing / debugging\n")
	//_ = client.Close()

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

	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] srcParts %v\n", srcParts)
		fmt.Fprintf(os.Stderr, "[debug] dstParts %v\n", dstParts)
	}

	servername := wconn.Servername()

	termination := Unknown
	scheme := None
	if "" != servername {
		dstAddr = servername
		//scheme = TLS
		scheme = HTTPS
	}
	if 80 == dstPort {
		scheme = HTTPS
	} else if 443 == dstPort {
		// TODO dstAddr = wconn.Servername()
		scheme = HTTP
	}

	src := NewAddr(
		scheme,
		termination,
		srcAddr,
		srcPort,
	)
	dst := NewAddr(
		scheme,
		termination,
		dstAddr,
		dstPort,
	)

	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] NewAddr src %+v\n", src)
		fmt.Fprintf(os.Stderr, "[debug] NewAddr dst %+v\n", dst)
	}

	err := s.MultiEncoder.Encode(wconn, *src, *dst)
	_ = wconn.Close()

	if dbg.Debug {
		fmt.Fprintf(os.Stderr, "[debug] Encoder Complete %+v %+v\n", id, err)
	}
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
			myLoad += 1
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
