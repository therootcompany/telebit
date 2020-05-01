package api

import (
	"time"
)

//StatusAPI -- Structure to support the server API
type StatusAPI struct {
	Name                     string              `json:"name"`
	Uptime                   float64             `json:"uptime"`
	WssDomain                string              `json:"wss_domain"`
	AdminDomain              string              `json:"admin_domain"`
	LoadbalanceDefaultMethod string              `json:"loadbalance_default_method"`
	DeadTime                 *StatusDeadTimeAPI  `json:"dead_time"`
	AdminStats               *TrafficAPI         `json:"admin_traffic"`
	TrafficStats             *TrafficAPI         `json:"traffic"`
	ExtConnections           *ConnectionStatsAPI `json:"ext_connections"`
	WSSConnections           *ConnectionStatsAPI `json:"wss_connections"`
}

//NewStatusAPI - Constructor
func NewStatusAPI(c *Status) (s *StatusAPI) {
	s = new(StatusAPI)
	s.Name = c.Name
	s.Uptime = time.Since(c.StartTime).Seconds()
	s.WssDomain = c.WssDomain
	s.AdminDomain = c.AdminDomain
	s.LoadbalanceDefaultMethod = c.LoadbalanceDefaultMethod
	s.DeadTime = NewStatusDeadTimeAPI(c.DeadTime.dwell, c.DeadTime.idle, c.DeadTime.Cancelcheck)
	s.AdminStats = NewTrafficAPI(c.AdminStats.Requests, c.AdminStats.Responses, c.AdminStats.BytesIn, c.AdminStats.BytesOut)
	s.TrafficStats = NewTrafficAPI(c.TrafficStats.Requests, c.TrafficStats.Responses, c.TrafficStats.BytesIn, c.TrafficStats.BytesOut)
	s.ExtConnections = NewConnectionStatsAPI(c.ExtConnections.Connections, c.ExtConnections.TotalConnections)
	s.WSSConnections = NewConnectionStatsAPI(c.WSSConnections.Connections, c.ExtConnections.TotalConnections)

	return
}
