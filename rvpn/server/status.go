package server

import (
	"context"
	"time"
)

//Status --
type Status struct {
	ctx                      context.Context
	Name                     string
	StartTime                time.Time
	WssDomain                string
	AdminDomain              string
	DeadTime                 *StatusDeadTime
	ConnectionTracking       *Tracking
	ConnectionTable          *Table
	servers                  *servers
	LoadbalanceDefaultMethod string
	AdminStats               *TrafficStats
	AdminReqTyoe             *AdminReqType
	TrafficStats             *TrafficStats
	ExtConnections           *ConnectionStats
	WSSConnections           *ConnectionStats
}

//NewStatus --
func NewStatus(ctx context.Context) (p *Status) {
	p = new(Status)
	p.ctx = ctx
	p.AdminStats = new(TrafficStats)
	p.TrafficStats = new(TrafficStats)
	p.ExtConnections = new(ConnectionStats)
	p.WSSConnections = new(ConnectionStats)
	return
}

// South Facing Functions

//WSSConnectionRegister --
func (p *Status) WSSConnectionRegister(newRegistration *Registration) {
	p.ConnectionTable.Register() <- newRegistration
	p.WSSConnections.IncConnections()
}

//WSSConnectionUnregister --
//unregisters a south facing connection
//intercept and update global statistics
func (p *Status) WSSConnectionUnregister() {
}

// External Facing Functions

//ExtConnectionRegister --
//registers an ext facing connection
//intercept and update global statistics
func (p *Status) ExtConnectionRegister(newTrack *Track) {
	p.ConnectionTracking.register <- newTrack
	p.ExtConnections.IncConnections()
}

//ExtConnectionUnregister --
//unregisters an ext facing connection
//intercept and update global statistics
func (p *Status) ExtConnectionUnregister(extConn *WedgeConn) {
	p.ConnectionTracking.unregister <- extConn
	p.ExtConnections.DecConnections()

}

//SendExtRequest --
//sends a request to a south facing connection
//intercept the send, update our global stats
func (p *Status) SendExtRequest(conn *Connection, sendTrack *SendTrack) {
	p.TrafficStats.IncRequests()
	p.TrafficStats.AddBytesOut(int64(len(sendTrack.data)))
	conn.SendCh() <- sendTrack
}
