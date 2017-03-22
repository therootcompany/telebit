package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/genericlistener"
)

var (
	loginfo                  *log.Logger
	logdebug                 *log.Logger
	logFlags                 = log.Ldate | log.Lmicroseconds | log.Lshortfile
	argWssClientListener     string
	argGenericBinding        int
	argServerBinding         string
	argServerAdminBinding    string
	argServerExternalBinding string
	argDeadTime              int
	connectionTable          *genericlistener.Table
	secretKey                = "abc123"
	wssHostName              = "localhost.daplie.me"
	adminHostName            = "rvpn.daplie.invalid"
	idle                     int
	dwell                    int
	cancelcheck              int
	lbDefaultMethod          string
	serverName               string
)

func init() {

}

//Main -- main entry point
func main() {
	flag.Parse()
	loginfo = log.New(os.Stdout, "INFO: main: ", logFlags)
	logdebug = log.New(os.Stdout, "DEBUG: main:", logFlags)
	viper.SetConfigName("go-rvpn-server")
	viper.AddConfigPath("./")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	flag.IntVar(&argDeadTime, "dead-time-counter", 5, "deadtime counter in seconds")

	wssHostName = viper.Get("rvpn.wssdomain").(string)
	adminHostName = viper.Get("rvpn.admindomain").(string)
	argGenericBinding = viper.GetInt("rvpn.genericlistener")
	deadtime := viper.Get("rvpn.deadtime")
	idle = deadtime.(map[string]interface{})["idle"].(int)
	dwell = deadtime.(map[string]interface{})["dwell"].(int)
	cancelcheck = deadtime.(map[string]interface{})["cancelcheck"].(int)
	lbDefaultMethod = viper.Get("rvpn.loadbalancing.defaultmethod").(string)
	serverName = viper.Get("rvpn.serverName").(string)

	loginfo.Println("startup")

	loginfo.Println(viper.Get("rvpn.genericlisteners"))
	loginfo.Println(viper.Get("rvpn.domains"))

	fmt.Println("-=-=-=-=-=-=-=-=-=-=")

	certbundle, err := tls.LoadX509KeyPair("certs/fullchain.pem", "certs/privkey.pem")
	if err != nil {
		loginfo.Println(err)
		return
	}

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()

	serverStatus := genericlistener.NewStatus(ctx)
	serverStatus.AdminDomain = adminHostName
	serverStatus.WssDomain = wssHostName
	serverStatus.Name = serverName
	serverStatus.StartTime = time.Now()
	serverStatus.DeadTime = genericlistener.NewStatusDeadTime(dwell, idle, cancelcheck)
	serverStatus.LoadbalanceDefaultMethod = lbDefaultMethod

	// Setup for GenericListenServe.
	// - establish context for the generic listener
	// - startup listener
	// - accept with peek buffer.
	// - peek at the 1st 30 bytes.
	// - check for tls
	// - if tls, establish, protocol peek buffer, else decrypted
	// - match protocol

	connectionTracking := genericlistener.NewTracking()
	serverStatus.ConnectionTracking = connectionTracking
	go connectionTracking.Run(ctx)

	connectionTable = genericlistener.NewTable(dwell, idle)
	serverStatus.ConnectionTable = connectionTable
	go connectionTable.Run(ctx)

	genericListeners := genericlistener.NewGenerListeners(ctx, secretKey, certbundle, serverStatus)
	serverStatus.GenericListeners = genericListeners

	go genericListeners.Run(ctx, argGenericBinding)

	select {}
}
