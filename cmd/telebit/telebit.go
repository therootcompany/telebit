package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"git.coolaj86.com/coolaj86/go-telebitd/client"

	"github.com/caddyserver/certmagic"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-acme/lego/v3/providers/dns/duckdns"

	_ "github.com/joho/godotenv/autoload"
)

var httpRegexp = regexp.MustCompile(`(?i)^http`)
var locals string
var domains string
var insecure bool
var relay string
var secret string
var token string

func init() {
	flag.StringVar(&locals, "locals", "", "comma separated list of <proto>:<port> or "+
		"<proto>:<hostname>:<port> to which matching incoming connections should forward. "+
		"Ex: smtps:8465,https:example.com:8443")
	flag.StringVar(&domains, "domains", "", "comma separated list of domain names to set to the tunnel")
	flag.BoolVar(&insecure, "insecure", false, "Allow TLS connections to telebit-relay without valid certs")
	flag.BoolVar(&insecure, "k", false, "alias of --insecure")
	flag.StringVar(&relay, "relay", "", "the domain (or ip address) at which the relay server is running")
	flag.StringVar(&secret, "secret", "", "the same secret used by telebit-relay (used for JWT authentication)")
	flag.StringVar(&token, "token", "", "a pre-generated token to give the server (instead of generating one with --secret)")
}

type proxy struct {
	protocol string
	hostname string
	port     int
}

func addLocals(proxies []proxy, location string) ([]proxy, error) {
	parts := strings.Split(location, ":")
	if len(parts) > 3 || "" == parts[0] {
		return nil, fmt.Errorf("provided invalid --locals %q", location)
	}

	// Format can be any of
	// <hostname> or <port> or <proto>:<port> or <proto>:<hostname>:<port>

	n := len(parts)
	i := n - 1
	last := parts[i]

	port, err := strconv.Atoi(last)
	if nil != err {
		// The last item is the hostname,
		// which means it should be the only item
		if n > 1 {
			return nil, fmt.Errorf("provided invalid --locals %q", location)
		}
		// accepting all defaults
		// If all that was provided as a "local" is the domain name we assume that domain
		last = strings.ToLower(strings.Trim(last, "/"))
		proxies = append(proxies, proxy{"http", last, 80})
		proxies = append(proxies, proxy{"https", last, 443})
		return proxies, nil
	}

	// the last item is the port, and it must be a valid port
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("local port forward must be between 1 and 65535, not %d", port)
	}

	switch n {
	case 1:
		// <port>
		proxies = append(proxies, proxy{"http", "*", port})
		proxies = append(proxies, proxy{"https", "*", port})
	case 2:
		// <hostname>:<port>
		// <scheme>:<port>
		parts[0] = strings.ToLower(strings.Trim(parts[0], "/"))
		if strings.Contains(parts[0], ".") {
			hostname := parts[0]
			proxies = append(proxies, proxy{"http", hostname, port})
			proxies = append(proxies, proxy{"https", hostname, port})
		} else {
			scheme := parts[0]
			proxies = append(proxies, proxy{scheme, "*", port})
		}
	case 3:
		// <scheme>:<hostname>:<port>
		scheme := strings.ToLower(strings.Trim(parts[0], "/"))
		hostname := strings.ToLower(strings.Trim(parts[1], "/"))
		proxies = append(proxies, proxy{scheme, hostname, port})
	}
	return proxies, nil
}

func addDomains(proxies []proxy, location string) ([]proxy, error) {
	parts := strings.Split(location, ":")
	if len(parts) > 3 || "" == parts[0] {
		return nil, fmt.Errorf("provided invalid --domains %q", location)
	}

	// Format is limited to
	// <hostname> or <proto>:<hostname>:<port>

	err := fmt.Errorf("invalid argument for --domains, use format <domainname> or <scheme>:<domainname>:<local-port>")
	switch len(parts) {
	case 1:
		// TODO test that it's a valid pattern for a domain
		hostname := parts[0]
		if !strings.Contains(hostname, ".") {
			return nil, err
		}
		proxies = append(proxies, proxy{"http", hostname, 80})
		proxies = append(proxies, proxy{"https", hostname, 443})
	case 2:
		return nil, err
	case 3:
		scheme := parts[0]
		hostname := parts[1]
		if "" == scheme {
			return nil, err
		}
		if !strings.Contains(hostname, ".") {
			return nil, err
		}
		port, _ := strconv.Atoi(parts[2])
		if port <= 0 || port > 65535 {
			return nil, err
		}
		proxies = append(proxies, proxy{scheme, hostname, port})
	}

	return proxies, nil
}

func extractServicePorts(proxies []proxy) client.RouteMap {
	result := make(client.RouteMap, 2)

	for _, p := range proxies {
		if p.protocol != "" && p.port != 0 {
			hostPorts := result[p.protocol]
			if hostPorts == nil {
				result[p.protocol] = make(map[client.DomainName]*client.TerminalConfig)
				hostPorts = result[p.protocol]
			}

			// Only HTTP and HTTPS allow us to determine the hostname from the request, so only
			// those protocols support different ports for the same service.
			if !httpRegexp.MatchString(p.protocol) || p.hostname == "" {
				p.hostname = "*"
			}
			if port, ok := hostPorts[p.hostname]; ok && port.Port != p.port {
				panic(fmt.Sprintf("duplicate ports for %s://%s", p.protocol, p.hostname))
			}
			hostPorts[p.hostname] = &client.TerminalConfig{
				Port: p.port,
			}
		}
	}

	// Make sure we have defaults for HTTPS and HTTP.
	if result["https"] == nil {
		result["https"] = make(map[client.DomainName]*client.TerminalConfig, 1)
	}
	if result["https"]["*"] == nil {
		result["https"]["*"] = &client.TerminalConfig{}
	}
	if result["https"]["*"].Port == 0 {
		result["https"]["*"].Port = 8443
	}

	if result["http"] == nil {
		result["http"] = make(map[client.DomainName]*client.TerminalConfig, 1)
	}
	if result["http"]["*"] == nil {
		result["http"]["*"] = &client.TerminalConfig{}
	}
	if result["http"]["*"].Port == 0 {
		result["http"]["*"] = result["https"]["*"]
	}

	return result
}

func main() {
	flag.Parse()

	var err error

	if "" == locals {
		locals = os.Getenv("LOCALS")
	}

	proxies := make([]proxy, 0)
	for _, option := range stringSlice(locals) {
		for _, location := range strings.Split(option, ",") {
			//fmt.Println("locals", location)
			proxies, err = addLocals(proxies, location)
			if nil != err {
				panic(err)
			}
		}
	}

	//fmt.Println("proxies:")
	//fmt.Printf("%+v\n\n", proxies)
	for _, option := range stringSlice(domains) {
		for _, location := range strings.Split(option, ",") {
			proxies, err = addDomains(proxies, location)
			if nil != err {
				panic(err)
			}
		}
	}

	servicePorts := extractServicePorts(proxies)
	domainMap := make(map[string]bool)
	for _, p := range proxies {
		if p.hostname != "" && p.hostname != "*" {
			domainMap[p.hostname] = true
		}
	}

	if relay == "" {
		relay = os.Getenv("RELAY")
	}
	if relay == "" {
		fmt.Fprintf(os.Stderr, "must provide remote relay server to connect to\n")
		os.Exit(1)
	}

	if secret == "" {
		secret = os.Getenv("SECRET")
	}

	if secret != "" {
		domains := make([]string, 0, len(domainMap))
		for name := range domainMap {
			domains = append(domains, name)
		}
		tokenData := jwt.MapClaims{"domains": domains}

		secret := []byte(secret)
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenData)
		if tokenStr, err := jwtToken.SignedString(secret); err != nil {
			panic(err)
		} else {
			token = tokenStr
		}
	} else if token != "" {
		fmt.Fprintf(os.Stderr, "must provide either token or secret\n")
		os.Exit(1)
	}

	ctx, quit := context.WithCancel(context.Background())
	defer quit()

	acmeStorage := "./acme.d/"
	acmeEmail := ""
	acmeStaging := false
	//
	// CertMagic is Greenlock for Go
	//
	directory := certmagic.LetsEncryptProductionCA
	if acmeStaging {
		directory = certmagic.LetsEncryptStagingCA
	}
	magic, err := newCertMagic(directory, acmeEmail, &certmagic.FileStorage{Path: acmeStorage})
	if nil != err {
		fmt.Fprintf(os.Stderr, "failed to initialize certificate management (discovery url? local folder perms?): %s\n", err)
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

	config := client.Config{
		Insecure:  insecure,
		Server:    relay,
		Services:  servicePorts,
		Token:     token,
		TLSConfig: tlsConfig,
	}

	fmt.Printf("config:\n%#v\n", config)
	log.Fatal(client.Run(ctx, &config))
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

func stringSlice(csv string) []string {
	list := []string{}
	for _, item := range strings.Split(csv, ", ") {
		if 0 == len(item) {
			continue
		}
		list = append(list, item)
	}
	return list
}
