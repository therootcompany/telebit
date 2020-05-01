package api

import (
	"time"

	
)

//ServersAPI -- Structure to support the server API
type ServersAPI struct {
	ServerName string             `json:"server_name"`
	ServerID   int64              `json:"server_id"`
	Domains    []*ServerDomainAPI `json:"domains"`
	Duration   float64            `json:"duration"`
	Idle       float64            `json:"idle"`
	BytesIn    int64              `json:"bytes_in"`
	BytesOut   int64              `json:"bytes_out"`
	Requests   int64              `json:"requests"`
	Responses  int64              `json:"responses"`
	Source     string             `json:"source_address"`
	State      bool               `json:"server_state"`
}

//NewServersAPI - Constructor
func NewServersAPI(c *Connection) (s *ServersAPI) {
	s = new(ServersAPI)
	s.ServerName = c.ServerName()
	s.ServerID = c.ConnectionID()
	s.Domains = make([]*ServerDomainAPI, 0)
	s.Duration = time.Since(c.ConnectTime()).Seconds()
	s.Idle = time.Since(c.LastUpdate()).Seconds()
	s.BytesIn = c.BytesIn()
	s.BytesOut = c.BytesOut()
	s.Requests = c.Requests
	s.Responses = c.Responses
	s.Source = c.Source()
	s.State = c.State()

	for d := range c.DomainTrack {
		dt := c.DomainTrack[d]
		domainAPI := NewServerDomainAPI(c, dt)
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
