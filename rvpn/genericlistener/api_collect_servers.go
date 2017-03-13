package genericlistener

import (
	"fmt"
	"time"
)

//ServersAPI -- Structure to support the server API
type ServersAPI struct {
	ServerName string       `json:"server_name"`
	ServerID   int64        `json:"server_id"`
	Domains    []*DomainAPI `json:"domains"`
	Duration   float64      `json:"duration"`
	BytesIn    int64        `json:"bytes_in"`
	BytesOut   int64        `json:"bytes_out"`
	Source     string       `json:"source_address"`
}

//NewServersAPI - Constructor
func NewServersAPI(c *Connection) (s *ServersAPI) {
	s = new(ServersAPI)
	s.ServerName = fmt.Sprintf("%p", c)
	s.ServerID = c.ConnectionID()
	s.Domains = make([]*DomainAPI, 0)
	s.Duration = time.Since(c.ConnectTime()).Seconds()
	s.BytesIn = c.BytesIn()
	s.BytesOut = c.BytesOut()
	s.Source = c.Source()

	for d := range c.DomainTrack {
		dt := c.DomainTrack[d]
		domainAPI := NewDomainAPI(dt.DomainName, dt.BytesIn(), dt.BytesOut())
		s.Domains = append(s.Domains, domainAPI)
	}
	return
}

//ServerAPIContainer -- Holder for all the Servers
type ServerAPIContainer struct {
	Servers []*ServersAPI `json:"servers"`
}

//NewServerAPIContainer -- Constructor
func NewServerAPIContainer() (p *ServerAPIContainer) {
	p = new(ServerAPIContainer)
	p.Servers = make([]*ServersAPI, 0)
	return p
}
