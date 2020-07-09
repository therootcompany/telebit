//go:generate go run -mod=vendor git.rootprojects.org/root/go-gitver

package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.coolaj86.com/coolaj86/go-telebitd/mgmt"
	"git.coolaj86.com/coolaj86/go-telebitd/mgmt/authstore"
	telebit "git.coolaj86.com/coolaj86/go-telebitd/mplexer"
	tbDns01 "git.coolaj86.com/coolaj86/go-telebitd/mplexer/dns01"
	httpshim "git.coolaj86.com/coolaj86/go-telebitd/relay/tunnel"
	"git.coolaj86.com/coolaj86/go-telebitd/table"
	legoDns01 "github.com/go-acme/lego/v3/challenge/dns01"

	"github.com/caddyserver/certmagic"
	"github.com/denisbrodbeck/machineid"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/providers/dns/duckdns"
	"github.com/go-acme/lego/v3/providers/dns/godaddy"
	_ "github.com/joho/godotenv/autoload"
)

var (
	// GitRev refers to the abbreviated commit hash
	GitRev = "0000000"
	// GitVersion refers to the most recent tag, plus any commits made since then
	GitVersion = "v0.0.0-pre0+0000000"
	// GitTimestamp refers to the timestamp of the most recent commit
	GitTimestamp = "0000-00-00T00:00:00+0000"
)

type Forward struct {
	scheme  string
	pattern string
	port    string
}

var authorizer telebit.Authorizer

var isHostname = regexp.MustCompile(`^[A-Za-z0-9_\.\-]+$`).MatchString

func main() {
	var domains []string
	var forwards []Forward
	var portForwards []Forward

	// TODO replace the websocket connection with a mock server
	appID := flag.String("app-id", "telebit.io", "a unique identifier for a deploy target environment")
	email := flag.String("acme-email", "", "email to use for Let's Encrypt / ACME registration")
	certpath := flag.String("acme-storage", "./acme.d/", "path to ACME storage directory")
	acmeAgree := flag.Bool("acme-agree", false, "agree to the terms of the ACME service provider (required)")
	acmeStaging := flag.Bool("acme-staging", false, "get fake certificates for testing")
	acmeDirectory := flag.String("acme-directory", "", "ACME Directory URL")
	enableHTTP01 := flag.Bool("acme-http-01", false, "enable HTTP-01 ACME challenges")
	enableTLSALPN01 := flag.Bool("acme-tls-alpn-01", false, "enable TLS-ALPN-01 ACME challenges")
	acmeRelay := flag.String("acme-relay", "", "the base url of the ACME DNS-01 relay, if not the same as the tunnel relay")
	authURL := flag.String("auth-url", "", "the base url for authentication, if not the same as the tunnel relay")
	relay := flag.String("relay", "", "the domain (or ip address) at which the relay server is running")
	apiHostname := flag.String("admin-hostname", "", "the hostname used to manage clients")
	secret := flag.String("secret", "", "the same secret used by telebit-relay (used for JWT authentication)")
	token := flag.String("token", "", "a pre-generated token to give the server (instead of generating one with --secret)")
	bindAddrsStr := flag.String("listen", "", "list of bind addresses on which to listen, such as localhost:80, or :443")
	locals := flag.String("locals", "", "a list of <from-domain>:<to-port>")
	portToPorts := flag.String("port-forward", "", "a list of <from-port>:<to-port> for raw port-forwarding")
	flag.Parse()

	if len(os.Args) >= 2 {
		if "version" == os.Args[1] {
			fmt.Printf("telebit %s %s %s", GitVersion, GitRev[:7], GitTimestamp)
			os.Exit(0)
		}
	}

	if len(*acmeDirectory) > 0 {
		if *acmeStaging {
			fmt.Fprintf(os.Stderr, "pick either acme-directory or acme-staging\n")
			os.Exit(1)
			return
		}
	}
	if *acmeStaging {
		*acmeDirectory = certmagic.LetsEncryptStagingCA
	}

	if 0 == len(*locals) {
		*locals = os.Getenv("LOCALS")
	}
	for _, cfg := range strings.Fields(strings.ReplaceAll(*locals, ",", " ")) {
		parts := strings.Split(cfg, ":")
		last := len(parts) - 1
		port := parts[last]
		domain := parts[last-1]
		scheme := ""
		if len(parts) > 2 {
			scheme = parts[0]
		}
		forwards = append(forwards, Forward{
			scheme:  scheme,
			pattern: domain,
			port:    port,
		})

		// don't load wildcard into jwt domains
		if "*" == domain {
			continue
		}
		domains = append(domains, domain)
	}

	if 0 == len(*portToPorts) {
		*portToPorts = os.Getenv("PORT_FORWARDS")
	}
	portForwards, err := parsePortForwards(portToPorts)
	if nil != err {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(1)
		return
	}

	bindAddrs, err := parseBindAddrs(*bindAddrsStr)
	if nil != err {
		fmt.Fprintf(os.Stderr, "invalid bind address(es) given to --listen\n")
		os.Exit(1)
		return
	}

	if 0 == len(*secret) {
		*secret = os.Getenv("SECRET")
	}
	ppid, err := machineid.ProtectedID(fmt.Sprintf("%s|%s", *appID, *secret))
	if nil != err {
		fmt.Fprintf(os.Stderr, "unauthorized device\n")
		os.Exit(1)
		return
	}
	ppidBytes, err := hex.DecodeString(ppid)
	ppid = base64.RawURLEncoding.EncodeToString(ppidBytes)

	if 0 == len(*token) {
		*token, err = authstore.HMACToken(ppid)
		if nil != err {
			fmt.Fprintf(os.Stderr, "neither secret nor token provided\n")
			os.Exit(1)
			return
		}
	}
	if 0 == len(*relay) {
		*relay = os.Getenv("RELAY") // "wss://example.com:443"
	}
	if 0 == len(*relay) {
		if len(bindAddrs) > 0 {
			fmt.Fprintf(os.Stderr, "Acting as Relay\n")
		} else {
			fmt.Fprintf(os.Stderr, "error: must provider or act as Relay\n")
			os.Exit(1)
			return
		}
	}
	if 0 == len(*acmeRelay) {
		*acmeRelay = strings.Replace(*relay, "ws", "http", 1) // "https://example.com:443"
	}

	if 0 == len(*authURL) {
		*authURL = os.Getenv("AUTH_URL")
	}
	if len(*relay) > 0 || len(*acmeRelay) > 0 {
		if "" == *authURL {
			*authURL = strings.Replace(*relay, "ws", "http", 1) // "https://example.com:443"
		}
		// TODO look at relay rather than authURL?
		grants, err := telebit.Inspect(*authURL, *token)
		if nil != err {
			_, err := mgmt.Register(*authURL, *secret, ppid)
			if nil != err {
				fmt.Fprintf(os.Stderr, "failed to register client: %s\n", err)
				os.Exit(1)
			}
			grants, err = telebit.Inspect(*authURL, *token)
			if nil != err {
				fmt.Fprintf(os.Stderr, "failed to authenticate after registering client: %s\n", err)
				os.Exit(1)
			}
		}
		fmt.Println("grants", grants)
	}
	authorizer = NewAuthorizer(*authURL)

	provider, err := getACMEProvider(acmeRelay, token)
	if nil != err {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
		return
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

	//mux := telebit.NewRouteMux(acme)
	mux := telebit.NewRouteMux()

	// Port forward without TerminatingTLS
	for _, fwd := range portForwards {
		fmt.Println("Fwd:", fwd.pattern, fwd.port)
		mux.ForwardTCP(fwd.pattern, "localhost:"+fwd.port, 120*time.Second)
	}
	// TODO close connection on invalid hostname
	mux.HandleTCP("*", telebit.HandlerFunc(routeSubscribersAndClients))
	mux.HandleTLS("*", acme, mux)

	if 0 == len(*apiHostname) {
		*apiHostname = os.Getenv("API_HOSTNAME")
	}
	if "" != *apiHostname {
		listener := httpshim.NewListener()
		go func() {
			httpsrv.Serve(listener)
		}()
		fmt.Printf("Will respond to Websocket and API requests to %q\n", *apiHostname)
		mux.HandleTCP(*apiHostname, telebit.HandlerFunc(func(client net.Conn) error {
			fmt.Printf("[debug] Accepting API or WebSocket client %q\n", *apiHostname)
			listener.Feed(client)
			fmt.Printf("[debug] done with %q client\n", *apiHostname)
			// nil now means handler in-progress (go routine)
			// EOF now means handler finished
			return nil
		}))
	}
	for _, fwd := range forwards {
		//mux.ForwardTCP("*", "localhost:"+fwd.port, 120*time.Second)
		mux.ForwardTCP(fwd.pattern, "localhost:"+fwd.port, 120*time.Second)
	}

	done := make(chan error)
	for _, addr := range bindAddrs {
		go func(addr string) {
			fmt.Printf("Listening on %s\n", addr)
			ln, err := net.Listen("tcp", addr)
			if nil != err {
				fmt.Fprintf(os.Stderr, "failed to bind to %q: %s", addr, err)
				done <- err
				return
			}
			if err := telebit.Serve(ln, mux); nil != err {
				fmt.Fprintf(os.Stderr, "failed to bind to %q: %s", addr, err)
				done <- err
				return
			}
		}(addr)
	}

	//connected := make(chan net.Conn)
	go func() {
		if "" == *relay {
			return
		}

		timeoutCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
		defer cancel()
		tun, err := telebit.DialWebsocketTunnel(timeoutCtx, *relay, *token)
		if nil != err {
			msg := ""
			if strings.Contains(err.Error(), "bad handshake") {
				msg = " (may be auth related)"
			}
			fmt.Fprintf(os.Stderr, "Error connecting to %s: %s%s\n", *relay, err, msg)
			os.Exit(1)
			return
		}

		err = mgmt.Ping(*authURL, *token)
		if nil != err {
			fmt.Fprintf(os.Stderr, "failed to ping mgmt server: %s\n", err)
			//os.Exit(1)
		}

		go func() {
			for {
				time.Sleep(10 * time.Minute)
				err = mgmt.Ping(*authURL, *token)
				if nil != err {
					fmt.Fprintf(os.Stderr, "failed to ping mgmt server: %s\n", err)
					//os.Exit(1)
				}
			}
		}()
		//connected <- tun
		//tun := <-connected
		fmt.Printf("Listening through %s\n", *relay)
		err = telebit.ListenAndServe(tun, mux)
		log.Fatal("Closed server: ", err)
		done <- err
	}()

	if err := <-done; nil != err {
		os.Exit(1)
	}
}

func routeSubscribersAndClients(client net.Conn) error {
	var wconn *telebit.ConnWrap
	switch conn := client.(type) {
	case *telebit.ConnWrap:
		wconn = conn
	default:
		panic("HandleTun is special in that it must receive &ConnWrap{ Conn: conn }")
	}

	// We know this to be two parts "ip:port"
	dstParts := strings.Split(client.LocalAddr().String(), ":")
	//dstAddr := dstParts[0]
	dstPort, _ := strconv.Atoi(dstParts[1])

	if 80 != dstPort && 443 != dstPort {
		// TODO handle by port without peeking at Servername / Hostname
		// if tryToServePort(client.LocalAddr().String(), wconn) {
		//   return io.EOF
		// }
	}

	// TODO hostname for plain http?
	servername := strings.ToLower(wconn.Servername())
	if "" != servername && !isHostname(servername) {
		_ = client.Close()
		fmt.Println("[debug] invalid servername")
		return fmt.Errorf("invalid servername")
	}

	// Match full servername "sub.domain.example.com"
	if tryToServeName(servername, wconn) {
		// TODO better non-error
		return nil
	}

	// Match wild names
	// - "*.domain.example.com"
	// - "*.example.com"
	// - (skip)
	labels := strings.Split(servername, ".")
	n := len(labels)
	if n < 3 {
		// skip
		return telebit.ErrNotHandled
	}
	for i := 1; i < n-1; i++ {
		wildname := "*." + strings.Join(labels[1:], ".")
		if tryToServeName(wildname, wconn) {
			return io.EOF
		}
	}

	// skip
	return telebit.ErrNotHandled
}

// tryToServeName picks the server tunnel with the least connections, if any
func tryToServeName(servername string, wconn *telebit.ConnWrap) bool {
	srv, ok := table.GetServer(servername)
	if !ok {
		fmt.Println("[debug] no server to server", servername)
		return false
	}

	// async so that the call stack can complete and be released
	//srv.clients.Store(wconn.LocalAddr().String(), wconn)
	go func() {
		err := srv.Serve(wconn)
		fmt.Printf("a browser client stream is done: %q\n", err)
		//srv.clients.Delete(wconn.LocalAddr().String())
	}()

	return true
}

func parsePortForwards(portToPorts *string) ([]Forward, error) {
	var portForwards []Forward

	for _, cfg := range strings.Fields(strings.ReplaceAll(*portToPorts, ",", " ")) {
		parts := strings.Split(cfg, ":")
		if 2 != len(parts) {
			return nil, fmt.Errorf("--port-forward should be in the format 1234:5678, not %q", cfg)
		}

		if _, err := strconv.Atoi(parts[0]); nil != err {
			return nil, fmt.Errorf("couldn't parse port %q of %q", parts[0], cfg)
		}
		if _, err := strconv.Atoi(parts[1]); nil != err {
			return nil, fmt.Errorf("couldn't parse port %q of %q", parts[1], cfg)
		}

		portForwards = append(portForwards, Forward{
			pattern: ":" + parts[0],
			port:    parts[1],
		})
	}

	return portForwards, nil
}

func parseBindAddrs(bindAddrsStr string) ([]string, error) {
	bindAddrs := []string{}

	for _, addr := range strings.Fields(strings.ReplaceAll(bindAddrsStr, ",", " ")) {
		parts := strings.Split(addr, ":")
		if len(parts) > 2 {
			return nil, fmt.Errorf("too many colons (:) in bind address %s", addr)
		}
		if "" == addr || "" == parts[0] {
			continue
		}

		var hostname, port string
		if 2 == len(parts) {
			hostname = parts[0]
			port = parts[1]
		} else {
			port = parts[0]
		}

		if _, err := strconv.Atoi(port); nil != err {
			return nil, fmt.Errorf("couldn't parse port of %q", addr)
		}
		bindAddrs = append(bindAddrs, hostname+":"+port)
	}

	return bindAddrs, nil
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

type ACMEProvider struct {
	BaseURL  string
	provider challenge.Provider
}

func (p *ACMEProvider) Present(domain, token, keyAuth string) error {
	return p.provider.Present(domain, token, keyAuth)
}

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
