package packer

import "bytes"

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

func (p *packerData) AppendString(dataString string) (n int, err error) {
	n, err = p.buffer.WriteString(dataString)
	return
}

func (p *packerData) AppendBytes(dataBytes []byte) (n int, err error) {
	n, err = p.buffer.Write(dataBytes)
	return
}

//Data --
func (p *packerData) Data() (b []byte) {
	b = p.buffer.Bytes()
	return
}
