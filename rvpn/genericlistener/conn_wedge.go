package genericlistener

import (
	"bufio"
	"encoding/hex"
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

//Peek - Get a number of bytes outof the buffer, but allow the buffer to be repled once read
func (w *WedgeConn) Peek(n int) ([]byte, error) {
	return w.reader.Peek(n)
}

//Read -- A normal reader.
func (w *WedgeConn) Read(p []byte) (int, error) {
	loginfo.Println("read", w.Conn)
	cnt, err := w.reader.Read(p)
	loginfo.Println("read", hex.Dump(p[0:cnt]))
	loginfo.Println(cnt, err)
	return cnt, err
}
