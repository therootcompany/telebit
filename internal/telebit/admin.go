package telebit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"git.rootprojects.org/root/telebit/assets/admin"
	"git.rootprojects.org/root/telebit/internal/dbg"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
)

var authorizer Authorizer

// RouteAdmin sets up the API, including the Mgmt proxy and ACME relay
func RouteAdmin(authURL string, r chi.Router) {
	var apiPingContent = []byte("{ \"success\": true, \"error\": \"\" }\n")

	authorizer = NewAuthorizer(authURL)

	r.Route("/", func(r chi.Router) {
		r.Use(middleware.Logger)
		//r.Use(middleware.Timeout(120 * time.Second))
		r.Use(middleware.Recoverer)

		/*
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					next.ServeHTTP(w, r)
				})
			})
		*/

		r.Mount("/ws", http.HandlerFunc(upgradeWebsocket))

		r.HandleFunc("/api/ping", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if dbg.Debug {
				fmt.Fprintf(os.Stderr, "[debug] hit /api/ping and replying\n")
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(apiPingContent)
		}))

		parsedAuthURL, err := url.Parse(authURL)
		if nil != err {
			panic(err)
		}

		proxyHandler := httputil.NewSingleHostReverseProxy(parsedAuthURL)
		proxyHandleFunc := func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api")
			proxyHandler.ServeHTTP(w, r)
		}

		// Proxy mgmt server Registration & Authentication
		r.Get("/api/inspect", proxyHandleFunc)
		r.Post("/api/register-device", proxyHandleFunc)
		r.Post("/api/register-device/*", proxyHandleFunc)

		// Proxy mgmt server ACME DNS 01 Challenges
		r.Get("/api/dns/*", proxyHandleFunc)
		r.Post("/api/dns/*", proxyHandleFunc)
		r.Delete("/api/dns/*", proxyHandleFunc)
		r.Get("/api/http/*", proxyHandleFunc)
		r.Post("/api/http/*", proxyHandleFunc)
		r.Delete("/api/http/*", proxyHandleFunc)
		r.Get("/api/acme-relay/*", proxyHandleFunc)
		r.Post("/api/acme-relay/*", proxyHandleFunc)
		r.Delete("/api/acme-relay/*", proxyHandleFunc)

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

			r.Get("/subscribers", getAllSubscribers)
			r.Get("/subscribers/{subject}", getSubscribers)
			r.Delete("/subscribers/{subject}", delSubscribers)
			r.NotFound(apiNotFoundHandler)
		})

		adminUI := http.FileServer(admin.AdminFS)
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			//rctx := chi.RouteContext(r.Context())
			//pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
			//fs := http.StripPrefix(pathPrefix, http.FileServer(root))
			fmt.Println("Request Path:", r.URL.Path)
			adminUI.ServeHTTP(w, r)
		})
	})
}

var apiNotFoundContent = []byte("{ \"error\": \"not found\" }\n")
var apiNotAuthorizedContent = []byte("{ \"error\": \"not authorized\" }\n")

func apiNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(apiNotFoundContent)
}

type SubscriberStatus struct {
	Since   *time.Time `json:"since,omitempty"`
	Subject string     `json:"sub"`
	Sockets []string   `json:"sockets"`
	Clients int        `json:"clients"`
	// TODO bytes read
}

func getAllSubscribers(w http.ResponseWriter, r *http.Request) {
	statuses := []*SubscriberStatus{}
	Servers.Range(func(key, value interface{}) bool {
		srvMap := value.(*sync.Map)
		status := getSubscribersHelper(srvMap)
		statuses = append(statuses, status)
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

func getSubscribers(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")
	statuses := &struct {
		Success     bool                `json:"success"`
		Subscribers []*SubscriberStatus `json:"subscribers"`
	}{
		Success:     true,
		Subscribers: []*SubscriberStatus{},
	}

	var srvMap *sync.Map
	srvMapX, ok := Servers.Load(subject)
	if ok {
		srvMap = srvMapX.(*sync.Map)
		statuses.Subscribers = append(statuses.Subscribers, getSubscribersHelper(srvMap))
	}

	_ = json.NewEncoder(w).Encode(statuses)
}

func getSubscribersHelper(srvMap *sync.Map) *SubscriberStatus {
	status := &SubscriberStatus{
		Since:   nil,
		Subject: "",
		Sockets: []string{},
		Clients: 0,
	}

	srvMap.Range(func(k, v interface{}) bool {
		status.Sockets = append(status.Sockets, k.(string))
		srv := v.(*SubscriberConn)
		if nil == status.Since || srv.Since.Sub(*status.Since) < 0 {
			copied := srv.Since.Truncate(time.Second)
			status.Since = &copied
		}
		status.Subject = srv.Grants.Subject
		srv.Clients.Range(func(k, v interface{}) bool {
			status.Clients++
			return true
		})

		return true
	})

	return status
}

func delSubscribers(w http.ResponseWriter, r *http.Request) {
	subject := chi.URLParam(r, "subject")

	ok := Remove(subject)
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

	wsTun := NewWebsocketTunnel(conn)
	fmt.Printf("New Authenticated WebSocket Remote Server\n")
	fmt.Printf("\thttp.req.RemoteAddr: %+v\n", r.RemoteAddr)
	fmt.Printf("\tconn.RemoteAddr(): %+v\n", conn.RemoteAddr())
	fmt.Printf("\tconn.LocalAddr(): %+v\n", conn.LocalAddr())
	fmt.Printf("\tgrants: %v\n", grants)

	// The remote address of the server is useful for identification.
	// The local address of the server (port to which it connected) is not very meaningful.
	// Rather the client's local address (the specific relay server) would be more useful.
	ctxEncoder, cancelEncoder := context.WithCancel(context.Background())
	now := time.Now()
	server := &SubscriberConn{
		Since:        &now,
		RemoteAddr:   r.RemoteAddr,
		WSConn:       conn,
		WSTun:        wsTun,
		Grants:       grants,
		Clients:      &sync.Map{},
		MultiEncoder: NewEncoder(ctxEncoder, wsTun),
		MultiDecoder: NewDecoder(wsTun),
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
		RemoveServer(server)
	}()

	Add(server)
}
