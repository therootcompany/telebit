package packer

import (
	"fmt"
	"net"
)

type addressFamily int

// packerHeader structure to hold our header information.
type packerHeader struct {
	family  addressFamily
	address net.IP
	Port    int
	Service string
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

func newPackerHeader() (p *packerHeader) {
	p = new(packerHeader)
	p.SetAddress("127.0.0.1")
	p.Port = 65535
	p.Service = "na"
	return
}

//SetAddress -- Set Address. which sets address family automatically
func (p *packerHeader) SetAddress(addr string) {
	p.address = net.ParseIP(addr)

	if p.address.To4() != nil {
		p.family = FamilyIPv4
	} else if p.address.To16() != nil {
		p.family = FamilyIPv6
	} else {
		panic(fmt.Sprintf("setAddress does not support %q", addr))
	}
}

func (p *packerHeader) AddressBytes() []byte {
	if ip4 := p.address.To4(); ip4 != nil {
		p.address = ip4
	}

	return []byte(p.address)
}

func (p *packerHeader) AddressString() string {
	return p.address.String()
}

func (p *packerHeader) Address() net.IP {
	return p.address
}

func (p *packerHeader) Family() addressFamily {
	return p.family
}

func (p *packerHeader) FamilyText() string {
	return addressFamilyText[p.family]
}
