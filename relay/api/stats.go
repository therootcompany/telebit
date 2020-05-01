package api

//TrafficStats --
type TrafficStats struct {
	Requests  int64
	Responses int64
	BytesIn   int64
	BytesOut  int64
}

//NewTrafficStats -- Consttuctor
func NewTrafficStats() (p *TrafficStats) {
	p = new(TrafficStats)
	p.Requests = 0
	p.Responses = 0
	p.BytesIn = 0
	p.BytesOut = 0

	return
}

//IncRequests --
func (p *TrafficStats) IncRequests() {
	p.Requests++
}

//IncResponses --
func (p *TrafficStats) IncResponses() {
	p.Responses++
}

//AddBytesIn --
func (p *TrafficStats) AddBytesIn(count int64) {
	p.BytesIn = p.BytesIn + count
}

//AddBytesOut --
func (p *TrafficStats) AddBytesOut(count int64) {
	p.BytesOut = p.BytesOut + count
}
