//go:generate go run -mod=vendor git.rootprojects.org/root/go-gitver/v2

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"git.rootprojects.org/root/telebit/internal/mgmt"
	"git.rootprojects.org/root/telebit/internal/mgmt/authstore"

	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/providers/dns/duckdns"
	"github.com/go-acme/lego/v3/providers/dns/godaddy"
	"github.com/go-acme/lego/v3/providers/dns/namedotcom"
	"github.com/go-chi/chi"

	_ "github.com/joho/godotenv/autoload"
)

var (
	// commit refers to the abbreviated commit hash
	commit = "0000000"
	// version refers to the most recent tag, plus any commits made since then
	version = "v0.0.0-pre0+0000000"
	// GitTimestamp refers to the timestamp of the most recent commit
	date = "0000-00-00T00:00:00+0000"

	// serviceName is the service name
	serviceName = "telebit-mgmt"

	// serviceDesc
	serviceDesc = "Telebit Device Management"
)

func ver() string {
	return fmt.Sprintf("%s v%s (%s) %s", serviceName, version, commit[:7], date)
}

var store authstore.Store
var secret string

func help() {
	fmt.Fprintf(os.Stderr, "Usage: mgmt --domain <devices.example.com> --secret <128-bit secret>\n")
}

func main() {
	if len(os.Args) >= 2 {
		if "version" == strings.TrimLeft(os.Args[1], "-") {
			fmt.Printf("telebit %s (%s) %s\n", version, commit[:7], date)
			os.Exit(0)
			return
		}
	}

	var err error

	var port string
	var lnAddr string
	var dbURL string
	var challengesPort string

	flag.StringVar(&port, "port", "",
		"port to listen to (default localhost 3000)")
	flag.StringVar(&lnAddr, "listen", "",
		"IPv4 or IPv6 bind address + port (instead of --port)")
	flag.StringVar(&challengesPort, "challenges-port", "",
		"port to use to respond to .well-known/acme-challenge tokens (should be 80, if used)")
	flag.StringVar(&dbURL, "db-url", "postgres://postgres:postgres@localhost:5432/postgres",
		"database (postgres) connection url")
	flag.StringVar(&secret, "secret", "",
		"a >= 16-character random string for JWT key signing")
	flag.StringVar(&mgmt.DeviceDomain, "domain", "",
		"the base domain to use for all clients")
	flag.StringVar(&mgmt.RelayDomain, "tunnel-domain", "",
		"the domain name of the tunnel relay service, if different from base domain")

	flag.Parse()

	if 0 == len(mgmt.DeviceDomain) {
		mgmt.DeviceDomain = os.Getenv("DOMAIN")
	}

	if 0 == len(mgmt.RelayDomain) {
		mgmt.RelayDomain = os.Getenv("TUNNEL_DOMAIN")
	}
	if 0 == len(mgmt.RelayDomain) {
		mgmt.RelayDomain = mgmt.DeviceDomain
	}

	if 0 == len(dbURL) {
		dbURL = os.Getenv("DB_URL")
	}

	if 0 == len(secret) {
		secret = os.Getenv("SECRET")
	}

	// prefer --listen (with address) over --port (localhost only)
	if 0 == len(lnAddr) {
		lnAddr = os.Getenv("LISTEN")
	}
	if 0 == len(lnAddr) {
		if 0 == len(port) {
			port = os.Getenv("PORT")
		}
		if 0 == len(port) {
			port = "3000"
		}
		lnAddr = "localhost:" + port
	}

	// TODO are these concurrency-safe?
	var provider challenge.Provider = nil
	if len(os.Getenv("GODADDY_API_KEY")) > 0 {
		id := os.Getenv("GODADDY_API_KEY")
		apiSecret := os.Getenv("GODADDY_API_SECRET")
		if provider, err = newGoDaddyDNSProvider(id, apiSecret); nil != err {
			panic(err)
		}
	} else if len(os.Getenv("DUCKDNS_TOKEN")) > 0 {
		if provider, err = newDuckDNSProvider(os.Getenv("DUCKDNS_TOKEN")); nil != err {
			panic(err)
		}
	} else if len(os.Getenv("NAMECOM_API_TOKEN")) > 0 {
		if provider, err = newNameDotComDNSProvider(
			os.Getenv("NAMECOM_USERNAME"),
			os.Getenv("NAMECOM_API_TOKEN"),
		); nil != err {
			panic(err)
		}
	} else {
		fmt.Println("DNS-01 relay disabled")
	}

	if 0 == len(mgmt.DeviceDomain) || 0 == len(secret) || 0 == len(dbURL) {
		help()
		os.Exit(1)
		return
	}

	connStr := dbURL
	// TODO url.Parse
	if strings.Contains(connStr, "@localhost/") || strings.Contains(connStr, "@localhost:") {
		connStr += "?sslmode=disable"
	} else {
		connStr += "?sslmode=required"
	}

	store, err = authstore.NewStore(connStr, mgmt.InitSQL)
	if nil != err {
		log.Fatal("connection error", err)
		return
	}
	_ = store.SetMaster(secret)
	defer store.Close()

	mgmt.Init(store, provider)

	if len(challengesPort) > 0 {
		go func() {
			fmt.Println("Listening for ACME challenges on :" + challengesPort)
			r := chi.NewRouter()
			r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(ver() + "\n"))
			})
			r.Get("/api/version", func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("TODO (json): " + ver() + "\n"))
			})
			if err := http.ListenAndServe(":"+challengesPort, mgmt.RouteStatic(r)); nil != err {
				log.Fatal(err)
				os.Exit(1)
			}
		}()
	}

	fmt.Println("Listening on", lnAddr)
	r := chi.NewRouter()
	r.Get("/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(ver() + "\n"))
	})
	r.Get("/api/version", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("TODO (json): " + ver() + "\n"))
	})
	mgmt.RouteAll(r)
	fmt.Fprintf(os.Stderr, "failed: %s", http.ListenAndServe(lnAddr, r))
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
