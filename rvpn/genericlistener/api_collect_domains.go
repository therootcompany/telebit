package genericlistener

//DomainsAPI -- Structure to support the server API
type DomainsAPI struct {
	DomainName string `json:"domain_name"`
	ServerID   int64  `json:"server_id"`
	BytesIn    int64  `json:"bytes_in"`
	BytesOut   int64  `json:"bytes_out"`
	Requests   int64  `json:"requests"`
	Responses  int64  `json:"responses"`
}

//NewDomainsAPI - Constructor
func NewDomainsAPI(c *Connection, d *DomainTrack) (s *DomainsAPI) {
	s = new(DomainsAPI)
	s.DomainName = d.DomainName
	s.ServerID = c.ConnectionID()
	s.BytesIn = d.BytesIn()
	s.BytesOut = d.BytesOut()
	s.Requests = d.requests
	s.Responses = d.responses

	return
}

//DomainsAPIContainer -- Holder for all the Servers
type DomainsAPIContainer struct {
	Domains []*DomainsAPI `json:"domains"`
}

//NewDomainsAPIContainer -- Constructor
func NewDomainsAPIContainer() (p *DomainsAPIContainer) {
	p = new(DomainsAPIContainer)
	p.Domains = make([]*DomainsAPI, 0)
	return p
}

//DomainAPI -- Structure to support the server API
type DomainAPI struct {
	DomainName string `json:"domain_name"`
	ServerID   int64  `json:"server_id"`
	BytesIn    int64  `json:"bytes_in"`
	BytesOut   int64  `json:"bytes_out"`
	Requests   int64  `json:"requests"`
	Responses  int64  `json:"responses"`
	Source     string `json:"source_addr"`
}

//NewDomainAPI - Constructor
func NewDomainAPI(c *Connection, d *DomainTrack) (s *DomainAPI) {
	s = new(DomainAPI)
	s.DomainName = d.DomainName
	s.ServerID = c.ConnectionID()
	s.BytesIn = d.BytesIn()
	s.BytesOut = d.BytesOut()
	s.Requests = d.requests
	s.Responses = d.responses
	s.Source = c.Source()
	return
}
