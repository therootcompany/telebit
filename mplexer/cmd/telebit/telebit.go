package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	telebit "git.coolaj86.com/coolaj86/go-telebitd/mplexer"

	"github.com/caddyserver/certmagic"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/providers/dns/duckdns"
	"github.com/go-acme/lego/v3/providers/dns/godaddy"
	_ "github.com/joho/godotenv/autoload"
)

type Forward struct {
	scheme  string
	pattern string
	port    string
}

func main() {
	var err error
	var provider challenge.Provider = nil
	var enableTLSALPN01 bool
	var domains []string
	var forwards []Forward

	// TODO replace the websocket connection with a mock server
	email := flag.String("acme-email", "", "email to use for Let's Encrypt / ACME registration")
	certpath := flag.String("acme-storage", "./acme.d/", "path to ACME storage directory")
	acmeAgree := flag.Bool("acme-agree", false, "agree to the terms of the ACME service provider (required)")
	acmeStaging := flag.Bool("acme-staging", false, "get fake certificates for testing")
	acmeDirectory := flag.String("acme-directory", "", "ACME Directory URL")
	enableHTTP01 := flag.Bool("acme-http-01", false, "enable HTTP-01 ACME challenges")
	relay := flag.String("relay", "", "the domain (or ip address) at which the relay server is running")
	secret := flag.String("secret", "", "the same secret used by telebit-relay (used for JWT authentication)")
	token := flag.String("token", "", "a pre-generated token to give the server (instead of generating one with --secret)")
	locals := flag.String("locals", "", "a list of <from-domain>:<to-port>")
	flag.Parse()

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
		enableTLSALPN01 = true
	}

	if "" == *relay {
		*relay = os.Getenv("RELAY") // "wss://example.com:443"
	}
	if "" == *token {
		if "" == *secret {
			*secret = os.Getenv("SECRET")
		}
		*token, err = getToken(*secret, domains)
	}
	if nil != err {
		panic(err)
	}

	ctx := context.Background()

	acme := &telebit.ACME{
		Email:                  *email,
		StoragePath:            *certpath,
		Agree:                  *acmeAgree,
		Directory:              *acmeDirectory,
		DNSProvider:            provider,
		EnableHTTPChallenge:    *enableHTTP01,
		EnableTLSALPNChallenge: enableTLSALPN01,
	}

	mux := telebit.NewRouteMux()
	mux.HandleTLS("*", acme, mux)
	for _, fwd := range forwards {
		mux.ForwardTCP("*", "localhost:"+fwd.port, 120*time.Second)
		//mux.ForwardTCP(fwd.pattern, "localhost:"+fwd.port, 120*time.Second)
	}

	tun, err := telebit.DialWebsocketTunnel(ctx, *relay, *token)
	if nil != err {
		fmt.Println("relay:", relay)
		log.Fatal(err)
		return
	}

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
