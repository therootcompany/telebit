package genericlistener

import (
	"bufio"
	"net"
)

//WedgeConn -- A buffered IO infront of a connection allowing peeking, and switching connections.
type WedgeConn struct {
	reader *bufio.Reader
	net.Conn
}

//NewWedgeConn -- Constructor
func NewWedgeConn(c net.Conn) (p *WedgeConn) {
	p = new(WedgeConn)
	p.reader = bufio.NewReader(c)
	p.Conn = c
	return
}

//NewWedgeConnSize -- Constructor
func NewWedgeConnSize(c net.Conn, size int) (p *WedgeConn) {
	p = new(WedgeConn)
	p.reader = bufio.NewReaderSize(c, size)
	p.Conn = c
	return
}

//Peek - Get a number of bytes outof the buffer, but allow the buffer to be replayed once read
func (w *WedgeConn) Peek(n int) ([]byte, error) {
	return w.reader.Peek(n)
}

//Read -- A normal reader.
func (w *WedgeConn) Read(p []byte) (int, error) {
	cnt, err := w.reader.Read(p)
	return cnt, err
}

//Buffered --
func (w *WedgeConn) Buffered() int {
	return w.reader.Buffered()
}

//PeekAll --
// - get all the chars available
// - pass then back
func (w *WedgeConn) PeekAll() (buf []byte, err error) {

	_, err = w.Peek(1)
	if err != nil {
		return nil, err
	}

	buf, err = w.Peek(w.Buffered())
	return
}
