package relay

import (
	"context"
	"crypto/tls"
	"log"
	"net"
	"net/http"

	"git.coolaj86.com/coolaj86/go-telebitd/relay/admin"
	"git.coolaj86.com/coolaj86/go-telebitd/relay/api"
	"git.coolaj86.com/coolaj86/go-telebitd/relay/mplexy"
	"git.coolaj86.com/coolaj86/go-telebitd/relay/tunnel"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Relay is probably a layer that doesn't need to exist
type Relay struct {
	ctx    context.Context
	status *api.Status
	mx     *mplexy.MPlexy
	table  *api.Table
}

// New initializes and returns a relay service
func New(ctx context.Context, tlsConfig *tls.Config, authz mplexy.Authorizer, status *api.Status, table *api.Table) *Relay {
	// TODO do we need this already setup here? or is it just for logging?
	status.ConnectionTracking = api.NewTracking()
	status.ConnectionTable = table
	authAdmin := authz
	r := &Relay{
		ctx:    ctx,
		status: status,
		table:  table,
		mx:     mplexy.New(ctx, tlsConfig, authAdmin, authz, status), // TODO Accept
	}
	return r
}

// ListenAndServe sets up all of the tcp, http, https, and tunnel servers
func (r *Relay) ListenAndServe(port int) error {

	serverStatus := r.status

	// Setup for GenericListenServe.
	// - establish context for the generic listener
	// - startup listener
	// - accept with peek buffer.
	// - peek at the 1st 30 bytes.
	// - check for tls
	// - if tls, establish, protocol peek buffer, else decrypted
	// - match protocol

	go r.status.ConnectionTracking.Run(r.ctx)
	go serverStatus.ConnectionTable.Run(r.ctx)

	//serverStatus.GenericListeners = genericListeners

	// blocks until it can listen, which it can't until started
	go r.mx.MultiListenAndServe(port)

	// funnel target devices into WebSocket pool
	tunnelListener := tunnel.NewListener()
	r.mx.AcceptTargetServer = func(conn net.Conn) {
		tunnelListener.Feed(conn)
	}
	go listenAndServeTargets(r.mx, tunnelListener)

	// funnel admin clients to API
	adminListener := tunnel.NewListener()
	r.mx.AcceptAdminClient = func(conn net.Conn) {
		adminListener.Feed(conn)
	}
	go admin.ListenAndServe(r.mx, adminListener)

	return r.mx.Run()
}

func listenAndServeTargets(mx *mplexy.MPlexy, listener net.Listener) error {
	serverStatus := mx.Status

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("HandleFunc /")
		switch url := r.URL.Path; url {
		case "/":
			log.Println("websocket opening ", r.RemoteAddr, " ", r.Host)

			authz, err := mx.AuthorizeTarget(r)
			if nil != err {
				log.Println("WebSocket authorization failed", err)
				return
			}
			var upgrader = websocket.Upgrader{
				ReadBufferSize:  65535,
				WriteBufferSize: 65535,
			}

			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				log.Println("WebSocket upgrade failed", err)
				return
			}

			log.Println("before connection table")

			serverName := authz.Domains[0]

			newRegistration := api.NewRegistration(conn, r.RemoteAddr, authz.Domains, serverStatus.ConnectionTracking, serverName)
			serverStatus.WSSConnectionRegister(newRegistration)

			if ok := <-newRegistration.CommCh(); !ok {
				log.Println("connection registration failed ", newRegistration)
				return
			}

			log.Println("connection registration accepted ", newRegistration)
		}
	})

	// TODO setup for http/2
	s := &http.Server{
		Addr:    ":80",
		Handler: router,
	}
	return s.Serve(listener)
}
