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

//Discard - discard a number of bytes, perhaps after peeking at the
func (w *WedgeConn) Discard(n int) (int, error) {
	return w.reader.Discard(n)
}

//Peek - Get a number of bytes outof the buffer, but allow the buffer to be replayed once read
func (w *WedgeConn) Peek(n int) ([]byte, error) {
	return w.reader.Peek(n)
}

//ReadByte -- A normal reader.
func (w *WedgeConn) ReadByte() (byte, error) {
	return w.reader.ReadByte()
}

//Read -- A normal reader.
func (w *WedgeConn) Read(p []byte) (int, error) {
	return w.reader.Read(p)
}

//Buffered --
func (w *WedgeConn) Buffered() int {
	return w.reader.Buffered()
}

//PeekAll --
// - get all the chars available
// - pass then back
func (w *WedgeConn) PeekAll() ([]byte, error) {
	// We first peek with 1 so that if there is no buffered data the reader will
	// fill the buffer before we read how much data is buffered.
	if _, err := w.Peek(1); err != nil {
		return nil, err
	}

	return w.Peek(w.Buffered())
}
