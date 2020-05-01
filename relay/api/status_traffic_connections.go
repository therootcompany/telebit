package api

//ConnectionStats --
type ConnectionStats struct {
	Connections      int64
	TotalConnections int64
}

//NewConnectionStats -- Consttuctor
func NewConnectionStats() (p *ConnectionStats) {
	p = new(ConnectionStats)
	p.Connections = 0
	p.TotalConnections = 0
	return
}

//IncConnections --
func (p *ConnectionStats) IncConnections() {
	p.Connections++
	p.TotalConnections++
}

//DecConnections --
func (p *ConnectionStats) DecConnections() {
	if p.Connections > 0 {
		p.Connections--
	}
}
