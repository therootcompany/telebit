package mplexer

import (
	"fmt"
	"strconv"
)

type Scheme string

const (
	HTTPS   = Scheme("https")
	HTTP    = Scheme("http")
	SSH     = Scheme("ssh")
	OpenVPN = Scheme("openvpn")
)

type Termination string

const (
	TCP = Termination("none")
	TLS = Termination("tls")
)

type Addr struct {
	scheme      Scheme
	termination Termination
	addr        string
	port        int
}

func NewAddr(s Scheme, t Termination, a string, p int) *Addr {
	return &Addr{
		scheme:      s,
		termination: t,
		addr:        a,
		port:        p,
	}
}

func (a *Addr) String() string {
	return fmt.Sprintf("%s:%s:%s:%d", a.Network(), a.Scheme(), a.addr, a.port)
}

func (a *Addr) Network() string {
	return a.addr + ":" + strconv.Itoa(a.port)
}

func (a *Addr) Scheme() Scheme {
	return a.scheme
}
