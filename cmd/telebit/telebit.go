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

	"git.coolaj86.com/coolaj86/go-telebitd/rvpn/client"
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

func addLocals(proxies []proxy, location string) []proxy {
	parts := strings.Split(location, ":")
	if len(parts) > 3 {
		panic(fmt.Sprintf("provided invalid location %q", location))
	}

	// If all that was provided as a "local" is the domain name we assume that domain
	// has HTTP and HTTPS handlers on the default ports.
	if len(parts) == 1 {
		proxies = append(proxies, proxy{"http", parts[0], 80})
		proxies = append(proxies, proxy{"https", parts[0], 443})
		return proxies
	}

	// Make everything lower case and trim any slashes in something like https://john.example.com
	parts[0] = strings.ToLower(parts[0])
	parts[1] = strings.ToLower(strings.Trim(parts[1], "/"))

	if len(parts) == 2 {
		if strings.Contains(parts[1], ".") {
			if parts[0] == "http" {
				parts = append(parts, "80")
			} else if parts[0] == "https" {
				parts = append(parts, "443")
			} else {
				panic(fmt.Sprintf("port must be specified for %q", location))
			}
		} else {
			// https:3443 -> https:*:3443
			parts = []string{parts[0], "*", parts[1]}
		}
	}

	if port, err := strconv.Atoi(parts[2]); err != nil {
		panic(fmt.Sprintf("port must be a valid number, not %q: %v", parts[2], err))
	} else if port <= 0 || port > 65535 {
		panic(fmt.Sprintf("%d is an invalid port for local services", port))
	} else {
		proxies = append(proxies, proxy{parts[0], parts[1], port})
	}
	return proxies
}

func addDomains(proxies []proxy, location string) []proxy {
	parts := strings.Split(location, ":")
	if len(parts) > 3 {
		panic(fmt.Sprintf("provided invalid location %q", location))
	} else if len(parts) == 2 {
		panic("invalid argument for --domains, use format <domainname> or <scheme>:<domainname>:<local-port>")
	}

	// If the scheme and port weren't provided use the zero values
	if len(parts) == 1 {
		return append(proxies, proxy{"", parts[0], 0})
	}

	if port, err := strconv.Atoi(parts[2]); err != nil {
		panic(fmt.Sprintf("port must be a valid number, not %q: %v", parts[2], err))
	} else if port <= 0 || port > 65535 {
		panic(fmt.Sprintf("%d is an invalid port for local services", port))
	} else {
		proxies = append(proxies, proxy{parts[0], parts[1], port})
	}
	return proxies
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

	proxies := make([]proxy, 0)
	for _, option := range viper.GetStringSlice("locals") {
		for _, location := range strings.Split(option, ",") {
			proxies = addLocals(proxies, location)
		}
	}
	for _, option := range viper.GetStringSlice("domains") {
		for _, location := range strings.Split(option, ",") {
			proxies = addDomains(proxies, location)
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
