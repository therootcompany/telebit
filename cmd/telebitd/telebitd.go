package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	golog "log"
	"net/http"
	"os"
	"strings"

	telebit "git.coolaj86.com/coolaj86/go-telebitd"
	"git.coolaj86.com/coolaj86/go-telebitd/log"
	"git.coolaj86.com/coolaj86/go-telebitd/relay"
	"git.coolaj86.com/coolaj86/go-telebitd/relay/api"
	"git.coolaj86.com/coolaj86/go-telebitd/relay/mplexy"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/spf13/viper"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	_ "github.com/joho/godotenv/autoload"
)

var Loginfo = log.Loginfo
var Logdebug = log.Logdebug

func init() {
	log.LogFlags = golog.Ldate | golog.Lmicroseconds | golog.Lshortfile
}

var (
	logfile    = "stdout"
	configPath = "./"
	configFile = "telebit-relay"

	argWssClientListener     string
	tcpPort                  int
	argServerBinding         string
	argServerAdminBinding    string
	argServerExternalBinding string
	argDeadTime              int
	connectionTable          *api.Table
	secretKey                string
	wssHostName              = "localhost.rootprojects.org"
	adminHostName            = telebit.InvalidAdminDomain
	idle                     int
	dwell                    int
	cancelcheck              int
	lbDefaultMethod          string
	nickname                 string
)

func init() {
	flag.StringVar(&logfile, "log", logfile, "Log file (or stdout/stderr; empty for none)")
	flag.StringVar(&configPath, "config-path", configPath, "Configuration File Path")
	flag.StringVar(&secretKey, "secret", "", "a >= 16-character random string for JWT key signing")
}

var logoutput io.Writer

//Main -- main entry point
func main() {
	flag.Parse()

	if "" == secretKey {
		secretKey = os.Getenv("TELEBIT_SECRET")
	}
	if len(secretKey) < 16 {
		fmt.Fprintf(os.Stderr, "Invalid secret: %q. See --help for details.\n", secretKey)
		os.Exit(1)
	}

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
	log.InitLogging(logoutput)

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
	tcpPort = viper.GetInt("rvpn.port")
	deadtime := viper.Get("rvpn.deadtime").(map[string]interface{})
	idle = deadtime["idle"].(int)
	dwell = deadtime["dwell"].(int)
	cancelcheck = deadtime["cancelcheck"].(int)
	lbDefaultMethod = viper.Get("rvpn.loadbalancing.defaultmethod").(string)
	nickname = viper.Get("rvpn.serverName").(string)

	Loginfo.Println("startup")

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()

	serverStatus := api.NewStatus(ctx)
	serverStatus.AdminDomain = adminHostName
	serverStatus.WssDomain = wssHostName
	serverStatus.Name = nickname
	serverStatus.DeadTime = api.NewStatusDeadTime(dwell, idle, cancelcheck)
	serverStatus.LoadbalanceDefaultMethod = lbDefaultMethod

	connectionTable := api.NewTable(dwell, idle, lbDefaultMethod)

	tlsConfig := &tls.Config{
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			// TODO
			// 1. call out to greenlock for validation
			// 2. push challenges through http channel
			// 3. receive certificates (or don't)
			certbundle, err := tls.LoadX509KeyPair("certs/fullchain.pem", "certs/privkey.pem")
			if err != nil {
				return nil, err
			}
			return &certbundle, nil
		},
	}

	authorizer := func(r *http.Request) (*mplexy.Authz, error) {
		// do we have a valid wss_client?

		var tokenString string
		if auth := strings.Split(r.Header.Get("Authorization"), " "); len(auth) > 1 {
			// TODO handle Basic auth tokens as well
			tokenString = auth[1]
		}
		if "" == tokenString {
			tokenString = r.URL.Query().Get("access_token")
		}

		_, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		if nil != err {
			return nil, err
		}

		authz := &mplexy.Authz{
			Domains: []string{
				"target.rootprojects.org",
			},
		}
		return authz, err

		/*
			tokenString := r.URL.Query().Get("access_token")
			result, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				return []byte(secretKey), nil
			})

			if err != nil || !result.Valid {
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("Not Authorized"))
				Loginfo.Println("access_token invalid...closing connection")
				return
			}

			// TODO
			claims := result.Claims.(jwt.MapClaims)
			domains, ok := claims["domains"].([]interface{})
		*/
	}

	r := relay.New(ctx, tlsConfig, authorizer, serverStatus, connectionTable)
	r.ListenAndServe(tcpPort)
}
