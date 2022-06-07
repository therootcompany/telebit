package mgmt

import (
	"git.rootprojects.org/root/telebit/internal/mgmt/authstore"
)

var store authstore.Store

// DeviceDomain is the base hostname used for devices, such as devices.example.com
// which has devices as foo.devices.example.com
var DeviceDomain string

// RelayDomain is the API hostname used for the tunnel
// ( currently NOT used, but will be used for wss://RELAY_DOMAIN/ )
var RelayDomain string

// Init initializes some package variables
func Init(s authstore.Store) {
	store = s
}
