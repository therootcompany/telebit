package matching

type headerTerm int
type matchType int
type protocolType int

//headerTerm -- ENUM for header terminator
const (
	CRLF2 headerTerm = iota
	ZER0
)

//Family -- ENUM for Address Family
const (
	BYTES matchType = iota
	REGEX
)

const (
	pHTTP = iota + 1
	pTLS
	pSSH
)

//Protocol --
type Protocol struct {
	HeaderTerm  headerTerm
	MatchType   matchType
	Type        protocolType
	SearchRegex string
	SearchBytes []byte
}

//NewProtocol -- Constructor
func NewProtocol() (p *Protocol) {
	p = new(Protocol)
	return
}

//Protocols --
type Protocols struct {
	protocols []*Protocol
}

func (p *Protocols) add(protocol *Protocol) []*Protocol {
	p.protocols = append(p.protocols, protocol)
	return p.protocols
}

//NewProtocols --
func NewProtocols() (p *Protocols) {
	p = new(Protocols)
	p.protocols = make([]*Protocol, 0)

	newp := NewProtocol()
	newp.MatchType = REGEX
	newp.HeaderTerm = CRLF2
	newp.MatchType = pHTTP
	p.add(newp)

	return
}
