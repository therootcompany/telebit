package connection

import "fmt"

const (
	initialDomains   = 0
	incrementDomains = 0
)

//Table maintains the set of connections
type Table struct {
	connections    map[*Connection][]string
	domains        map[string]*Connection
	register       chan *Connection
	unregister     chan *Connection
	domainAnnounce chan *DomainMapping
	domainRevoke   chan *DomainMapping
}

//NewTable -- consructor
func NewTable() (p *Table) {
	p = new(Table)
	p.connections = make(map[*Connection][]string)
	p.domains = make(map[string]*Connection)
	p.register = make(chan *Connection)
	p.unregister = make(chan *Connection)
	p.domainAnnounce = make(chan *DomainMapping)
	p.domainRevoke = make(chan *DomainMapping)
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

//Run -- Execute
func (c *Table) Run() {
	loginfo.Println("ConnectionTable starting")
	for {
		select {
		case connection := <-c.register:
			loginfo.Println("register fired")
			c.connections[connection] = make([]string, initialDomains)
			connection.commCh <- true

			// handle initial domain additions
			for _, domain := range connection.initialDomains {
				// add to the domains regirstation

				newDomain := string(domain.(string))
				loginfo.Println("adding domain ", newDomain, " to connection ", connection)
				c.domains[newDomain] = connection

				// add to the connection domain list
				s := c.connections[connection]
				c.connections[connection] = append(s, newDomain)
			}

			loginfo.Println("register exiting")

		case connection := <-c.unregister:
			loginfo.Println("closing connection ", connection)
			if _, ok := c.connections[connection]; ok {
				for _, domain := range c.connections[connection] {
					fmt.Println("removing domain ", domain)
					if _, ok := c.domains[domain]; ok {
						delete(c.domains, domain)
					}
				}

				delete(c.connections, connection)
				close(connection.send)
			}

		case domainMapping := <-c.domainAnnounce:
			loginfo.Println("domainMapping fired ", domainMapping)
			//check to make sure connection is already regiered, you can no register a domain without an apporved connection
			//if connection, ok := connections[domainMapping.connection]; ok {

			//} else {

			//}

		}
		fmt.Println("domain ", c.domains)
		fmt.Println("connections ", c.connections)
	}
}

//Register -- Property
func (c *Table) Register() (r chan *Connection) {
	r = c.register
	return
}
