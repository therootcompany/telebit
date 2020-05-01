package api

//ConnectionStatsAPI --
type ConnectionStatsAPI struct {
	Connections      int64 `json:"current_connections"`
	TotalConnections int64 `json:"total_connections"`
}

//NewConnectionStatsAPI -- Consttuctor
func NewConnectionStatsAPI(connections int64, totalConnections int64) (p *ConnectionStatsAPI) {
	p = new(ConnectionStatsAPI)
	p.Connections = connections
	p.TotalConnections = totalConnections
	return
}
