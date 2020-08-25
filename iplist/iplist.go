package iplist

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

var fields []string
var initialized bool

// Init should be called with domain that has valid SPF-like records
// to populate the IP whitelist, or with an empty string "" to disable
func Init(txtDomain string) []string {
	initialized = true
	if "" == txtDomain {
		return []string{}
	}

	err := updateTxt(txtDomain)
	if nil != err {
		panic(err)
	}
	go func() {
		for {
			time.Sleep(5 * time.Minute)
			if err := updateTxt(txtDomain); nil != err {
				fmt.Fprintf(os.Stderr, "warn: could not update iplist: %s\n", err)
				continue
			}
		}
	}()

	for _, section := range fields {
		parts := strings.Split(section, ":")
		if 2 != len(parts) || !strings.HasPrefix(parts[0], "ip") {
			// ignore unsupported bits
			// (i.e. +mx +ip include:xxx)
			continue
		}
		ip := parts[1]

		if strings.Contains(ip, "/") {
			_, _, err := net.ParseCIDR(ip)
			if nil != err {
				panic(fmt.Errorf("invalid CIDR %q", ip))
			}
			continue
		}

		ipAddr := net.ParseIP(ip)
		if nil == ipAddr {
			panic(fmt.Errorf(
				"IP %q from SPF record could not be parsed",
				ipAddr.String(),
			))
		}
	}
	return fields
}

func updateTxt(txtDomain string) error {
	var newFields []string
	records, err := net.LookupTXT(txtDomain)
	if nil != err {
		return fmt.Errorf("bad spf-domain: %s", err)
	}
	for _, record := range records {
		newFields, err = parseSpf(record)
		if nil != err {
			continue
		}
		if len(fields) > 0 {
			break
		}
	}

	// TODO put a lock here?
	fields = newFields
	return nil
}

// IsAllowed returns true if the given IP matches an IP or CIDR in
// the whitelist, or if the spf-domain is an empty string explicitly
func IsAllowed(remoteIP net.IP) (bool, error) {
	if !initialized {
		panic(fmt.Errorf("was not initialized"))
	}
	if 0 == len(fields) {
		return true, nil
	}
	if nil == remoteIP {
		return false, nil
	}

	for _, section := range fields {
		parts := strings.Split(section, ":")
		if 2 != len(parts) || !strings.HasPrefix(parts[0], "ip") {
			// ignore unsupported bits
			// (i.e. +mx +ip include:xxx)
			continue
		}
		ip := parts[1]

		if strings.Contains(ip, "/") {
			_, ipNet, err := net.ParseCIDR(ip)
			if nil != err {
				return false, fmt.Errorf("invalid CIDR %q", ip)
			}
			return ipNet.Contains(remoteIP), nil
		}

		ipAddr := net.ParseIP(ip)
		if nil == ipAddr {
			return false, fmt.Errorf(
				"IP %q from SPF record could not be parsed",
				ipAddr.String(),
			)
		}
		if remoteIP.Equal(ipAddr) {
			return true, nil
		}
	}
	return false, nil
}

func parseSpf(spf1 string) ([]string, error) {
	fields := strings.Fields(spf1)
	if len(fields) < 1 ||
		len(fields[0]) < 1 ||
		!strings.HasPrefix(fields[0], "v=") {
		return nil, errors.New("missing v=")
	}
	fields = fields[1:]

	return fields, nil
}
