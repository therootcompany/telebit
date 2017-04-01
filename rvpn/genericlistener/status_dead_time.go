package server

//StatusDeadTime -- structure for deadtime configuration
type StatusDeadTime struct {
	dwell       int
	idle        int
	cancelcheck int
}

//NewStatusDeadTime -- constructor
func NewStatusDeadTime(dwell, idle, cancelcheck int) (p *StatusDeadTime) {
	p = new(StatusDeadTime)
	p.dwell = dwell
	p.idle = idle
	p.cancelcheck = cancelcheck
	return
}
