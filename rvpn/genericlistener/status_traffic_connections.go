package genericlistener

//ConnectionStats --
type ConnectionStats struct {
	Connections int64
}

//NewConnectionStats -- Consttuctor
func NewConnectionStats() (p *ConnectionStats) {
	p = new(ConnectionStats)
	p.Connections = 0
	return
}

//IncConnections --
func (p *ConnectionStats) IncConnections() {
	p.Connections++
}

//DecConnections --
func (p *ConnectionStats) DecConnections() {
	p.Connections--
}
