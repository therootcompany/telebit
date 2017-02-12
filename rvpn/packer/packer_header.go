package packer

import "net"
import "fmt"

// packerHeader structure to hold our header information.
type packerHeader struct {
	family  addressFamily
	address net.IP
	Port    int
	Service string
}

type addressFamily int
type addressFamilyString string

//Family -- ENUM for Address Family
const (
	FamilyIPv4 addressFamily = iota
	FamilyIPv6
)

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
	err := p.address.To4()

	if err != nil {
		p.family = FamilyIPv4
	} else {
		err := p.address.To16()
		if err != nil {
			p.family = FamilyIPv6
		} else {
			panic(fmt.Sprintf("setAddress does not support %s", addr))
		}
	}
}

func (p *packerHeader) Address() (address net.IP) {
	address = p.address
	return
}

func (p *packerHeader) Family() (family addressFamily) {
	family = p.family
	return
}
