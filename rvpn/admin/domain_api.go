package admin

//DomainAPI -- Structure to hold the domain tracking for JSON
type DomainAPI struct {
	Domain   string
	BytesIn  int64
	BytesOut int64
}

//NewDomainAPI - Constructor
func NewDomainAPI(domain string, bytesin int64, bytesout int64) (d *DomainAPI) {
	d = new(DomainAPI)
	d.Domain = domain
	d.BytesIn = bytesin
	d.BytesOut = bytesout
	return
}

// //DomainAPIContainer --
// type DomainAPIContainer struct {
// 	Domains []*DomainAPI
// }

// //NewDomainAPIContainer -- Constructor
// func NewDomainAPIContainer() (p *DomainAPIContainer) {
// 	p = new(DomainAPIContainer)
// 	p.Domains = make([]*DomainAPI, 0)
// 	return p
// }
