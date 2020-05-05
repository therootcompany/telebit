package packer

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	_             = iota // skip the iota value of 0
	packerV1 byte = 255 - iota
	packerV2
)

//Packer -- contains both header and data
type Packer struct {
	Header
	Data packerData
}

// NewPacker creates a new Packer struct using the information from the provided header as
// its own header. (Because the header is stored directly and not as a pointer/reference
// it should be safe to override items like the service without affecting the template header.)
func NewPacker(header *Header) *Packer {
	p := new(Packer)
	if header != nil {
		p.Header = *header
	}
	return p
}

func splitHeader(header []byte, names []string) (map[string]string, error) {
	parts := strings.Split(string(header), ",")
	if p, n := len(parts), len(names); p > n {
		return nil, fmt.Errorf("Header contains %d extra fields", p-n)
	} else if p < n {
		return nil, fmt.Errorf("Header missing fields %q", names[p:])
	}

	result := make(map[string]string, len(names))
	for ind, key := range names {
		result[key] = parts[ind]
	}
	return result, nil
}

// ReadMessage checks the protocol and switches accordingly
func ReadMessage(b []byte) (*Packer, error) {
	// Detect protocol in use
	if b[0] == packerV1 {
		return ReadV1Message(b)
	}

	return nil, fmt.Errorf("Version %d not supported", 255-b[0])
}

// ReadV1Message parses a v1-formatted message
func ReadV1Message(b []byte) (*Packer, error) {
	// Separate the header and body using the header length in the second byte.
	p := NewPacker(nil)
	header := b[2 : b[1]+2]
	data := b[b[1]+2:]

	// Handle the different parts of the header.
	parts, err := splitHeader(header, []string{"address family", "address", "port", "data length", "service"})
	if err != nil {
		return nil, err
	}

	if familyText := parts["address family"]; familyText == addressFamilyText[FamilyIPv4] {
		p.Header.family = FamilyIPv4
	} else if familyText == addressFamilyText[FamilyIPv6] {
		p.Header.family = FamilyIPv6
	} else {
		return nil, fmt.Errorf("Address family %q not supported", familyText)
	}

	p.Header.address = net.ParseIP(parts["address"])
	if p.Header.address == nil {
		return nil, fmt.Errorf("Invalid network address %q", parts["address"])
	} else if p.Header.family == FamilyIPv4 && p.Header.address.To4() == nil {
		return nil, fmt.Errorf("Address %q is not in address family %s", parts["address"], p.Header.Family())
	}

	//handle port
	if port, err := strconv.Atoi(parts["port"]); err != nil {
		return nil, fmt.Errorf("Error converting port %q: %v", parts["port"], err)
	} else if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("Port %d out of range", port)
	} else {
		p.Header.port = port
	}

	//handle data length
	if dataLen, err := strconv.Atoi(parts["data length"]); err != nil {
		return nil, fmt.Errorf("Error converting data length %q: %v", parts["data length"], err)
	} else if dataLen != len(data) {
		return nil, fmt.Errorf("Data length %d doesn't match received length %d", dataLen, len(data))
	}

	//handle Service
	p.Header.service = parts["service"]

	//handle payload
	p.Data.AppendBytes(data)
	return p, nil
}

//PackV1 -- Outputs version 1 of packer
func (p *Packer) PackV1() bytes.Buffer {
	header := strings.Join([]string{
		p.Header.Family(),
		p.Header.Address(),
		strconv.Itoa(p.Header.Port()),
		strconv.Itoa(p.Data.DataLen()),
		p.Header.Service(),
	}, ",")

	var buf bytes.Buffer
	buf.WriteByte(packerV1)
	buf.WriteByte(byte(len(header)))
	buf.WriteString(header)
	buf.Write(p.Data.Data())

	return buf
}
