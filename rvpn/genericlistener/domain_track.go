package genericlistener

//DomainTrack -- Tracking specifics for domains
type DomainTrack struct {
	DomainName string
	bytesIn    int64
	bytesOut   int64
	requests   int64
	responses  int64
}

//NewDomainTrack -- Constructor
func NewDomainTrack(domainName string) (p *DomainTrack) {
	p = new(DomainTrack)
	p.DomainName = domainName
	p.bytesIn = 0
	p.bytesOut = 0
	p.requests = 0
	p.responses = 0
	return
}

//BytesIn -- Property
func (c *DomainTrack) BytesIn() int64 {
	return c.bytesIn
}

//BytesOut -- Property
func (c *DomainTrack) BytesOut() int64 {
	return c.bytesOut
}

//AddIn - Property
func (c *DomainTrack) AddIn(num int64) {
	c.bytesIn = c.bytesIn + num
}

//AddOut -- Property
func (c *DomainTrack) AddOut(num int64) {
	c.bytesOut = c.bytesOut + num
}

//AddRequests - Property
func (c *DomainTrack) AddRequests() {
	c.requests = c.requests + 1
}

//AddResponses - Property
func (c *DomainTrack) AddResponses() {
	c.responses = c.responses + 1
}

//Requests -- Property
func (c *DomainTrack) Requests() int64 {
	return c.requests
}

//Responses -- Property
func (c *DomainTrack) Responses() int64 {
	return c.responses
}
