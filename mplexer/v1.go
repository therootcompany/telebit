package telebit

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	// HeaderLengthState is the 2nd (1) state
	HeaderLengthState State = 1 + iota
	// HeaderState is the 3rd (2) state
	HeaderState
	// PayloadState is the 4th (3) state
	PayloadState
)

const (
	// FamilyIndex is the 1st (0) address element, either IPv4 or IPv6
	FamilyIndex int = iota
	// AddressIndex is the 2nd (1) address element, the IP or Hostname
	AddressIndex
	// PortIndex is the 3rd (2) address element, the Port
	PortIndex
	// LengthIndex is the 4th (3) address element, the Payload size
	LengthIndex
	// ServiceIndex is the 5th (4) address element, the Scheme or Control message type
	ServiceIndex
	// RelayPortIndex is the 6th (5) address element, the port on which the connection was established
	RelayPortIndex
	// ServernameIndex is the 7th (6) address element, the SNI Servername or Hostname
	ServernameIndex
)

// Header is the MPLEXY address/control meta data that comes before a packet
type Header struct {
	Family  string
	Address string
	Port    string
	Service string
}

func (p *Parser) unpackV1(b []byte) (int, error) {
	z := 0
	for {
		if z > 20 {
			panic("stuck in an infinite loop?")
		}
		z++
		n := len(b)
		if n < 1 {
			//fmt.Println("[debug] v1 end", z, n)
			break
		}

		var err error
		switch p.parseState {
		case VersionState:
			//fmt.Println("[debug] version state", b[0])
			p.state.version = b[0]
			b = b[1:]
			p.consumed++
			p.parseState++
		case HeaderLengthState:
			//fmt.Println("[debug] v1 h len")
			b = p.unpackV1HeaderLength(b)
		case HeaderState:
			//fmt.Println("[debug] v1 header")
			b, err = p.unpackV1Header(b, n)
			if nil != err {
				//fmt.Println("[debug] v1 header err", err)
				consumed := p.consumed
				p.consumed = 0
				return consumed, err
			}
		case PayloadState:
			//fmt.Println("[debug] v1 payload")
			// if this payload is complete, reset all state
			if p.state.payloadWritten == p.state.payloadLen {
				p.state = ParserState{}
				p.parseState = 0
			}
			b, err = p.unpackV1Payload(b, n)
			if nil != err {
				consumed := p.consumed
				p.consumed = 0
				return consumed, err
			}
		default:
			fmt.Println("[debug] v1 unknown state")
			// do nothing
			consumed := p.consumed
			p.consumed = 0
			return consumed, errors.New("error unpacking")
		}
	}

	consumed := p.consumed
	p.consumed = 0
	return consumed, nil
}

func (p *Parser) unpackV1HeaderLength(b []byte) []byte {
	p.state.headerLen = int(b[0])
	//fmt.Println("[debug] unpacked header len", p.state.headerLen)
	b = b[1:]
	p.consumed++
	p.parseState++
	return b
}

func (p *Parser) unpackV1Header(b []byte, n int) ([]byte, error) {
	//fmt.Println("[debug] got", len(b), "bytes", string(b))
	m := len(p.state.header)
	k := p.state.headerLen - m
	if n < k {
		k = n
	}
	p.consumed += k
	c := b[0:k]
	b = b[k:]
	//fmt.Println("[debug] has", m, "want", k, "more and have", len(b), "more")
	p.state.header = append(p.state.header, c...)
	if p.state.headerLen != len(p.state.header) {
		return b, nil
	}
	parts := strings.Split(string(p.state.header), ",")
	p.state.header = nil
	if len(parts) < 5 {
		return nil, errors.New("error unpacking header")
	}

	payloadLenStr := parts[LengthIndex]
	payloadLen, err := strconv.Atoi(payloadLenStr)
	if nil != err {
		return nil, errors.New("error unpacking header payload length")
	}
	p.state.payloadLen = payloadLen
	port, _ := strconv.Atoi(parts[PortIndex])
	service := parts[ServiceIndex]

	if "control" == service {
		return nil, errors.New("'control' messages not implemented")
	}

	src := Addr{
		family: parts[FamilyIndex],
		addr:   parts[AddressIndex],
		port:   port,
		//scheme: Scheme(service),
	}
	dst := Addr{
		scheme: Scheme(service),
	}
	if len(parts) > RelayPortIndex {
		port, _ := strconv.Atoi(parts[RelayPortIndex])
		dst.port = port
	}
	if len(parts) > ServernameIndex {
		dst.addr = parts[ServernameIndex]
	}
	p.state.srcAddr = src
	p.state.dstAddr = dst
	/*
		p.state.conn = p.conns[addr.Network()]
		if nil == p.state.conn {
			rconn, wconn := net.Pipe()
			conn := Conn{
				updated:         time.Now(),
				relayTargetAddr: addr,
				relay:           rconn,
				local:           wconn,
			}
			copied := conn
			p.state.conn = &copied
			p.conns[addr.Network()] = p.state.conn
			p.newConns <- p.state.conn
		}
	*/
	p.parseState++

	return b, nil
}

func (p *Parser) unpackV1Payload(b []byte, n int) ([]byte, error) {
	// Handle "connect" and "end"
	if 0 == p.state.payloadLen {
		/*
			p.newMsg <- msg{
				addr:  Addr,
				bytes: []byte{},
			}

			addr := &p.state.conn.relayTargetAddr
			if "end" == string(addr.scheme) {
				if err := p.state.conn.Close(); nil != err {
					// TODO log potential error?
				}
			}
			return b, nil
		*/

		//fmt.Printf("[debug] [2] payload written: %d | payload length: %d\n", p.state.payloadWritten, p.state.payloadLen)
		p.handler.RouteBytes(p.state.srcAddr, p.state.dstAddr, []byte{})
		return b, nil
	}

	k := p.state.payloadLen - p.state.payloadWritten
	if n < k {
		k = n
	}
	c := b[0:k]
	b = b[k:]
	// TODO don't let a write on one connection block others,
	// and also put backpressure on just that connection
	/*
		m, err := p.state.conn.local.Write(c)
		p.state.payloadWritten += m
		if nil != err {
			// TODO we want to surface this error somewhere, but not to the websocket
			return b, nil
		}
	*/
	p.handler.RouteBytes(p.state.srcAddr, p.state.dstAddr, c)
	p.consumed += k
	p.state.payloadWritten += k

	//fmt.Printf("[debug] [1] payload written: %d | payload length: %d\n", p.state.payloadWritten, p.state.payloadLen)
	// if this payload is complete, reset all state
	if p.state.payloadWritten == p.state.payloadLen {
		p.state = ParserState{}
		p.parseState = 0
	}
	return b, nil
}
