package admin

import (
	"log"
	"net"
	"net/http"
	"runtime"
	"strconv"

	"git.coolaj86.com/coolaj86/go-telebitd/relay/api"
	"git.coolaj86.com/coolaj86/go-telebitd/relay/mplexy"

	"github.com/gorilla/mux"
)

const (
	endPointPrefix = "/api/org.rootprojects.tunnel/"
)

var connectionTable *api.Table
var serverStatus *api.Status
var serverStatusAPI *api.Status

//ListenAndServe -
// - expecting an existing oneConnListener with a qualified wss client connected.
// - auth will happen again since we were just peeking at the token.
func ListenAndServe(mx *mplexy.MPlexy, adminListener net.Listener) error {
	//serverStatus = mx.ctx.Value(ctxServerStatus).(*Status)

	connectionTable = mx.Status.ConnectionTable
	serverStatusAPI = mx.Status

	router := mux.NewRouter().StrictSlash(true)
	router.PathPrefix("/admin/").Handler(http.StripPrefix("/admin/", http.FileServer(http.Dir("html/admin"))))
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("HandleFunc /")

		_, err := mx.AuthorizeAdmin(r)
		if err == nil {
			// TODO
			w.Write([]byte("TODO: handle bad auth"))
			return
		}

		serverStatus.AdminStats.IncRequests()

		switch url := r.URL.Path; url {
		case "/":
			http.Redirect(w, r, "/admin", 301)
			serverStatus.AdminStats.IncResponses()
			return
		default:
			http.Error(w, "Not Found", 404)
		}
	})

	router.HandleFunc(endPointPrefix+"domains", getDomainsEndpoint).Methods("GET")
	router.HandleFunc(endPointPrefix+"domain/", getDomainEndpoint).Methods("GET")
	router.HandleFunc(endPointPrefix+"domain/{domain-name}", getDomainEndpoint).Methods("GET")
	router.HandleFunc(endPointPrefix+"servers", getServersEndpoint).Methods("GET")
	router.HandleFunc(endPointPrefix+"server/", getServerEndpoint).Methods("GET")
	router.HandleFunc(endPointPrefix+"server/{server-id}", getServerEndpoint).Methods("GET")
	router.HandleFunc(endPointPrefix+"status/", getStatusEndpoint).Methods("GET")

	s := &http.Server{
		Addr:    ":80",
		Handler: router,
	}
	return s.Serve(adminListener)
}

func getStatusEndpoint(w http.ResponseWriter, r *http.Request) {
	pc, _, _, _ := runtime.Caller(0)
	log.Println(runtime.FuncForPC(pc).Name())

	serverStatus.AdminStats.IncRequests()

	statusContainer := api.NewStatusAPI(serverStatusAPI)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	env := NewResponse("domains/GET")
	env.Result = statusContainer
	env.GenerateWriter(w)
	serverStatus.AdminStats.IncResponses()
}

func getDomainsEndpoint(w http.ResponseWriter, r *http.Request) {
	pc, _, _, _ := runtime.Caller(0)
	log.Println(runtime.FuncForPC(pc).Name())

	serverStatus.AdminStats.IncRequests()

	domainsContainer := api.NewDomainsAPI(connectionTable.Domains)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	env := NewResponse("domains/GET")
	env.Result = domainsContainer
	env.GenerateWriter(w)
	serverStatus.AdminStats.IncResponses()
}

func getDomainEndpoint(w http.ResponseWriter, r *http.Request) {
	pc, _, _, _ := runtime.Caller(0)
	log.Println(runtime.FuncForPC(pc).Name())

	serverStatus.AdminStats.IncRequests()

	env := NewResponse("domain/GET")

	params := mux.Vars(r)
	if id, ok := params["domain-name"]; !ok {
		env.Error = "domain-name is missing"
		env.ErrorURI = r.RequestURI
		env.ErrorDescription = "domain API requires a domain-name"
	} else {
		domainName := id
		if domainLB, ok := connectionTable.Domains[domainName]; !ok {
			env.Error = "domain-name was not found"
			env.ErrorURI = r.RequestURI
			env.ErrorDescription = "domain-name not found"
		} else {
			var domainAPIContainer []*api.ServerDomainAPI
			conns := domainLB.Connections()
			for pos := range conns {
				conn := conns[pos]
				domainAPI := api.NewServerDomainAPI(conn, conn.DomainTrack[domainName])
				domainAPIContainer = append(domainAPIContainer, domainAPI)
			}
			env.Result = domainAPIContainer
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	env.GenerateWriter(w)
	serverStatus.AdminStats.IncResponses()
}

func getServersEndpoint(w http.ResponseWriter, r *http.Request) {
	pc, _, _, _ := runtime.Caller(0)
	log.Println(runtime.FuncForPC(pc).Name())

	serverStatus.AdminStats.IncRequests()

	serverContainer := api.NewServerAPIContainer()

	for c := range connectionTable.Connections() {
		serverAPI := api.NewServersAPI(c)
		serverContainer.Servers = append(serverContainer.Servers, serverAPI)

	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	env := NewResponse("servers/GET")
	env.Result = serverContainer
	env.GenerateWriter(w)
	serverStatus.AdminStats.IncResponses()
}

func getServerEndpoint(w http.ResponseWriter, r *http.Request) {
	pc, _, _, _ := runtime.Caller(0)
	log.Println(runtime.FuncForPC(pc).Name())

	serverStatus.AdminStats.IncRequests()

	env := NewResponse("server/GET")

	params := mux.Vars(r)
	if id, ok := params["server-id"]; !ok {
		env.Error = "server-id is missing"
		env.ErrorURI = r.RequestURI
		env.ErrorDescription = "server API requires a server-id"
	} else {
		serverID, err := strconv.Atoi(id)
		if err != nil {
			env.Error = "server-id is not an integer"
			env.ErrorURI = r.RequestURI
			env.ErrorDescription = "server API requires a server-id"

		} else {
			conn, err := connectionTable.GetConnection(int64(serverID))
			if err != nil {
				env.Error = "server-id was not found"
				env.ErrorURI = r.RequestURI
				env.ErrorDescription = "missing server-id, make sure desired service-id is in servers"
			} else {
				log.Println("test")
				serverAPI := api.NewServerAPI(conn)
				env.Result = serverAPI

			}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	env.GenerateWriter(w)
	serverStatus.AdminStats.IncResponses()
}
