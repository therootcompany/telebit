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
	appID := flag.String("app-id", "", "a unique identifier for a deploy target environment")
	authURL := flag.String("auth-url", "", "the base url for authentication, if not the same as the tunnel relay")
	clientSecret := flag.String("client-secret", "", "the same secret used by telebit-relay (used for JWT authentication)")
	machinePPID := flag.Bool("machine-ppid", false, "just print the machine ppid, not the token")
	relaySecret := flag.String("relay-secret", "", "the same secret used by telebit-relay (used for JWT authentication)")
	flag.Parse()

	if 0 == len(*appID) {
		*appID = os.Getenv("APP_ID")
	}
	if 0 == len(*appID) {
		*appID = "telebit.io"
	}
	if 0 == len(*clientSecret) {
		*clientSecret = os.Getenv("CLIENT_SECRET")
	}
	if 0 == len(*relaySecret) {
		*relaySecret = os.Getenv("RELAY_SECRET")
		if 0 == len(*relaySecret) {
			*relaySecret = os.Getenv("SECRET")
		}
	}

	if 0 == len(*authURL) {
		*authURL = os.Getenv("AUTH_URL")
	}

	if len(flag.Args()) >= 2 {
		*relaySecret = flag.Args()[1]
	}
	if "" == *relaySecret && "" == *clientSecret {
		fmt.Fprintf(os.Stderr, "Usage: signjwt <secret>\n")
		os.Exit(1)
		return
	}

	secret := *clientSecret
	if 0 == len(secret) {
		secret = *relaySecret
	}
	if len(flag.Args()) >= 2 {
		secret = flag.Args()[1]
	}

	if len(flag.Args()) >= 3 || *machinePPID || "" != *clientSecret {
		muid, err := machineid.ProtectedID(*appID + "|" + secret)
		if nil != err {
			panic(err)
		}
		muidBytes, _ := hex.DecodeString(muid)
		ppid := base64.RawURLEncoding.EncodeToString(muidBytes)
		fmt.Fprintf(os.Stderr, "[debug] appID = %s\n", *appID)
		fmt.Fprintf(os.Stderr, "[debug] secret = %s\n", secret)
		pub := authstore.ToPublicKeyString(ppid)
		if len(flag.Args()) >= 3 || *machinePPID {
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
		secret = ppid
	}

	tok, err := authstore.HMACToken(secret)
	if nil != err {
		fmt.Fprintf(os.Stderr, "signing error: %s\n", err)
		os.Exit(1)
		return
	}

	fmt.Fprintf(os.Stderr, "[debug] <token>\n")
	fmt.Fprintf(os.Stdout, tok)

	_, err = telebit.Inspect(*authURL, tok)
	if nil != err {
		fmt.Fprintf(os.Stderr, "inpsect relay token failed:\n%s\n", err)
		os.Exit(1)
	}
}
