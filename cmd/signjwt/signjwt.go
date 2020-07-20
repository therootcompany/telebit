package main

import (
	"crypto/hmac"
	"crypto/sha256"
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

	machinePPID := flag.String("machine-ppid", "", "spoof the machine ppid")
	machineID := flag.String("machine-id", "", "spoof the raw machine id")
	vendorID := flag.String("vendor-id", "", "a unique identifier for a deploy target environment")
	authURL := flag.String("auth-url", "", "the base url for authentication, if not the same as the tunnel relay")
	getMachinePPID := flag.Bool("machine-ppid-only", false, "just print the machine ppid, not the token")
	flag.StringVar(&secret, "secret", "", "either the remote server or the tunnel relay secret (used for JWT authentication)")
	flag.Parse()

	if 0 == len(*authURL) {
		*authURL = os.Getenv("AUTH_URL")
	}

	if 0 == len(*vendorID) {
		*vendorID = os.Getenv("VENDOR_ID")
	}
	if 0 == len(*vendorID) {
		*vendorID = "telebit.io"
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

	ppid := *machinePPID
	if 0 == len(ppid) {
		appID := fmt.Sprintf("%s|%s", *vendorID, secret)
		var muid string
		var err error
		if 0 == len(*machineID) {
			muid, err = machineid.ProtectedID(appID)
			if nil != err {
				fmt.Fprintf(os.Stderr, "unauthorized device: %s\n", err)
				os.Exit(1)
				return
			}
		} else {
			muid = ProtectMachineID(appID, *machineID)
		}
		muidBytes, _ := hex.DecodeString(muid)
		ppid = base64.RawURLEncoding.EncodeToString(muidBytes)
	}

	fmt.Fprintf(os.Stderr, "[debug] vendorID = %s\n", *vendorID)
	fmt.Fprintf(os.Stderr, "[debug] secret = %s\n", secret)
	pub := authstore.ToPublicKeyString(ppid)

	if *getMachinePPID {
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

func ProtectMachineID(appID, machineID string) string {
	mac := hmac.New(sha256.New, []byte(machineID))
	mac.Write([]byte(appID))
	return hex.EncodeToString(mac.Sum(nil))
}
