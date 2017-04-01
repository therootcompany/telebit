package packer

import (
	"bytes"
)

//packerData -- Contains packer data
type packerData struct {
	buffer bytes.Buffer
}

func newPackerData() *packerData {
	return new(packerData)
}

func (p *packerData) AppendString(dataString string) (int, error) {
	return p.buffer.WriteString(dataString)
}

func (p *packerData) AppendBytes(dataBytes []byte) (int, error) {
	return p.buffer.Write(dataBytes)
}

//Data --
func (p *packerData) Data() []byte {
	return p.buffer.Bytes()
}

func (p *packerData) DataLen() int {
	return p.buffer.Len()
}
