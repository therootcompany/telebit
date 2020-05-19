package packer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	HeaderLengthState State = 1 + iota
	HeaderState
	PayloadState
)

const (
	FamilyIndex int = iota
	AddressIndex
	PortIndex
	LengthIndex
	ServiceIndex
)

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

	addr := Addr{
		family: parts[FamilyIndex],
		addr:   parts[AddressIndex],
		port:   port,
		scheme: Scheme(service),
	}
	p.state.addr = addr
	/*
		p.state.conn = p.conns[addr.Network()]
		if nil == p.state.conn {
			rconn, wconn := net.Pipe()
			conn := Conn{
				updated:         time.Now(),
				relayRemoteAddr: addr,
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

			addr := &p.state.conn.relayRemoteAddr
			if "end" == string(addr.scheme) {
				if err := p.state.conn.Close(); nil != err {
					// TODO log potential error?
				}
			}
			return b, nil
		*/

		//fmt.Printf("[debug] [2] payload written: %d | payload length: %d\n", p.state.payloadWritten, p.state.payloadLen)
		p.handler.WriteMessage(p.state.addr, []byte{})
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
	p.handler.WriteMessage(p.state.addr, c)
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
