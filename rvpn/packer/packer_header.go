package packer

import "net"
import "fmt"

type addressFamily int

// packerHeader structure to hold our header information.
type packerHeader struct {
	family    addressFamily
	address   net.IP
	Port      int
	Service   string
	HeaderLen byte
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
	p.HeaderLen = 0
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

func (p *packerHeader) AddressBytes() (b []byte) {
	b = make([]byte, 16)

	switch {
	case p.address.To4() != nil:
		b = make([]byte, 4)
		for pos := range b {
			b[pos] = p.address[pos+12]
		}
		return
	}
	return
}

func (p *packerHeader) Address() (address net.IP) {
	address = p.address
	return
}

func (p *packerHeader) Family() (family addressFamily) {
	family = p.family
	return
}

func (p *packerHeader) FamilyText() (familyText string) {
	familyText = addressFamilyText[p.family]
	return
}
