package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"git.coolaj86.com/coolaj86/go-telebitd/mgmt/authstore"
	telebit "git.coolaj86.com/coolaj86/go-telebitd/mplexer"

	"github.com/denisbrodbeck/machineid"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	var secret, clientSecret, relaySecret string

	appID := flag.String("app-id", "", "a unique identifier for a deploy target environment")
	authURL := flag.String("auth-url", "", "the base url for authentication, if not the same as the tunnel relay")
	machinePPID := flag.Bool("machine-ppid", false, "just print the machine ppid, not the token")
	flag.StringVar(&secret, "secret", "", "either the remote server or the tunnel relay secret (used for JWT authentication)")
	flag.Parse()

	if 0 == len(*authURL) {
		*authURL = os.Getenv("AUTH_URL")
	}

	if 0 == len(*appID) {
		*appID = os.Getenv("APP_ID")
	}
	if 0 == len(*appID) {
		*appID = os.Getenv("CLIENT_ID")
	}
	if 0 == len(*appID) {
		*appID = "telebit.io"
	}

	if 0 == len(secret) {
		clientSecret = os.Getenv("CLIENT_SECRET")
		relaySecret = os.Getenv("RELAY_SECRET")
		if 0 == len(relaySecret) {
			relaySecret = os.Getenv("SECRET")
		}
	}
	if 0 == len(secret) {
		secret = clientSecret
	}
	if 0 == len(secret) {
		secret = relaySecret
	}

	if 0 == len(secret) && 0 == len(clientSecret) && 0 == len(relaySecret) {
		fmt.Fprintf(os.Stderr, "See usage: signjwt --help\n")
		os.Exit(1)
		return
	} else if 0 != len(clientSecret) && 0 != len(relaySecret) {
		fmt.Fprintf(os.Stderr, "Use only one of $SECRET or --relay-secret or --client-secret\n")
		os.Exit(1)
		return
	}

	var ppid string
	muid, err := machineid.ProtectedID(fmt.Sprintf("%s|%s", *appID, secret))
	//muid, err := machineid.ProtectedID(fmt.Sprintf("%s|%s", ClientID, ClientSecret))
	if nil != err {
		fmt.Fprintf(os.Stderr, "unauthorized device: %s\n", err)
		os.Exit(1)
		return
	}
	muidBytes, _ := hex.DecodeString(muid)
	ppid = base64.RawURLEncoding.EncodeToString(muidBytes)

	fmt.Fprintf(os.Stderr, "[debug] appID = %s\n", *appID)
	fmt.Fprintf(os.Stderr, "[debug] secret = %s\n", secret)
	pub := authstore.ToPublicKeyString(ppid)

	if *machinePPID {
		fmt.Fprintf(os.Stderr, "[debug]: <ppid> <pub>\n")
		fmt.Fprintf(
			os.Stdout,
			"%s %s\n",
			ppid,
			pub,
		)
		return
	}

	fmt.Fprintf(os.Stderr, "[debug] ppid = %s\n", ppid)
	fmt.Fprintf(os.Stderr, "[debug] pub = %s\n", pub)

	tok, err := authstore.HMACToken(ppid)
	if nil != err {
		fmt.Fprintf(os.Stderr, "signing error: %s\n", err)
		os.Exit(1)
		return
	}

	fmt.Fprintf(os.Stderr, "[debug] <token>\n")
	fmt.Fprintf(os.Stdout, "%s\n", tok)

	if "" != *authURL {
		grants, err := telebit.Inspect(*authURL, tok)
		if nil != err {
			fmt.Fprintf(os.Stderr, "inspect relay token failed:\n%s\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "[debug] <grants>\n")
		fmt.Fprintf(os.Stderr, "%+v\n", grants)
	}
}
