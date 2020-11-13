package mgmt

import (
	"git.rootprojects.org/root/telebit/internal/mgmt/authstore"

	"github.com/go-acme/lego/v3/challenge"
)

var store authstore.Store

var provider challenge.Provider = nil

// DeviceDomain is the base hostname used for devices, such as devices.example.com
// which has devices as foo.devices.example.com
var DeviceDomain string

// RelayDomain is the API hostname used for the tunnel
// ( currently NOT used, but will be used for wss://RELAY_DOMAIN/ )
var RelayDomain string

// MWKey is a type guard
type MWKey string

// Init initializes some package variables
func Init(s authstore.Store, p challenge.Provider) {
	store = s
	provider = p
}
