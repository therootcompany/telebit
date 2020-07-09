package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	telebit "git.coolaj86.com/coolaj86/go-telebitd/mplexer"
	tbDns01 "git.coolaj86.com/coolaj86/go-telebitd/mplexer/dns01"
	"git.coolaj86.com/coolaj86/go-telebitd/table"

	"github.com/caddyserver/certmagic"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-acme/lego/v3/challenge"
	legoDns01 "github.com/go-acme/lego/v3/challenge/dns01"
	"github.com/go-acme/lego/v3/providers/dns/duckdns"
	"github.com/go-acme/lego/v3/providers/dns/godaddy"
	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"

	_ "github.com/joho/godotenv/autoload"
)

var authorizer telebit.Authorizer
var httpsrv *http.Server

var apiNotFoundContent = []byte("{ \"error\": \"not found\" }\n")
var apiNotAuthorizedContent = []byte("{ \"error\": \"not authorized\" }\n")

func init() {
	r := chi.NewRouter()

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("[debug] should be handled as websocket quickly")
			next.ServeHTTP(w, r)
		})
	})

	r.Mount("/ws", http.HandlerFunc(upgradeWebsocket))

	httpsrv = &http.Server{
		Handler: r,
	}
}

func main() {
	certpath := flag.String("acme-storage", "./acme.d/", "path to ACME storage directory")
	email := flag.String("acme-email", "", "email to use for Let's Encrypt / ACME registration")
	acmeAgree := flag.Bool("acme-agree", false, "agree to the terms of the ACME service provider (required)")
	//acmeStaging := flag.Bool("acme-staging", false, "get fake certificates for testing")
	acmeDirectory := flag.String("acme-directory", "", "ACME Directory URL")
	enableHTTP01 := flag.Bool("acme-http-01", false, "enable HTTP-01 ACME challenges")
	enableTLSALPN01 := flag.Bool("acme-tls-alpn-01", false, "enable TLS-ALPN-01 ACME challenges")
	acmeRelay := flag.String("acme-relay", "", "the base url of the ACME DNS-01 relay, if not the same as the tunnel relay")
	authURL := flag.String("auth-url", "", "the base url for authentication, if not the same as the tunnel relay")
	//apiHostname := flag.String("admin-hostname", "", "the hostname used to manage clients")
	token := flag.String("token", "", "a pre-generated token to give the server (instead of generating one with --secret)")
	flag.Parse()

	if 0 == len(*authURL) {
		*authURL = os.Getenv("AUTH_URL")
	}
	authorizer = NewAuthorizer(*authURL)

	if 0 == len(*acmeRelay) {
		*acmeRelay = os.Getenv("ACME_RELAY_BASEURL")
	}
	provider, err := getACMEProvider(acmeRelay, token)
	if nil != err {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
		return
	}
	if 0 == len(*email) {
		*email = os.Getenv("ACME_EMAIL")
	}
	fmt.Printf("Email: %q\n", *email)
	acme := &telebit.ACME{
		Email:       *email,
		StoragePath: *certpath,
		Agree:       *acmeAgree,
		Directory:   *acmeDirectory,
		DNSProvider: provider,
		//DNSChallengeOption:     legoDns01.DNSProviderOption,
		DNSChallengeOption: legoDns01.WrapPreCheck(func(domain, fqdn, value string, orig legoDns01.PreCheckFunc) (bool, error) {
			ok, err := orig(fqdn, value)
			if ok {
				fmt.Println("[Telebit-ACME-DNS] sleeping an additional 5 seconds")
				time.Sleep(5 * time.Second)
			}
			return ok, err
		}),
		EnableHTTPChallenge:    *enableHTTP01,
		EnableTLSALPNChallenge: *enableTLSALPN01,
	}

	// TODO ports
	netListener, err := net.Listen("tcp", ":3020")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Bad things are happening: %s", err)
		os.Exit(1)
	}

	tlsListener := tls.NewListener(netListener, NewTLSConfig(acme))
	httpsrv.Serve(tlsListener)
}

// NewTLSConfig returns a certmagic-enabled config
func NewTLSConfig(acme *telebit.ACME) *tls.Config {
	acme.Storage = &certmagic.FileStorage{Path: acme.StoragePath}

	if "" == acme.Directory {
		acme.Directory = certmagic.LetsEncryptProductionCA
	}

	magic, err := telebit.NewCertMagic(acme)
	if nil != err {
		fmt.Fprintf(
			os.Stderr,
			"failed to initialize certificate management (discovery url? local folder perms?): %s\n",
			err,
		)
		os.Exit(1)
	}

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
	return tlsConfig
}

func apiNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.Write(apiNotFoundContent)
}

// NewAuthorizer TODO
func NewAuthorizer(authURL string) telebit.Authorizer {
	return func(r *http.Request) (*telebit.Grants, error) {
		// do we have a valid wss_client?

		var tokenString string
		if auth := strings.Split(r.Header.Get("Authorization"), " "); len(auth) > 1 {
			// TODO handle Basic auth tokens as well
			tokenString = auth[1]
		}
		if "" == tokenString {
			// Browsers do not allow Authorization Headers and must use access_token query string
			tokenString = r.URL.Query().Get("access_token")
		}
		if "" != r.URL.Query().Get("access_token") {
			r.URL.Query().Set("access_token", "[redacted]")
		}

		grants, err := telebit.Inspect(authURL, tokenString)

		if nil != err {
			fmt.Println("[wsserve] return an error, do not go on")
			return nil, err
		}
		if "" != r.URL.Query().Get("access_token") {
			r.URL.Query().Set("access_token", "[redacted:"+grants.Subject+"]")
		}

		return grants, err
	}
}

func upgradeWebsocket(w http.ResponseWriter, r *http.Request) {
	log.Println("websocket opening ", r.RemoteAddr, " ", r.Host)
	w.Header().Set("Content-Type", "application/json")

	if "Upgrade" != r.Header.Get("Connection") && "WebSocket" != r.Header.Get("Upgrade") {
		w.Write(apiNotFoundContent)
		return
	}

	grants, err := authorizer(r)
	if nil != err {
		log.Println("WebSocket authorization failed", err)
		w.Write(apiNotAuthorizedContent)
		return
	}
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	fmt.Println("[debug] grants", grants)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed", err)
		return
	}

	wsTun := telebit.NewWebsocketTunnel(conn)
	server := &table.SubscriberConn{
		RemoteAddr:   r.RemoteAddr,
		WSConn:       conn,
		WSTun:        wsTun,
		Grants:       grants,
		Clients:      &sync.Map{},
		MultiEncoder: telebit.NewEncoder(context.TODO(), wsTun),
		MultiDecoder: telebit.NewDecoder(wsTun),
	}
	// TODO should this happen at NewEncoder()?
	// (or is it even necessary anymore?)
	_ = server.MultiEncoder.Start()

	go func() {
		// (this listener is also a telebit.Router)
		err := server.MultiDecoder.Decode(server)

		// The tunnel itself must be closed explicitly because
		// there's an encoder with a callback between the websocket
		// and the multiplexer, so it doesn't know to stop listening otherwise
		_ = wsTun.Close()
		fmt.Printf("a subscriber stream is done: %q\n", err)
	}()

	table.Add(server)
}

func getACMEProvider(acmeRelay, token *string) (challenge.Provider, error) {
	var err error
	var provider challenge.Provider = nil

	if "" != os.Getenv("GODADDY_API_KEY") {
		id := os.Getenv("GODADDY_API_KEY")
		apiSecret := os.Getenv("GODADDY_API_SECRET")
		if provider, err = newGoDaddyDNSProvider(id, apiSecret); nil != err {
			return nil, err
		}
	} else if "" != os.Getenv("DUCKDNS_TOKEN") {
		if provider, err = newDuckDNSProvider(os.Getenv("DUCKDNS_TOKEN")); nil != err {
			return nil, err
		}
	} else {
		if "" == *acmeRelay {
			return nil, fmt.Errorf("No relay for ACME DNS-01 challenges given to --acme-relay")
		}
		endpoint := *acmeRelay
		if strings.HasSuffix(endpoint, "/") {
			endpoint = endpoint[:len(endpoint)-1]
		}
		//endpoint += "/api/dns/"
		if provider, err = newAPIDNSProvider(endpoint, *token); nil != err {
			return nil, err
		}
	}

	return provider, nil
}

// ACMEProvider TODO
type ACMEProvider struct {
	BaseURL  string
	provider challenge.Provider
}

// Present presents a prepared ACME challenge
func (p *ACMEProvider) Present(domain, token, keyAuth string) error {
	return p.provider.Present(domain, token, keyAuth)
}

// CleanUp removes a used, expired, or otherwise complete ACME challenge
func (p *ACMEProvider) CleanUp(domain, token, keyAuth string) error {
	return p.provider.CleanUp(domain, token, keyAuth)
}

// newDuckDNSProvider is for the sake of demoing the tunnel
func newDuckDNSProvider(token string) (*duckdns.DNSProvider, error) {
	config := duckdns.NewDefaultConfig()
	config.Token = token
	return duckdns.NewDNSProviderConfig(config)
}

// newGoDaddyDNSProvider is for the sake of demoing the tunnel
func newGoDaddyDNSProvider(id, secret string) (*godaddy.DNSProvider, error) {
	config := godaddy.NewDefaultConfig()
	config.APIKey = id
	config.APISecret = secret
	return godaddy.NewDNSProviderConfig(config)
}

// newAPIDNSProvider is for the sake of demoing the tunnel
func newAPIDNSProvider(baseURL string, token string) (*tbDns01.DNSProvider, error) {
	config := tbDns01.NewDefaultConfig()
	config.Token = token
	endpoint, err := url.Parse(baseURL)
	if nil != err {
		return nil, err
	}
	config.Endpoint = endpoint
	return tbDns01.NewDNSProviderConfig(config)
}

/*
	// TODO for http proxy
	return mplexer.TargetOptions {
		Hostname // default localhost
		Termination // default TLS
		XFWD // default... no?
		Port // default 0
		Conn // should be dialed beforehand
	}, nil
*/

/*
	t := telebit.New(token)
	mux := telebit.RouteMux{}
	mux.HandleTLS("*", mux) // go back to itself
	mux.HandleProxy("example.com", "localhost:3000")
	mux.HandleTCP("example.com", func (c *telebit.Conn) {
		return httpmux.Serve()
	})

	l := t.Listen("wss://example.com")
	conn := l.Accept()
	telebit.Serve(listener, mux)
	t.ListenAndServe("wss://example.com", mux)
*/

func getToken(secret string, domains, ports []string) (token string, err error) {
	tokenData := jwt.MapClaims{"domains": domains, "ports": ports}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenData)
	if token, err = jwtToken.SignedString([]byte(secret)); err != nil {
		return "", err
	}
	return token, nil
}
