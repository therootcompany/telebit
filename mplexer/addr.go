package telebit

import "fmt"

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
	family      string // TODO what should be the format? "tcpv6"?
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
	//return a.addr + ":" + strconv.Itoa(a.port)
	return fmt.Sprintf("%s+%s:%s:%d", a.family, a.Scheme(), a.addr, a.port)
}

// Network s typically network "family", such as "tcp" or "ip",
// but in this case will be "tun", which is a cue to do a `switch`
// to actually use the specific features of a telebit.Addr
func (a *Addr) Network() string {
	return "tun"
}

func (a *Addr) Port() int {
	return a.port
}

func (a *Addr) Hostname() string {
	return a.addr
}

func (a *Addr) Scheme() Scheme {
	return a.scheme
}
