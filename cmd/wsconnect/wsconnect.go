package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"git.coolaj86.com/coolaj86/go-telebitd/mgmt"
	"git.coolaj86.com/coolaj86/go-telebitd/mgmt/authstore"
	telebit "git.coolaj86.com/coolaj86/go-telebitd/mplexer"

	"github.com/denisbrodbeck/machineid"
	"github.com/gorilla/websocket"
	_ "github.com/joho/godotenv/autoload"
)

var authorizer telebit.Authorizer

func main() {
	// TODO replace the websocket connection with a mock server
	appID := flag.String("app-id", "", "a unique identifier for a deploy target environment")
	authURL := flag.String("auth-url", "", "the base url for authentication, if not the same as the tunnel relay")
	relay := flag.String("relay", "", "the domain (or ip address) at which the relay server is running")
	secret := flag.String("secret", "", "the same secret used by telebit-relay (used for JWT authentication)")
	token := flag.String("token", "", "a pre-generated token to give the server (instead of generating one with --secret)")
	flag.Parse()

	if 0 == len(*appID) {
		*appID = os.Getenv("APP_ID")
	}
	if 0 == len(*appID) {
		*appID = "telebit.io"
	}
	if 0 == len(*secret) {
		*secret = os.Getenv("CLIENT_SECRET")
	}
	ppid, err := machineid.ProtectedID(fmt.Sprintf("%s|%s", *appID, *secret))
	if nil != err {
		fmt.Fprintf(os.Stderr, "unauthorized device\n")
		os.Exit(1)
		return
	}
	ppidBytes, err := hex.DecodeString(ppid)
	ppid = base64.RawURLEncoding.EncodeToString(ppidBytes)
	fmt.Println("[debug] app-id, secret, ppid", *appID, *secret, ppid)
	if 0 == len(*token) {
		*token, err = authstore.HMACToken(ppid)
		if nil != err {
			fmt.Fprintf(os.Stderr, "neither secret nor token provided\n")
			os.Exit(1)
			return
		}
	}
	if 0 == len(*relay) {
		*relay = os.Getenv("RELAY") // "wss://example.com:443"
	}
	if 0 == len(*relay) {
		fmt.Fprintf(os.Stderr, "Missing relay url\n")
		//os.Exit(1)
		//return
	}

	if "" == *authURL {
		*authURL = os.Getenv("AUTH_URL")
	}
	if "" == *authURL {
		*authURL = strings.Replace(*relay, "ws", "http", 1) // "https://example.com:443"
	}
	authorizer = NewAuthorizer(*authURL)
	// TODO look at relay rather than authURL?
	grants, err := telebit.Inspect(*authURL, *token)
	if nil != err {
		fmt.Println("[debug] inpsect failed:")
		fmt.Println(err)
		_, err := mgmt.Register(*authURL, *secret, ppid)
		if nil != err {
			fmt.Fprintf(os.Stderr, "failed to register client: %s\n", err)
			os.Exit(1)
		}
		grants, err = telebit.Inspect(*authURL, *token)
		if nil != err {
			fmt.Fprintf(os.Stderr, "failed to authenticate after registering client: %s\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("grants", grants)

	wsd := websocket.Dialer{}
	headers := http.Header{}
	headers.Set("Authorization", fmt.Sprintf("Bearer %s", *token))
	// *http.Response
	sep := "?"
	if strings.Contains(*relay, sep) {
		sep = "&"
	}
	timeoutCtx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	wsconn, _, err := wsd.DialContext(timeoutCtx, *relay+sep+"access_token="+*token+"&versions=v1", headers)
	if nil != err {
		fmt.Println("[debug] err")
		fmt.Println(err)
		return
	}

	for {
		_, msgr, err := wsconn.NextReader()
		if nil != err {
			fmt.Println("debug wsconn NextReader err:", err)
			return
			//return 0, err
		}
		for {
			b := make([]byte, 512)
			n, err := msgr.Read(b)
			if nil != err {
				fmt.Println("debug msgr Read err:", err)
				return
			}
			fmt.Println(n, string(b[0:n]))
		}
	}
}

func NewAuthorizer(authURL string) telebit.Authorizer {
	return func(r *http.Request) (*telebit.Grants, error) {
		// do we have a valid wss_client?

		var tokenString string
		if auth := strings.Split(r.Header.Get("Authorization"), " "); len(auth) > 1 {
			// TODO handle Basic auth tokens as well
			tokenString = auth[1]
		}
		if "" == tokenString {
			// Browsers do not allow Authorization Headers and must use access_token query string
			tokenString = r.URL.Query().Get("access_token")
		}
		if "" != r.URL.Query().Get("access_token") {
			r.URL.Query().Set("access_token", "[redacted]")
		}

		grants, err := telebit.Inspect(authURL, tokenString)

		if nil != err {
			fmt.Println("[wsconnect] return an error, do not go on")
			return nil, err
		}
		if "" != r.URL.Query().Get("access_token") {
			r.URL.Query().Set("access_token", "[redacted:"+grants.Subject+"]")
		}

		return grants, err
	}
}
