package connection

//DomainMapping --
type DomainMapping struct {
	connection *Connection
	domainName string
	err        int
	connCh     chan bool
}

//ConnCh -- Property
func (c *DomainMapping) ConnCh() chan bool {
	return c.connCh
}

//NewDomainMapping -- Constructor
func NewDomainMapping(connection *Connection, domain string) (p *DomainMapping) {
	p = new(DomainMapping)
	p.connection = connection
	p.domainName = domain
	p.err = -1
	p.connCh = make(chan bool)
	return
}
