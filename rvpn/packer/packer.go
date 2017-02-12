package packer

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
