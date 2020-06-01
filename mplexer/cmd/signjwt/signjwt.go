package main

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"

	"git.coolaj86.com/coolaj86/go-telebitd/mplexer/mgmt/authstore"

	"github.com/denisbrodbeck/machineid"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	var secret string

	if len(os.Args) >= 2 {
		secret = os.Args[1]
	}
	if "" == secret {
		secret = os.Getenv("SECRET")
	}
	if "" == secret {
		fmt.Fprintf(os.Stderr, "Usage: signjwt <secret>")
		os.Exit(1)
		return
	}

	if len(os.Args) >= 3 {
		muid, err := machineid.ProtectedID("test-id|" + secret)
		if nil != err {
			panic(err)
		}
		muidBytes, _ := hex.DecodeString(muid)
		muid = base64.RawURLEncoding.EncodeToString(muidBytes)
		fmt.Println(
			muid,
			authstore.ToPublicKeyString(muid),
		)
		return
	}

	tok, err := authstore.HMACToken(secret)
	if nil != err {
		fmt.Fprintf(os.Stderr, "signing error: %s", err)
		os.Exit(1)
		return
	}

	fmt.Println(tok)
}
