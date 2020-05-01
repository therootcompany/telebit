package main

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"

	"git.coolaj86.com/coolaj86/go-telebitd/client"
)

var httpRegexp = regexp.MustCompile(`(?i)^http`)

func init() {
	flag.StringSlice("locals", []string{}, "comma separated list of <proto>:<port> or "+
		"<proto>:<hostname>:<port> to which matching incoming connections should forward. "+
		"Ex: smtps:8465,https:example.com:8443")
	flag.StringSlice("domains", []string{}, "comma separated list of domain names to set to the tunnel")
	viper.BindPFlag("locals", flag.Lookup("locals"))
	viper.BindPFlag("domains", flag.Lookup("domains"))

	flag.BoolP("insecure", "k", false, "Allow TLS connections to stunneld without valid certs")
	flag.String("stunneld", "", "the domain (or ip address) at which the RVPN server is running")
	flag.String("secret", "", "the same secret used by stunneld (used for JWT authentication)")
	flag.String("token", "", "a pre-generated token to give the server (instead of generating one with --secret)")
	viper.BindPFlag("raw.insecure", flag.Lookup("insecure"))
	viper.BindPFlag("raw.stunneld", flag.Lookup("stunneld"))
	viper.BindPFlag("raw.secret", flag.Lookup("secret"))
	viper.BindPFlag("raw.token", flag.Lookup("token"))
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

func extractServicePorts(proxies []proxy) map[string]map[string]int {
	result := make(map[string]map[string]int, 2)

	for _, p := range proxies {
		if p.protocol != "" && p.port != 0 {
			hostPorts := result[p.protocol]
			if hostPorts == nil {
				result[p.protocol] = make(map[string]int)
				hostPorts = result[p.protocol]
			}

			// Only HTTP and HTTPS allow us to determine the hostname from the request, so only
			// those protocols support different ports for the same service.
			if !httpRegexp.MatchString(p.protocol) || p.hostname == "" {
				p.hostname = "*"
			}
			if port, ok := hostPorts[p.hostname]; ok && port != p.port {
				panic(fmt.Sprintf("duplicate ports for %s://%s", p.protocol, p.hostname))
			}
			hostPorts[p.hostname] = p.port
		}
	}

	// Make sure we have defaults for HTTPS and HTTP.
	if result["https"] == nil {
		result["https"] = make(map[string]int, 1)
	}
	if result["https"]["*"] == 0 {
		result["https"]["*"] = 8443
	}

	if result["http"] == nil {
		result["http"] = make(map[string]int, 1)
	}
	if result["http"]["*"] == 0 {
		result["http"]["*"] = result["https"]["*"]
	}

	return result
}

func main() {
	flag.Parse()

	var err error
	proxies := make([]proxy, 0)
	for _, option := range viper.GetStringSlice("locals") {
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
	for _, option := range viper.GetStringSlice("domains") {
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

	if viper.GetString("raw.stunneld") == "" {
		panic("must provide remote RVPN server to connect to")
	}

	var token string
	if viper.GetString("raw.token") != "" {
		token = viper.GetString("raw.token")
	} else if viper.GetString("raw.secret") != "" {
		domains := make([]string, 0, len(domainMap))
		for name := range domainMap {
			domains = append(domains, name)
		}
		tokenData := jwt.MapClaims{"domains": domains}

		secret := []byte(viper.GetString("raw.secret"))
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenData)
		if tokenStr, err := jwtToken.SignedString(secret); err != nil {
			panic(err)
		} else {
			token = tokenStr
		}
	} else {
		panic("must provide either token or secret")
	}

	ctx, quit := context.WithCancel(context.Background())
	defer quit()

	config := client.Config{
		Insecure: viper.GetBool("raw.insecure"),
		Server:   viper.GetString("raw.stunneld"),
		Services: servicePorts,
		Token:    token,
	}
	panic(client.Run(ctx, &config))
}
