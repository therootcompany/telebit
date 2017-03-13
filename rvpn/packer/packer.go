package packer

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
)

const (
	packerV1 byte = 255 - 1
	packerV2 byte = 255 - 2
)

//Packer -- contains both header and data
type Packer struct {
	Header *packerHeader
	Data   *packerData
}

//NewPacker -- Structre
func NewPacker() (p *Packer) {
	p = new(Packer)
	p.Header = newPackerHeader()
	p.Data = newPackerData()
	return
}

//ReadMessage -
func ReadMessage(b []byte) (p *Packer, err error) {
	fmt.Println("ReadMessage")
	var pos int

	err = nil
	// detect protocol in use
	if b[0] == packerV1 {
		p = NewPacker()

		// Handle Header Length
		pos = pos + 1
		p.Header.HeaderLen = b[pos]

		//handle address family
		pos = pos + 1
		end := bytes.IndexAny(b[pos:], ",")
		if end == -1 {
			err = fmt.Errorf("missing , while parsing address family")
			return nil, err
		}

		bAddrFamily := b[pos : pos+end]
		if bytes.ContainsAny(bAddrFamily, addressFamilyText[FamilyIPv4]) {
			p.Header.family = FamilyIPv4
		} else if bytes.ContainsAny(bAddrFamily, addressFamilyText[FamilyIPv6]) {
			p.Header.family = FamilyIPv6
		} else {
			err = fmt.Errorf("Address family not supported %d", bAddrFamily)
		}

		//handle address
		pos = pos + end + 1
		end = bytes.IndexAny(b[pos:], ",")
		if end == -1 {
			err = fmt.Errorf("missing , while parsing address")
			return nil, err
		}
		p.Header.address = net.ParseIP(string(b[pos : pos+end]))

		//handle import
		pos = pos + end + 1
		end = bytes.IndexAny(b[pos:], ",")
		if end == -1 {
			err = fmt.Errorf("missing , while parsing address")
			return nil, err
		}

		p.Header.Port, err = strconv.Atoi(string(b[pos : pos+end]))
		if err != nil {
			err = fmt.Errorf("error converting port %s", err)
		}

		//handle data length
		pos = pos + end + 1
		end = bytes.IndexAny(b[pos:], ",")
		if end == -1 {
			err = fmt.Errorf("missing , while parsing address")
			return nil, err
		}

		p.Data.DataLen, err = strconv.Atoi(string(b[pos : pos+end]))
		if err != nil {
			err = fmt.Errorf("error converting data length %s", err)
		}

		//handle Service
		pos = pos + end + 1
		end = pos + int(p.Header.HeaderLen)
		p.Header.Service = string(b[pos : p.Header.HeaderLen+2])

		//handle payload
		pos = int(p.Header.HeaderLen + 2)
		p.Data.AppendBytes(b[pos:])

	} else {
		err = fmt.Errorf("Version %d not supported", b[0:0])
	}

	return

}

//PackV1 -- Outputs version 1 of packer
func (p *Packer) PackV1() (b bytes.Buffer) {
	version := packerV1

	var headerBuf bytes.Buffer
	headerBuf.WriteString(p.Header.FamilyText())
	headerBuf.WriteString(",")
	headerBuf.Write([]byte(p.Header.Address().String()))
	headerBuf.WriteString(",")
	headerBuf.WriteString(fmt.Sprintf("%d", p.Header.Port))
	headerBuf.WriteString(",")
	headerBuf.WriteString(fmt.Sprintf("%d", p.Data.buffer.Len()))
	headerBuf.WriteString(",")
	headerBuf.WriteString(p.Header.Service)

	var metaBuf bytes.Buffer
	metaBuf.WriteByte(version)
	metaBuf.WriteByte(byte(headerBuf.Len()))

	var buf bytes.Buffer
	buf.Write(metaBuf.Bytes())
	buf.Write(headerBuf.Bytes())
	buf.Write(p.Data.buffer.Bytes())

	//fmt.Println("header: ", headerBuf.String())
	//fmt.Println("meta: ", metaBuf)
	//fmt.Println("Data: ", p.Data.buffer)
	//fmt.Println("Buffer: ", buf.Bytes())
	//fmt.Println("Buffer: ", hex.Dump(buf.Bytes()))
	//fmt.Printf("Buffer %s", buf.Bytes())

	b = buf

	return
}
