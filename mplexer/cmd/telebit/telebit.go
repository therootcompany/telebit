package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"git.coolaj86.com/coolaj86/go-telebitd/mplexer/packer"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/websocket"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// TODO replace the websocket connection with a mock server

	relay := os.Getenv("RELAY") // "wss://roottest.duckdns.org:8443"
	authz, err := getToken(os.Getenv("SECRET"))
	if nil != err {
		panic(err)
	}

	ctx := context.Background()
	wsd := websocket.Dialer{}
	headers := http.Header{}
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", authz))
	// *http.Response
	sep := "?"
	if strings.Contains(relay, sep) {
		sep = "&"
	}
	wsconn, _, err := wsd.DialContext(ctx, relay+sep+"access_token="+authz, headers)
	if nil != err {
		fmt.Println("relay:", relay)
		log.Fatal(err)
		return
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

	mux := packer.NewRouteMux()
	//mux.HandleTLS("*", mux.TerminateTLS(mux))
	mux.ForwardTCP("*", "localhost:3000", 120*time.Second)
	// TODO set failure
	log.Fatal("Closed server: ", packer.ListenAndServe(wsconn, mux))
}

func getToken(secret string) (token string, err error) {
	domains := []string{"dandel.duckdns.org"}
	tokenData := jwt.MapClaims{"domains": domains}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenData)
	if token, err = jwtToken.SignedString([]byte(secret)); err != nil {
		return "", err
	}
	return token, nil
}
