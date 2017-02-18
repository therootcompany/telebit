package packer

import (
	"bytes"
	"encoding/hex"
	"fmt"
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

//PackV1 -- Outputs version 1 of packer
func (p *Packer) PackV1() (b bytes.Buffer) {
	version := byte(1)

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
	metaBuf.WriteByte(byte(255) - version)
	metaBuf.WriteByte(byte(headerBuf.Len()))

	var buf bytes.Buffer
	buf.Write(metaBuf.Bytes())
	buf.Write(headerBuf.Bytes())
	buf.Write(p.Data.buffer.Bytes())

	fmt.Println("header: ", headerBuf.String())
	fmt.Println("meta: ", metaBuf)
	fmt.Println("Data: ", p.Data.buffer)
	fmt.Println("Buffer: ", buf.Bytes())
	fmt.Println("Buffer: ", hex.Dump(buf.Bytes()))
	fmt.Printf("Buffer %s", buf.Bytes())

	b = buf

	return
}
