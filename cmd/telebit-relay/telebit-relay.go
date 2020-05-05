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
	"strconv"
	"strings"

	"git.coolaj86.com/coolaj86/go-telebitd/log"
	"git.coolaj86.com/coolaj86/go-telebitd/relay"
	"git.coolaj86.com/coolaj86/go-telebitd/relay/api"
	"git.coolaj86.com/coolaj86/go-telebitd/relay/mplexy"

	"github.com/caddyserver/certmagic"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-acme/lego/v3/providers/dns/duckdns"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	_ "github.com/joho/godotenv/autoload"
)

// Loginfo TODO remove
var Loginfo = log.Loginfo

// Logdebug TODO remove
var Logdebug = log.Logdebug

func init() {
	log.LogFlags = golog.Ldate | golog.Lmicroseconds | golog.Lshortfile
}

var (
	logfile    = "stdout"
	configPath = "./"
	configFile = "telebit-relay"

	tcpPort           int
	argDeadTime       int
	connectionTable   *api.Table
	secretKey         string
	wssHostName       string
	adminHostName     string
	idle              int
	dwell             int
	cancelcheck       int
	loadBalanceMethod api.LoadBalanceStrategy
	nickname          string
	acmeEmail         string
	acmeStorage       string
	acmeAgree         bool
	acmeStaging       bool
	allclients        string
)

func init() {
	flag.StringVar(&allclients, "clients", "", "list of client:secret pairings such as example.com:secret123,foo.com:secret321")
	flag.StringVar(&acmeEmail, "acme-email", "", "email to use for Let's Encrypt / ACME registration")
	flag.StringVar(&acmeStorage, "acme-storage", "./acme.d/", "path to ACME storage directory")
	flag.BoolVar(&acmeAgree, "acme-agree", false, "agree to the terms of the ACME service provider (required)")
	flag.BoolVar(&acmeStaging, "staging", false, "get fake certificates for testing")
	flag.StringVar(&adminHostName, "admin-hostname", "", "the management domain")
	flag.StringVar(&wssHostName, "wss-hostname", "", "the wss domain for connecting devices, if different from admin")
	flag.StringVar(&configPath, "config-path", configPath, "Configuration File Path")
	flag.StringVar(&secretKey, "secret", "", "a >= 16-character random string for JWT key signing") // SECRET
	flag.StringVar(&logfile, "log", logfile, "Log file (or stdout/stderr; empty for none)")
	flag.IntVar(&tcpPort, "port", 0, "tcp port on which to listen")                           // PORT
	flag.StringVar(&nickname, "nickname", "", "a nickname for this server, as an identifier") // NICKNAME
}

var logoutput io.Writer

// Client is a domain and secret pair
type Client struct {
	domain string
	secret string
}

//Main -- main entry point
func main() {
	flag.Parse()

	if !acmeAgree {
		fmt.Fprintf(os.Stderr, "set --acme-agree=true to accept the terms of the ACME service provider.\n")
		os.Exit(1)
	}

	clients := []Client{}
	for _, pair := range strings.Split(allclients, ", ") {
		if 0 == len(pair) {
			continue
		}
		keyval := strings.Split(pair, ":")
		clients = append(clients, Client{
			domain: keyval[0],
			secret: keyval[1],
		})
	}

	if 0 == len(secretKey) {
		secretKey = os.Getenv("SECRET")
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

	flag.IntVar(&argDeadTime, "dead-time-counter", 5, "deadtime counter in seconds")

	if 0 == tcpPort {
		tcpPort, _ = strconv.Atoi(os.Getenv("PORT"))
		if 0 == tcpPort {
			fmt.Fprintf(os.Stderr, "must specify --port or PORT\n")
			os.Exit(1)
		}
	}

	if 0 == len(adminHostName) {
		adminHostName = os.Getenv("ADMIN_HOSTNAME")
	}
	if 0 == len(wssHostName) {
		wssHostName = os.Getenv("WSS_HOSTNAME")
	}
	if 0 == len(wssHostName) {
		wssHostName = adminHostName
	}

	// load balancer method
	loadBalanceMethod = api.RoundRobin
	if 0 == len(nickname) {
		nickname = os.Getenv("NICKNAME")
	}

	// TODO what do these "deadtimes" do exactly?
	dwell := 120
	idle := 60
	cancelcheck := 10

	Loginfo.Println("startup")

	ctx, cancelContext := context.WithCancel(context.Background())
	defer cancelContext()

	// CertMagic is Greenlock for Go
	directory := certmagic.LetsEncryptProductionCA
	if acmeStaging {
		directory = certmagic.LetsEncryptStagingCA
	}
	magic, err := newCertMagic(directory, acmeEmail, &certmagic.FileStorage{Path: acmeStorage})
	if nil != err {
		fmt.Fprintf(os.Stderr, "failed to initialize certificate management (discovery url? local folder perms?): %s\n", err)
		os.Exit(1)
	}

	serverStatus := api.NewStatus(ctx)
	serverStatus.AdminDomain = adminHostName
	serverStatus.WssDomain = wssHostName
	serverStatus.Name = nickname
	serverStatus.DeadTime = api.NewStatusDeadTime(dwell, idle, cancelcheck)
	serverStatus.LoadbalanceDefaultMethod = string(loadBalanceMethod)

	connectionTable := api.NewTable(dwell, idle, loadBalanceMethod)

	tlsConfig := &tls.Config{
		GetCertificate: func(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return magic.GetCertificate(hello)
			/*
				if false {
					_, _ = magic.GetCertificate(hello)
				}

				// TODO
				// 1. call out to greenlock for validation
				// 2. push challenges through http channel
				// 3. receive certificates (or don't)
				certbundleT, err := tls.LoadX509KeyPair("certs/fullchain.pem", "certs/privkey.pem")
				certbundle := &certbundleT
				if err != nil {
					return nil, err
				}
				return certbundle, nil
			*/
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

		tok, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(secretKey), nil
		})
		if nil != err {
			fmt.Println("return an error, do not go on")
			return nil, err
		}
		fmt.Printf("client claims:\n%+v\n", tok.Claims)

		domains := []string{}
		for _, name := range tok.Claims.(jwt.MapClaims)["domains"].([]interface{}) {
			domains = append(domains, name.(string))
		}
		authz := &mplexy.Authz{
			Domains: domains,
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

func newCertMagic(directory string, email string, storage certmagic.Storage) (*certmagic.Config, error) {
	cache := certmagic.NewCache(certmagic.CacheOptions{
		GetConfigForCert: func(cert certmagic.Certificate) (*certmagic.Config, error) {
			// do whatever you need to do to get the right
			// configuration for this certificate; keep in
			// mind that this config value is used as a
			// template, and will be completed with any
			// defaults that are set in the Default config
			return &certmagic.Config{}, nil
		},
	})
	provider, err := newDuckDNSProvider(os.Getenv("DUCKDNS_TOKEN"))
	if err != nil {
		return nil, err
	}
	magic := certmagic.New(cache, certmagic.Config{
		Storage: storage,
		OnDemand: &certmagic.OnDemandConfig{
			DecisionFunc: func(name string) error {
				return nil
			},
		},
	})
	// Ummm... just a little confusing
	magic.Issuer = certmagic.NewACMEManager(magic, certmagic.ACMEManager{
		DNSProvider:             provider,
		CA:                      directory,
		Email:                   email,
		Agreed:                  true,
		DisableHTTPChallenge:    true,
		DisableTLSALPNChallenge: true,
		// plus any other customizations you need
	})
	return magic, nil
}

// newDuckDNSProvider is for the sake of demoing the tunnel
func newDuckDNSProvider(token string) (*duckdns.DNSProvider, error) {
	config := duckdns.NewDefaultConfig()
	config.Token = token
	return duckdns.NewDNSProviderConfig(config)
}
