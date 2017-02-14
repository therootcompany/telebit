package rvpnmain

import (
	"flag"
	"fmt"
	"log"
	"os"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/admin"
	"git.daplie.com/Daplie/go-rvpn-server/rvpn/client"
	"git.daplie.com/Daplie/go-rvpn-server/rvpn/connection"
	"git.daplie.com/Daplie/go-rvpn-server/rvpn/external"
	"git.daplie.com/Daplie/go-rvpn-server/rvpn/packer"
	"git.daplie.com/Daplie/go-rvpn-server/rvpn/xlate"
)

var (
	loginfo                  *log.Logger
	logdebug                 *log.Logger
	logFlags                 = log.Ldate | log.Lmicroseconds | log.Lshortfile
	argServerBinding         string
	argServerAdminBinding    string
	argServerExternalBinding string
	connectionTable          *connection.Table
	wssMapping               *xlate.WssMapping
	secretKey                = "abc123"
)

func init() {
	flag.StringVar(&argServerBinding, "server-port", "127.0.0.1:3502", "server Bind listener")
	flag.StringVar(&argServerAdminBinding, "admin-server-port", "127.0.0.2:8000", "admin server Bind listener")
	flag.StringVar(&argServerExternalBinding, "external-server-port", "127.0.0.1:8080", "external server Bind listener")

}

//Run -- main entry point
func Run() {
	flag.Parse()

	loginfo = log.New(os.Stdout, "INFO: packer: ", logFlags)
	logdebug = log.New(os.Stdout, "DEBUG: packer:", logFlags)

	loginfo.Println("startup")

	p := packer.NewPacker()
	p.Header.SetAddress("127.0.0.2")
	p.Header.Port = 32768
	p.Data.AppendString("A test message")
	p.PackV1()

	fmt.Println("-=-=-=-=-=-=-=-=-=-=")

	wssMapping = xlate.NewwssMapping()
	go wssMapping.Run()

	connectionTable = connection.NewTable()
	go connectionTable.Run()
	go client.LaunchClientListener(connectionTable, &secretKey, &argServerBinding)
	go external.LaunchWebRequestExternalListener(&argServerExternalBinding, connectionTable)

	err := admin.LaunchAdminListener(&argServerAdminBinding)
	if err != nil {
		loginfo.Println("LauchAdminListener failed: ", err)
	}
}
