package genericlistener

//SendTrack -- Used as a channel communication to id domain asssociated to domain for outbound WSS
type SendTrack struct {
	data   []byte
	domain string
}

//NewSendTrack -- Constructor
func NewSendTrack(data []byte, domain string) (p *SendTrack) {
	p = new(SendTrack)
	p.data = data
	p.domain = domain
	return

}
