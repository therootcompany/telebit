//go:generate go run -mod=vendor git.rootprojects.org/root/go-gitver

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"git.rootprojects.org/root/telebit/mgmt/authstore"

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

// MWKey is a type guard
type MWKey string

var store authstore.Store
var provider challenge.Provider = nil // TODO is this concurrency-safe?
var secret string
var primaryDomain string
var relayDomain string

func help() {
	fmt.Fprintf(os.Stderr, "Usage: mgmt --domain <devices.example.com> --secret <128-bit secret>\n")
}

func main() {
	var err error

	addr := flag.String("address", "", "IPv4 or IPv6 bind address")
	port := flag.String("port", "3000", "port to listen to")
	challengesPort := flag.String("challenges-port", "80", "port to use to respond to .well-known/acme-challenge tokens")
	dbURL := flag.String(
		"db-url",
		"postgres://postgres:postgres@localhost/postgres",
		"database (postgres) connection url",
	)
	flag.StringVar(&secret, "secret", "", "a >= 16-character random string for JWT key signing")
	flag.StringVar(&primaryDomain, "domain", "", "the base domain to use for all clients")
	flag.StringVar(&relayDomain, "tunnel-domain", "", "the domain name of the tunnel relay service, if different from base domain")
	flag.Parse()

	if "" == primaryDomain {
		help()
		os.Exit(1)
	}
	if "" == relayDomain {
		relayDomain = primaryDomain
	}

	if "" != os.Getenv("GODADDY_API_KEY") {
		id := os.Getenv("GODADDY_API_KEY")
		apiSecret := os.Getenv("GODADDY_API_SECRET")
		if provider, err = newGoDaddyDNSProvider(id, apiSecret); nil != err {
			panic(err)
		}
	} else if "" != os.Getenv("DUCKDNS_TOKEN") {
		if provider, err = newDuckDNSProvider(os.Getenv("DUCKDNS_TOKEN")); nil != err {
			panic(err)
		}
	} else {
		panic("Must provide either DUCKDNS or GODADDY credentials")
	}

	if "" == secret {
		secret = os.Getenv("SECRET")
	}
	if "" == secret {
		help()
		os.Exit(1)
		return
	}

	connStr := *dbURL
	// TODO url.Parse
	if strings.Contains(connStr, "@localhost/") || strings.Contains(connStr, "@localhost:") {
		connStr += "?sslmode=disable"
	} else {
		connStr += "?sslmode=required"
	}

	store, err = authstore.NewStore(connStr, initSQL)
	if nil != err {
		log.Fatal("connection error", err)
		return
	}
	_ = store.SetMaster(secret)
	defer store.Close()

	go func() {
		fmt.Println("Listening for ACME challenges on :" + *challengesPort)
		if err := http.ListenAndServe(":"+*challengesPort, routeStatic()); nil != err {
			log.Fatal(err)
			os.Exit(1)
		}
	}()

	bind := *addr + ":" + *port
	fmt.Println("Listening on", bind)
	fmt.Fprintf(os.Stderr, "failed: %s", http.ListenAndServe(bind, routeAll()))
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
