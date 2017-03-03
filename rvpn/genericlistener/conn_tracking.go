package genericlistener

import "net"
import "context"
import "fmt"

//Tracking --
type Tracking struct {
	connections map[string]net.Conn
	register    chan net.Conn
	unregister  chan net.Conn
}

//NewTracking -- Constructor
func NewTracking() (p *Tracking) {
	p = new(Tracking)
	p.connections = make(map[string]net.Conn)
	p.register = make(chan net.Conn)
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
			key := connection.RemoteAddr().String()
			loginfo.Println("register fired", key)
			p.connections[key] = connection
			p.list()

		case connection := <-p.unregister:
			key := connection.RemoteAddr().String()
			loginfo.Println("unregister fired", key)
			p.connections[key] = connection
			if _, ok := p.connections[key]; ok {
				delete(p.connections, key)
			}
			p.list()
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
func (p *Tracking) Lookup(key string) (c net.Conn, err error) {
	if _, ok := p.connections[key]; ok {
		c = p.connections[key]
	} else {
		err = fmt.Errorf("Lookup failed for %s", key)
		c = nil
	}
	return
}
