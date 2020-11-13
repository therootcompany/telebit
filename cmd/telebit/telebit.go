//go:generate go run -mod=vendor git.rootprojects.org/root/go-gitver/v2

package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.rootprojects.org/root/telebit"
	"git.rootprojects.org/root/telebit/dbg"
	"git.rootprojects.org/root/telebit/internal/dns01"
	"git.rootprojects.org/root/telebit/internal/http01"
	"git.rootprojects.org/root/telebit/internal/mgmt"
	"git.rootprojects.org/root/telebit/internal/mgmt/authstore"
	"git.rootprojects.org/root/telebit/internal/service"
	telebitX "git.rootprojects.org/root/telebit/internal/telebit"
	"git.rootprojects.org/root/telebit/iplist"
	"git.rootprojects.org/root/telebit/table"
	"git.rootprojects.org/root/telebit/tunnel"

	"github.com/coolaj86/certmagic"
	"github.com/denisbrodbeck/machineid"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-acme/lego/v3/challenge"
	legoDNS01 "github.com/go-acme/lego/v3/challenge/dns01"
	"github.com/go-acme/lego/v3/providers/dns/duckdns"
	"github.com/go-acme/lego/v3/providers/dns/godaddy"
	"github.com/go-acme/lego/v3/providers/dns/namedotcom"
	"github.com/go-chi/chi"
	"github.com/joho/godotenv"
	_ "github.com/joho/godotenv/autoload"
)

const (
	// exitOk is for normal exits, such as a graceful disconnect or shutdown
	exitOk = 0

	// exitBadArguments is for positive failures as a result of arguments
	exitBadArguments = 1

	// exitBadConfig is for positive failures from an external service
	exitBadConfig = 2

	// exitRetry is for potentially false negative failures from temporary
	// conditions such as a DNS resolution or network failure for which it would
	// be reasonable to wait 10 seconds and try again
	exitRetry = 29
)

var (
	// commit refers to the abbreviated commit hash
	commit = "0000000"
	// version refers to the most recent tag, plus any commits made since then
	version = "v0.0.0-pre0+0000000"
	// GitTimestamp refers to the timestamp of the most recent commit
	date = "0000-00-00T00:00:00+0000"

	// serviceName is the service name
	serviceName = "telebit"

	// serviceDesc
	serviceDesc = "securely relay traffic through telebit.io"
)

// Forward describes how to route a network connection
type Forward struct {
	scheme   string
	pattern  string
	port     string
	localTLS bool
}

var isHostname = regexp.MustCompile(`^[A-Za-z0-9_\.\-]+$`).MatchString

// VendorID may be baked in, or supplied via ENVs or --args
var VendorID string

// ClientSecret may be baked in, or supplied via ENVs or --args
var ClientSecret string

func main() {
	if len(os.Args) >= 2 {
		if "version" == strings.TrimLeft(os.Args[1], "-") {
			fmt.Printf("telebit %s (%s) %s\n", version, commit[:7], date)
			os.Exit(exitOk)
			return
		}
	}

	if len(os.Args) >= 2 {
		if "install" == os.Args[1] {
			if err := service.Install(serviceName, serviceDesc); nil != err {
				fmt.Fprintf(os.Stderr, "%v", err)
			}
			return
		}
	}

	var domains []string
	var forwards []Forward
	var portForwards []Forward
	var resolvers []string

	debug := flag.Bool("debug", true, "show debug output")
	spfDomain := flag.String("spf-domain", "", "domain with SPF-like list of IP addresses which are allowed to connect to clients")
	// TODO replace the websocket connection with a mock server
	vendorID := flag.String("vendor-id", "", "a unique identifier for a deploy target environment")
	envpath := flag.String("env", "", "path to .env file")
	email := flag.String("acme-email", "", "email to use for Let's Encrypt / ACME registration")
	certpath := flag.String("acme-storage", "./acme.d/", "path to ACME storage directory")
	acmeAgree := flag.Bool("acme-agree", false, "agree to the terms of the ACME service provider (required)")
	acmeStaging := flag.Bool("acme-staging", false, "get fake certificates for testing")
	acmeDirectory := flag.String("acme-directory", "", "ACME Directory URL")
	enableHTTP01 := flag.Bool("acme-http-01", false, "enable HTTP-01 ACME challenges")
	enableTLSALPN01 := flag.Bool("acme-tls-alpn-01", false, "enable TLS-ALPN-01 ACME challenges")
	acmeRelay := flag.String("acme-relay-url", "", "the base url of the ACME DNS-01 relay, if not the same as the tunnel relay")
	acmeHTTP01Relay := flag.String("acme-http-01-relay-url", "", "the base url of the ACME HTTP-01 relay, if not the same as the DNS-01 relay")
	var dnsPropagationDelay time.Duration
	flag.DurationVar(&dnsPropagationDelay, "dns-01-delay", 0, "add an extra delay after dns self-check to allow DNS-01 challenges to propagate")
	resolverList := flag.String("dns-resolvers", "", "a list of resolvers in the format 8.8.8.8:53,8.8.4.4:53")
	authURL := flag.String("auth-url", "", "the base url for authentication, if not the same as the tunnel relay")
	relay := flag.String("tunnel-relay-url", "", "the websocket url at which to connect to the tunnel relay")
	apiHostname := flag.String("api-hostname", "", "the hostname used to manage clients")
	secret := flag.String("secret", "", "the same secret used by telebit-relay (used for JWT authentication)")
	token := flag.String("token", "", "an auth token for the server (instead of generating --secret); use --token=false to ignore any $TOKEN in env")
	leeway := flag.Duration("leeway", 15*time.Minute, "allow for time drift / skew (hard-coded to 15 minutes)")
	bindAddrsStr := flag.String("listen", "", "list of bind addresses on which to listen, such as localhost:80, or :443")
	tlsLocals := flag.String("tls-locals", "", "like --locals, but TLS will be used to connect to the local port")
	locals := flag.String("locals", "", "a list of <from-domain>:<to-port>")
	portToPorts := flag.String("port-forward", "", "a list of <from-port>:<to-port> for raw port-forwarding")
	verbose := flag.Bool("verbose", false, "log excessively")
	flag.Parse()

	if !dbg.Debug {
		dbg.Debug = *verbose || *debug
	}

	if len(*envpath) > 0 {
		if err := godotenv.Load(*envpath); nil != err {
			fmt.Fprintf(os.Stderr, "%v", err)
			os.Exit(exitBadArguments)
			return
		}
	}

	spfRecords := iplist.Init(*spfDomain)
	if len(spfRecords) > 0 {
		fmt.Println(
			"Allow client connections from:",
			strings.Join(spfRecords, " "),
		)
	}

	if len(*acmeDirectory) > 0 {
		if *acmeStaging {
			fmt.Fprintf(os.Stderr, "pick either acme-directory or acme-staging\n")
			os.Exit(exitBadArguments)
			return
		}
	}
	if *acmeStaging {
		*acmeDirectory = certmagic.LetsEncryptStagingCA
	}
	if !*acmeAgree {
		if "true" == os.Getenv("ACME_AGREE") {
			*acmeAgree = true
		}
	}
	if 0 == len(*acmeRelay) {
		*acmeRelay = os.Getenv("ACME_RELAY_URL")
	}
	if 0 == len(*acmeHTTP01Relay) {
		*acmeHTTP01Relay = os.Getenv("ACME_HTTP_01_RELAY_URL")
	}

	if 0 == len(*email) {
		*email = os.Getenv("ACME_EMAIL")
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

	if 0 == len(*tlsLocals) {
		*tlsLocals = os.Getenv("TLS_LOCALS")
	}
	for _, cfg := range strings.Fields(strings.ReplaceAll(*tlsLocals, ",", " ")) {
		parts := strings.Split(cfg, ":")
		last := len(parts) - 1
		port := parts[last]
		domain := parts[last-1]
		scheme := ""
		if len(parts) > 2 {
			scheme = parts[0]
		}
		forwards = append(forwards, Forward{
			scheme:   scheme,
			pattern:  domain,
			port:     port,
			localTLS: true,
		})

		// don't load wildcard into jwt domains
		if "*" == domain {
			continue
		}
		domains = append(domains, domain)
	}

	var err error
	if 0 == dnsPropagationDelay {
		dnsPropagationDelay, err = time.ParseDuration(os.Getenv("DNS_01_DELAY"))
	}
	if 0 == dnsPropagationDelay {
		dnsPropagationDelay = 5 * time.Second
	}

	if 0 == len(*resolverList) {
		*resolverList = os.Getenv("DNS_RESOLVERS")
	}
	if len(*resolverList) > 0 {
		for _, resolver := range strings.Fields(strings.ReplaceAll(*resolverList, ",", " ")) {
			resolvers = append(resolvers, resolver)
		}
		legoDNS01.AddRecursiveNameservers(resolvers)
	}

	if 0 == len(*portToPorts) {
		*portToPorts = os.Getenv("PORT_FORWARDS")
	}
	portForwards, err = parsePortForwards(portToPorts)
	if nil != err {
		fmt.Fprintf(os.Stderr, "%s", err)
		os.Exit(exitBadArguments)
		return
	}

	if 0 == len(*bindAddrsStr) {
		*bindAddrsStr = os.Getenv("LISTEN")
	}
	bindAddrs, err := parseBindAddrs(*bindAddrsStr)
	if nil != err {
		fmt.Fprintf(os.Stderr, "invalid bind address(es) given to --listen\n")
		os.Exit(exitBadArguments)
		return
	}

	// Baked-in takes precedence
	if 0 == len(VendorID) {
		VendorID = *vendorID
	} else if 0 != len(*vendorID) {
		if VendorID != *vendorID {
			fmt.Fprintf(os.Stderr, "invalid --vendor-id\n")
			os.Exit(exitBadArguments)
		}
	}
	if 0 == len(VendorID) {
		VendorID = os.Getenv("VENDOR_ID")
	} else if 0 != len(os.Getenv("VENDOR_ID")) {
		if VendorID != os.Getenv("VENDOR_ID") {
			fmt.Fprintf(os.Stderr, "invalid VENDOR_ID\n")
			os.Exit(exitBadArguments)
		}
	}
	if 0 == len(ClientSecret) {
		ClientSecret = *secret
	} else if 0 != len(*secret) {
		if ClientSecret != *secret {
			fmt.Fprintf(os.Stderr, "invalid --secret\n")
			os.Exit(exitBadArguments)
		}
	}
	if 0 == len(ClientSecret) {
		ClientSecret = os.Getenv("SECRET")
	} else if 0 != len(os.Getenv("SECRET")) {
		if ClientSecret != os.Getenv("SECRET") {
			fmt.Fprintf(os.Stderr, "invalid SECRET\n")
			os.Exit(exitBadArguments)
		}
	}
	ppid, err := machineid.ProtectedID(fmt.Sprintf("%s|%s", VendorID, ClientSecret))
	if nil != err {
		fmt.Fprintf(os.Stderr, "unauthorized device\n")
		os.Exit(exitBadConfig)
		return
	}
	ppidBytes, err := hex.DecodeString(ppid)
	ppid = base64.RawURLEncoding.EncodeToString(ppidBytes)

	if 0 == len(*token) {
		*token = os.Getenv("TOKEN")
	}
	if "false" == *token {
		*token = ""
	}
	if 0 == len(*token) {
		*token, err = authstore.HMACToken(ppid, *leeway)
		if dbg.Debug {
			fmt.Printf("[debug] app_id: %q\n", VendorID)
			//fmt.Printf("[debug] client_secret: %q\n", ClientSecret)
			//fmt.Printf("[debug] ppid: %q\n", ppid)
			//fmt.Printf("[debug] ppid: [redacted]\n")
			fmt.Printf("[debug] token: %q\n", *token)
		}
		if nil != err {
			fmt.Fprintf(os.Stderr, "neither secret nor token provided\n")
			os.Exit(exitBadArguments)
			return
		}
	}
	if 0 == len(*relay) {
		*relay = os.Getenv("TUNNEL_RELAY_URL") // "wss://example.com:443"
	}
	if 0 == len(*relay) {
		if len(bindAddrs) > 0 {
			fmt.Fprintf(os.Stderr, "Acting as Relay\n")
		} else {
			fmt.Fprintf(os.Stderr, "error: must provide Relay, or act as Relay\n")
			os.Exit(exitBadArguments)
			return
		}
	}

	if 0 == len(*authURL) {
		*authURL = os.Getenv("AUTH_URL")
	}
	var grants *telebit.Grants
	if len(*relay) > 0 /* || len(*acmeRelay) > 0 */ {
		directory, err := tunnel.Discover(*relay)
		if nil != err {
			fmt.Fprintf(os.Stderr, "Error: invalid Tunnel Relay URL %q: %s\n", *relay, err)
			os.Exit(exitRetry)
		}
		fmt.Printf("[Directory] %s\n", *relay)
		jsonb, _ := json.Marshal(directory)
		fmt.Printf("\t%s\n", string(jsonb))

		// TODO trimming this should no longer be necessary, but I need to double check
		authBase := strings.TrimSuffix(directory.Authenticate.URL, "/inspect")
		if "" == *authURL {
			*authURL = authBase
		} else {
			fmt.Println("Suggested Auth URL:", authBase)
			fmt.Println("--auth-url Auth URL:", *authURL)
		}
		if "" == *authURL {
			fmt.Fprintf(os.Stderr, "Discovered Directory Endpoints: %+v\n", directory)
			fmt.Fprintf(os.Stderr, "No Auth URL detected, nor supplied\n")
			os.Exit(exitBadConfig)
			return
		}
		fmt.Println("Auth URL", *authURL)

		dns01Base := directory.DNS01Proxy.URL
		if 0 == len(*acmeRelay) {
			*acmeRelay = dns01Base
		} else {
			fmt.Println("Suggested ACME DNS 01 Proxy URL:", dns01Base)
			fmt.Println("--acme-relay-url ACME DNS 01 Proxy URL:", *acmeRelay)
		}
		if 0 == len(*acmeHTTP01Relay) {
			*acmeHTTP01Relay = directory.HTTP01Proxy.URL
		} else {
			fmt.Println("Suggested ACME HTTP 01 Proxy URL:", dns01Base)
			fmt.Println("--acme-http-01-relay-url ACME DNS 01 Proxy URL:", *acmeRelay)
		}

		grants, err = telebit.Inspect(*authURL, *token)
		if nil != err {
			if dbg.Debug {
				fmt.Fprintf(os.Stderr, "failed to inspect token: %s\n", err)
			}
			_, err := mgmt.Register(*authURL, ClientSecret, ppid)
			if nil != err {
				if strings.Contains(err.Error(), `"E_NOT_FOUND"`) {
					fmt.Fprintf(os.Stderr, "invalid client credentials: %s\n", err)
					// the server confirmed that the client is bad
					os.Exit(exitBadConfig)
				} else {
					// there may have been a network error
					fmt.Fprintf(os.Stderr, "failed to register client: %s\n", err)
					os.Exit(exitRetry)
				}
				return
			}
			grants, err = telebit.Inspect(*authURL, *token)
			if nil != err {
				fmt.Fprintf(os.Stderr, "failed to authenticate after registering client: %s\n", err)
				// there was no error registering the client, yet there was one authenticating
				// therefore this may be an error that will  be resolved
				os.Exit(exitRetry)
			}
		}
		fmt.Printf("[Grants]\n\t%#v\n", grants)
		*relay = grants.Audience
	}

	fmt.Printf("Email: %q\n", *email)

	acme := &telebit.ACME{
		Email:                  *email,
		StoragePath:            *certpath,
		Agree:                  *acmeAgree,
		Directory:              *acmeDirectory,
		EnableTLSALPNChallenge: *enableTLSALPN01,
	}

	// TODO
	// Blog about the stupidity of this typing
	// var dns01Solver *dns01.Solver = nil
	if len(*acmeRelay) > 0 {
		provider, err := getACMEProvider(acmeRelay, token)
		if nil != err {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			// it's possible for some providers this could be a failed network request,
			// but I think in the case of what we specifically support it's bad arguments
			os.Exit(exitBadArguments)
			return
		}
		/*
			options: legoDNS01.WrapPreCheck(func(domain, fqdn, value string, orig legoDNS01.PreCheckFunc) (bool, error) {
				ok, err := orig(fqdn, value)
				if ok && dnsPropagationDelay > 0 {
					fmt.Printf("[Telebit-ACME-DNS] sleeping an additional %s\n", dnsPropagationDelay)
					time.Sleep(dnsPropagationDelay)
				}
				return ok, err
			}),
		*/
		//DNSChallengeOption:     legoDNS01.DNSProviderOption,
		acme.DNS01Solver = dns01.NewSolver(provider)
		fmt.Println("Using DNS-01 solver for ACME Challenges")
	}

	if *enableHTTP01 {
		acme.EnableHTTPChallenge = true
	}
	if len(*acmeHTTP01Relay) > 0 {
		acme.EnableHTTPChallenge = true
		endpoint, err := url.Parse(*acmeHTTP01Relay)
		if nil != err {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(exitBadArguments)
			return
		}
		http01Solver, err := http01.NewSolver(&http01.Config{
			Endpoint: endpoint,
			Token:    *token,
		})

		acme.HTTP01Solver = http01Solver
		fmt.Println("Using HTTP-01 solver for ACME Challenges")
	}

	if nil == acme.HTTP01Solver && nil == acme.DNS01Solver {
		fmt.Fprintf(os.Stderr, "Neither ACME HTTP 01 nor DNS 01 proxy URL detected, nor supplied\n")
		os.Exit(1)
		return
	}

	mux := muxAll(portForwards, forwards, acme, apiHostname, authURL, grants)

	done := make(chan error)
	if dbg.Debug {
		fmt.Println("[debug] bindAddrs", bindAddrs, *bindAddrsStr)
	}
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
			os.Exit(exitRetry)
			return
		}

		err = mgmt.Ping(*authURL, *token)
		if nil != err {
			fmt.Fprintf(os.Stderr, "failed to ping mgmt server: %s\n", err)
			//os.Exit(exitRetry)
		}

		go func() {
			for {
				time.Sleep(10 * time.Minute)
				if "" != ClientSecret {
					// re-create token unless no secret was supplied
					*token, err = authstore.HMACToken(ppid, *leeway)
				}
				err = mgmt.Ping(*authURL, *token)
				if nil != err {
					fmt.Fprintf(os.Stderr, "failed to ping mgmt server: %s\n", err)
					//os.Exit(exitRetry)
				}
			}
		}()
		//connected <- tun
		//tun := <-connected
		fmt.Printf("Listening through %s\n", *relay)
		err = telebit.ListenAndServe(tun, mux)
		fmt.Fprintf(os.Stderr, "Closed server: %s\n", err)
		os.Exit(exitRetry)
		done <- err
	}()

	if err := <-done; nil != err {
		os.Exit(exitRetry)
	}
}

func muxAll(
	portForwards, forwards []Forward,
	acme *telebit.ACME,
	apiHostname, authURL *string,
	grants *telebit.Grants,
) *telebit.RouteMux {
	//mux := telebit.NewRouteMux(acme)
	mux := telebit.NewRouteMux()

	// Port forward without TerminatingTLS
	for _, fwd := range portForwards {
		msg := fmt.Sprintf("Fwd: %s %s", fwd.pattern, fwd.port)
		fmt.Println(msg)
		mux.ForwardTCP(fwd.pattern, "localhost:"+fwd.port, 120*time.Second, msg, "[Port Forward]")
	}

	if 0 == len(*apiHostname) {
		*apiHostname = os.Getenv("API_HOSTNAME")
	}
	if "" != *apiHostname {
		// this is a generic net listener
		r := chi.NewRouter()
		telebitX.RouteAdmin(*authURL, r)
		apiListener := tunnel.NewListener()
		go func() {
			httpsrv := &http.Server{Handler: r}
			httpsrv.Serve(apiListener)
		}()
		fmt.Printf("Will respond to Websocket and API requests to %q\n", *apiHostname)
		mux.HandleTLS(*apiHostname, acme, mux, "[Terminate TLS & Recurse] for "+*apiHostname)
		mux.HandleTCP(*apiHostname, telebit.HandlerFunc(func(client net.Conn) error {
			if dbg.Debug {
				fmt.Printf("[debug] Accepting API or WebSocket client %q\n", *apiHostname)
			}
			apiListener.Feed(client)
			if dbg.Debug {
				fmt.Printf("[debug] done with %q client\n", *apiHostname)
			}
			// nil now means handler in-progress (go routine)
			// EOF now means handler finished
			return nil
		}), "[Admin API & Server Relays]")
	}

	// TODO close connection on invalid hostname
	mux.HandleTCP("*", telebit.HandlerFunc(routeSubscribersAndClients), "[Tun => Remote Servers]")

	if nil != grants {
		for i, domainname := range grants.Domains {
			fmt.Printf("[%d] Will decrypt remote requests to %q\n", i, domainname)
			mux.HandleTLS(domainname, acme, mux, "[Terminate TLS & Recurse] for (tunnel) "+domainname)
		}
	}

	for i, fwd := range forwards {
		fmt.Printf("[%d] Will decrypt local requests to \"%s://%s\"\n", i, fwd.scheme, fwd.pattern)
		mux.HandleTLS(fwd.pattern, acme, mux, "[Terminate TLS & Recurse] for (local) "+fwd.pattern)
	}

	//mux.HandleTLSFunc(func (sni) bool {
	//	// do whatever
	//	return false
	//}, acme, mux, "[Terminate TLS & Recurse]")
	for _, fwd := range forwards {
		//mux.ForwardTCP("*", "localhost:"+fwd.port, 120*time.Second)
		if "https" == fwd.scheme {
			if fwd.localTLS {
				// this doesn't make much sense, but... security theatre
				mux.ReverseProxyHTTPS(fwd.pattern, "localhost:"+fwd.port, 120*time.Second, "[Servername Reverse Proxy TLS]")
			} else {
				mux.ReverseProxyHTTP(fwd.pattern, "localhost:"+fwd.port, 120*time.Second, "[Servername Reverse Proxy]")
			}
		}
		mux.ForwardTCP(fwd.pattern, "localhost:"+fwd.port, 120*time.Second, "[Servername Forward]")
	}

	return mux
}

func routeSubscribersAndClients(client net.Conn) error {
	var wconn *telebit.ConnWrap
	switch conn := client.(type) {
	case *telebit.ConnWrap:
		wconn = conn
	default:
		panic("routeSubscribersAndClients is special in that it must receive &ConnWrap{ Conn: conn }")
	}

	// We know this to be two parts "ip:port"
	dstParts := strings.Split(client.LocalAddr().String(), ":")
	//dstAddr := dstParts[0]
	dstPort, _ := strconv.Atoi(dstParts[1])

	if dbg.Debug {
		fmt.Printf("[debug] wconn.LocalAddr() %+v\n", wconn.LocalAddr())
		fmt.Printf("[debug] wconn.RemoteAddr() %+v\n", wconn.RemoteAddr())
	}

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

		if dbg.Debug {
			fmt.Println("[debug] invalid servername")
		}
		return fmt.Errorf("invalid servername")
	}

	if dbg.Debug {
		fmt.Printf("[debug] wconn.Servername() %+v\n", servername)
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
		wildname := "*." + strings.Join(labels[i:], ".")
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
	if !ok || nil == srv {
		if ok {
			// TODO BUG: Sometimes srv=nil & ok=true, which should not be possible
			fmt.Println("[bug] found 'srv=nil'", servername, srv)
		}
		if dbg.Debug {
			fmt.Println("[debug] no server to server", servername)
		}
		return false
	}
	// Note: timing can reveal if the client exists

	if allowAll, _ := iplist.IsAllowed(nil); !allowAll {
		addr := wconn.RemoteAddr()
		if nil == addr {
			// handled by denial
			wconn.Close()
			return true
		}

		// 192.168.1.100:2345
		// [::fe12]:2345
		remoteIP := addr.String()
		index := strings.LastIndex(remoteIP, ":")
		if index < 1 {
			// TODO how to handle unexpected invalid address?
			wconn.Close()
			return true
		}
		remoteIP = remoteIP[:index]

		fmt.Println("remote addr:", remoteIP)

		if "127.0.0.1" != remoteIP &&
			"::1" != remoteIP &&
			"localhost" != remoteIP {
			ipAddr := net.ParseIP(remoteIP)
			if nil == ipAddr {
				wconn.Close()
				return true
			}

			if ok, err := iplist.IsAllowed(ipAddr); !ok || nil != err {
				wconn.Close()
				return true
			}
		}
	}

	// async so that the call stack can complete and be released
	//srv.clients.Store(wconn.LocalAddr().String(), wconn)
	go func() {
		if dbg.Debug {
			fmt.Printf("[debug] found server to handle client:\n%#v\n", srv)
		}
		err := srv.Serve(wconn)
		if dbg.Debug {
			fmt.Printf("[debug] a browser client stream is done: %v\n", err)
		}
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
		if "" == addr {
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
	} else if "" != os.Getenv("NAMECOM_API_TOKEN") {
		if provider, err = newNameDotComDNSProvider(
			os.Getenv("NAMECOM_USERNAME"),
			os.Getenv("NAMECOM_API_TOKEN"),
		); nil != err {
			return nil, err
		}
	} else if "" != os.Getenv("DUCKDNS_TOKEN") {
		if provider, err = newDuckDNSProvider(os.Getenv("DUCKDNS_TOKEN")); nil != err {
			return nil, err
		}
	} else {
		if "" == *acmeRelay {
			return nil, fmt.Errorf("No relay for ACME DNS-01 challenges given to --acme-relay-url")
		}
		endpoint := *acmeRelay
		if !strings.HasSuffix(endpoint, "/") {
			endpoint += "/"
		}
		/*
			if strings.HasSuffix(endpoint, "/") {
				endpoint = endpoint[:len(endpoint)-1]
			}
			endpoint += "/api/dns/"
		*/
		if provider, err = newAPIDNSProvider(endpoint, *token); nil != err {
			return nil, err
		}
	}

	return provider, nil
}

// newNameDotComDNSProvider is for the sake of demoing the tunnel
func newNameDotComDNSProvider(username, apitoken string) (*namedotcom.DNSProvider, error) {
	config := namedotcom.NewDefaultConfig()
	config.Username = username
	config.APIToken = apitoken
	return namedotcom.NewDNSProviderConfig(config)
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
func newAPIDNSProvider(baseURL string, token string) (*dns01.DNSProvider, error) {
	config := dns01.NewDefaultConfig()
	config.Token = token
	endpoint, err := url.Parse(baseURL)
	if nil != err {
		return nil, err
	}
	config.Endpoint = endpoint
	return dns01.NewDNSProviderConfig(config)
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
