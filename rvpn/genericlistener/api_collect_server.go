package genericlistener

import (
	"fmt"
	"time"
)

//ServerAPI -- Structure to support the server API
type ServerAPI struct {
	ServerName string       `json:"server_name"`
	ServerID   int64        `json:"server_id"`
	Domains    []*DomainAPI `json:"domains"`
	Duration   float64      `json:"duration"`
	BytesIn    int64        `json:"bytes_in"`
	BytesOut   int64        `json:"bytes_out"`
	Source     string       `json:"source_address"`
}

//NewServerAPI - Constructor
func NewServerAPI(c *Connection) (s *ServerAPI) {
	s = new(ServerAPI)
	s.ServerName = fmt.Sprintf("%p", c)
	s.ServerID = c.ConnectionID()
	s.Domains = make([]*DomainAPI, 0)
	s.Duration = time.Since(c.ConnectTime()).Seconds()
	s.BytesIn = c.BytesIn()
	s.BytesOut = c.BytesOut()
	s.Source = c.source

	for domainName := range c.DomainTrack {

		domainAPI := NewDomainAPI(c, c.DomainTrack[domainName])
		s.Domains = append(s.Domains, domainAPI)
	}
	return
}
