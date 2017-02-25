package genericlistener

import (
	"encoding/hex"
	"io"
	"net"
	"sync"
)

//WedgeListener -- used to hand off connections to other protocols via Listen
type WedgeListener struct {
	conn net.Conn
	once sync.Once
}

//Accept --
func (s *WedgeListener) Accept() (net.Conn, error) {
	var c net.Conn

	loginfo.Println("Accept")

	if 1 == 2 {

		var buffer [512]byte
		cnt, err := s.conn.Read(buffer[0:])
		if err != nil {
			loginfo.Println("Errpr radomg")
		}
		loginfo.Println("buffer")
		loginfo.Println(hex.Dump(buffer[0:cnt]))
	}

	s.once.Do(func() {
		loginfo.Println("Do Once")
		c = s.conn
	})

	if c != nil {
		loginfo.Println("accepted")
		return c, nil
	}
	return nil, io.EOF
}

//Close --
func (s *WedgeListener) Close() error {
	s.once.Do(func() {
		loginfo.Println("close called")
		s.conn.Close()
	})
	return nil
}

//Addr --
func (s *WedgeListener) Addr() net.Addr {
	loginfo.Println("Add Called", s.conn.LocalAddr())
	return s.conn.LocalAddr()
}
