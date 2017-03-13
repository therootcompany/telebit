package genericlistener

import (
	"context"
	"fmt"
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

//handleAdminClient -
// - expecting an existing oneConnListener with a qualified wss client connected.
// - auth will happen again since we were just peeking at the token.
func handleAdminClient(ctx context.Context, oneConn *oneConnListener) {
	connectionTable = ctx.Value(ctxConnectionTable).(*Table)
	router := mux.NewRouter().StrictSlash(true)

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

	router.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintln(w, "<html>Welcome..press <a href=/api/com.daplie.rvpn/servers>Servers</a> to access stats</html>")
	})

	router.HandleFunc(endPointPrefix+"servers", getServersEndpoint).Methods("GET")
	router.HandleFunc(endPointPrefix+"server/", getServerEndpoint).Methods("GET")
	router.HandleFunc(endPointPrefix+"server/{server-id}", getServerEndpoint).Methods("GET")

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

	//json.NewEncoder(w).Encode(serverContainer)

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
