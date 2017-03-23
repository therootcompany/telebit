package genericlistener

import "sync"

type adminReqType string

const (
	adminStatus  adminReqType = "admin_status"
	adminDomain  adminReqType = "admin_domain"
	adminDomains adminReqType = "admin_domains"
	adminServer  adminReqType = "admin_server"
	adminServers adminReqType = "admin_servers"
)

//AdminReqType --
type AdminReqType struct {
	mutex       *sync.Mutex
	RequestType map[adminReqType]int64
}

//NewAdminReqType -- Constructor
func NewAdminReqType() (p *AdminReqType) {
	p = new(AdminReqType)
	p.mutex = &sync.Mutex{}
	p.RequestType = make(map[adminReqType]int64)
	return
}

func (p *AdminReqType) add(reqType adminReqType) {
	p.mutex.Lock()

	defer p.mutex.Unlock()

	if _, ok := p.RequestType[reqType]; ok {
		p.RequestType[reqType]++
	} else {
		p.RequestType[reqType] = int64(1)
	}
}

func (p *AdminReqType) get(reqType adminReqType) (total int64) {
	p.mutex.Lock()

	defer p.mutex.Unlock()

	if _, ok := p.RequestType[reqType]; ok {
		total = p.RequestType[reqType]
	} else {
		total = 0
	}
	return
}
