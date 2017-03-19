package genericlistener

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

//GenericListeners -
type GenericListeners struct {
	listeners          map[*net.Listener]int
	ctx                context.Context
	connnectionTable   *Table
	connectionTracking *Tracking
	secretKey          string
	certbundle         tls.Certificate
	register           chan *ListenerRegistration
	genericListeners   *GenericListeners
	wssHostName        string
	adminHostName      string
	cancelCheck        int
	lbDefaultMethod    string
}

//NewGenerListeners --
func NewGenerListeners(ctx context.Context, connectionTable *Table, connectionTrack *Tracking, secretKey string, certbundle tls.Certificate,
	wssHostName string, adminHostName string, cancelCheck int, lbDefaultMethod string) (p *GenericListeners) {
	p = new(GenericListeners)
	p.listeners = make(map[*net.Listener]int)
	p.ctx = ctx
	p.connnectionTable = connectionTable
	p.connectionTracking = connectionTrack
	p.secretKey = secretKey
	p.certbundle = certbundle
	p.register = make(chan *ListenerRegistration)
	p.wssHostName = wssHostName
	p.adminHostName = adminHostName
	p.cancelCheck = cancelCheck
	p.lbDefaultMethod = lbDefaultMethod
	return
}

//Run -- Execute
// - execute the GenericLister
// - pass initial port, we'll announce that
func (gl *GenericListeners) Run(ctx context.Context, initialPort int) {
	loginfo.Println("ConnectionTable starting")

	config := &tls.Config{Certificates: []tls.Certificate{gl.certbundle}}

	ctx = context.WithValue(ctx, ctxSecretKey, gl.secretKey)
	ctx = context.WithValue(ctx, ctxConnectionTable, gl.connnectionTable)

	loginfo.Println(gl.connectionTracking)

	ctx = context.WithValue(ctx, ctxConnectionTrack, gl.connectionTracking)
	ctx = context.WithValue(ctx, ctxConfig, config)
	ctx = context.WithValue(ctx, ctxListenerRegistration, gl.register)
	ctx = context.WithValue(ctx, ctxWssHostName, gl.wssHostName)
	ctx = context.WithValue(ctx, ctxAdminHostName, gl.adminHostName)
	ctx = context.WithValue(ctx, ctxCancelCheck, gl.cancelCheck)
	ctx = context.WithValue(ctx, ctxLoadbalanceDefaultMethod, gl.lbDefaultMethod)

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
