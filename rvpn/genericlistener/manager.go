package server

import (
	"context"
	"crypto/tls"
	"net"
)

//ListenerRegistrationStatus - post registration status
type ListenerRegistrationStatus int

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
	return
}

//servers -
type servers struct {
	listeners          map[*net.Listener]int
	ctx                context.Context
	connnectionTable   *Table
	connectionTracking *Tracking
	secretKey          string
	certbundle         tls.Certificate
	register           chan *ListenerRegistration
	servers   *servers
	wssHostName        string
	adminHostName      string
	cancelCheck        int
	lbDefaultMethod    string
	serverStatus       *Status
}

//NewGenerListeners --
func NewGenerListeners(ctx context.Context, secretKey string, certbundle tls.Certificate, serverStatus *Status) (p *servers) {
	p = new(servers)
	p.listeners = make(map[*net.Listener]int)
	p.ctx = ctx
	p.connnectionTable = serverStatus.ConnectionTable
	p.connectionTracking = serverStatus.ConnectionTracking
	p.secretKey = secretKey
	p.certbundle = certbundle
	p.register = make(chan *ListenerRegistration)
	p.wssHostName = serverStatus.WssDomain
	p.adminHostName = serverStatus.AdminDomain
	p.cancelCheck = serverStatus.DeadTime.cancelcheck
	p.lbDefaultMethod = serverStatus.LoadbalanceDefaultMethod
	p.serverStatus = serverStatus
	return
}

//Run -- Execute
// - execute the GenericLister
// - pass initial port, we'll announce that
func (gl *servers) Run(ctx context.Context, initialPort int) {
	loginfo.Println("ConnectionTable starting")

	config := &tls.Config{Certificates: []tls.Certificate{gl.certbundle}}

	ctx = context.WithValue(ctx, ctxSecretKey, gl.secretKey)

	loginfo.Println(gl.connectionTracking)

	ctx = context.WithValue(ctx, ctxConnectionTrack, gl.connectionTracking)
	ctx = context.WithValue(ctx, ctxConfig, config)
	ctx = context.WithValue(ctx, ctxListenerRegistration, gl.register)
	ctx = context.WithValue(ctx, ctxWssHostName, gl.wssHostName)
	ctx = context.WithValue(ctx, ctxAdminHostName, gl.adminHostName)
	ctx = context.WithValue(ctx, ctxCancelCheck, gl.cancelCheck)
	ctx = context.WithValue(ctx, ctxLoadbalanceDefaultMethod, gl.lbDefaultMethod)
	ctx = context.WithValue(ctx, ctxServerStatus, gl.serverStatus)

	go func(ctx context.Context) {
		for {
			select {

			case <-ctx.Done():
				loginfo.Println("Cancel signal hit")
				return

			case registration := <-gl.register:
				loginfo.Println("register fired", registration.port)

				// check to see if port is already running
				for listener := range gl.listeners {
					if gl.listeners[listener] == registration.port {
						loginfo.Println("listener already running", registration.port)
						registration.status = listenerExists
						registration.commCh <- registration
					}
				}
				loginfo.Println("listener starting up ", registration.port)
				loginfo.Println(ctx.Value(ctxConnectionTrack).(*Tracking))
				go GenericListenAndServe(ctx, registration)

				status := <-registration.commCh
				if status.status == listenerAdded {
					gl.listeners[status.listener] = status.port
				} else if status.status == listenerFault {
					loginfo.Println("Unable to create a new listerer", registration.port)
				}
			}
		}

	}(ctx)

	newListener := NewListenerRegistration(initialPort)
	gl.register <- newListener
}
