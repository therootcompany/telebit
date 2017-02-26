package rvpnmain

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"context"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/connection"
	"git.daplie.com/Daplie/go-rvpn-server/rvpn/genericlistener"
	"git.daplie.com/Daplie/go-rvpn-server/rvpn/xlate"
)

var (
	loginfo                  *log.Logger
	logdebug                 *log.Logger
	logFlags                 = log.Ldate | log.Lmicroseconds | log.Lshortfile
	argWssClientListener     string
	argGenericBinding        string
	argServerBinding         string
	argServerAdminBinding    string
	argServerExternalBinding string
	argDeadTime              int
	connectionTable          *connection.Table
	wssMapping               *xlate.WssMapping
	secretKey                = "abc123"
)

func init() {
	flag.IntVar(&argDeadTime, "dead-time-counter", 5, "deadtime counter in seconds")
	flag.StringVar(&argGenericBinding, "generic-listener", ":8443", "generic SSL Listener")
	flag.StringVar(&argWssClientListener, "wss-client-listener", ":3502", "wss client listener address:port")
	flag.StringVar(&argServerAdminBinding, "admin-server-port", "127.0.0.2:8000", "admin server Bind listener")
	flag.StringVar(&argServerExternalBinding, "external-server-port", "127.0.0.1:8080", "external server Bind listener")
}

//Run -- main entry point
func Run() {
	flag.Parse()

	loginfo = log.New(os.Stdout, "INFO: packer: ", logFlags)
	logdebug = log.New(os.Stdout, "DEBUG: packer:", logFlags)

	loginfo.Println("startup")

	fmt.Println("-=-=-=-=-=-=-=-=-=-=")

	certbundle, err := tls.LoadX509KeyPair("certs/fullchain.pem", "certs/privkey.pem")
	if err != nil {
		loginfo.Println(err)
		return
	}

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()

	connectionTable = connection.NewTable()
	go connectionTable.Run(ctx)

	// Setup for GenericListenServe.
	// - establish context for the generic listener
	// - startup listener
	// - accept with peek buffer.
	// - peek at the 1st 30 bytes.
	// - check for tls
	// - if tls, establish, protocol peek buffer, else decrypted
	// - match protocol

	go genericlistener.GenericListenAndServe(ctx, connectionTable, secretKey, argGenericBinding, certbundle, argDeadTime)

	time.Sleep(20 * time.Second)
	cancelContext()
	time.Sleep(60 * time.Second)

	//wssMapping = xlate.NewwssMapping()
	//go wssMapping.Run()

	//go client.LaunchClientListener(connectionTable, &secretKey, &argServerBinding)
	//go external.LaunchWebRequestExternalListener(&argServerExternalBinding, connectionTable)
	//go external.LaunchExternalServer(argServerExternalBinding, connectionTable)
	//err = admin.LaunchAdminListener(&argServerAdminBinding, connectionTable)
	//if err != nil {
	//	loginfo.Println("LauchAdminListener failed: ", err)
	//}

	//genericlistener.LaunchWssListener(connectionTable, secretKey, argWssClientListener, "certs/fullchain.pem", "certs/privkey.pem")
}
