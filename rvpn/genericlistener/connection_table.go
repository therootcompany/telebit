package genericlistener

import "fmt"
import "time"
import "context"

const (
	initialDomains   = 0
	incrementDomains = 0
)

//Table maintains the set of connections
type Table struct {
	connections    map[*Connection][]string
	domains        map[string]*Connection
	register       chan *Registration
	unregister     chan *Connection
	domainAnnounce chan *DomainMapping
	domainRevoke   chan *DomainMapping
	dwell          int
	idle           int
}

//NewTable -- consructor
func NewTable(dwell int, idle int) (p *Table) {
	p = new(Table)
	p.connections = make(map[*Connection][]string)
	p.domains = make(map[string]*Connection)
	p.register = make(chan *Registration)
	p.unregister = make(chan *Connection)
	p.domainAnnounce = make(chan *DomainMapping)
	p.domainRevoke = make(chan *DomainMapping)
	p.dwell = dwell
	p.idle = idle
	return
}

//Connections Property
func (c *Table) Connections() (table map[*Connection][]string) {
	table = c.connections
	return
}

//ConnByDomain -- Obtains a connection from a domain announcement.
func (c *Table) ConnByDomain(domain string) (conn *Connection, ok bool) {
	conn, ok = c.domains[domain]
	return
}

//reaper --
func (c *Table) reaper(delay int, idle int) {
	_ = "breakpoint"
	for {
		loginfo.Println("Reaper waiting for ", delay, " seconds")
		time.Sleep(time.Duration(delay) * time.Second)

		loginfo.Println("Running scanning ", len(c.connections))
		for d := range c.connections {
			if d.GetState() == false {
				if time.Since(d.lastUpdate).Seconds() > float64(idle) {
					loginfo.Println("reaper removing ", d.lastUpdate, time.Since(d.lastUpdate).Seconds())
					delete(c.connections, d)
				}
			}
		}
	}
}

//Run -- Execute
func (c *Table) Run(ctx context.Context) {
	loginfo.Println("ConnectionTable starting")

	go c.reaper(c.dwell, c.idle)

	for {
		select {

		case <-ctx.Done():
			loginfo.Println("Cancel signal hit")
			return

		case registration := <-c.register:
			loginfo.Println("register fired")

			connection := NewConnection(c, registration.conn, registration.source, registration.initialDomains, registration.connectionTrack)
			c.connections[connection] = make([]string, initialDomains)
			registration.commCh <- true

			// handle initial domain additions
			for _, domain := range connection.initialDomains {
				// add to the domains regirstation

				newDomain := string(domain.(string))
				loginfo.Println("adding domain ", newDomain, " to connection ", connection.conn.RemoteAddr().String())
				c.domains[newDomain] = connection

				// add to the connection domain list
				s := c.connections[connection]
				c.connections[connection] = append(s, newDomain)
			}
			go connection.Writer()
			go connection.Reader(ctx)

		case connection := <-c.unregister:
			loginfo.Println("closing connection ", connection.conn.RemoteAddr().String())
			if _, ok := c.connections[connection]; ok {
				for _, domain := range c.connections[connection] {
					fmt.Println("removing domain ", domain)
					if _, ok := c.domains[domain]; ok {
						delete(c.domains, domain)
					}
				}

				//delete(c.connections, connection)
				//close(connection.send)
			}

		case domainMapping := <-c.domainAnnounce:
			loginfo.Println("domainMapping fired ", domainMapping)
			//check to make sure connection is already regiered, you can no register a domain without an apporved connection
			//if connection, ok := connections[domainMapping.connection]; ok {

			//} else {

			//}

		}
	}
}

//Register -- Property
func (c *Table) Register() (r chan *Registration) {
	r = c.register
	return
}
