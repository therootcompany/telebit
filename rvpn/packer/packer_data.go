package packer

import (
	"bytes"
)

//packerData -- Contains packer data
type packerData struct {
	buffer  *bytes.Buffer
	DataLen int
}

func newPackerData() (p *packerData) {
	p = new(packerData)
	p.buffer = new(bytes.Buffer)
	return
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
