package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"

	"io"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/genericlistener"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var (
	logfile    = "stdout"
	configPath = "./"
	configFile = "go-rvpn-server.yaml"

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
	flag.StringVar(&logfile, "log", logfile, "Log file (or stdout/stderr; empty for none)")
	flag.StringVar(&configPath, "config-path", configPath, "Configuration File Path")
	flag.StringVar(&configFile, "config-file", configFile, "Configuration File Name")

}

var logoutput io.Writer

//Main -- main entry point
func main() {
	flag.Parse()
	switch logfile {
	case "stdout":
		logoutput = os.Stdout
	case "stderr":
		logoutput = os.Stderr
	case "":
		logoutput = ioutil.Discard
	default:
		logoutput = &lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    100,
			MaxAge:     120,
			MaxBackups: 10,
		}
	}

	// send the output io.Writing to the other packages
	genericlistener.InitLogging(logoutput)

	loginfo = log.New(logoutput, "INFO: main: ", logFlags)
	logdebug = log.New(logoutput, "DEBUG: main:", logFlags)

	viper.SetConfigName(configPath)
	viper.AddConfigPath("./")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	flag.IntVar(&argDeadTime, "dead-time-counter", 5, "deadtime counter in seconds")

	wssHostName = viper.Get("rvpn.wssdomain").(string)
	adminHostName = viper.Get("rvpn.admindomain").(string)
	argGenericBinding = viper.GetInt("rvpn.genericlistener")
	deadtime := viper.Get("rvpn.deadtime").(map[string]interface{})
	idle = deadtime["idle"].(int)
	dwell = deadtime["dwell"].(int)
	cancelcheck = deadtime["cancelcheck"].(int)
	lbDefaultMethod = viper.Get("rvpn.loadbalancing.defaultmethod").(string)
	serverName = viper.Get("rvpn.serverName").(string)

	loginfo.Println("startup")

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
	go connectionTable.Run(ctx, lbDefaultMethod)

	genericListeners := genericlistener.NewGenerListeners(ctx, secretKey, certbundle, serverStatus)
	serverStatus.GenericListeners = genericListeners

	go genericListeners.Run(ctx, argGenericBinding)

	select {}
}
