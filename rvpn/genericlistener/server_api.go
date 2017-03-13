package genericlistener

import (
	"fmt"
	"time"
)

//ServerAPI -- Structure to support the server API
type ServerAPI struct {
	ServerName string       `json:"server_name"`
	Domains    []*DomainAPI `json:"domains"`
	Duration   float64      `json:"duration"`
	BytesIn    int64        `json:"bytes_in"`
	BytesOut   int64        `json:"bytes_out"`
}

//NewServerAPI - Constructor
func NewServerAPI(c *Connection) (s *ServerAPI) {
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
	Servers []*ServerAPI `json:"servers"`
}

//NewServerAPIContainer -- Constructor
func NewServerAPIContainer() (p *ServerAPIContainer) {
	p = new(ServerAPIContainer)
	p.Servers = make([]*ServerAPI, 0)
	return p
}
