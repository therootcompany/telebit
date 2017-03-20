package genericlistener

//TrafficStats --
type TrafficAPI struct {
	Requests  int64
	Responses int64
	BytesIn   int64
	BytesOut  int64
}

//NewTrafficStats -- Consttuctor
func NewTrafficAPI(requests int64, responses int64, bytes_in int64, bytes_out int64) (p *TrafficAPI) {
	p = new(TrafficAPI)
	p.Requests = requests
	p.Responses = responses
	p.BytesIn = bytes_in
	p.BytesOut = bytes_out

	return
}