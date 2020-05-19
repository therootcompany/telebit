package packer

import (
	"context"
	"errors"
)

type Parser struct {
	ctx        context.Context
	handler    Handler
	newConns   chan *Conn
	conns      map[string]*Conn
	state      ParserState
	parseState State
	dataReady  chan struct{}
	data       []byte
	consumed   int
}

type ParserState struct {
	consumed       int
	version        byte
	headerLen      int
	header         []byte
	payloadLen     int
	addr           Addr
	payloadWritten int
}

type State int

const (
	V1 byte = 255 - (1 + iota)
	V2
)

const (
	VersionState State = 0
)

func NewParser(ctx context.Context, handler Handler) *Parser {
	return &Parser{
		ctx:       ctx,
		conns:     make(map[string]*Conn),
		newConns:  make(chan *Conn, 2), // Buffered to make testing easier
		dataReady: make(chan struct{}, 2),
		data:      []byte{},
		handler:   handler,
	}
}

type Handler interface {
	WriteMessage(Addr, []byte)
}

// Write receives tunnel data and creates or writes to connections
func (p *Parser) Write(b []byte) (int, error) {
	if len(b) < 1 {
		return 0, errors.New("developer error: wrote 0 bytes")
	}

	/*
		// so that we can overwrite the main state
		// as soon as a full message has completed
		// but still keep the number of bytes written
		if 0 == p.state.written {
			p.written = 0
		}
	*/

	switch p.parseState {
	case VersionState:
		//fmt.Println("[debug] version state", b[0])
		p.state.version = b[0]
		b = b[1:]
		p.consumed += 1
		p.parseState += 1
	default:
		// do nothing
	}

	switch p.state.version {
	case V1:
		//fmt.Println("[debug] v1 unmarshal")
		return p.unpackV1(b)
	default:
		return 0, errors.New("incorrect version or version not implemented")
	}
}
