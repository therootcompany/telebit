package api

//TrafficStats --
type TrafficAPI struct {
	Requests  int64
	Responses int64
	BytesIn   int64
	BytesOut  int64
}

//NewTrafficStats -- Consttuctor
func NewTrafficAPI(requests, responses, bytesIn, bytesOut int64) (p *TrafficAPI) {
	p = new(TrafficAPI)
	p.Requests = requests
	p.Responses = responses
	p.BytesIn = bytesIn
	p.BytesOut = bytesOut

	return
}
