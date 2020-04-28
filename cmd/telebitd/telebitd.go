package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/spf13/viper"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	"git.coolaj86.com/coolaj86/go-telebitd/rvpn/server"
)

var (
	logfile    = "stdout"
	configPath = "./"
	configFile = "go-rvpn-server"

	loginfo                  *log.Logger
	logdebug                 *log.Logger
	logFlags                 = log.Ldate | log.Lmicroseconds | log.Lshortfile
	argWssClientListener     string
	argGenericBinding        int
	argServerBinding         string
	argServerAdminBinding    string
	argServerExternalBinding string
	argDeadTime              int
	connectionTable          *server.Table
	secretKey                = "abc123"
	wssHostName              = "localhost.rootprojects.org"
	adminHostName            = "rvpn.rootprojects.invalid"
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
	server.InitLogging(logoutput)

	loginfo = log.New(logoutput, "INFO: main: ", logFlags)
	logdebug = log.New(logoutput, "DEBUG: main:", logFlags)

	viper.SetConfigName(configFile)
	viper.AddConfigPath(configPath)
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

	serverStatus := server.NewStatus(ctx)
	serverStatus.AdminDomain = adminHostName
	serverStatus.WssDomain = wssHostName
	serverStatus.Name = serverName
	serverStatus.StartTime = time.Now()
	serverStatus.DeadTime = server.NewStatusDeadTime(dwell, idle, cancelcheck)
	serverStatus.LoadbalanceDefaultMethod = lbDefaultMethod

	// Setup for GenericListenServe.
	// - establish context for the generic listener
	// - startup listener
	// - accept with peek buffer.
	// - peek at the 1st 30 bytes.
	// - check for tls
	// - if tls, establish, protocol peek buffer, else decrypted
	// - match protocol

	connectionTracking := server.NewTracking()
	serverStatus.ConnectionTracking = connectionTracking
	go connectionTracking.Run(ctx)

	connectionTable = server.NewTable(dwell, idle)
	serverStatus.ConnectionTable = connectionTable
	go connectionTable.Run(ctx, lbDefaultMethod)

	genericListeners := server.NewGenerListeners(ctx, secretKey, certbundle, serverStatus)
	//serverStatus.GenericListeners = genericListeners

	go genericListeners.Run(ctx, argGenericBinding)

	select {}
}
