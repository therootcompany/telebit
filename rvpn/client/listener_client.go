package client

import (
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/connection"
)

//LaunchClientListener - starts up http listeners and handles various URI paths
func LaunchClientListener(connectionTable *connection.Table, secretKey *string, serverBinding *string) (err error) {
	loginfo.Println("starting WebRequestExternal Listener ", *serverBinding)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch url := r.URL.Path; url {
		case "/":
			handleConnectionWebSocket(connectionTable, w, r, *secretKey, false)

		default:
			http.Error(w, "Not Found", 404)
		}
	})

	s := &http.Server{
		Addr:    *serverBinding,
		Handler: mux,
	}

	err = s.ListenAndServeTLS("certs/fullchain.pem", "certs/privkey.pem")
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
