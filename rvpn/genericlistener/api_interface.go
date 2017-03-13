package genericlistener

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

const (
	rDNSPackageName = "com.daplie.rvpn"
)

var connectionTable *Table

//handleAdminClient -
// - expecting an existing oneConnListener with a qualified wss client connected.
// - auth will happen again since we were just peeking at the token.
func handleAdminClient(ctx context.Context, oneConn *oneConnListener) {
	connectionTable = ctx.Value(ctxConnectionTable).(*Table)
	router := mux.NewRouter().StrictSlash(true)

	endpoints := make(map[string]string)

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
		fmt.Fprintln(w, "<html>Welcome..press <a href=/api/servers>Servers</a> to access stats</html>")
	})

	router.HandleFunc("/api/"+rDNSPackageName+"servers", apiServers)

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

func apiServers(w http.ResponseWriter, r *http.Request) {
	fmt.Println("here")
	serverContainer := NewServerAPIContainer()

	for c := range connectionTable.Connections() {
		serverAPI := NewServerAPI(c)
		serverContainer.Servers = append(serverContainer.Servers, serverAPI)

	}

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	json.NewEncoder(w).Encode(serverContainer)

}
