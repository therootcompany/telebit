package main

import (
	"fmt"
	"net"
	"os"

	"git.rootprojects.org/root/telebit/iplist"
)

func help() {
	fmt.Fprintf(os.Stderr, "Usage: iplist domain.tld 123.45.6.78\n")
	fmt.Fprintf(os.Stderr, "(`dig TXT +short domain.tld` should return a list like `v=spf1 ip4:123.45.6.78 ip4:123.45.6.1/24`\n")
	os.Exit(1)
}

func main() {
	if 3 != len(os.Args) {
		help()
		return
	}

	txtDomain := os.Args[1]
	remoteIP := net.ParseIP(os.Args[2])
	if nil == remoteIP {
		fmt.Fprintf(os.Stderr, "bad remote IP\n")
		os.Exit(1)
		return
	}

	iplist.Init(txtDomain)

	allowed, err := iplist.IsAllowed(remoteIP)
	if nil != err {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
		return
	}
	if !allowed {
		fmt.Fprintf(os.Stderr, "not allowed\n")
		os.Exit(1)
		return
	}

	fmt.Println("allowed")
}
