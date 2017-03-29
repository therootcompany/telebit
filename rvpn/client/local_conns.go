package client

import (
	"fmt"
	"net"
	"sync"

	"github.com/gorilla/websocket"

	"io"

	"git.daplie.com/Daplie/go-rvpn-server/rvpn/packer"
)

type localConns struct {
	lock     sync.RWMutex
	locals   map[string]net.Conn
	services map[string]int
	remote   *websocket.Conn
}

func newLocalConns(remote *websocket.Conn, services map[string]int) *localConns {
	l := new(localConns)
	l.services = services
	l.remote = remote
	l.locals = make(map[string]net.Conn)
	return l
}

func (l *localConns) Write(p *packer.Packer) error {
	l.lock.RLock()
	defer l.lock.RUnlock()

	key := fmt.Sprintf("%s:%d", p.Header.Address(), p.Header.Port)
	if conn := l.locals[key]; conn != nil {
		_, err := conn.Write(p.Data.Data())
		return err
	}

	go l.startConnection(p)
	return nil
}

func (l *localConns) startConnection(orig *packer.Packer) {
	key := fmt.Sprintf("%s:%d", orig.Header.Address(), orig.Header.Port)
	addr := fmt.Sprintf("127.0.0.1:%d", l.services[orig.Header.Service])
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		loginfo.Println("failed to open connection to", addr, err)
		return
	}
	loginfo.Println("opened connection to", addr, "with key", key)
	defer loginfo.Println("finished connection to", addr, "with key", key)

	conn.Write(orig.Data.Data())

	l.lock.Lock()
	l.locals[key] = conn
	l.lock.Unlock()
	defer func() {
		l.lock.Lock()
		delete(l.locals, key)
		l.lock.Unlock()
		conn.Close()
	}()

	buf := make([]byte, 4096)
	for {
		size, err := conn.Read(buf)
		if err != nil {
			if err != io.EOF {
				loginfo.Println("failed to read from local connection to", addr, err)
			}
			return
		}

		p := packer.NewPacker()
		p.Header = orig.Header
		p.Data.AppendBytes(buf[:size])
		packed := p.PackV1()
		l.remote.WriteMessage(websocket.BinaryMessage, packed.Bytes())
	}
}
