package packer

import (
	"fmt"
	"net"
)

type addressFamily int

// The Header struct holds most of the information contained in the header for packets
// between the client and the server (the length of the data is not included here). It
// is used to uniquely identify remote connections on the servers end and to communicate
// which service the remote client is trying to connect to.
type Header struct {
	family  addressFamily
	address net.IP
	port    int
	service string
}

//Family -- ENUM for Address Family
const (
	FamilyIPv4 addressFamily = iota
	FamilyIPv6
)

var addressFamilyText = [...]string{
	"IPv4",
	"IPv6",
}

// NewHeader create a new Header object.
func NewHeader(address string, port int, service string) (*Header, error) {
	h := new(Header)
	if err := h.setAddress(address); err != nil {
		return nil, err
	}
	h.port = port
	h.service = service
	return h, nil
}

// setAddress parses the provided address string and automatically sets the IP family.
func (p *Header) setAddress(addr string) error {
	p.address = net.ParseIP(addr)

	if p.address.To4() != nil {
		p.family = FamilyIPv4
	} else if p.address.To16() != nil {
		p.family = FamilyIPv6
	} else {
		return fmt.Errorf("invalid IP address %q", addr)
	}
	return nil
}

// Family returns the string corresponding to the address's IP family.
func (p *Header) Family() string {
	return addressFamilyText[p.family]
}

// Address returns the string form of the header's remote address.
func (p *Header) Address() string {
	return p.address.String()
}

// Port returns the connected port of the remote connection.
func (p *Header) Port() int {
	return p.port
}

// SetService overrides the header's original service. This is primarily useful
// for sending 'error' and 'end' messages.
func (p *Header) SetService(service string) {
	p.service = service
}

// Service returns the service stored in the header.
func (p *Header) Service() string {
	return p.service
}
