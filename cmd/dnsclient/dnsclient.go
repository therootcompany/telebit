//go:generate go run -mod=vendor git.rootprojects.org/root/go-gitver/v2

package main

import (
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"git.rootprojects.org/root/telebit/internal/dns01"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-acme/lego/v3/challenge"
	_ "github.com/joho/godotenv/autoload"
)

var (
	// commit refers to the abbreviated commit hash
	commit = "0000000"
	// version refers to the most recent tag, plus any commits made since then
	version = "v0.0.0-pre0+0000000"
	// GitTimestamp refers to the timestamp of the most recent commit
	date = "0000-00-00T00:00:00+0000"
)

func main() {
	var err error
	var provider challenge.Provider = nil
	var domains []string

	// TODO replace the websocket connection with a mock server
	acmeRelay := flag.String("acme-relay-url", "", "the base url of the ACME DNS-01 relay, if not the same as the tunnel relay")
	secret := flag.String("secret", "", "the same secret used by telebit-relay (used for JWT authentication)")
	token := flag.String("token", "", "a pre-generated token to give the server (instead of generating one with --secret)")
	flag.Parse()

	if len(os.Args) >= 2 {
		if "version" == strings.TrimLeft(os.Args[1], "-") {
			fmt.Printf("telebit %s (%s) %s\n", version, commit[:7], date)
			os.Exit(0)
			return
		}
	}

	if "" == *token {
		if "" == *secret {
			*secret = os.Getenv("SECRET")
		}
		*token, err = getToken(*secret, domains)
	}
	if nil != err {
		fmt.Fprintf(os.Stderr, "neither secret nor token provided")
		os.Exit(1)
		return
	}

	if "" == *acmeRelay {
		panic(errors.New("ACME relay should be specified"))
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
		panic(err)
	}

	err = provider.Present(os.Getenv("HOSTNAME"), "xxx", "yyy")
	if nil != err {
		fmt.Fprintf(os.Stderr, err.Error())
	}
	err = provider.Present(os.Getenv("HOSTNAME"), "xxx", "yyy")
	if nil != err {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}

	fmt.Println("quite possibly successful")
}

// newAPIDNSProvider is for the sake of demoing the tunnel
func newAPIDNSProvider(baseURL string, token string) (*dns01.DNSProvider, error) {
	config := dns01.NewDefaultConfig()
	config.Tokener = func() string {
		return token
	}
	endpoint, err := url.Parse(baseURL)
	if nil != err {
		return nil, err
	}
	config.Endpoint = endpoint
	return dns01.NewDNSProviderConfig(config)
}

func getToken(secret string, domains []string) (token string, err error) {
	tokenData := jwt.MapClaims{"domains": domains}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenData)
	if token, err = jwtToken.SignedString([]byte(secret)); err != nil {
		return "", err
	}
	return token, nil
}
