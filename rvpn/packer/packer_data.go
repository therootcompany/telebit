package packer

import "bytes"

//packerData -- Contains packer data
type packerData struct {
	buffer *bytes.Buffer
}

func newPackerData() (p *packerData) {
	p = new(packerData)
	p.buffer = new(bytes.Buffer)
	return
}

func (p packerData) AppendString(dataString string) (n int, err error) {
	n, err = p.buffer.WriteString(dataString)
	return
}
