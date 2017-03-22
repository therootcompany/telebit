package genericlistener

import (
	"context"
	"fmt"
	"net"
	"sync"
)

//Track -- used to track connection + domain
type Track struct {
	conn   net.Conn
	domain string
}

//NewTrack -- Constructor
func NewTrack(conn net.Conn, domain string) (p *Track) {
	p = new(Track)
	p.conn = conn
	p.domain = domain
	return
}

//Tracking --
type Tracking struct {
	mutex       *sync.Mutex
	connections map[string]*Track
	register    chan *Track
	unregister  chan net.Conn
}

//NewTracking -- Constructor
func NewTracking() (p *Tracking) {
	p = new(Tracking)
	p.mutex = &sync.Mutex{}
	p.connections = make(map[string]*Track)
	p.register = make(chan *Track)
	p.unregister = make(chan net.Conn)
	return
}

//Run -
func (p *Tracking) Run(ctx context.Context) {
	loginfo.Println("Tracking Running")

	for {
		select {

		case <-ctx.Done():
			loginfo.Println("Cancel signal hit")
			return

		case connection := <-p.register:
			p.mutex.Lock()
			key := connection.conn.RemoteAddr().String()
			loginfo.Println("register fired", key)
			p.connections[key] = connection
			p.list()
			p.mutex.Unlock()

		case connection := <-p.unregister:
			p.mutex.Lock()
			key := connection.RemoteAddr().String()
			loginfo.Println("unregister fired", key)
			if _, ok := p.connections[key]; ok {
				delete(p.connections, key)
			}
			p.list()
			p.mutex.Unlock()
		}
	}
}

func (p *Tracking) list() {
	for c := range p.connections {
		loginfo.Println(c)
	}
}

//Lookup --
// - get connection from key
func (p *Tracking) Lookup(key string) (*Track, error) {
	defer func() {
		p.mutex.Unlock()
	}()
	p.mutex.Lock()

	if _, ok := p.connections[key]; ok {
		return p.connections[key], nil
	}
	return nil, fmt.Errorf("Lookup failed for %s", key)
}
