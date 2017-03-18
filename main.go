package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"

	"context"

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
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	flag.IntVar(&argDeadTime, "dead-time-counter", 5, "deadtime counter in seconds")

	wssHostName = viper.Get("rvpn.wssdomain").(string)
	adminHostName = viper.Get("rvpn.admindomain").(string)
	argGenericBinding = viper.GetInt("rvpn.genericlistener")
	deadtime := viper.Get("rvpn.deadtime")
	idle = deadtime.(map[string]interface{})["idle"].(int)
	dwell = deadtime.(map[string]interface{})["dwell"].(int)
	cancelcheck = deadtime.(map[string]interface{})["cancelcheck"].(int)

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

	// Setup for GenericListenServe.
	// - establish context for the generic listener
	// - startup listener
	// - accept with peek buffer.
	// - peek at the 1st 30 bytes.
	// - check for tls
	// - if tls, establish, protocol peek buffer, else decrypted
	// - match protocol

	connectionTracking := genericlistener.NewTracking()
	go connectionTracking.Run(ctx)

	connectionTable = genericlistener.NewTable(dwell, idle)
	go connectionTable.Run(ctx)

	genericListeners := genericlistener.NewGenerListeners(ctx, connectionTable, connectionTracking, secretKey, certbundle, wssHostName, adminHostName, cancelcheck)
	go genericListeners.Run(ctx, argGenericBinding)

	//Run for 10 minutes and then shutdown cleanly
	time.Sleep(6000 * time.Second)
	cancelContext()
}
