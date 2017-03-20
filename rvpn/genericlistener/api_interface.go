package genericlistener

import (
	"context"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/envelope"
	"github.com/gorilla/mux"
)

const (
	endPointPrefix = "/api/com.daplie.rvpn/"
)

var connectionTable *Table
var serverStatusAPI *Status

//handleAdminClient -
// - expecting an existing oneConnListener with a qualified wss client connected.
// - auth will happen again since we were just peeking at the token.
func handleAdminClient(ctx context.Context, oneConn *oneConnListener) {
	serverStatus := ctx.Value(ctxServerStatus).(*Status)

	connectionTable = serverStatus.ConnectionTable
	serverStatusAPI = serverStatus
	router := mux.NewRouter().StrictSlash(true)

	router.PathPrefix("/admin/").Handler(http.StripPrefix("/admin/", http.FileServer(http.Dir("html/admin"))))

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		loginfo.Println("HandleFunc /")
		switch url := r.URL.Path; url {
		case "/":
			// check to see if we are using the administrative Host
			if strings.Contains(r.Host, "rvpn.daplie.invalid") {
				http.Redirect(w, r, "/admin", 301)
			}

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

	err := s.Serve(oneConn)
	if err != nil {
		loginfo.Println("Serve error: ", err)
	}

	select {
	case <-ctx.Done():
		loginfo.Println("Cancel signal hit")
		return
	}
}

func getStatusEndpoint(w http.ResponseWriter, r *http.Request) {
	pc, _, _, _ := runtime.Caller(0)
	loginfo.Println(runtime.FuncForPC(pc).Name())

	statusContainer := NewStatusAPI(serverStatusAPI)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	env := envelope.NewEnvelope("domains/GET")
	env.Result = statusContainer
	env.GenerateWriter(w)

}

func getDomainsEndpoint(w http.ResponseWriter, r *http.Request) {
	pc, _, _, _ := runtime.Caller(0)
	loginfo.Println(runtime.FuncForPC(pc).Name())

	domainsContainer := NewDomainsAPIContainer()

	for domain := range connectionTable.domains {
		conn := connectionTable.domains[domain]
		domainAPI := NewDomainsAPI(conn, conn.DomainTrack[domain])
		domainsContainer.Domains = append(domainsContainer.Domains, domainAPI)

	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	env := envelope.NewEnvelope("domains/GET")
	env.Result = domainsContainer
	env.GenerateWriter(w)

}

func getDomainEndpoint(w http.ResponseWriter, r *http.Request) {
	pc, _, _, _ := runtime.Caller(0)
	loginfo.Println(runtime.FuncForPC(pc).Name())

	env := envelope.NewEnvelope("domain/GET")

	params := mux.Vars(r)
	if id, ok := params["domain-name"]; !ok {
		env.Error = "domain-name is missing"
		env.ErrorURI = r.RequestURI
		env.ErrorDescription = "domain API requires a domain-name"
	} else {
		domainName := id
		if conn, ok := connectionTable.domains[domainName]; !ok {
			env.Error = "domain-name was not found"
			env.ErrorURI = r.RequestURI
			env.ErrorDescription = "domain-name not found"
		} else {

			domainAPI := NewDomainAPI(conn, conn.DomainTrack[domainName])
			env.Result = domainAPI
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	env.GenerateWriter(w)
}

func getServersEndpoint(w http.ResponseWriter, r *http.Request) {
	pc, _, _, _ := runtime.Caller(0)
	loginfo.Println(runtime.FuncForPC(pc).Name())

	serverContainer := NewServerAPIContainer()

	for c := range connectionTable.Connections() {
		serverAPI := NewServersAPI(c)
		serverContainer.Servers = append(serverContainer.Servers, serverAPI)

	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	env := envelope.NewEnvelope("servers/GET")
	env.Result = serverContainer
	env.GenerateWriter(w)
}

func getServerEndpoint(w http.ResponseWriter, r *http.Request) {
	pc, _, _, _ := runtime.Caller(0)
	loginfo.Println(runtime.FuncForPC(pc).Name())

	env := envelope.NewEnvelope("server/GET")

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
				loginfo.Println("test")
				serverAPI := NewServerAPI(conn)
				env.Result = serverAPI

			}
		}
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	env.GenerateWriter(w)
}
