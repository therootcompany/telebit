//go:generate go run -mod=vendor git.rootprojects.org/root/go-gitver

package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"git.coolaj86.com/coolaj86/go-telebitd/mgmt"
	"git.coolaj86.com/coolaj86/go-telebitd/mgmt/authstore"
	telebit "git.coolaj86.com/coolaj86/go-telebitd/mplexer"
	"git.coolaj86.com/coolaj86/go-telebitd/mplexer/dns01"

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

func main() {
	var err error
	var provider challenge.Provider = nil
	var domains []string
	var forwards []Forward

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
	secret := flag.String("secret", "", "the same secret used by telebit-relay (used for JWT authentication)")
	token := flag.String("token", "", "a pre-generated token to give the server (instead of generating one with --secret)")
	locals := flag.String("locals", "", "a list of <from-domain>:<to-port>")
	flag.Parse()

	if len(os.Args) >= 2 {
		if "version" == os.Args[1] {
			fmt.Printf("telebit %s %s %s", GitVersion, GitRev[:7], GitTimestamp)
			os.Exit(0)
		}
	}

	if "" != *acmeDirectory {
		if *acmeStaging {
			fmt.Fprintf(os.Stderr, "pick either acme-directory or acme-staging\n")
			os.Exit(1)
		}
	}
	if *acmeStaging {
		*acmeDirectory = certmagic.LetsEncryptStagingCA
	}

	if "" == *locals {
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

	ppid, err := machineid.ProtectedID(fmt.Sprintf("%s|%s", *appID, *secret))
	if nil != err {
		fmt.Fprintf(os.Stderr, "unauthorized device")
		os.Exit(1)
	}
	ppidBytes, err := hex.DecodeString(ppid)
	ppid = base64.RawURLEncoding.EncodeToString(ppidBytes)

	if "" == *token {
		if "" == *secret {
			*secret = os.Getenv("SECRET")
		}
		*token, err = authstore.HMACToken(ppid)
	}
	if nil != err {
		fmt.Fprintf(os.Stderr, "neither secret nor token provided")
		os.Exit(1)
		return
	}

	if "" == *relay {
		*relay = os.Getenv("RELAY") // "wss://example.com:443"
	}
	if "" == *relay {
		fmt.Fprintf(os.Stderr, "Missing relay url")
		os.Exit(1)
		return
	}
	if "" == *acmeRelay {
		*acmeRelay = strings.Replace(*relay, "ws", "http", 1) // "https://example.com:443"
	}
	if "" == *authURL {
		*authURL = strings.Replace(*relay, "ws", "http", 1) // "https://example.com:443"
	}

	if "" != os.Getenv("GODADDY_API_KEY") {
		id := os.Getenv("GODADDY_API_KEY")
		secret := os.Getenv("GODADDY_API_SECRET")
		if provider, err = newGoDaddyDNSProvider(id, secret); nil != err {
			panic(err)
		}
	} else if "" != os.Getenv("DUCKDNS_TOKEN") {
		if provider, err = newDuckDNSProvider(os.Getenv("DUCKDNS_TOKEN")); nil != err {
			panic(err)
		}
	} else {
		endpoint := *acmeRelay
		if strings.HasSuffix(endpoint, "/") {
			endpoint = endpoint[:len(endpoint)-1]
		}
		//endpoint += "/api/dns/"
		if provider, err = newAPIDNSProvider(endpoint, *token); nil != err {
			panic(err)
		}
	}

	grants, err := telebit.Inspect(*authURL, *token)
	if nil != err {
		_, err := mgmt.Register(*authURL, *secret, ppid)
		if nil != err {
			fmt.Fprintf(os.Stderr, "failed to register client: %s", err)
			os.Exit(1)
		}
		grants, err = telebit.Inspect(*authURL, *token)
		if nil != err {
			fmt.Fprintf(os.Stderr, "failed to authenticate after registering client: %s", err)
			os.Exit(1)
		}
	}
	fmt.Println("grants", grants)

	acme := &telebit.ACME{
		Email:                  *email,
		StoragePath:            *certpath,
		Agree:                  *acmeAgree,
		Directory:              *acmeDirectory,
		DNSProvider:            provider,
		EnableHTTPChallenge:    *enableHTTP01,
		EnableTLSALPNChallenge: *enableTLSALPN01,
	}

	mux := telebit.NewRouteMux()
	mux.HandleTLS("*", acme, mux)
	for _, fwd := range forwards {
		mux.ForwardTCP("*", "localhost:"+fwd.port, 120*time.Second)
		//mux.ForwardTCP(fwd.pattern, "localhost:"+fwd.port, 120*time.Second)
	}

	connected := make(chan net.Conn)
	go func() {
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
			fmt.Fprintf(os.Stderr, "failed to ping mgmt server: %s", err)
			//os.Exit(1)
		}

		connected <- tun
	}()

	go func() {
		for {
			time.Sleep(10 * time.Minute)
			err = mgmt.Ping(*authURL, *token)
			if nil != err {
				fmt.Fprintf(os.Stderr, "failed to ping mgmt server: %s", err)
				//os.Exit(1)
			}
		}
	}()

	tun := <-connected
	fmt.Printf("Listening at %s\n", *relay)
	log.Fatal("Closed server: ", telebit.ListenAndServe(tun, mux))
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

func getToken(secret string, domains []string) (token string, err error) {
	tokenData := jwt.MapClaims{"domains": domains}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenData)
	if token, err = jwtToken.SignedString([]byte(secret)); err != nil {
		return "", err
	}
	return token, nil
}
