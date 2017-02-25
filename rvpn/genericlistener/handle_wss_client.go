package genericlistener

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/admin"
	"git.daplie.com/Daplie/go-rvpn-server/rvpn/connection"
)

//LaunchWssListener - obtains a onetime connection from wedge listener
func LaunchWssListener(connectionTable *connection.Table, secretKey string, serverBind string, certfile string, keyfile string) (err error) {
	loginfo.Println("starting LaunchWssListener ")

	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		loginfo.Println("HandleFunc /")
		switch url := r.URL.Path; url {
		case "/":
			// check to see if we are using the administrative Host
			if strings.Contains(r.Host, "rvpn.daplie.invalid") {
				http.Redirect(w, r, "/admin", 301)
			}

			handleConnectionWebSocket(connectionTable, w, r, secretKey, false)

		default:
			http.Error(w, "Not Found", 404)
		}
	})

	router.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Welcome!")
	})

	router.HandleFunc("/api/servers", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("here")
		serverContainer := admin.NewServerAPIContainer()

		for c := range connectionTable.Connections() {
			serverAPI := admin.NewServerAPI(c)
			serverContainer.Servers = append(serverContainer.Servers, serverAPI)

		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		json.NewEncoder(w).Encode(serverContainer)

	})

	s := &http.Server{
		Addr:    serverBind,
		Handler: router,
	}

	err = s.ListenAndServeTLS(certfile, keyfile)
	if err != nil {
		loginfo.Println("ListenAndServeTLS: ", err)
	}
	return
}

// handleConnectionWebSocket handles websocket requests from the peer.
func handleConnectionWebSocket(connectionTable *connection.Table, w http.ResponseWriter, r *http.Request, secretKey string, admin bool) {
	loginfo.Println("websocket opening ", r.RemoteAddr, " ", r.Host)

	tokenString := r.URL.Query().Get("access_token")
	result, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secretKey), nil
	})

	if err != nil || !result.Valid {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Not Authorized"))
		loginfo.Println("access_token invalid...closing connection")
		return
	}

	loginfo.Println("help access_token valid")

	claims := result.Claims.(jwt.MapClaims)
	domains, ok := claims["domains"].([]interface{})

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		loginfo.Println("WebSocket upgrade failed", err)
		return
	}

	loginfo.Println("before connection table")

	//newConnection := connection.NewConnection(connectionTable, conn, r.RemoteAddr, domains)

	newRegistration := connection.NewRegistration(conn, r.RemoteAddr, domains)
	connectionTable.Register() <- newRegistration
	ok = <-newRegistration.CommCh()
	if !ok {
		loginfo.Println("connection registration failed ", newRegistration)
		return
	}

	loginfo.Println("connection registration accepted ", newRegistration)
}
