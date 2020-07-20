package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	telebit "git.rootprojects.org/root/telebit"
	"git.rootprojects.org/root/telebit/admin"
	"git.rootprojects.org/root/telebit/dbg"
	"git.rootprojects.org/root/telebit/table"

	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
)

var httpsrv *http.Server

func init() {
	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r)
		})
	})

	r.Mount("/ws", http.HandlerFunc(upgradeWebsocket))

	r.HandleFunc("/api/ping", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if dbg.Debug {
			fmt.Fprintf(os.Stderr, "[debug] hit /api/ping and replying\n")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(apiPingContent)
	}))

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
					return
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

var apiPingContent = []byte("{ \"success\": true, \"error\": \"\" }\n")
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
	table.Servers.Range(func(key, value interface{}) bool {
		tunnels := 0
		clients := 0
		//subject := key.(string)
		srvMap := value.(*sync.Map)
		srvMap.Range(func(k, v interface{}) bool {
			tunnels += 1
			srv := v.(*table.SubscriberConn)
			srv.Clients.Range(func(k, v interface{}) bool {
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

	ok := table.Remove(subject)
	if !ok {
		// TODO should this be an error?
		_ = json.NewEncoder(w).Encode(&struct {
			Success bool `json:"success"`
		}{
			Success: true,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(&struct {
		Success bool `json:"success"`
	}{
		Success: true,
	})
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
	fmt.Printf("New Authenticated WebSocket Remote Server\n")
	fmt.Printf("\thttp.req.RemoteAddr: %+v\n", r.RemoteAddr)
	fmt.Printf("\tconn.RemoteAddr(): %+v\n", conn.RemoteAddr())
	fmt.Printf("\tconn.LocalAddr(): %+v\n", conn.LocalAddr())
	fmt.Printf("\tgrants: %v\n", grants)

	// The remote address of the server is useful for identification.
	// The local address of the server (port to which it connected) is not very meaningful.
	// Rather the client's local address (the specific relay server) would be more useful.
	ctxEncoder, cancelEncoder := context.WithCancel(context.Background())
	server := &table.SubscriberConn{
		RemoteAddr:   r.RemoteAddr,
		WSConn:       conn,
		WSTun:        wsTun,
		Grants:       grants,
		Clients:      &sync.Map{},
		MultiEncoder: telebit.NewEncoder(ctxEncoder, wsTun),
		MultiDecoder: telebit.NewDecoder(wsTun),
	}
	// TODO should this happen at NewEncoder()?
	// (or is it even necessary anymore?)
	_ = server.MultiEncoder.Start()

	go func() {
		// (this listener is also a telebit.Router)
		err := server.MultiDecoder.Decode(server)
		cancelEncoder() // TODO why don't failed writes solve this?
		//_ = server.MultiEncoder.Close()

		// The tunnel itself must be closed explicitly because
		// there's an encoder with a callback between the websocket
		// and the multiplexer, so it doesn't know to stop listening otherwise
		_ = wsTun.Close()
		// TODO close all clients
		fmt.Printf("a subscriber stream is done: %q\n", err)
		// TODO check what happens when we leave a junk connection
		//fmt.Println("[debug] [warn] removing server turned off")
		table.RemoveServer(server)
	}()

	table.Add(server)
}
