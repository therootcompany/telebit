package genericlistener

import (
	"io"
	"net"
)

type oneConnListener struct {
	conn net.Conn
}

func (l *oneConnListener) Accept() (c net.Conn, err error) {
	c = l.conn

	if c == nil {
		err = io.EOF
		loginfo.Println("Accept")
		return
	}
	err = nil
	l.conn = nil
	loginfo.Println("Accept", c.LocalAddr().String(), c.RemoteAddr().String())
	return
}

func (l *oneConnListener) Close() error {
	loginfo.Println("close")
	return nil
}

func (l *oneConnListener) Addr() net.Addr {
	loginfo.Println("addr")
	return nil
}
