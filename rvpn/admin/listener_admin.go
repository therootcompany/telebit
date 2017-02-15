package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/connection"

	"github.com/gorilla/mux"
)

var (
	connTable *connection.Table
)

//LaunchAdminListener - starts up http listeners and handles various URI paths
func LaunchAdminListener(serverBinding *string, connectionTable *connection.Table) (err error) {
	loginfo.Println("starting launchAdminListener", *serverBinding)

	connTable = connectionTable

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", index)
	router.HandleFunc("/api/servers", apiServers)

	s := &http.Server{
		Addr:    *serverBinding,
		Handler: router,
	}

	err = s.ListenAndServeTLS("certs/fullchain.pem", "certs/privkey.pem")
	if err != nil {
		loginfo.Println("ListenAndServe: ", err)
	}
	return
}

func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Welcome!")
}

//ServerAPI -- Structure to support the server API
type ServerAPI struct {
	ServerName string
	Duration   float64
	BytesIn    int64
	BytesOut   int64
}

//NewServerAPI - Constructor
func NewServerAPI(c *connection.Connection) (s *ServerAPI) {
	s = new(ServerAPI)
	s.ServerName = fmt.Sprintf("%p", c)

	fmt.Println(s.ServerName)

	s.Duration = time.Since(c.ConnectTime()).Seconds()
	s.BytesIn = c.BytesIn()
	s.BytesOut = c.BytesOut()
	return

}

//ServerAPIContainer -- Holder for all the Servers
type ServerAPIContainer struct {
	Servers []*ServerAPI
}

//NewServerAPIContainer -- Constructor
func NewServerAPIContainer() (p *ServerAPIContainer) {
	p = new(ServerAPIContainer)
	p.Servers = make([]*ServerAPI, 0)
	return p
}

func apiServers(w http.ResponseWriter, r *http.Request) {
	fmt.Println("here")
	serverContainer := NewServerAPIContainer()

	for c := range connTable.Connections() {
		serverAPI := NewServerAPI(c)
		serverContainer.Servers = append(serverContainer.Servers, serverAPI)
	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(serverContainer)

}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Domain not supported", http.StatusBadRequest)
}
