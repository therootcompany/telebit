package relay

import (
	"context"
	"crypto/tls"

	"git.coolaj86.com/coolaj86/go-telebitd/server"
)

type Relay struct {
	ctx    context.Context
	status *server.Status
	mx     *server.MPlexy
	table  *server.Table
}

func New(ctx context.Context, tlsConfig *tls.Config, authz server.Authorizer, status *server.Status, table *server.Table) *Relay {
	return &Relay{
		ctx:    ctx,
		status: status,
		table:  table,
		mx:     server.New(ctx, tlsConfig, authz, status),
	}
}

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

	connectionTracking := server.NewTracking()
	serverStatus.ConnectionTracking = connectionTracking
	go connectionTracking.Run(r.ctx)

	serverStatus.ConnectionTable = r.table
	go serverStatus.ConnectionTable.Run(r.ctx)

	//serverStatus.GenericListeners = genericListeners

	// blocks until it can listen, which it can't until started
	go r.mx.MultiListenAndServe(port)

	return r.mx.Run()
}
