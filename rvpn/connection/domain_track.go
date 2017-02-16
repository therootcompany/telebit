package connection

//DomainTrack -- Tracking specifics for domains
type DomainTrack struct {
	DomainName string
	bytesIn    int64
	bytesOut   int64
}

//NewDomainTrack -- Constructor
func NewDomainTrack(domainName string) (p *DomainTrack) {
	p = new(DomainTrack)
	p.DomainName = domainName
	p.bytesIn = 0
	p.bytesOut = 0
	return
}

//BytesIn -- Property
func (c *DomainTrack) BytesIn() (b int64) {
	b = c.bytesIn
	return
}

//BytesOut -- Property
func (c *DomainTrack) BytesOut() (b int64) {
	b = c.bytesOut
	return
}

//AddIn - Porperty
func (c *DomainTrack) AddIn(num int64) {
	c.bytesIn = c.bytesIn + num
}

//AddOut -- Property
func (c *DomainTrack) AddOut(num int64) {
	c.bytesOut = c.bytesOut + num
}
