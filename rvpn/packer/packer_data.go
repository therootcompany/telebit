package packer

import "bytes"

//packerData -- Contains packer data
type packerData struct {
	Buffer *bytes.Buffer
}

func newPackerData() (p *packerData) {
	p = new(packerData)
	p.Buffer = new(bytes.Buffer)
	return
}
