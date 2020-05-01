package api

//StatusDeadTimeAPI -- structure for deadtime configuration
type StatusDeadTimeAPI struct {
	Dwell       int `json:"dwell"`
	Idle        int `json:"idle"`
	Cancelcheck int `json:"cancel_check"`
}

//NewStatusDeadTimeAPI -- constructor
func NewStatusDeadTimeAPI(dwell int, idle int, cancelcheck int) (p *StatusDeadTimeAPI) {
	p = new(StatusDeadTimeAPI)
	p.Dwell = dwell
	p.Idle = idle
	p.Cancelcheck = cancelcheck
	return
}
