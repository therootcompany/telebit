package server

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"git.coolaj86.com/coolaj86/go-telebitd/relay/api"
)

//ListenerRegistrationStatus - post registration status
type ListenerRegistrationStatus int

// Authz represents grants or privileges of a client
// clientID
// domains that may be forwarded
// # of domains that may be forwarded
// ports that may be forwarded (i.e. allow special ports < 1024, exclude 443, 25, etc)
// # of ports that may be forwarded
// # of concurrent conections
// # bandwith rate (i.e. 5 mbps)
// # bandwith cap per time period (i.e. 100 MB / hour)
// # throttled rate (i.e. 0 (kill), or 1 mbps)
type Authz struct {
	Domains []string
}

// Authorizer is called when a new client connects and we need to know something about it
type Authorizer func(*http.Request) (*Authz, error)

const (
	listenerAdded ListenerRegistrationStatus = iota
	listenerExists
	listenerFault
)

//ListenerRegistration -- A connection registration structure used to bring up a connection
//connection table will then handle additing and sdtarting up the various readers
//else error.
type ListenerRegistration struct {
	// The websocket connection.
	listener *net.Listener

	// The listener port
	port int

	// The status
	status ListenerRegistrationStatus

	// The error
	err error

	// communications channel between go routines
	commCh chan *ListenerRegistration
}

//NewListenerRegistration -- Constructor
func NewListenerRegistration(port int) (p *ListenerRegistration) {
	p = new(ListenerRegistration)
	p.port = port
	p.commCh = make(chan *ListenerRegistration)
	return p
}

// MPlexy -
type MPlexy struct {
	listeners          map[*net.Listener]int
	ctx                context.Context
	connnectionTable   *api.Table
	connectionTracking *api.Tracking
	AuthorizeTarget    Authorizer
	AuthorizeAdmin     Authorizer
	tlsConfig          *tls.Config
	register           chan *ListenerRegistration
	wssHostName        string
	adminHostName      string
	cancelCheck        int
	lbDefaultMethod    string
	Status             *api.Status
	AcceptTargetServer func(net.Conn)
	AcceptAdminClient  func(net.Conn)
}

// New creates tcp (and https and wss?) listeners
func New(
	ctx context.Context,
	tlsConfig *tls.Config,
	authAdmin Authorizer,
	authz Authorizer,
	serverStatus *api.Status,
) (mx *MPlexy) {
	mx = &MPlexy{
		listeners:          make(map[*net.Listener]int),
		ctx:                ctx,
		connnectionTable:   serverStatus.ConnectionTable,
		connectionTracking: serverStatus.ConnectionTracking,
		AuthorizeTarget:    authz,
		AuthorizeAdmin:     authz,
		tlsConfig:          tlsConfig,
		register:           make(chan *ListenerRegistration),
		wssHostName:        serverStatus.WssDomain,
		adminHostName:      serverStatus.AdminDomain,
		cancelCheck:        serverStatus.DeadTime.Cancelcheck,
		lbDefaultMethod:    serverStatus.LoadbalanceDefaultMethod,
		Status:             serverStatus,
	}
	return mx
}

//Run -- Execute
// - execute the GenericLister
// - pass initial port, we'll announce that
func (mx *MPlexy) Run() error {
	loginfo.Println("ConnectionTable starting")

	loginfo.Println(mx.connectionTracking)

	ctx := mx.ctx

	// For just this bit
	ctx = context.WithValue(ctx, ctxConnectionTrack, mx.connectionTracking)

	// For all Listeners
	ctx = context.WithValue(ctx, ctxConfig, mx.tlsConfig)
	ctx = context.WithValue(ctx, ctxListenerRegistration, mx.register)
	ctx = context.WithValue(ctx, ctxWssHostName, mx.wssHostName)
	ctx = context.WithValue(ctx, ctxAdminHostName, mx.adminHostName)
	ctx = context.WithValue(ctx, ctxCancelCheck, mx.cancelCheck)
	ctx = context.WithValue(ctx, ctxLoadbalanceDefaultMethod, mx.lbDefaultMethod)
	ctx = context.WithValue(ctx, ctxServerStatus, mx.Status)

	for {
		select {

		case <-ctx.Done():
			loginfo.Println("Cancel signal hit")
			return nil

		case registration := <-mx.register:
			loginfo.Println("register fired", registration.port)

			// check to see if port is already running
			for listener := range mx.listeners {
				if mx.listeners[listener] == registration.port {
					loginfo.Println("listener already running", registration.port)
					registration.status = listenerExists
					registration.commCh <- registration
				}
			}
			loginfo.Println("listener starting up ", registration.port)
			loginfo.Println(ctx.Value(ctxConnectionTrack).(*api.Tracking))
			go mx.multiListenAndServe(ctx, registration)

			status := <-registration.commCh
			if status.status == listenerAdded {
				mx.listeners[status.listener] = status.port
			} else if status.status == listenerFault {
				loginfo.Println("Unable to create a new listerer", registration.port)
			}
		}
	}

	return nil
}

func (mx *MPlexy) Start() {
	go mx.Run()
}

// MultiListenAndServe starts another listener (to the same application) on a new port
func (mx *MPlexy) MultiListenAndServe(port int) {
	// TODO how to associate a listening device with a given plain port
	mx.register <- NewListenerRegistration(port)
}
