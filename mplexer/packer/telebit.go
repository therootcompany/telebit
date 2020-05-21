package packer

import "errors"

// Note: 64k is the TCP max, but 1460b is the 100mbit Ethernet max (1500 MTU - overhead),
// but 1Gbit Ethernet (Jumbo frame) has an 9000b MTU
// Nerds posting benchmarks on SO show that 8k seems about right,
// but even 1024b could work well.
var defaultBufferSize = 8192

// ErrBadGateway means that the target did not accept the connection
var ErrBadGateway = errors.New("EBADGATEWAY")
