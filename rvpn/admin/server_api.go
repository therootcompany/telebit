package admin

import (
	"fmt"
	"time"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/connection"
)

//ServerAPI -- Structure to support the server API
type ServerAPI struct {
	ServerName string
	Domains    []*DomainAPI
	Duration   float64
	BytesIn    int64
	BytesOut   int64
}

//NewServerAPI - Constructor
func NewServerAPI(c *connection.Connection) (s *ServerAPI) {
	s = new(ServerAPI)
	s.ServerName = fmt.Sprintf("%p", c)
	s.Domains = make([]*DomainAPI, 0)
	s.Duration = time.Since(c.ConnectTime()).Seconds()
	s.BytesIn = c.BytesIn()
	s.BytesOut = c.BytesOut()

	for d := range c.DomainTrack {
		dt := c.DomainTrack[d]
		domainAPI := NewDomainAPI(dt.DomainName, dt.BytesIn(), dt.BytesOut())
		s.Domains = append(s.Domains, domainAPI)
	}
	return
}

//ServerAPIContainer -- Holder for all the Servers
type ServerAPIContainer struct {
	Servers []*ServerAPI
}

//NewServerAPIContainer -- Constructor
func NewServerAPIContainer() (p *ServerAPIContainer) {
	p = new(ServerAPIContainer)
	p.Servers = make([]*ServerAPI, 0)
	return p
}
