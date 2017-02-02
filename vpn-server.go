package main

import (
	"flag"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	//Info ..
	loginfo         *log.Logger
	logfatal        *log.Logger
	logFlags        = log.Ldate | log.Ltime | log.Lshortfile
	argServerPort   = flag.String("server-port", ":8000", "serverPort listener")
	connectionTable *ConnectionTable
)

func logInit(infoHandle io.Writer) {
	loginfo = log.New(infoHandle, "INFO: ", logFlags)
	logfatal = log.New(infoHandle, "FATAL : ", logFlags)
}

/*
handlerServeContent -- Handles generic URI paths /
"/" - normal client activities for websocket, marked admin=false
"/admin" - marks incoming connection as admin, however must authenticate
"/ws/client" & "/ws/admin" websocket terminations
*/
func handlerServeContent(w http.ResponseWriter, r *http.Request) {
	switch url := r.URL.Path; url {
	case "/":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		template.Must(template.ParseFiles("html/client.html")).Execute(w, r.Host)

	case "/admin":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		template.Must(template.ParseFiles("html/admin.html")).Execute(w, r.Host)

	case "/ws/client":
		handleConnectionWebSocket(connectionTable, w, r, false)

	case "/ws/admin":
		handleConnectionWebSocket(connectionTable, w, r, true)

	default:
		http.Error(w, "Not Found", 404)

	}
}

//launchListener - starts up http listeners and handles various URI paths
func launchListener() {
	loginfo.Println("starting Listener")

	connectionTable = newConnectionTable()
	go connectionTable.run()
	http.HandleFunc("/", handlerServeContent)

	err := http.ListenAndServeTLS(*argServerPort, "server.crt", "server.key", nil)
	if err != nil {
		logfatal.Println("ListenAndServe: ", err)
		panic(err)
	}
}

func main() {
	logInit(os.Stdout)
	loginfo.Println("startup")
	flag.Parse()
	loginfo.Println(*argServerPort)

	go launchListener()
	time.Sleep(600 * time.Second)
}
