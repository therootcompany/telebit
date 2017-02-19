package connection

import "github.com/gorilla/websocket"

//Registration -- A connection registration structure used to bring up a connection
//connection table will then handle additing and sdtarting up the various readers
//else error.
type Registration struct {
	// The websocket connection.
	conn *websocket.Conn

	// Address of the Remote End Point
	source string

	// communications channel between go routines
	commCh chan bool

	//initialDomains - a list of domains from the JWT
	initialDomains []interface{}
}

//NewRegistration -- Constructor
func NewRegistration(conn *websocket.Conn, remoteAddress string, initialDomains []interface{}) (p *Registration) {
	p = new(Registration)
	p.conn = conn
	p.source = remoteAddress
	p.commCh = make(chan bool)
	p.initialDomains = initialDomains
	return
}

//CommCh -- Property
func (c *Registration) CommCh() chan bool {
	return c.commCh
}
