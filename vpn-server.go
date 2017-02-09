package main

import (
	"flag"
	"io"
	"log"
	"os"
	"time"

	"git.daplie.com/Daplie/go-rvpn-server/logging"
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
	loginfo                  *log.Logger
	logfatal                 *log.Logger
	logFlags                 = log.Ldate | log.Lmicroseconds | log.Lshortfile
	argServerBinding         = flag.String("server-port", "127.0.0.1:3502", "server Bind listener")
	argServerAdminBinding    = flag.String("admin-server-port", "127.0.0.2:8000", "admin server Bind listener")
	argServerExternalBinding = flag.String("external-server-port", "127.0.0.1:8080", "external server Bind listener")
	connectionTable          *ConnectionTable
	secretKey                = "abc123"
)

func logInit(infoHandle io.Writer) {
	loginfo = log.New(infoHandle, "INFO: ", logFlags)
	logfatal = log.New(infoHandle, "FATAL : ", logFlags)
}

func main() {
	logging.Init(os.Stdout, logFlags)
	linfo, lfatal := logging.Get()
	loginfo = linfo
	logfatal = lfatal

	loginfo.Println("startup")
	flag.Parse()

	connectionTable = newConnectionTable()
	go connectionTable.run()
	go launchClientListener()
	go launchWebRequestExternalListener()
	launchAdminListener()
}
