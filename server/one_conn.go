package server

import (
	"io"
	"net"
)

type oneConnListener struct {
	conn net.Conn
}

func (l *oneConnListener) Accept() (net.Conn, error) {
	if l.conn == nil {
		loginfo.Println("oneConnListener Accept EOF")
		return nil, io.EOF
	}

	c := l.conn
	l.conn = nil
	loginfo.Println("Accept", c.LocalAddr().String(), c.RemoteAddr().String())
	return c, nil
}

func (l *oneConnListener) Close() error {
	loginfo.Println("oneConnListener close")
	return nil
}

func (l *oneConnListener) Addr() net.Addr {
	loginfo.Println("oneConnLister addr")
	return nil
}
