package genericlistener

import "golang.org/x/net/websocket"

type domain string

//WssRegistration --
type WssRegistration struct {
	domainName domain
	connection *websocket.Conn
}

//WssMapping --
type WssMapping struct {
	register         chan *websocket.Conn
	unregister       chan *websocket.Conn
	domainRegister   chan *WssRegistration
	domainUnregister chan *WssRegistration
	connections      map[*websocket.Conn][]domain
	domains          map[domain]*websocket.Conn
}

//NewwssMapping  -- constructor
func NewwssMapping() (p *WssMapping) {
	p = new(WssMapping)
	p.connections = make(map[*websocket.Conn][]domain)
	return
}

//Run -- Execute
func (c *WssMapping) Run() {
	loginfo.Println("WSSMapping starting")
	for {
		select {
		case wssConn := <-c.register:
			loginfo.Println("register fired")
			c.connections[wssConn] = make([]domain, initialDomains)

			for conn := range c.connections {
				loginfo.Println(conn)
			}

		case wssConn := <-c.unregister:
			loginfo.Println("closing connection ", wssConn)
			if _, ok := c.connections[wssConn]; ok {
				delete(c.connections, wssConn)
			}
		}
	}
}

// register a wss connection first -- initialize the domain slice
// add a domain
//      find the connectino add to the slice.
//      find the domain set the connection in the map.

// domain(s) -> connection
// connection -> domains
