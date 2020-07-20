package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"os"

	"git.coolaj86.com/coolaj86/go-telebitd/mgmt/authstore"

	"github.com/denisbrodbeck/machineid"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	var secret, clientSecret, relaySecret string
	var raw bool

	flag.BoolVar(&raw, "raw", false, "output the raw machine id")
	vendorID := flag.String("vendor-id", "", "a unique identifier for a deploy target environment")
	flag.StringVar(&secret, "secret", "", "either the remote server or the tunnel relay secret (used for JWT authentication)")
	flag.Parse()

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
		fmt.Fprintf(os.Stderr, "See usage: machineid --help\n")
		os.Exit(1)
		return
	} else if 0 != len(clientSecret) && 0 != len(relaySecret) {
		fmt.Fprintf(os.Stderr, "Use only one of $SECRET or --relay-secret or --client-secret\n")
		os.Exit(1)
		return
	}

	if raw {
		rawID, err := machineid.ID()
		if nil != err {
			fmt.Fprintf(os.Stderr, "Error: %q", err)
			os.Exit(1)
			return
		}
		fmt.Println("Raw Machine ID:", rawID)
	}

	fmt.Println("Vendor ID:", *vendorID)
	fmt.Println("Secret:", secret)

	var ppid string
	muid, err := machineid.ProtectedID(fmt.Sprintf("%s|%s", *vendorID, secret))
	//muid, err := machineid.ProtectedID(fmt.Sprintf("%s|%s", VendorID, ClientSecret))
	if nil != err {
		fmt.Fprintf(os.Stderr, "unauthorized device: %s\n", err)
		os.Exit(1)
		return
	}
	muidBytes, _ := hex.DecodeString(muid)
	ppid = base64.RawURLEncoding.EncodeToString(muidBytes)

	fmt.Println("PPID:", ppid)
	pub := authstore.ToPublicKeyString(ppid)
	fmt.Println("Pub:", pub)
}
