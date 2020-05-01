package api

//DomainsAPI -- A collections of all the domains
//List of Domains -> DomainAPI
//DomainsAPI -> DomainServerAPI
//

//DomainServerAPI -- Container for Server Stats related to a domain
type DomainServerAPI struct {
	ServerName string     `json:"server_name"`
	Traffic    TrafficAPI `json:"traffic"`
}

//NewDomainServerAPI -- Constructor
func NewDomainServerAPI(domain string, conn *Connection) (p *DomainServerAPI) {
	p = new(DomainServerAPI)
	dt := conn.DomainTrack[domain]
	p.Traffic.BytesIn = dt.BytesIn()
	p.Traffic.BytesOut = dt.BytesOut()
	p.Traffic.Requests = dt.Requests()
	p.Traffic.Responses = dt.Responses()
	p.ServerName = conn.ServerName()

	return
}

//DomainAPI -- Container for domain and related servers
type DomainAPI struct {
	DomainName   string             `json:"domain_name"`
	TotalServers int                `json:"server_total"`
	Servers      []*DomainServerAPI `json:"servers"`
	Traffic      TrafficAPI         `json:"traffic"`
}

//NewDomainAPI -- Constructor
func NewDomainAPI(domain string, domainLoadBalance *DomainLoadBalance) (p *DomainAPI) {
	p = new(DomainAPI)
	p.DomainName = domain
	for pos := range domainLoadBalance.Connections() {
		ds := NewDomainServerAPI(domain, domainLoadBalance.Connections()[pos])
		p.Servers = append(p.Servers, ds)
		p.TotalServers++
		p.Traffic.BytesIn += domainLoadBalance.Connections()[pos].BytesIn()
		p.Traffic.BytesOut += domainLoadBalance.Connections()[pos].BytesOut()
		p.Traffic.Requests += domainLoadBalance.Connections()[pos].Requests
		p.Traffic.Responses += domainLoadBalance.Connections()[pos].Responses
	}
	return
}

//DomainsAPI -- Container for Domains
type DomainsAPI struct {
	TotalDomains int          `json:"domain_total"`
	Domains      []*DomainAPI `json:"domains"`
	Traffic      TrafficAPI   `json:"traffic"`
}

//NewDomainsAPI -- Constructor
func NewDomainsAPI(domains map[string]*DomainLoadBalance) (p *DomainsAPI) {
	p = new(DomainsAPI)
	for domain := range domains {
		d := NewDomainAPI(domain, domains[domain])
		p.Domains = append(p.Domains, d)
		p.Traffic.BytesIn += d.Traffic.BytesIn
		p.Traffic.BytesOut += d.Traffic.BytesOut
		p.Traffic.Requests += d.Traffic.Requests
		p.Traffic.Responses += d.Traffic.Responses

	}
	return
}
