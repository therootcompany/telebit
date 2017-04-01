package server

//ServerDomainsAPI -- Structure to support the server API
type ServerDomainsAPI struct {
	DomainName string `json:"domain_name"`
	ServerID   int64  `json:"server_id"`
	BytesIn    int64  `json:"bytes_in"`
	BytesOut   int64  `json:"bytes_out"`
	Requests   int64  `json:"requests"`
	Responses  int64  `json:"responses"`
}

//NewServerDomainsAPI - Constructor
func NewServerDomainsAPI(c *Connection, d *DomainTrack) (s *ServerDomainsAPI) {
	s = new(ServerDomainsAPI)
	s.DomainName = d.DomainName
	s.ServerID = c.ConnectionID()
	s.BytesIn = d.BytesIn()
	s.BytesOut = d.BytesOut()
	s.Requests = d.requests
	s.Responses = d.responses

	return
}

//ServerDomainsAPIContainer -- Holder for all the Servers
type ServerDomainsAPIContainer struct {
	Domains []*ServerDomainsAPI `json:"domains"`
}

//NewServerDomainsAPIContainer -- Constructor
func NewServerDomainsAPIContainer() (p *ServerDomainsAPIContainer) {
	p = new(ServerDomainsAPIContainer)
	p.Domains = make([]*ServerDomainsAPI, 0)
	return p
}

//ServerDomainAPI -- Structure to support the server API
type ServerDomainAPI struct {
	DomainName string `json:"domain_name"`
	ServerID   int64  `json:"server_id"`
	BytesIn    int64  `json:"bytes_in"`
	BytesOut   int64  `json:"bytes_out"`
	Requests   int64  `json:"requests"`
	Responses  int64  `json:"responses"`
	Source     string `json:"source_addr"`
}

//NewServerDomainAPI - Constructor
func NewServerDomainAPI(c *Connection, d *DomainTrack) (s *ServerDomainAPI) {
	s = new(ServerDomainAPI)
	s.DomainName = d.DomainName
	s.ServerID = c.ConnectionID()
	s.BytesIn = d.BytesIn()
	s.BytesOut = d.BytesOut()
	s.Requests = d.requests
	s.Responses = d.responses
	s.Source = c.Source()
	return
}
