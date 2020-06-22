package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	telebit "git.coolaj86.com/coolaj86/go-telebitd/mplexer"
	"git.coolaj86.com/coolaj86/go-telebitd/mplexer/admin"

	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
)

var httpsrv *http.Server

// Servers represent actual connections
var Servers *sync.Map

// Table makes sense to be in-memory, but it could be serialized if needed
var Table *sync.Map

func init() {
	Servers = &sync.Map{}
	Table = &sync.Map{}
	r := chi.NewRouter()

	r.HandleFunc("/ws", upgradeWebsocket)

	r.Route("/api", func(r chi.Router) {
		// TODO token needs a globally unique subject

		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				grants, err := authorizer(r)
				if nil != err {
					log.Println("authorization failed", err)
					w.Write(apiNotAuthorizedContent)
					return
				}

				// TODO define Admins in a better way
				if "*" != grants.Subject {
					log.Println("only admins allowed", err)
					w.Write(apiNotAuthorizedContent)
				}

				next.ServeHTTP(w, r)
			})
		})

		r.Get("/subscribers", getSubscribers)
		r.Delete("/subscribers/{subject}", delSubscribers)
		r.NotFound(apiNotFoundHandler)
	})

	adminUI := http.FileServer(admin.AdminFS)
	r.Get("/", adminUI.ServeHTTP)

	httpsrv = &http.Server{
		Handler: r,
	}
}

var apiNotFoundContent = []byte("{ \"error\": \"not found\" }\n")
var apiNotAuthorizedContent = []byte("{ \"error\": \"not authorized\" }\n")

func apiNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(apiNotFoundContent)
}

type SubscriberStatus struct {
	Subject string
	Tunnels int
	Clients int
	// TODO bytes read
}

func getSubscribers(w http.ResponseWriter, r *http.Request) {
	statuses := []*SubscriberStatus{}
	Servers.Range(func(key, value interface{}) bool {
		tunnels := 0
		clients := 0
		//subject := key.(string)
		srvMap := value.(*sync.Map)
		srvMap.Range(func(k, v interface{}) bool {
			tunnels += 1
			srv := v.(*SubscriberConn)
			srv.clients.Range(func(k, v interface{}) bool {
				clients += 1
				return true
			})

			statuses = append(statuses, &SubscriberStatus{
				Subject: k.(string),
				Tunnels: tunnels,
				Clients: clients,
			})
			return true
		})
		return true
	})
	_ = json.NewEncoder(w).Encode(&struct {
		Success     bool                `json:"success"`
		Subscribers []*SubscriberStatus `json:"subscribers"`
	}{
		Success:     true,
		Subscribers: statuses,
	})
}

func delSubscribers(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")

	srvMapX, ok := Servers.Load(subject)
	if !ok {
		// TODO should this be an error?
		_ = json.NewEncoder(w).Encode(&struct {
			Success bool `json:"success"`
		}{
			Success: true,
		})
		return
	}

	srvMap := srvMapX.(*sync.Map)
	srvMap.Range(func(k, v interface{}) bool {
		srv := v.(*SubscriberConn)
		srv.clients.Range(func(k, v interface{}) bool {
			conn := v.(net.Conn)
			_ = conn.Close()
			return true
		})
		srv.wsConn.Close()
		return true
	})
	Servers.Delete(subject)

	_ = json.NewEncoder(w).Encode(&struct {
		Success bool `json:"success"`
	}{
		Success: true,
	})
}

// SubscriberConn represents a tunneled server, its grants, and its clients
type SubscriberConn struct {
	remoteAddr string
	wsConn     *websocket.Conn
	wsTun      net.Conn // *telebit.WebsocketTunnel
	grants     *telebit.Grants
	clients    *sync.Map

	// TODO is this the right codec type?
	multiEncoder *telebit.Encoder
	multiDecoder *telebit.Decoder

	// to fulfill Router interface
}

func (s *SubscriberConn) RouteBytes(src, dst telebit.Addr, payload []byte) {
	id := src.String()
	fmt.Println("Routing some more bytes:")
	fmt.Println("src", id, src)
	fmt.Println("dst", dst)
	clientX, ok := s.clients.Load(id)
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
	s.clients.Store(id, client)

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

	err := s.multiEncoder.Encode(wconn, *src, *dst)
	s.clients.Delete(id)
	return err
}

func upgradeWebsocket(w http.ResponseWriter, r *http.Request) {
	log.Println("websocket opening ", r.RemoteAddr, " ", r.Host)
	w.Header().Set("Content-Type", "application/json")

	if "Upgrade" != r.Header.Get("Connection") && "WebSocket" != r.Header.Get("Upgrade") {
		w.Write(apiNotFoundContent)
		return
	}

	grants, err := authorizer(r)
	if nil != err {
		log.Println("WebSocket authorization failed", err)
		w.Write(apiNotAuthorizedContent)
		return
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed", err)
		return
	}

	wsTun := telebit.NewWebsocketTunnel(conn)
	server := SubscriberConn{
		remoteAddr:   r.RemoteAddr,
		wsConn:       conn,
		wsTun:        wsTun,
		grants:       grants,
		clients:      &sync.Map{},
		multiEncoder: telebit.NewEncoder(context.TODO(), wsTun),
		multiDecoder: telebit.NewDecoder(wsTun),
	}

	go func() {
		// (this listener is also a telebit.Router)
		err := server.multiDecoder.Decode(&server)

		// The tunnel itself must be closed explicitly because
		// there's an encoder with a callback between the websocket
		// and the multiplexer, so it doesn't know to stop listening otherwise
		_ = wsTun.Close()
		fmt.Printf("a subscriber stream is done: %q\n", err)
	}()

	var srvMap *sync.Map
	srvMapX, ok := Servers.Load(grants.Subject)
	if ok {
		srvMap = srvMapX.(*sync.Map)
	} else {
		srvMap = &sync.Map{}
	}
	srvMap.Store(r.RemoteAddr, server)
	Servers.Store(grants.Subject, srvMap)

	// Add this server to the domain name matrix
	for _, name := range grants.Domains {
		var srvMap *sync.Map
		srvMapX, ok := Table.Load(name)
		if ok {
			srvMap = srvMapX.(*sync.Map)
		} else {
			srvMap = &sync.Map{}
		}
		srvMap.Store(r.RemoteAddr, server)
		Table.Store(name, srvMap)
	}
}
