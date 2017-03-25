package genericlistener

import (
	"context"
	"fmt"
	"time"
)

const (
	initialDomains   = 0
	incrementDomains = 0
)

//Table maintains the set of connections
type Table struct {
	connections    map[*Connection][]string
	domains        map[string]*DomainLoadBalance
	register       chan *Registration
	unregister     chan *Connection
	domainAnnounce chan *DomainMapping
	domainRevoke   chan *DomainMapping
	dwell          int
	idle           int
}

//NewTable -- consructor
func NewTable(dwell, idle int) (p *Table) {
	p = new(Table)
	p.connections = make(map[*Connection][]string)
	p.domains = make(map[string]*DomainLoadBalance)
	p.register = make(chan *Registration)
	p.unregister = make(chan *Connection)
	p.domainAnnounce = make(chan *DomainMapping)
	p.domainRevoke = make(chan *DomainMapping)
	p.dwell = dwell
	p.idle = idle
	return
}

//Connections Property
func (c *Table) Connections() map[*Connection][]string {
	return c.connections
}

//ConnByDomain -- Obtains a connection from a domain announcement.  A domain may be announced more than once
//if that is the case the system stores these connections and then sends traffic back round-robin
//back to the WSS connections
func (c *Table) ConnByDomain(domain string) (*Connection, bool) {
	for dn := range c.domains {
		loginfo.Println(dn, domain)
	}
	if domainsLB, ok := c.domains[domain]; ok {
		loginfo.Println("found")
		conn := domainsLB.NextMember()
		return conn, ok
	}
	return nil, false
}

//reaper --
func (c *Table) reaper(delay int, idle int) {
	_ = "breakpoint"
	for {
		loginfo.Println("Reaper waiting for ", delay, " seconds")
		time.Sleep(time.Duration(delay) * time.Second)

		loginfo.Println("Running scanning ", len(c.connections))
		for d := range c.connections {
			if !d.State() {
				if time.Since(d.lastUpdate).Seconds() > float64(idle) {
					loginfo.Println("reaper removing ", d.lastUpdate, time.Since(d.lastUpdate).Seconds())
					delete(c.connections, d)
				}
			}
		}
	}
}

//GetConnection -- find connection by server-id
func (c *Table) GetConnection(serverID int64) (*Connection, error) {
	for conn := range c.connections {
		if conn.ConnectionID() == serverID {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("Server-id %d not found", serverID)
}

//Run -- Execute
func (c *Table) Run(ctx context.Context, defaultMethod string) {
	loginfo.Println("ConnectionTable starting")

	go c.reaper(c.dwell, c.idle)

	for {
		select {

		case <-ctx.Done():
			loginfo.Println("Cancel signal hit")
			return

		case registration := <-c.register:
			loginfo.Println("register fired")

			connection := NewConnection(c, registration.conn, registration.source, registration.initialDomains,
				registration.connectionTrack, registration.serverName)
			c.connections[connection] = make([]string, initialDomains)
			registration.commCh <- true

			// handle initial domain additions
			for _, domain := range connection.initialDomains {
				// add to the domains regirstation

				newDomain := string(domain.(string))
				loginfo.Println("adding domain ", newDomain, " to connection ", connection.conn.RemoteAddr().String())

				//check to see if domain is already present.
				if _, ok := c.domains[newDomain]; ok {

					//append to a list of connections for that domain
					c.domains[newDomain].AddConnection(connection)
				} else {
					//if not, then add as the 1st to the list of connections
					c.domains[newDomain] = NewDomainLoadBalance(defaultMethod)
					c.domains[newDomain].AddConnection(connection)
				}

				// add to the connection domain list
				s := c.connections[connection]
				c.connections[connection] = append(s, newDomain)
			}
			go connection.Writer()
			go connection.Reader(ctx)

		case connection := <-c.unregister:
			loginfo.Println("closing connection ", connection.conn.RemoteAddr().String())

			//does connection exist in the connection table -- should never be an issue
			if _, ok := c.connections[connection]; ok {

				//iterate over the connections for the domain
				for _, domain := range c.connections[connection] {
					loginfo.Println("remove domain", domain)

					//removing domain, make sure it is present (should never be a problem)
					if _, ok := c.domains[domain]; ok {

						domainLB := c.domains[domain]
						domainLB.RemoveConnection(connection)

						//check to see if domain is free of connections, if yes, delete map entry
						if domainLB.count > 0 {
							//ignore...perhaps we will do something here dealing wtih the lb method
						} else {
							delete(c.domains, domain)
						}
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
func (c *Table) Register() chan *Registration {
	return c.register
}
