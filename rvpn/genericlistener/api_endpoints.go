package genericlistener

import "net/http"

type apiEndPoint struct {
	pack     string
	endpoint string
	method   func(w http.ResponseWriter, r *http.Request)
}

type APIEndPoints struct {
	endPoint map[string]*apiEndPoint
}

//NewAPIEndPoints -- Constructor
func NewAPIEndPoints() (p *APIEndPoints) {
	p = new(apiEndPoints)
	p.endPoint = make(map[string]*apiEndPoint)
	return
}

func (p *apiEndPoints) add(pack string, endpoint string, method func(w http.ResponseWriter, r *http.Request)) {

	router.HandleFunc("/api/"+rDNSPackageName+"servers", apiServers)
}
